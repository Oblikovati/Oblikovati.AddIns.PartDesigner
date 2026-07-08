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
	"fmt"
	"strconv"

	"oblikovati.org/api/client"
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

// Sketch creates a 2D sketch on the named plane ("XY"/"XZ"/"YZ") and returns a handle for
// adding constrained, parameter-driven geometry.
func (b *PartBuilder) Sketch(plane string) (*SketchContext, error) {
	res, err := b.api.Sketch().Create(wire.CreateSketchArgs{Plane: plane})
	if err != nil {
		return nil, fmt.Errorf("create sketch on %s: %w", plane, err)
	}
	return &SketchContext{b: b, index: res.SketchIndex}, nil
}

// Extrude extrudes the sketch's first profile to a distance expression (a parameter name or
// formula) with the given operation (new|join|cut|intersect).
func (b *PartBuilder) Extrude(sk *SketchContext, distanceExpr, operation string) error {
	_, err := client.AddFeature(b.api.Features(), featureargs.Extrude{
		SketchIndex: sk.index, ProfileIndex: 0, Distance: distanceExpr, Operation: operation,
	})
	if err != nil {
		return fmt.Errorf("extrude sketch %d by %q: %w", sk.index, distanceExpr, err)
	}
	return nil
}
