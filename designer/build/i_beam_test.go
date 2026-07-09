// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// iBeamMember builds a synthetic resolved I-beam member: a text designation label plus the five
// section dimensions (including the root-fillet radius) and the stock length the generator drives.
func iBeamMember(designation string, h, b, tw, tf, r, l float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-ibeam", Generator: "i_beam", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"designation"},
		Columns: []catalog.Column{
			{Name: "designation", Param: "designation", Type: catalog.ColumnText},
			{Name: "h", Param: "height", Type: catalog.ColumnLength},
			{Name: "b", Param: "flange_width", Type: catalog.ColumnLength},
			{Name: "tw", Param: "web_thickness", Type: catalog.ColumnLength},
			{Name: "tf", Param: "flange_thickness", Type: catalog.ColumnLength},
			{Name: "r", Param: "root_radius", Type: catalog.ColumnLength},
			{Name: "l", Param: "length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{
			Key:    "designation=" + designation,
			Values: map[string]float64{"h": h, "b": b, "tw": tw, "tf": tf, "r": r, "l": l},
			Labels: map[string]string{"designation": designation},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestIBeamBuildsSectionExtrude is the C2 wide-flange acceptance check: the five section
// parameters are published (the text designation is a label, not a param), the filleted outline is
// aligned to the axes and pinned by offset dimensions to the parameters, its four root fillets
// share one radius, and it extrudes to `length` as one new solid.
func TestIBeamBuildsSectionExtrude(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (IBeam{}).Build(newBuilder(h, catalog.UnitsMillimetre),
		iBeamMember("IPE 200", 200, 100, 5.6, 8.5, 12, 6000)); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	assertParam(t, h.added, "height", "200 mm")
	assertParam(t, h.added, "flange_width", "100 mm")
	assertParam(t, h.added, "web_thickness", "5.6 mm")
	assertParam(t, h.added, "flange_thickness", "8.5 mm")
	assertParam(t, h.added, "root_radius", "12 mm")
	assertParam(t, h.added, "length", "6000 mm")
	// The designation is a text label; it must not become a driving parameter.
	for _, p := range h.added {
		if p.Name == "designation" {
			t.Errorf("designation published as a parameter (%+v); it is a label", p)
		}
	}

	if len(h.extrudes) != 1 || h.extrudes[0].Distance != "length" || h.extrudes[0].Operation != "new" {
		t.Errorf("extrudes = %+v, want one length/new I-section", h.extrudes)
	}

	// The section is pinned by eight offset dimensions from the grounded origin — height & web at
	// half, the flange-inner faces at h/2−tf — so the whole outline re-drives with the parameters.
	wantOffsets := []string{half("height"), half("flange_width"), half("web_thickness"),
		"(height) / 2 - (flange_thickness)"}
	for _, expr := range wantOffsets {
		if !hasDimension(h.dimensions, expr) {
			t.Errorf("offset dimension %q not applied; have %+v", expr, h.dimensions)
		}
	}
	// Four concave root fillets share one radius (EqualRadius chain of 3) driven by a single radius
	// dimension, so every web-flange junction re-drives with root_radius.
	if got := countKind(h.constraints, "equalRadius"); got != 3 {
		t.Errorf("equalRadius constraints = %d, want 3 (chain the four root fillets)", got)
	}
	if !hasDimension(h.dimensions, "root_radius") {
		t.Errorf("root-fillet radius dimension missing; have %+v", h.dimensions)
	}
	if got := countKind(h.constraints, "fix"); got != 1 {
		t.Errorf("fix count = %d, want 1 (the grounded origin)", got)
	}
	// With host inference disabled every straight edge is oriented explicitly (6 H + 6 V) plus the
	// 8 arc centre-pins (4 V + 4 H) — verified against the solver at DOF 0, redundant 0.
	if got := countKind(h.constraints, "horizontal") + countKind(h.constraints, "vertical"); got != 20 {
		t.Errorf("axis alignments = %d, want 20 (12 edges + 8 arc pins)", got)
	}
}

// TestIBeamUnderConstrainedFails ensures a non-zero DOF aborts before extruding a floppy section.
func TestIBeamUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 5}
	if err := (IBeam{}).Build(newBuilder(h, catalog.UnitsMillimetre),
		iBeamMember("IPE 200", 200, 100, 5.6, 8.5, 12, 6000)); err == nil {
		t.Fatal("Build accepted an under-constrained I-section; want an error")
	}
	if len(h.extrudes) != 0 {
		t.Errorf("extrudes = %d, want 0 when under-constrained", len(h.extrudes))
	}
}

// TestIBeamBuildErrorsPropagate injects a host failure at each wire method the build uses.
func TestIBeamBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodParametersList, wire.MethodSketchCreate, wire.MethodSketchAddEntity,
		wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodFeaturesAdd,
	}
	for _, m := range methods {
		h := &fakeHost{dof: 0, failMethod: m}
		if err := (IBeam{}).Build(newBuilder(h, catalog.UnitsMillimetre),
			iBeamMember("IPE 200", 200, 100, 5.6, 8.5, 12, 6000)); err == nil {
			t.Errorf("failMethod %q: Build succeeded, want an error", m)
		}
	}
}

// TestDefaultRegistryHasIBeam checks the generator is wired into the built-in set.
func TestDefaultRegistryHasIBeam(t *testing.T) {
	g, ok := DefaultRegistry().Get("i_beam")
	if !ok || g.Kind() != "i_beam" {
		t.Fatalf("DefaultRegistry i_beam = (%v,%v), want the IBeam generator", g, ok)
	}
}

// countKind counts recorded geometric-constraint kinds.
func countKind(kinds []string, want string) int {
	n := 0
	for _, k := range kinds {
		if k == want {
			n++
		}
	}
	return n
}
