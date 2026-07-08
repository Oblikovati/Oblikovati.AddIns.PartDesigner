// SPDX-License-Identifier: GPL-2.0-only

// Package build is the generator framework of the Part Designer add-in: the PartGenerator
// seam every procedural part builder implements, a registry keyed by generator kind, and
// the PartBuilder that drives the host to realize a resolved catalogue member as a
// fully-constrained (DOF-0) parametric part — parameters published from the member's
// columns, sketch dimensions bound to those parameters, and features built on top.
//
// A generator assumes an active part document (the placement service creates + activates
// it, then calls Build); it never creates the document itself.
package build

import (
	"encoding/json"
	"fmt"
	"strconv"

	"oblikovati.org/api/client"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/part-designer/designer/catalog"
)

// PartBuilder is the thin, testable seam a generator drives the host through. It bundles the
// MotorDesigner-proven idioms: publish parameters first, then build geometry whose
// dimensions reference those parameters by name (never literal coordinates), so editing a
// parameter re-drives the part.
type PartBuilder struct {
	api  *client.Client
	unit string // length-unit token for expressions ("mm"/"in"), from the family
}

// NewPartBuilder binds a builder to the host client, using the family's unit for the
// length-valued parameter expressions it publishes.
func NewPartBuilder(api *client.Client, unit catalog.Units) *PartBuilder {
	return &PartBuilder{api: api, unit: string(unit)}
}

// API exposes the typed client for geometry a generator builds directly.
func (b *PartBuilder) API() *client.Client { return b.api }

// PublishParams upserts every numeric column of the member as a model parameter (named by
// the column's Param, valued by the cell + the right unit), so the generator's dimensions
// can reference them. It is idempotent — Set when the parameter already exists, else Add —
// so re-running on the same document (Change-Size) re-drives rather than duplicates.
func (b *PartBuilder) PublishParams(rm ResolvedMember) error {
	existing, err := b.existingParams()
	if err != nil {
		return err
	}
	for _, col := range rm.Family.Columns {
		if !col.Type.Numeric() {
			continue // text columns (grade/designation) are labels, not dimensional params
		}
		expr := b.paramExpr(col.Type, rm.Member.Values[col.Name])
		if err := b.upsertParam(existing, col.Param, expr); err != nil {
			return fmt.Errorf("publish param %q: %w", col.Param, err)
		}
	}
	return nil
}

// existingParams returns the set of parameter names already on the active document.
func (b *PartBuilder) existingParams() (map[string]bool, error) {
	list, err := b.api.Parameters().List()
	if err != nil {
		return nil, fmt.Errorf("list parameters: %w", err)
	}
	set := make(map[string]bool, len(list.Parameters))
	for _, p := range list.Parameters {
		set[p.Name] = true
	}
	return set, nil
}

// upsertParam sets the parameter when present, else adds it, and records the new name.
func (b *PartBuilder) upsertParam(existing map[string]bool, name, expr string) error {
	args := wire.ParameterSetArgs{Name: name, Expression: expr}
	if existing[name] {
		_, err := b.api.Parameters().Set(args)
		return err
	}
	_, err := b.api.Parameters().Add(args)
	existing[name] = true
	return err
}

// paramExpr renders a cell value as a unit-bearing expression per its column type.
func (b *PartBuilder) paramExpr(t catalog.ColumnType, v float64) string {
	num := strconv.FormatFloat(v, 'g', -1, 64)
	switch t {
	case catalog.ColumnAngle:
		return num + " deg"
	case catalog.ColumnCount:
		return num
	default: // ColumnLength
		return num + " " + b.unit
	}
}

// DeriveParam upserts a formula parameter — one whose expression is derived from the member's
// published columns rather than a raw cell (a bearing's pitch diameter, ball diameter and race
// diameters computed from bore/outer_dia). Like PublishParams it is idempotent, so re-driving a
// Change-Size recomputes rather than duplicating. The derived parameter shows in the part's list,
// keeping the whole part parametric: editing bore re-drives the balls' pitch circle.
func (b *PartBuilder) DeriveParam(name, expr string) error {
	existing, err := b.existingParams()
	if err != nil {
		return err
	}
	if err := b.upsertParam(existing, name, expr); err != nil {
		return fmt.Errorf("derive param %q = %q: %w", name, expr, err)
	}
	return nil
}

