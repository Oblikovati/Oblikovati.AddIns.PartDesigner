// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/api/client"
	"oblikovati.org/api/wire"
	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/part-designer/designer/catalog"
)

// roundBarMember builds a synthetic resolved member whose family exposes the nominal_dia +
// length parameters the RoundBar reference generator needs.
func roundBarMember(d, l float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t", Generator: "round_bar", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"d"},
		Columns: []catalog.Column{
			{Name: "d", Param: "nominal_dia", Type: catalog.ColumnLength},
			{Name: "l", Param: "length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{Key: "d=8", Values: map[string]float64{"d": d, "l": l}}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

func newBuilder(h *fakeHost, unit catalog.Units) *PartBuilder {
	return NewPartBuilder(client.New(h), unit)
}

// TestRoundBarBuildsParametricCylinder is the A3 acceptance check: the reference generator
// publishes the member's parameters and builds a DOF-0 cylinder whose geometry is driven by
// those parameters (never literal coordinates), in the expected call order.
func TestRoundBarBuildsParametricCylinder(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (RoundBar{}).Build(newBuilder(h, catalog.UnitsMillimetre), roundBarMember(8, 40)); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	// Parameters published from the member columns (all new -> Add), with units.
	wantAdd := []wire.ParameterSetArgs{
		{Name: "nominal_dia", Expression: "8 mm"},
		{Name: "length", Expression: "40 mm"},
	}
	if len(h.added) != 2 || h.added[0] != wantAdd[0] || h.added[1] != wantAdd[1] {
		t.Errorf("added params = %+v, want %+v", h.added, wantAdd)
	}

	// The circle's radius and its binding dimension are parameter expressions, not literals.
	if h.circleRadius != "(nominal_dia)/2" {
		t.Errorf("circle radius expr = %q, want (nominal_dia)/2", h.circleRadius)
	}
	if len(h.constraints) != 1 || h.constraints[0] != "fix" {
		t.Errorf("constraints = %v, want [fix] (pinned centre)", h.constraints)
	}
	if len(h.dimensions) != 1 || h.dimensions[0].Kind != "diameter" ||
		h.dimensions[0].Expression != "nominal_dia" || h.dimensions[0].SketchIndex != 1 {
		t.Errorf("dimensions = %+v, want one diameter=nominal_dia on sketch 1", h.dimensions)
	}

	// Extrude is parameter-driven and on the built sketch.
	if h.extrudeKind != featureargs.KindExtrude {
		t.Errorf("feature kind = %q, want %q", h.extrudeKind, featureargs.KindExtrude)
	}
	if h.extrude.Distance != "length" || h.extrude.Operation != "new" || h.extrude.SketchIndex != 1 {
		t.Errorf("extrude = %+v, want distance=length op=new sketch=1", h.extrude)
	}

	// DOF is checked before the extrude commits, so an under-constrained profile is caught.
	if indexOf(h.methods, wire.MethodSketchConstraintStatus) >= indexOf(h.methods, wire.MethodFeaturesAdd) {
		t.Errorf("constraint status must be checked before extrude; methods = %v", h.methods)
	}
}

// TestPublishParamsUpsert checks the idempotent path: a parameter already on the document is
// Set (re-driven), a new one is Added — the Change-Size behaviour.
func TestPublishParamsUpsert(t *testing.T) {
	h := &fakeHost{existing: []string{"nominal_dia"}}
	err := newBuilder(h, catalog.UnitsMillimetre).PublishParams(roundBarMember(10, 50))
	if err != nil {
		t.Fatalf("PublishParams error = %v", err)
	}
	if len(h.set) != 1 || h.set[0].Name != "nominal_dia" || h.set[0].Expression != "10 mm" {
		t.Errorf("set = %+v, want nominal_dia=10 mm (re-driven)", h.set)
	}
	if len(h.added) != 1 || h.added[0].Name != "length" {
		t.Errorf("added = %+v, want only length (new)", h.added)
	}
}

// TestParamExprFormatting covers the unit-bearing expression rendering per column type.
func TestParamExprFormatting(t *testing.T) {
	mm := &PartBuilder{unit: "mm"}
	inch := &PartBuilder{unit: "in"}
	for _, tc := range []struct {
		b    *PartBuilder
		typ  catalog.ColumnType
		v    float64
		want string
	}{
		{mm, catalog.ColumnLength, 12.5, "12.5 mm"},
		{inch, catalog.ColumnLength, 0.25, "0.25 in"},
		{mm, catalog.ColumnAngle, 30, "30 deg"},
		{mm, catalog.ColumnCount, 6, "6"},
	} {
		if got := tc.b.paramExpr(tc.typ, tc.v); got != tc.want {
			t.Errorf("paramExpr(%q,%v) = %q, want %q", tc.typ, tc.v, got, tc.want)
		}
	}
}

// TestUnderConstrainedProfileFails ensures a non-zero DOF from the solver aborts the build
// (with the offending DOF in the message) rather than producing a floppy part.
func TestUnderConstrainedProfileFails(t *testing.T) {
	h := &fakeHost{dof: 2}
	err := (RoundBar{}).Build(newBuilder(h, catalog.UnitsMillimetre), roundBarMember(8, 40))
	if err == nil {
		t.Fatal("Build accepted an under-constrained sketch; want an error")
	}
	if h.hasMethod(wire.MethodFeaturesAdd) {
		t.Error("extrude ran on an under-constrained sketch; DOF check must gate it")
	}
}

// indexOf returns the first position of s in xs, or len(xs) when absent.
func indexOf(xs []string, s string) int {
	for i, x := range xs {
		if x == s {
			return i
		}
	}
	return len(xs)
}
