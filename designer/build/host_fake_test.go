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
	existing   []string // parameter names already on the document
	dof        int      // DOF returned by sketch.constraintStatus
	failMethod string   // when non-empty, this wire method returns an error
	noPoints   bool     // when true, sketch.addEntity returns a circle with no centre point

	methods      []string
	added        []wire.ParameterSetArgs
	set          []wire.ParameterSetArgs
	circleRadius string
	constraints  []string // geometric constraint kinds, in order
	dimensions   []wire.AddDimensionArgs
	extrude      featureargs.Extrude
	extrudeKind  string
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
		h.circleRadius = decode[wire.AddSketchEntityArgs](req).Radius
		if h.noPoints {
			return []byte(`{"entityId":10,"pointIds":[]}`), nil
		}
		return []byte(`{"entityId":10,"pointIds":[11,12]}`), nil
	case wire.MethodSketchAddConstraint:
		h.constraints = append(h.constraints, decode[wire.AddConstraintArgs](req).Kind)
	case wire.MethodSketchAddDimension:
		h.dimensions = append(h.dimensions, decode[wire.AddDimensionArgs](req))
	case wire.MethodSketchConstraintStatus:
		return json.Marshal(wire.ConstraintStatusResult{DOF: h.dof, Status: "checked"})
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

// featureReply records the feature kind + decoded extrude args.
func (h *fakeHost) featureReply(req []byte) ([]byte, error) {
	args := decode[wire.AddFeatureArgs](req)
	h.extrudeKind = args.Kind
	if args.Kind == featureargs.KindExtrude {
		_ = json.Unmarshal(args.Args, &h.extrude)
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
