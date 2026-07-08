// SPDX-License-Identifier: GPL-2.0-only

package build

// splitGapAngle is the arc a retaining ring is revolved through: just under a full turn, so the
// missing wedge is the split gap the ring is sprung open at. It is a representational gap, not a
// per-size standard dimension (like the spring washer's sub-one-turn coil).
const splitGapAngle = "330 deg"

// Circlip generates a retaining ring / circlip (DIN 471 external, DIN 472 internal) as a flat
// split ring: a rectangular radial cross-section (inner_dia/2 → outer_dia/2, thickness tall)
// revolved about the axis through splitGapAngle, leaving the split gap. The ring's bore/outer
// diameter and thickness are parameter-driven; the lug ears and their assembly holes are a tracked
// refinement (the ring is representational, per the milestone plan).
type Circlip struct{}

// Kind is the family `generator` binding for retaining rings.
func (Circlip) Kind() string { return "circlip" }

// Build publishes the member's parameters and revolves the ring's radial section through the split
// gap. It expects the family to expose `inner_dia`, `outer_dia`, `thickness` (and drives the
// revolve to `length` via the fixed split-gap angle).
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
	return b.Revolve(sk, splitGapAngle, "new")
}
