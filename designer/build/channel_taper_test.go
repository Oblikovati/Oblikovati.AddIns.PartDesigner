// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/part-designer/designer/catalog"
)

// taperedChannelMember builds a synthetic resolved UPN member: the four section dimensions plus
// the flange taper angle the taper-flange generator needs (the fillet radii are derived from tf).
func taperedChannelMember(designation string, h, b, tw, tf, taper, l float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-upn", Generator: "channel_taper", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"designation"},
		Columns: []catalog.Column{
			{Name: "designation", Param: "designation", Type: catalog.ColumnText},
			{Name: "h", Param: "height", Type: catalog.ColumnLength},
			{Name: "b", Param: "flange_width", Type: catalog.ColumnLength},
			{Name: "tw", Param: "web_thickness", Type: catalog.ColumnLength},
			{Name: "tf", Param: "flange_thickness", Type: catalog.ColumnLength},
			{Name: "taper", Param: "flange_taper", Type: catalog.ColumnAngle},
			{Name: "l", Param: "length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{
			Key:    "designation=" + designation,
			Values: map[string]float64{"h": h, "b": b, "tw": tw, "tf": tf, "taper": taper, "l": l},
			Labels: map[string]string{"designation": designation},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestTaperedChannelBuildsFilletedSection: the UPN generator publishes its parameters (including
// the flange taper angle + toe radius), rounds the four inner corners, and extrudes the DOF-0
// tapered-flange section to length (#69).
func TestTaperedChannelBuildsFilletedSection(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (TaperedChannel{}).Build(newBuilder(h, catalog.UnitsMillimetre),
		taperedChannelMember("UPN 200", 200, 75, 8.5, 11.5, 4.5739, 6000)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "flange_thickness", "11.5 mm")
	assertParam(t, h.added, "flange_taper", "4.5739 deg")

	if len(h.extrudes) != 1 || h.extrudes[0].Distance != "length" {
		t.Errorf("extrudes = %+v, want one length extrude of the tapered section", h.extrudes)
	}
	// Four fillet radius dimensions: two roots at tf, two toes at tf/2.
	if !hasDimension(h.dimensions, "flange_thickness") || !hasDimension(h.dimensions, "(flange_thickness) / 2") {
		t.Errorf("fillet radius dimensions missing; have %+v", h.dimensions)
	}
}

// TestTaperedChannelUnderConstrainedFails: a non-zero DOF (an under-pinned section) aborts before
// extruding, so a floppy tapered channel never becomes a solid.
func TestTaperedChannelUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 3}
	if err := (TaperedChannel{}).Build(newBuilder(h, catalog.UnitsMillimetre),
		taperedChannelMember("UPN 200", 200, 75, 8.5, 11.5, 4.5739, 6000)); err == nil {
		t.Fatal("Build accepted an under-constrained tapered channel; want an error")
	}
	if len(h.extrudes) != 0 {
		t.Errorf("extrudes = %d, want 0 when under-constrained", len(h.extrudes))
	}
}