// Sketch creates a 2D sketch on the named plane ("XY"/"XZ"/"YZ") and returns a handle for
// adding constrained, parameter-driven geometry.
func (b *PartBuilder) Sketch(plane string) (*SketchContext, error) {
	res, err := b.api.Sketch().Create(wire.CreateSketchArgs{Plane: plane})
	if err != nil {
		return nil, fmt.Errorf("create sketch on %s: %w", plane, err)
	}
	return &SketchContext{b: b, index: res.SketchIndex}, nil
}

// OffsetPlaneSketch creates a work plane parallel to XY, offset along +Z by offsetExpr (a
// parameter name or formula; a negative expression offsets downward), then a sketch on it. A
// headed screw builds its shank from the head-underside plane so the shank is a fresh solid
// created after the socket is cut — keeping the shank cylinder analytic for the thread. The
// plane is created hidden: it is construction scaffolding, not part of the placed standard part.
func (b *PartBuilder) OffsetPlaneSketch(offsetExpr string) (*SketchContext, error) {
	wp, err := b.api.WorkPlanes().OffsetHidden(types.WorkRefXYPlane, offsetExpr)
	if err != nil {
		return nil, fmt.Errorf("create XY-offset work plane %q: %w", offsetExpr, err)
	}
	res, err := b.api.Sketch().Create(wire.CreateSketchArgs{WorkPlaneIndex: &wp.Index})
	if err != nil {
		return nil, fmt.Errorf("create sketch on offset plane %q (index %d): %w", offsetExpr, wp.Index, err)
	}
	return &SketchContext{b: b, index: res.SketchIndex}, nil
}

// Extrude extrudes the sketch's first profile to a distance expression (a parameter name or
// formula) with the given operation (new|join|cut|intersect), in the default (+Z) direction.
func (b *PartBuilder) Extrude(sk *SketchContext, distanceExpr, operation string) error {
	return b.ExtrudeDirected(sk, distanceExpr, operation, "")
}

// ExtrudeDirected extrudes with an explicit direction ("positive"|"negative"|"symmetric"; ""
// ⇒ positive). A headed fastener grows its head down (−) from the base plane and its shank
// further down, so they meet and join.
func (b *PartBuilder) ExtrudeDirected(sk *SketchContext, distanceExpr, operation, direction string) error {
	_, err := b.ExtrudeNamed(sk, distanceExpr, operation, direction)
	return err
}

// ExtrudeNamed extrudes like ExtrudeDirected but returns the created feature's name, so the solid
// can be patterned — a cylindrical roller extruded symmetric about the mid-plane, then arrayed
// around the pitch circle.
func (b *PartBuilder) ExtrudeNamed(sk *SketchContext, distanceExpr, operation, direction string) (string, error) {
	res, err := client.AddFeature(b.api.Features(), featureargs.Extrude{
		SketchIndex: sk.index, ProfileIndex: 0, Distance: distanceExpr,
		Operation: operation, Direction: direction,
	})
	if err != nil {
		return "", fmt.Errorf("extrude sketch %d by %q (%s %s): %w",
			sk.index, distanceExpr, operation, direction, err)
	}
	return featureName(res), nil
}

// Loft blends the bottom sketch's first profile into the top sketch's first profile as a solid
// (operation new|join|cut). A countersunk screw head is a loft from the head-diameter circle
// (top) to the shank-diameter circle (bottom, on the head-underside plane): the cone angle is
// implied by the two parameter-driven diameters and the plane offset, so it re-drives with the
// size — unlike an extrude taper, whose angle the host cannot express as a formula.
func (b *PartBuilder) Loft(bottom, top *SketchContext, operation string) error {
	_, err := client.AddFeature(b.api.Features(), featureargs.Loft{
		Sections: []featureargs.LoftSection{
			{SketchIndex: bottom.index, ProfileIndex: 0},
			{SketchIndex: top.index, ProfileIndex: 0},
		},
		Operation: operation,
	})
	if err != nil {
		return fmt.Errorf("loft sketch %d→%d (%s): %w", bottom.index, top.index, operation, err)
	}
	return nil
}

