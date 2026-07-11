// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// circlipMember builds a synthetic resolved retaining-ring member: nominal, ring bore/outer, s.
// It carries no Category (neither External nor Internal), which is fine for the plain-ring tests
// below since those don't touch the ear branch.
func circlipMember(nominal, innerDia, outerDia, thickness float64) ResolvedMember {
	return circlipMemberWithCategory(nominal, innerDia, outerDia, thickness, nil)
}

// circlipMemberWithCategory is circlipMember plus an explicit family Category, the signal
// circlipIsExternal/circlipEarsFit key the external-vs-internal branch off.
func circlipMemberWithCategory(nominal, innerDia, outerDia, thickness float64, category catalog.CategoryPath) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-circlip", Generator: "circlip", Units: catalog.UnitsMillimetre,
		Category:   category,
		KeyColumns: []string{"d"},
		Columns: []catalog.Column{
			{Name: "d", Param: "nominal_dia", Type: catalog.ColumnLength},
			{Name: "di", Param: "inner_dia", Type: catalog.ColumnLength},
			{Name: "do", Param: "outer_dia", Type: catalog.ColumnLength},
			{Name: "s", Param: "thickness", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{
			Key:    "d=20",
			Values: map[string]float64{"d": nominal, "di": innerDia, "do": outerDia, "s": thickness},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// circlipExtMember builds an external (shaft) retaining-ring member — category ".../External" —
// with di/do in the same reading order as circlip_din471.json's columns (di < do): the groove/shaft
// diameter first, the larger free outer diameter second.
func circlipExtMember(key string, nominal, innerDia, outerDia, thickness float64) ResolvedMember {
	rm := circlipMemberWithCategory(nominal, innerDia, outerDia, thickness,
		catalog.CategoryPath{"Shaft Parts", "Retaining Rings", "External"})
	rm.Family.Members[0].Key = key
	return rm
}

// circlipIntMember builds an internal (bore) retaining-ring member — category ".../Internal".
// Args are (key, nominal, outerDia, innerDia, thickness): DIN 472's free outer diameter "do" is
// always larger than the bore/groove diameter "di" (circlip_din472.json, e.g. d20: di=15.0,
// do=21.0), so outerDia is listed first to read in size order and to let call sites hand this
// helper the real catalogue values directly.
func circlipIntMember(key string, nominal, outerDia, innerDia, thickness float64) ResolvedMember {
	rm := circlipMemberWithCategory(nominal, innerDia, outerDia, thickness,
		catalog.CategoryPath{"Shaft Parts", "Retaining Rings", "Internal"})
	rm.Family.Members[0].Key = key
	return rm
}

// TestCirclipRevolvesSplitRing is the D3 acceptance check: the ring parameters are published and a
// radial section is revolved through the split-gap angle (under a full turn) as one new solid.
func TestCirclipRevolvesSplitRing(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (Circlip{}).Build(newBuilder(h, catalog.UnitsMillimetre), circlipMember(20, 19, 27, 1.2)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "inner_dia", "19 mm")
	assertParam(t, h.added, "outer_dia", "27 mm")
	assertParam(t, h.added, "thickness", "1.2 mm")

	if len(h.revolves) != 1 {
		t.Fatalf("revolves = %d, want 1 (the split ring)", len(h.revolves))
	}
	rv := h.revolves[0]
	if rv.Angle != splitGapAngle || rv.Operation != "new" || rv.AxisRef != "origin/axis/z" {
		t.Errorf("revolve = %+v, want %s about z / new", rv, splitGapAngle)
	}
	if len(h.extrudes) != 0 {
		t.Errorf("extrudes = %d, want 0 (the ring is revolved, not extruded)", len(h.extrudes))
	}
}

// TestCirclipUnderConstrainedFails ensures a non-zero DOF aborts before the revolve.
func TestCirclipUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 3}
	if err := (Circlip{}).Build(newBuilder(h, catalog.UnitsMillimetre), circlipMember(20, 19, 27, 1.2)); err == nil {
		t.Fatal("Build accepted an under-constrained ring; want an error")
	}
	if len(h.revolves) != 0 {
		t.Errorf("revolves = %d, want 0 when under-constrained", len(h.revolves))
	}
}

// TestCirclipBuildErrorsPropagate injects a host failure at each wire method the build uses.
func TestCirclipBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodParametersList, wire.MethodSketchCreate, wire.MethodSketchAddEntity,
		wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodFeaturesAdd,
	}
	for _, m := range methods {
		h := &fakeHost{dof: 0, failMethod: m}
		if err := (Circlip{}).Build(newBuilder(h, catalog.UnitsMillimetre), circlipMember(20, 19, 27, 1.2)); err == nil {
			t.Errorf("failMethod %q: Build succeeded, want an error", m)
		}
	}
}

