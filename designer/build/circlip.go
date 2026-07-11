// SPDX-License-Identifier: GPL-2.0-only

package build

import "oblikovati.org/part-designer/designer/catalog"

// splitGapAngle is the arc a retaining ring is revolved through: just under a full turn, so the
// missing wedge is the split gap the ring is sprung open at. It is a representational gap, not a
// per-size standard dimension (like the spring washer's sub-one-turn coil).
const splitGapAngle = "330 deg"

// Lug-ear proportions (representational — DIN 471/472 catalogues carry no lug/hole columns;
// see the geometry-math-advisor derivation, #61). Every additive constant carries a unit
// because the parameter evaluator is unit-strict (a bare literal + a length evaluates to 0).
const (
	earEyeFracExternal = "1.0"  // eye Ø as a fraction of the ring's radial band width
	earEyeFracInternal = "0.9"  // internal ears sit at a smaller radius → trimmed to avoid collision
	earHoleFrac        = "0.45" // plier-hole Ø as a fraction of the eye Ø (leaves a real rim)
	earOutwardFrac     = "0.3"  // eye centre this fraction of its own Ø past the band edge (60% overlap)
	earMinClr          = 0.3    // mm; rim floor + two-ear non-collision clearance (float, for the guard)
	mmPerInch          = 25.4   // exact by definition (1 in = 25.4 mm)
)

// Circlip generates a retaining ring / circlip (DIN 471 external, DIN 472 internal) as a flat
// split ring: a rectangular radial cross-section (inner_dia/2 → outer_dia/2, thickness tall)
// revolved about the axis through splitGapAngle, leaving the split gap. The ring's bore/outer
// diameter and thickness are parameter-driven. When the plier-lug guard passes, the two eyes'
// derived parameters are published (#61 Task 1) and their geometry — a flat annulus extruded
// through the thickness at each gap edge — is built too (#61 Task 2, see buildCirclipEar).
type Circlip struct{}

// Kind is the family `generator` binding for retaining rings.
func (Circlip) Kind() string { return "circlip" }

// Build publishes the member's parameters, revolves the ring's radial section through the split
// gap, and — when circlipEarsFit(rm) allows it — publishes the derived plier-lug-ear parameters
// and builds the two eyes (one per gap edge: 0° and splitGapAngle; external/internal branch from
// the family category, see circlipIsExternal). It expects the family to expose `inner_dia`,
// `outer_dia`, `thickness` (and drives the revolve to `length` via the fixed split-gap angle).
func (Circlip) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedRadialSection("inner_dia", "outer_dia", "thickness"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	if _, err := b.Revolve(sk, "origin/axis/z", splitGapAngle, "new"); err != nil {
		return err
	}
	if !circlipEarsFit(rm) {
		return nil // guard fails (e.g. an undersized custom member) → ring only, no ears
	}
	if err := deriveCirclipEarParams(b, circlipIsExternal(rm)); err != nil {
		return err
	}
	if err := b.buildCirclipEar("0 deg"); err != nil {
		return err
	}
	return b.buildCirclipEar(splitGapAngle)
}

// buildCirclipEar builds one plier-lug eye at the given gap-edge azimuth: a flat annulus on a
// work plane at the ring's axial mid-level, extruded through the ring thickness as a separate body
// overlapping the band end. Called once per gap edge (0° and splitGapAngle).
func (b *PartBuilder) buildCirclipEar(azimuthExpr string) error {
	sk, err := b.OffsetPlaneSketch("thickness / 2")
	if err != nil {
		return err
	}
	if err := sk.GroundedEyeSection("eye_radius_pos", azimuthExpr, "eye_outer_dia", "plier_hole_dia"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.ExtrudeDirected(sk, "thickness", "new", "symmetric")
}

// circlipIsExternal reports whether the ring is an external (shaft) ring — ears project radially
// OUTWARD — vs an internal (bore) ring — ears project INWARD. Keyed off the family category
// ("Shaft Parts/Retaining Rings/External" | ".../Internal"), the same signal the data encodes.
func circlipIsExternal(rm ResolvedMember) bool {
	c := rm.Family.Category
	return len(c) == 0 || c[len(c)-1] != "Internal"
}

// deriveCirclipEarParams publishes the lug-eye geometry: the ring's radial band width, the eye and
// plier-hole diameters, and the eye-centre radius (beyond OD for external, inside ID for internal).
func deriveCirclipEarParams(b *PartBuilder, external bool) error {
	eyeFrac := earEyeFracInternal
	radius := "inner_dia / 2 - eye_outer_dia * " + earOutwardFrac
	if external {
		eyeFrac, radius = earEyeFracExternal, "outer_dia / 2 + eye_outer_dia * "+earOutwardFrac
	}
	steps := []struct{ name, expr string }{
		{"ear_band_width", "(outer_dia - inner_dia) / 2"},
		{"eye_outer_dia", "ear_band_width * " + eyeFrac},
		{"plier_hole_dia", "eye_outer_dia * " + earHoleFrac},
		{"eye_radius_pos", radius},
	}
	for _, s := range steps {
		if err := b.DeriveParam(s.name, s.expr); err != nil {
			return err
		}
	}
	return nil
}

// earClearance scales the earMinClr mm floor into the family's own unit. rm.Value returns column
// numbers in that unit (the loader does no mm conversion), so circlipEarsFit's float comparisons
// must be against a clearance in the same unit — otherwise a bare mm constant checked against inch
// magnitudes reads as ~25x too large and silently fails an inch family's fit guard (#61).
func earClearance(rm ResolvedMember) float64 {
	if rm.Family.Units == catalog.UnitsInch {
		return earMinClr / mmPerInch
	}
	return earMinClr
}

// circlipEarsFit reports whether both lug ears fit: a positive eye rim, and the two ears (30° apart
// on the eye-centre circle) not colliding — 2·R·sin15° ≥ eye_dia + clearance. Internal rings are
// the binding case (smaller R). Mirrors the parametric formulas so the Go build decision matches.
func circlipEarsFit(rm ResolvedMember) bool {
	di, do := rm.Value("di"), rm.Value("do")
	band := (do - di) / 2
	external := circlipIsExternal(rm)
	eye := band
	if !external {
		eye = band * 0.9
	}
	hole := eye * 0.45
	var r float64
	if external {
		r = do/2 + eye*0.3
	} else {
		r = di/2 - eye*0.3
	}
	const sin15 = 0.2588190451
	clr := earClearance(rm)
	rimOK := hole+2*clr <= eye
	noCollide := 2*r*sin15 >= eye+clr
	posRadius := r-eye/2 > 0
	return rimOK && noCollide && posRadius
}