// Revolve sweeps the sketch's first profile about axisRef ("origin/axis/z" for a ring, or
// "origin/axis/x" to turn an off-axis half-disk into a ball) by angleExpr — a parameter name or
// formula; an angle just under a full turn leaves the split gap of a retaining ring. It returns the
// created feature's name so it can be patterned (the bearing's ball). It is how a
// turned/rotationally-symmetric part is built from a cross-section, so the part re-drives with its
// parameters.
func (b *PartBuilder) Revolve(sk *SketchContext, axisRef, angleExpr, operation string) (string, error) {
	res, err := client.AddFeature(b.api.Features(), featureargs.Revolve{
		SketchIndex: sk.index, ProfileIndex: 0, AxisRef: axisRef,
		Angle: angleExpr, Operation: operation,
	})
	if err != nil {
		return "", fmt.Errorf("revolve sketch %d about %s by %q (%s): %w", sk.index, axisRef, angleExpr, operation, err)
	}
	return featureName(res), nil
}

// PatternCircular replicates the named source feature countExpr times evenly around the world Z
// axis — how a bearing's single ball is arrayed into its full ball complement.
func (b *PartBuilder) PatternCircular(sourceFeature, countExpr string) error {
	_, err := b.api.Features().PatternCircular(wire.CircularPatternFeatureArgs{
		SourceFeatures: []string{sourceFeature}, CountExpr: countExpr,
		Angle: "360 deg", AxisPoint: []float64{0, 0, 0}, AxisDir: []float64{0, 0, 1},
	})
	if err != nil {
		return fmt.Errorf("circular-pattern %q x%q: %w", sourceFeature, countExpr, err)
	}
	return nil
}

// featureName reads the created feature's tree name from a features.add reply (the "feature"
// field), so a follow-up pattern can reference it.
func featureName(res json.RawMessage) string {
	var out struct {
		Feature string `json:"feature"`
	}
	_ = json.Unmarshal(res, &out)
	return out.Feature
}

// Coil sweeps the sketch's first profile along a helix about the world Z axis — how a helical
// spring washer is modelled. revolutions just under one turn leaves the split gap; the coil's
// height (the total axial rise) offsets the two ends, giving the free height a DIN 127 spring
// lock washer stands at. Height and revolutions are expressions, so the coil re-drives with the
// size (two of pitch/revolutions/height fix the helix).
func (b *PartBuilder) Coil(sk *SketchContext, heightExpr, revolutionsExpr string) error {
	_, err := client.AddFeature(b.api.Features(), featureargs.Coil{
		SketchIndex: sk.index, ProfileIndex: 0, AxisRef: "origin/axis/z",
		Height: heightExpr, Revolutions: revolutionsExpr,
	})
	if err != nil {
		return fmt.Errorf("coil sketch %d (height %q, revolutions %q): %w", sk.index, heightExpr, revolutionsExpr, err)
	}
	return nil
}

// CosmeticThread tags a cylindrical face (by reference key) with a representational, non-cut
// thread of the given designation (e.g. "M8x1.25") — the Content-Center convention for
// standard fasteners, which show thread lines without modelling the helical cut. The thread
// runs the full length of the face.
func (b *PartBuilder) CosmeticThread(faceRef, designation string) error {
	return b.CosmeticThreadSpan(faceRef, designation, "", "")
}

// CosmeticThreadSpan is CosmeticThread limited to an axial window of the face: the thread runs
// for lengthExpr starting offsetExpr up from the face's start edge (both distance expressions;
// empty offset ⇒ 0, empty length ⇒ the full face). A double-ended stud threads its two ends by
// applying two spans to its single cylindrical face — the metal end at offset 0 and the nut end
// at the far end — leaving the plain shank between them bare.
func (b *PartBuilder) CosmeticThreadSpan(faceRef, designation, offsetExpr, lengthExpr string) error {
	_, err := client.AddFeature(b.api.Features(), featureargs.Thread{
		FaceRef: faceRef, Designation: designation, Cut: false,
		Offset: offsetExpr, Length: lengthExpr,
	})
	if err != nil {
		return fmt.Errorf("thread face %q as %q (offset %q, length %q): %w",
			faceRef, designation, offsetExpr, lengthExpr, err)
	}
	return nil
}