// TestCirclipEarParamsExternal checks the external-branch derived-ear-parameter expressions
// (exact strings) published after the ring revolve, for a DIN 471 d30 member.
func TestCirclipEarParamsExternal(t *testing.T) {
	h := &fakeHost{dof: 0}
	// DIN 471 d30: di=28.6, do=38.6, s=1.5, external
	if err := (Circlip{}).Build(newBuilder(h, catalog.UnitsMillimetre), circlipExtMember("d30", 30, 28.6, 38.6, 1.5)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "ear_band_width", "(outer_dia - inner_dia) / 2")
	assertParam(t, h.added, "eye_outer_dia", "ear_band_width * 1.0")
	assertParam(t, h.added, "plier_hole_dia", "eye_outer_dia * 0.45")
	assertParam(t, h.added, "eye_radius_pos", "outer_dia / 2 + eye_outer_dia * 0.3")
}

// TestCirclipEarParamsInternal checks the internal-branch derived-ear-parameter expressions
// (smaller kEye, eye centre inside the bore) for a DIN 472 d20 member.
func TestCirclipEarParamsInternal(t *testing.T) {
	h := &fakeHost{dof: 0}
	// DIN 472 d20: di=15 (bore side)… internal kEye=0.9, eye projects inward
	if err := (Circlip{}).Build(newBuilder(h, catalog.UnitsMillimetre), circlipIntMember("d20", 20, 21.0, 15.0, 1.0)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "eye_outer_dia", "ear_band_width * 0.9")
	assertParam(t, h.added, "eye_radius_pos", "inner_dia / 2 - eye_outer_dia * 0.3")
}

// TestCirclipEarsFit checks the Go-side fit guard mirrors the parametric formulas: DIN 471 d30
// (external) and DIN 472 d20 (internal, the binding worst case, 15.4% margin) both fit; a
// synthetic tiny internal ring (small bore + wide band) does not.
func TestCirclipEarsFit(t *testing.T) {
	if !circlipEarsFit(circlipExtMember("d30", 30, 28.6, 38.6, 1.5)) {
		t.Error("ears should fit DIN471 d30 (external)")
	}
	if !circlipEarsFit(circlipIntMember("d20", 20, 21.0, 15.0, 1.0)) {
		t.Error("ears should fit DIN472 d20 (internal, the binding worst case, 15.4% margin)")
	}
	// Synthetic tiny internal: small bore + wide band → ears collide → skip.
	if circlipEarsFit(circlipIntMember("x", 6, 9.0, 4.0, 1.0)) {
		t.Error("ears must NOT fit a tiny internal ring; want skip-ears fallback")
	}
}

// TestCirclipEarsFallbackSkipsParams is the Build-level regression for circlipEarsFit's false
// branch (mirrors TestRollerCageFallbackSkipsBar in roller_cage_test.go): with the same tiny
// internal fixture TestCirclipEarsFit already proved fails the guard, Build must publish none of
// the four ear-derived parameters and still succeed — the ring builds exactly as it does today
// (Fall-back-to-plain, #61 design). It would fail if circlipEarsFit's result stopped gating
// deriveCirclipEarParams.
func TestCirclipEarsFallbackSkipsParams(t *testing.T) {
	h := &fakeHost{dof: 0}
	rm := circlipIntMember("x", 6, 9.0, 4.0, 1.0)
	if circlipEarsFit(rm) {
		t.Fatal("test fixture unexpectedly passes circlipEarsFit; no longer degenerate")
	}
	if err := (Circlip{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	for _, name := range []string{"ear_band_width", "eye_outer_dia", "plier_hole_dia", "eye_radius_pos"} {
		for _, p := range h.added {
			if p.Name == name {
				t.Errorf("param %q published; want no ear params in the skip-ears fallback", name)
			}
		}
	}
}

// TestCirclipEarDeriveParamsErrorsPropagate walks deriveCirclipEarParams' own four-step
// derivation chain by letting parameters.list succeed failAfter times (the one call inside
// PublishParams, already covered by TestCirclipBuildErrorsPropagate, plus every earlier ear-param
// step's own DeriveParam→existingParams call) before failing the next one — reaching each step's
// "if err != nil" guard (circlip.go's deriveCirclipEarParams loop) one at a time instead of always
// tripping the very first list call. Mirrors TestRollerDeriveParamsErrorsPropagate's pattern
// (roller_bearing_test.go) for the analogous rollerCage derivation chain.
func TestCirclipEarDeriveParamsErrorsPropagate(t *testing.T) {
	cases := []struct {
		name      string
		failAfter int
	}{
		{"ear_band_width", 1},
		{"eye_outer_dia", 2},
		{"plier_hole_dia", 3},
		{"eye_radius_pos", 4},
	}
	for _, c := range cases {
		h := &fakeHost{dof: 0, failMethod: wire.MethodParametersList, failAfter: c.failAfter}
		rm := circlipExtMember("d30", 30, 28.6, 38.6, 1.5) // passes circlipEarsFit
		err := (Circlip{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm)
		if err == nil {
			t.Errorf("%s: Build succeeded, want an error", c.name)
		}
	}
}

// TestDefaultRegistryHasCirclip checks the generator is wired into the built-in set.
func TestDefaultRegistryHasCirclip(t *testing.T) {
	g, ok := DefaultRegistry().Get("circlip")
	if !ok || g.Kind() != "circlip" {
		t.Fatalf("DefaultRegistry circlip = (%v,%v), want the Circlip generator", g, ok)
	}
}
