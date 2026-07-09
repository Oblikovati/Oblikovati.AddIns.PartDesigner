// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"math"
	"testing"

	"oblikovati.org/api/wire"
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

// TestAngularContactBuildsTiltedGrooveRings is the acceptance check: the tabulated dimensions,
// contact angle and ball count are published; the pitch/ball/groove diameters, the axial groove
// offset for the contact angle, and each ring's asymmetric high/relief shoulder diameters are
// derived; the balls are patterned FIRST by ball_count; then the inner and outer tilted-groove rings
// are revolved about the Z axis.
func TestAngularContactBuildsTiltedGrooveRings(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (AngularContact{}).Build(newBuilder(h, catalog.UnitsMillimetre), angularMember("7206-B", 30, 62, 16, 40, 13)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "bore", "30 mm")
	assertParam(t, h.added, "contact_angle", "40 deg")
	assertParam(t, h.added, "ball_count", "13")
	assertParam(t, h.added, "ball_dia", "(outer_dia - bore) * 0.28")
	assertParam(t, h.added, "groove_radius", "ball_dia * 0.52")
	// The groove centre is displaced along the α-tilted contact normal by (r_g − R): its axial and
	// radial components tilt the contact line and straddle the pitch circle.
	assertParam(t, h.added, "groove_axial_offset", "(groove_radius - ball_dia / 2) * sin(contact_angle)")
	assertParam(t, h.added, "groove_radial_offset", "(groove_radius - ball_dia / 2) * cos(contact_angle)")
	assertParam(t, h.added, "outer_groove_dia", "pitch_dia + 2 * groove_radial_offset")
	assertParam(t, h.added, "inner_groove_dia", "pitch_dia - 2 * groove_radial_offset")
	// The shoulders are asymmetric: a tall retaining land (0.55·r_g) and a relieved counterbore
	// (0.85·r_g), opened outward on the outer ring and inward on the inner ring.
	assertParam(t, h.added, "outer_high_shoulder_dia", "outer_groove_dia + 1.1 * groove_radius")
	assertParam(t, h.added, "outer_relief_shoulder_dia", "outer_groove_dia + 1.7 * groove_radius")
	assertParam(t, h.added, "inner_high_shoulder_dia", "inner_groove_dia - 1.1 * groove_radius")
	assertParam(t, h.added, "inner_relief_shoulder_dia", "inner_groove_dia - 1.7 * groove_radius")

	if len(h.revolves) != 3 {
		t.Fatalf("revolves = %d, want 3 (one ball + inner ring + outer ring)", len(h.revolves))
	}
	if h.revolves[0].AxisRef != "origin/axis/x" {
		t.Errorf("first revolve axis = %q, want origin/axis/x (the ball)", h.revolves[0].AxisRef)
	}
	for i := 1; i < 3; i++ {
		if h.revolves[i].AxisRef != "origin/axis/z" || h.revolves[i].Angle != "360 deg" {
			t.Errorf("ring revolve[%d] = %+v, want origin/axis/z / 360 deg", i, h.revolves[i])
		}
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

// TestAngularContactBuildErrorsPropagate injects a host failure at each wire method the build uses.
func TestAngularContactBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodParametersList, wire.MethodParametersAdd, wire.MethodSketchCreate,
		wire.MethodSketchAddEntity, wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodFeaturesAdd,
	}
	for _, m := range methods {
		h := &fakeHost{dof: 0, failMethod: m}
		if err := (AngularContact{}).Build(newBuilder(h, catalog.UnitsMillimetre), angularMember("7206-B", 30, 62, 16, 40, 13)); err == nil {
			t.Errorf("failMethod %q: Build succeeded, want an error", m)
		}
	}
}

// TestAngularGrooveFitsRaceway guards the tilted-groove geometry against the groove breaching the
// contact-side axial face. The groove centre is pushed off the mid-plane by gc_axial = (r_g−R)·sin α
// toward the tall retaining shoulder, and the groove arc reaches sqrt(1−0.55²)·r_g ≈ 0.835·r_g beyond
// that centre on the high side, so gc_axial + 0.835·r_g must stay inside width/2 or the groove would
// cut the axial face and the ring section would self-intersect on revolve. This is the static,
// per-member check the fakeHost cannot make (it does not evaluate expressions).
func TestAngularGrooveFitsRaceway(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("catalog.Load() error = %v", err)
	}
	const zsHigh = 0.835 // sqrt(1 − 0.55²), the groove's high-side axial reach as a fraction of r_g
	for _, fam := range cat.Families() {
		if fam.Generator != "angular_contact" {
			continue
		}
		cols := map[string]string{}
		for _, c := range fam.Columns {
			cols[c.Param] = c.Name
		}
		for _, m := range fam.Members {
			bore, outer := m.Values[cols["bore"]], m.Values[cols["outer_dia"]]
			width, alpha := m.Values[cols["width"]], m.Values[cols["contact_angle"]]
			ballDia := 0.28 * (outer - bore)
			grooveRadius := 0.52 * ballDia
			gcAxial := (grooveRadius - ballDia/2) * math.Sin(alpha*math.Pi/180)
			if land := width/2 - (gcAxial + zsHigh*grooveRadius); land <= 0 {
				t.Errorf("family %q member %q: tilted groove breaches the axial face "+
					"(width/2=%.2f ≤ gc_axial+0.835·r_g=%.2f, land=%.2f)",
					fam.ID, m.Key, width/2, gcAxial+zsHigh*grooveRadius, land)
			}
		}
	}
}

// TestDefaultRegistryHasAngularContact checks the generator is wired into the built-in set.
func TestDefaultRegistryHasAngularContact(t *testing.T) {
	g, ok := DefaultRegistry().Get("angular_contact")
	if !ok || g.Kind() != "angular_contact" {
		t.Fatalf("DefaultRegistry angular_contact = (%v,%v), want the AngularContact generator", g, ok)
	}
}
