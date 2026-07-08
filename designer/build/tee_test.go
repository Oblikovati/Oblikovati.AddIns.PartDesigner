// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// teeMember builds a synthetic resolved tee member: a text designation plus the four section
// dimensions and the stock length.
func teeMember(designation string, h, b, tw, tf, l float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-tee", Generator: "tee", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"designation"},
		Columns: []catalog.Column{
			{Name: "designation", Param: "designation", Type: catalog.ColumnText},
			{Name: "h", Param: "height", Type: catalog.ColumnLength},
			{Name: "b", Param: "flange_width", Type: catalog.ColumnLength},
			{Name: "tw", Param: "web_thickness", Type: catalog.ColumnLength},
			{Name: "tf", Param: "flange_thickness", Type: catalog.ColumnLength},
			{Name: "l", Param: "length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{
			Key:    "designation=" + designation,
			Values: map[string]float64{"h": h, "b": b, "tw": tw, "tf": tf, "l": l},
			Labels: map[string]string{"designation": designation},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestTeeBuildsSectionExtrude is the C3 tee acceptance check: the four section parameters are
// published, the 8-vertex outline is aligned and pinned by offset dimensions to the parameters,
// and it extrudes to `length` as one new solid.
func TestTeeBuildsSectionExtrude(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (Tee{}).Build(newBuilder(h, catalog.UnitsMillimetre),
		teeMember("T 60", 60, 60, 7, 7, 6000)); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	assertParam(t, h.added, "height", "60 mm")
	assertParam(t, h.added, "flange_width", "60 mm")
	assertParam(t, h.added, "web_thickness", "7 mm")
	assertParam(t, h.added, "flange_thickness", "7 mm")
	assertParam(t, h.added, "length", "6000 mm")

	if len(h.extrudes) != 1 || h.extrudes[0].Distance != "length" || h.extrudes[0].Operation != "new" {
		t.Errorf("extrudes = %+v, want one length/new tee section", h.extrudes)
	}
	wantOffsets := []string{half("height"), half("flange_width"), half("web_thickness"),
		"(height) / 2 - (flange_thickness)"}
	for _, expr := range wantOffsets {
		if !hasDimension(h.dimensions, expr) {
			t.Errorf("offset dimension %q not applied; have %+v", expr, h.dimensions)
		}
	}
	if got := countKind(h.constraints, "fix"); got != 1 {
		t.Errorf("fix count = %d, want 1 (the grounded origin)", got)
	}
	if got := countKind(h.constraints, "horizontal") + countKind(h.constraints, "vertical"); got != 9 {
		t.Errorf("axis alignments = %d, want 9 (tee section)", got)
	}
}

// TestTeeUnderConstrainedFails ensures a non-zero DOF aborts before extruding.
func TestTeeUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 3}
	if err := (Tee{}).Build(newBuilder(h, catalog.UnitsMillimetre),
		teeMember("T 60", 60, 60, 7, 7, 6000)); err == nil {
		t.Fatal("Build accepted an under-constrained tee; want an error")
	}
}

// TestTeeBuildErrorsPropagate injects a host failure at each wire method the build uses.
func TestTeeBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodParametersList, wire.MethodSketchCreate, wire.MethodSketchAddEntity,
		wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodFeaturesAdd,
	}
	for _, m := range methods {
		h := &fakeHost{dof: 0, failMethod: m}
		if err := (Tee{}).Build(newBuilder(h, catalog.UnitsMillimetre),
			teeMember("T 60", 60, 60, 7, 7, 6000)); err == nil {
			t.Errorf("failMethod %q: Build succeeded, want an error", m)
		}
	}
}

// TestDefaultRegistryHasTee checks the generator is wired into the built-in set.
func TestDefaultRegistryHasTee(t *testing.T) {
	g, ok := DefaultRegistry().Get("tee")
	if !ok || g.Kind() != "tee" {
		t.Fatalf("DefaultRegistry tee = (%v,%v), want the Tee generator", g, ok)
	}
}
