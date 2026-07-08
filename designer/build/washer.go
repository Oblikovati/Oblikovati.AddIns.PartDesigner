// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// Washer is the washer generator. A plain washer (ISO 7089 / DIN 125) is a flat annular ring; a
// spring-lock washer (DIN 127) is a helical single-turn coil — its rectangular cross-section
// swept just under a full turn about the axis so the two split ends stand off axially at the
// free height, the defining spring form. Every dimension is parameter-driven, so editing the
// published size (inner_dia, outer_dia, thickness, and for the coil free_height) re-drives it.
//
// The plain ring is an outer disk with a concentric bore cut — the disk-minus-bore idiom the hex
// nut uses, two single-circle profiles keeping each extrude unambiguous (ProfileIndex 0) and
// DOF-0. The family's Variant selects the ring or the coil.
type Washer struct{}

// Kind is the family `generator` binding for washers.
func (Washer) Kind() string { return "washer" }

// Build realizes the DOF-0 parametric washer. A plain washer needs inner_dia, outer_dia and
// thickness; a spring washer (Variant "spring") additionally needs free_height for the coil.
func (Washer) Build(b *PartBuilder, rm ResolvedMember) error {
	split, err := washerHasSplit(rm.Family.Variant)
	if err != nil {
		return err
	}
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	if split {
		return buildSpringCoil(b)
	}
	return buildWasherRing(b)
}

// washerHasSplit resolves the family Variant: a plain washer is a flat ring, a spring washer is a
// helical coil. An unknown variant is rejected rather than silently building a plain ring.
func washerHasSplit(variant string) (bool, error) {
	switch variant {
	case "", "plain":
		return false, nil
	case "spring":
		return true, nil
	default:
		return false, fmt.Errorf("washer variant %q unknown; want \"plain\" or \"spring\"", variant)
	}
}

// buildWasherRing extrudes the outer disk up from the XY base plane by thickness, then cuts the
// concentric bore through it — leaving the flat annulus a plain washer is.
func buildWasherRing(b *PartBuilder) error {
	disk, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := disk.GroundedCircle(0, 0, "outer_dia"); err != nil {
		return err
	}
	if err := disk.AssertFullyConstrained(); err != nil {
		return err
	}
	if err := b.Extrude(disk, "thickness", "new"); err != nil {
		return err
	}
	bore, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := bore.GroundedCircle(0, 0, "inner_dia"); err != nil {
		return err
	}
	if err := bore.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.Extrude(bore, "thickness", "cut")
}

// coilRevolutions sweeps just under a full turn so the two ends leave the split — a thin kerf,
// not a wide wedge, since DIN 127's split is a near-radial cut, not a large angular gap.
const coilRevolutions = "0.97"

// buildSpringCoil realizes the DIN 127 spring-lock washer as a helical single-turn coil: the
// ring's rectangular cross-section (radial width from the diameters, thickness tall) swept about
// the axis. The coil's total height is the standard free height, so its axial rise offsets the
// split ends by free_height − thickness — the spring set. Unlike the plain washer it is a coil,
// so it needs no bore or split cut.
func buildSpringCoil(b *PartBuilder) error {
	sec, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sec.GroundedRadialSection("inner_dia", "outer_dia", "thickness"); err != nil {
		return err
	}
	if err := sec.AssertFullyConstrained(); err != nil {
		return err
	}
	// The coil's height is the free height; its rise over the sweep is that minus the section it
	// starts at, which is the axial offset of the split ends.
	return b.Coil(sec, "free_height - thickness", coilRevolutions)
}
