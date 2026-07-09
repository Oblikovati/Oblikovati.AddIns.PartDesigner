// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/part-designer/designer/catalog"
)

// taperedMember builds a synthetic resolved tapered-roller-bearing member: designation, bore d,
// outer diameter D, width T, contact angle alpha, roller count Z (the 30206: 30×62×17.25, 14°, 16).
func taperedMember(designation string, bore, outerDia, width, angle, rollers float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-tapered-roller", Generator: "tapered_roller", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"designation"},
		Columns: []catalog.Column{
			{Name: "designation", Param: "designation", Type: catalog.ColumnText},
			{Name: "d", Param: "bore", Type: catalog.ColumnLength},
			{Name: "D", Param: "outer_dia", Type: catalog.ColumnLength},
			{Name: "T", Param: "width", Type: catalog.ColumnLength},
			{Name: "alpha", Param: "contact_angle", Type: catalog.ColumnAngle},
			{Name: "Z", Param: "roller_count", Type: catalog.ColumnCount},
		},
		Members: []catalog.Member{{
			Key:    "designation=30206",
			Values: map[string]float64{"d": bore, "D": outerDia, "T": width, "alpha": angle, "Z": rollers},
			Labels: map[string]string{"designation": designation},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestTaperedRollerBuildsRollersAndRaces is the acceptance check: the tabulated dimensions, contact
// angle and roller count are published, the roller/race geometry derived, one tapered roller lofted
// and patterned FIRST by roller_count, then the cone and cup revolved about the Z axis.
func TestTaperedRollerBuildsRollersAndRaces(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (TaperedRoller{}).Build(newBuilder(h, catalog.UnitsMillimetre), taperedMember("30206", 30, 62, 17.25, 14, 16)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "bore", "30 mm")
	assertParam(t, h.added, "contact_angle", "14 deg")
	assertParam(t, h.added, "roller_count", "16")
	// On-apex angles: cone ray = 0.75·α, roller axis δ = 0.875·α; the apex arm is p/tan δ.
	assertParam(t, h.added, "cone_ray_angle", "contact_angle * 0.75")
	assertParam(t, h.added, "axis_angle", "contact_angle * 0.875")
	assertParam(t, h.added, "apex_arm", "(pitch_dia / 2) / tan(axis_angle)")
	// Raceway diameters are the shared apex rays 2·ζ·tan γ; the roller diameter falls out of them.
	assertParam(t, h.added, "cup_big_dia", "2 * zeta_big * tan(contact_angle)")
	assertParam(t, h.added, "roller_big_dia", "(cup_big_dia - cone_big_dia) / 2")
	assertParam(t, h.added, "roller_big_pos", "(cone_big_dia + cup_big_dia) / 2")
	// The cone big rib: foot beyond the roller big end, crest proud of the roller but clear of the cup.
	assertParam(t, h.added, "rib_inner_z", "roller_axial / 2 + width * 0.04")
	assertParam(t, h.added, "rib_crest_dia",
		"min(roller_big_pos + 0.8 * roller_big_dia, cup_big_dia - 0.3 * roller_big_dia)")

	if len(h.lofts) != 1 {
		t.Fatalf("lofts = %d, want 1 (the tapered roller)", len(h.lofts))
	}
	if len(h.patterns) != 1 || h.patterns[0].CountExpr != "roller_count" {
		t.Errorf("patterns = %+v, want one roller_count pattern", h.patterns)
	}
	if len(h.revolves) != 2 {
		t.Fatalf("revolves = %d, want 2 (cone + cup)", len(h.revolves))
	}
	for i, rv := range h.revolves {
		if rv.AxisRef != "origin/axis/z" || rv.Operation != "new" {
			t.Errorf("revolve[%d] = %+v, want origin/axis/z / new", i, rv)
		}
	}
}

// TestTaperedRollerUnderConstrainedFails ensures a non-zero DOF aborts before any solid is made.
func TestTaperedRollerUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 2}
	if err := (TaperedRoller{}).Build(newBuilder(h, catalog.UnitsMillimetre), taperedMember("30206", 30, 62, 17.25, 14, 16)); err == nil {
		t.Fatal("Build accepted an under-constrained tapered bearing; want an error")
	}
	if len(h.lofts) != 0 || len(h.revolves) != 0 {
		t.Errorf("made geometry despite bad DOF: lofts=%d revolves=%d", len(h.lofts), len(h.revolves))
	}
}

// TestDefaultRegistryHasTaperedRoller checks the generator is wired into the built-in set.
func TestDefaultRegistryHasTaperedRoller(t *testing.T) {
	g, ok := DefaultRegistry().Get("tapered_roller")
	if !ok || g.Kind() != "tapered_roller" {
		t.Fatalf("DefaultRegistry tapered_roller = (%v,%v), want the TaperedRoller generator", g, ok)
	}
}
