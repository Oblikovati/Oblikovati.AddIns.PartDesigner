// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"encoding/json"
	"errors"

	"oblikovati.org/api/wire"
	"oblikovati.org/api/wire/featureargs"
)

// fakeHost is a HostCaller double for the generator framework: it records the wire calls a
// generator makes and decodes the ones the tests assert on (published parameters, the
// circle's radius expression, the constraint + dimension, the extrude), returning canned
// replies. `existing` seeds parameters.list so the Set-vs-Add upsert path can be exercised;
// `dof` is what constraintStatus reports (0 = fully constrained).
type fakeHost struct {
	existing     []string // parameter names already on the document
	dof          int      // DOF returned by sketch.constraintStatus
	failMethod   string   // when non-empty, this wire method returns an error
	noPoints     bool     // when true, sketch.addEntity returns a circle with no centre point
	noCylinder   bool     // when true, model.referenceKeys reports no cylindrical face
	headCylinder bool     // when true, referenceKeys adds a second (head) cylinder above the shank
	shortPolygon bool     // when true, a polygon add returns too few points (missing centre)

	methods      []string
	added        []wire.ParameterSetArgs
	set          []wire.ParameterSetArgs
	circleRadius string
	constraints  []string // geometric constraint kinds, in order
	dimensions   []wire.AddDimensionArgs
	extrude      featureargs.Extrude        // the last extrude (back-compat with the round-bar test)
	extrudeKind  string                     // the last feature kind
	extrudes     []featureargs.Extrude      // every extrude, in order
	threads      []featureargs.Thread       // every cosmetic/cut thread, in order
	lofts        []featureargs.Loft         // every loft, in order
	coils        []featureargs.Coil         // every coil, in order
	revolves     []featureargs.Revolve      // every revolve, in order
	workPlanes   []wire.CreateWorkPlaneArgs // every work plane created, in order
}

// Call records the method and returns a minimal reply per the wire method.
func (h *fakeHost) Call(method string, req []byte) ([]byte, error) {
	h.methods = append(h.methods, method)
	if method == h.failMethod {
		return nil, errors.New("fake host: forced failure for " + method)
	}
	switch method {
	case wire.MethodParametersList:
		return h.listReply()
	case wire.MethodParametersAdd:
		h.added = append(h.added, decode[wire.ParameterSetArgs](req))
	case wire.MethodParametersSet:
		h.set = append(h.set, decode[wire.ParameterSetArgs](req))
	case wire.MethodSketchCreate:
		return []byte(`{"sketchIndex":1}`), nil
	case wire.MethodSketchAddEntity:
		return h.addEntityReply(req)
	case wire.MethodSketchAddConstraint:
		h.constraints = append(h.constraints, decode[wire.AddConstraintArgs](req).Kind)
	case wire.MethodSketchAddDimension:
		h.dimensions = append(h.dimensions, decode[wire.AddDimensionArgs](req))
	case wire.MethodSketchConstraintStatus:
		return json.Marshal(wire.ConstraintStatusResult{DOF: h.dof, Status: "checked"})
	case wire.MethodWorkPlanesCreate:
		h.workPlanes = append(h.workPlanes, decode[wire.CreateWorkPlaneArgs](req))
		return json.Marshal(wire.CreateWorkPlaneResult{Index: 2, Ref: "wp/2", Name: "Offset", Healthy: true})
	case wire.MethodModelReferenceKeys:
		return h.referenceKeysReply()
	case wire.MethodFeaturesAdd:
		return h.featureReply(req)
	}
	return []byte("{}"), nil
}

// listReply reports the seeded existing parameters for parameters.list.
func (h *fakeHost) listReply() ([]byte, error) {
	res := wire.ListParametersResult{}
	for _, name := range h.existing {
		res.Parameters = append(res.Parameters, wire.ParameterInfo{Name: name})
	}
	return json.Marshal(res)
}

