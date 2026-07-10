// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/part-designer/designer/catalog"
)

// selfAligningMember builds a synthetic resolved 532xx member (the 53206: 30×47×11, 16 balls) —
// same columns as the 511xx thrust bearing but bound to the self-aligning generator.
func selfAligningMember(designation string, bore, outerDia, height, balls float64) ResolvedMember {
	rm := thrustMember(designation, bore, outerDia, height, balls)
	rm.Family.Generator = "thrust_self_aligning"
	return rm
}

// TestThrustSelfAligningBuildsSphericalSeat is the acceptance check: the boundary dimensions and the
// shared 511xx groove geometry are published, the seat sphere is derived (radius from the cap depth,
// its axis centre anchoring the OD rim at +height/2, a slightly smaller seat-washer sphere for the
// clearance), the balls are patterned FIRST, then three washers are revolved about Z — the grooved
// shaft washer, the sphered-back housing washer, and the seat washer.
func TestThrustSelfAligningBuildsSphericalSeat(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (ThrustSelfAligning{}).Build(newBuilder(h, catalog.UnitsMillimetre), selfAligningMember("53206", 30, 47, 11, 16)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	// Shared grooved-race params + the derived seat sphere.
	assertParam(t, h.added, "groove_radius", "ball_dia * 0.53")
	assertParam(t, h.added, "seat_cap", "0.06 * outer_dia")
	assertParam(t, h.added, "seat_sphere_r", "((outer_dia / 2) * (outer_dia / 2) + seat_cap * seat_cap) / (2 * seat_cap)")
	assertParam(t, h.added, "seat_centre_z", "height / 2 + sqrt(seat_sphere_r * seat_sphere_r - (outer_dia / 2) * (outer_dia / 2))")
	assertParam(t, h.added, "seat_washer_r", "seat_sphere_r - 0.02 * height")

	// One ball revolve about X + three washer revolves about Z (shaft, sphered housing, seat).
	if len(h.revolves) != 4 {
		t.Fatalf("revolves = %d, want 4 (ball + shaft + sphered housing + seat washer)", len(h.revolves))
	}
	if h.revolves[0].AxisRef != "origin/axis/x" {
		t.Errorf("first revolve = %+v, want the ball about origin/axis/x", h.revolves[0])
	}
	for i := 1; i < 4; i++ {
		if h.revolves[i].AxisRef != "origin/axis/z" || h.revolves[i].Angle != "360 deg" {
			t.Errorf("washer revolve[%d] = %+v, want origin/axis/z / 360 deg", i, h.revolves[i])
		}
	}
	if len(h.patterns) != 1 || h.patterns[0].CountExpr != "ball_count" {
		t.Errorf("patterns = %+v, want one ball_count pattern", h.patterns)
	}
	if len(h.extrudes) != 0 {
		t.Errorf("extrudes = %d, want 0 (all revolved)", len(h.extrudes))
	}
}

// TestDefaultRegistryHasThrustSelfAligning checks the generator is wired into the built-in set.
func TestDefaultRegistryHasThrustSelfAligning(t *testing.T) {
	g, ok := DefaultRegistry().Get("thrust_self_aligning")
	if !ok || g.Kind() != "thrust_self_aligning" {
		t.Fatalf("DefaultRegistry thrust_self_aligning = (%v,%v), want the ThrustSelfAligning generator", g, ok)
	}
}
