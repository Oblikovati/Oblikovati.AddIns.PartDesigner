// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/part-designer/designer/catalog"
)

// angularMember builds a synthetic resolved angular-contact-bearing member: designation, bore d,
// outer diameter D, width B, contact angle alpha, ball count Z (the 7206-B: 30×62×16, 40°, 13).
func angularMember(designation string, bore, outerDia, width, angle, balls float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-angular-contact", Generator: "angular_contact", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"designation"},
		Columns: []catalog.Column{
			{Name: "designation", Param: "designation", Type: catalog.ColumnText},
			{Name: "d", Param: "bore", Type: catalog.ColumnLength},
			{Name: "D", Param: "outer_dia", Type: catalog.ColumnLength},
			{Name: "B", Param: "width", Type: catalog.ColumnLength},
			{Name: "alpha", Param: "contact_angle", Type: catalog.ColumnAngle},
			{Name: "Z", Param: "ball_count", Type: catalog.ColumnCount},
		},
		Members: []catalog.Member{{
			Key:    "designation=7206-B",
			Values: map[string]float64{"d": bore, "D": outerDia, "B": width, "alpha": angle, "Z": balls},
			Labels: map[string]string{"designation": designation},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestAngularContactBuildsRelievedOuterRing is the acceptance check: the tabulated dimensions,
// contact angle and ball count are published, the pitch/ball/race diameters and the relieved
// shoulder diameter derived, the balls patterned FIRST by ball_count, then the plain inner ring and
// the relieved outer ring revolved about the Z axis.
func TestAngularContactBuildsRelievedOuterRing(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (AngularContact{}).Build(newBuilder(h, catalog.UnitsMillimetre), angularMember("7206-B", 30, 62, 16, 40, 13)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "bore", "30 mm")
	assertParam(t, h.added, "contact_angle", "40 deg")
	assertParam(t, h.added, "ball_count", "13")
	assertParam(t, h.added, "ball_dia", "(outer_dia - bore) * 0.28")
	// The relieved shoulder opens OUTWARD (larger than the race) to expose the balls.
	assertParam(t, h.added, "relief_dia", "(outer_race_dia + outer_dia) / 2")

	if len(h.revolves) != 3 {
		t.Fatalf("revolves = %d, want 3 (one ball + inner ring + relieved outer ring)", len(h.revolves))
	}
	if h.revolves[0].AxisRef != "origin/axis/x" {
		t.Errorf("first revolve axis = %q, want origin/axis/x (the ball)", h.revolves[0].AxisRef)
	}
	if len(h.patterns) != 1 || h.patterns[0].CountExpr != "ball_count" {
		t.Errorf("patterns = %+v, want one ball_count pattern", h.patterns)
	}
}

// TestAngularContactUnderConstrainedFails ensures a non-zero DOF aborts before any solid is made.
func TestAngularContactUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 2}
	if err := (AngularContact{}).Build(newBuilder(h, catalog.UnitsMillimetre), angularMember("7206-B", 30, 62, 16, 40, 13)); err == nil {
		t.Fatal("Build accepted an under-constrained angular-contact bearing; want an error")
	}
}

// TestDefaultRegistryHasAngularContact checks the generator is wired into the built-in set.
func TestDefaultRegistryHasAngularContact(t *testing.T) {
	g, ok := DefaultRegistry().Get("angular_contact")
	if !ok || g.Kind() != "angular_contact" {
		t.Fatalf("DefaultRegistry angular_contact = (%v,%v), want the AngularContact generator", g, ok)
	}
}
