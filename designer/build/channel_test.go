// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// channelMember builds a synthetic resolved channel member: a text designation plus the four
// section dimensions and the stock length.
func channelMember(designation string, h, b, tw, tf, l float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-channel", Generator: "channel", Units: catalog.UnitsMillimetre,
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

// TestChannelBuildsSectionExtrude is the C2 channel acceptance check: the four section parameters
// are published, the 8-vertex outline is aligned and pinned by offset dimensions (including the
// web-front face at b/2−tw), and it extrudes to `length` as one new solid.
func TestChannelBuildsSectionExtrude(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (Channel{}).Build(newBuilder(h, catalog.UnitsMillimetre),
		channelMember("UPN 200", 200, 75, 8.5, 11.5, 6000)); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	assertParam(t, h.added, "height", "200 mm")
	assertParam(t, h.added, "flange_width", "75 mm")
	assertParam(t, h.added, "web_thickness", "8.5 mm")
	assertParam(t, h.added, "flange_thickness", "11.5 mm")
	assertParam(t, h.added, "length", "6000 mm")

	if len(h.extrudes) != 1 || h.extrudes[0].Distance != "length" || h.extrudes[0].Operation != "new" {
		t.Errorf("extrudes = %+v, want one length/new channel section", h.extrudes)
	}

	// The web-front face sits at b/2−tw from the origin — the offset that gives the section its
	// U shape rather than a plain rectangle.
	wantOffsets := []string{half("height"), half("flange_width"),
		"(height) / 2 - (flange_thickness)", "(flange_width) / 2 - (web_thickness)"}
	for _, expr := range wantOffsets {
		if !hasDimension(h.dimensions, expr) {
			t.Errorf("offset dimension %q not applied; have %+v", expr, h.dimensions)
		}
	}
	// 9 axis alignments + 1 origin fix.
	if got := countKind(h.constraints, "fix"); got != 1 {
		t.Errorf("fix count = %d, want 1 (the grounded origin)", got)
	}
	if got := countKind(h.constraints, "horizontal") + countKind(h.constraints, "vertical"); got != 9 {
		t.Errorf("axis alignments = %d, want 9 (channel section)", got)
	}
}

// TestChannelUnderConstrainedFails ensures a non-zero DOF aborts before extruding.
func TestChannelUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 4}
	if err := (Channel{}).Build(newBuilder(h, catalog.UnitsMillimetre),
		channelMember("UPN 200", 200, 75, 8.5, 11.5, 6000)); err == nil {
		t.Fatal("Build accepted an under-constrained channel; want an error")
	}
	if len(h.extrudes) != 0 {
		t.Errorf("extrudes = %d, want 0 when under-constrained", len(h.extrudes))
	}
}

// TestChannelBuildErrorsPropagate injects a host failure at each wire method the build uses.
func TestChannelBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodParametersList, wire.MethodSketchCreate, wire.MethodSketchAddEntity,
		wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodFeaturesAdd,
	}
	for _, m := range methods {
		h := &fakeHost{dof: 0, failMethod: m}
		if err := (Channel{}).Build(newBuilder(h, catalog.UnitsMillimetre),
			channelMember("UPN 200", 200, 75, 8.5, 11.5, 6000)); err == nil {
			t.Errorf("failMethod %q: Build succeeded, want an error", m)
		}
	}
}

// TestDefaultRegistryHasChannel checks the generator is wired into the built-in set.
func TestDefaultRegistryHasChannel(t *testing.T) {
	g, ok := DefaultRegistry().Get("channel")
	if !ok || g.Kind() != "channel" {
		t.Fatalf("DefaultRegistry channel = (%v,%v), want the Channel generator", g, ok)
	}
}