// addEntityReply serves sketch.addEntity: a polygon returns six corner points followed by the
// centre (the shape GroundedHexagon expects), any other kind returns a single centre point (or
// none when noPoints exercises the missing-centre guard). Only circles carry a Radius, so the
// recorded circleRadius is left untouched by a polygon add.
func (h *fakeHost) addEntityReply(req []byte) ([]byte, error) {
	a := decode[wire.AddSketchEntityArgs](req)
	if a.Radius != "" {
		h.circleRadius = a.Radius
	}
	if a.Kind == "polygon" {
		points := []uint64{30, 31, 32, 33, 34, 35, 36} // 6 corners + centre
		if h.shortPolygon {
			points = points[:3] // a malformed reply the GroundedHexagon guard must reject
		}
		return json.Marshal(wire.AddSketchEntityResult{
			EntityID: 20, Kind: "polygon",
			EntityIDs: []uint64{20, 21, 22, 23, 24, 25},
			PointIDs:  points,
		})
	}
	if a.Kind == "rectangle" {
		return json.Marshal(wire.AddSketchEntityResult{
			EntityID: 40, Kind: "rectangle",
			EntityIDs: []uint64{40, 41, 42, 43}, // 4 edges
			PointIDs:  []uint64{44, 45, 46, 47}, // 4 corners: BL, BR, TR, TL
		})
	}
	if a.Kind == "polyline" {
		n := len(a.Points) // one corner + one edge per point in a closed loop
		pts, edges := make([]uint64, n), make([]uint64, n)
		for i := 0; i < n; i++ {
			pts[i], edges[i] = uint64(50+i), uint64(70+i)
		}
		return json.Marshal(wire.AddSketchEntityResult{
			EntityID: 70, Kind: "polyline", EntityIDs: edges, PointIDs: pts,
		})
	}
	if h.noPoints {
		return json.Marshal(wire.AddSketchEntityResult{EntityID: 10})
	}
	return json.Marshal(wire.AddSketchEntityResult{EntityID: 10, PointIDs: []uint64{11, 12}})
}

// referenceKeysReply reports a plane head-face plus the shank's cylindrical face (suppressed by
// noCylinder, to exercise the missing-cylinder guard). Each face carries a representative point;
// the shank sits deepest (z=-20). With headCylinder a second, higher cylinder (the socket cap
// screw's cylindrical head, z=-4) is added so ShankCylinderFaceKey's lowest-face pick is tested.
func (h *fakeHost) referenceKeysReply() ([]byte, error) {
	faces := []wire.TopologyRef{{Key: "head-top", Kind: "plane", Point: []float64{0, 0, 0}}}
	if h.headCylinder {
		faces = append(faces, wire.TopologyRef{Key: "head-cyl", Kind: "cylinder", Point: []float64{0, 0, -4}})
	}
	if !h.noCylinder {
		faces = append(faces, wire.TopologyRef{Key: "shank-cyl", Kind: "cylinder", Point: []float64{0, 0, -20}})
	}
	return json.Marshal(wire.ReferenceKeysResult{Bodies: []wire.BodyTopology{{Faces: faces}}})
}

// featureReply records each feature: extrudes (kept individually and as the last one) and
// threads.
func (h *fakeHost) featureReply(req []byte) ([]byte, error) {
	args := decode[wire.AddFeatureArgs](req)
	h.extrudeKind = args.Kind
	switch args.Kind {
	case featureargs.KindExtrude:
		var ex featureargs.Extrude
		_ = json.Unmarshal(args.Args, &ex)
		h.extrude = ex
		h.extrudes = append(h.extrudes, ex)
	case featureargs.KindThread:
		var t featureargs.Thread
		_ = json.Unmarshal(args.Args, &t)
		h.threads = append(h.threads, t)
	case featureargs.KindLoft:
		var l featureargs.Loft
		_ = json.Unmarshal(args.Args, &l)
		h.lofts = append(h.lofts, l)
	case featureargs.KindCoil:
		var cl featureargs.Coil
		_ = json.Unmarshal(args.Args, &cl)
		h.coils = append(h.coils, cl)
	case featureargs.KindRevolve:
		var rv featureargs.Revolve
		_ = json.Unmarshal(args.Args, &rv)
		h.revolves = append(h.revolves, rv)
	}
	return []byte("{}"), nil
}

// decode unmarshals a recorded request body into T (best-effort; malformed input yields the
// zero value, which the assertions then catch).
func decode[T any](req []byte) T {
	var v T
	_ = json.Unmarshal(req, &v)
	return v
}

// hasMethod reports whether method was called at least once.
func (h *fakeHost) hasMethod(method string) bool {
	for _, m := range h.methods {
		if m == method {
			return true
		}
	}
	return false
}
