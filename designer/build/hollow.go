// SPDX-License-Identifier: GPL-2.0-only

package build

// HollowRect generates a square or rectangular hollow structural section (EN 10219 SHS/RHS) as a
// tube: a `width`×`height` outer rectangle extruded to `length`, then a concentric inner rectangle
// inset by the wall `thickness` on every side cut through it. One generator serves both SHS
// (width == height) and RHS. The cold-formed corners carry a real bend radius — the outer corner at
// outer_radius, the inner corner one wall-thickness tighter (outer_radius − thickness).
type HollowRect struct{}

// Kind is the family `generator` binding for rectangular/square hollow sections.
func (HollowRect) Kind() string { return "hollow_rect" }

// Build publishes the member's parameters, derives the outer/inner corner radii from the wall
// thickness (EN 10219: outer ≈ 2·t, inner = outer − t), and builds the tube from a rounded outer
// rectangle and an inset rounded inner rectangle. It expects the family to expose `width`,
// `height`, `thickness`, and `length`.
func (HollowRect) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	if err := b.DeriveParam("outer_radius", "2 * (thickness)"); err != nil {
		return err
	}
	if err := b.DeriveParam("inner_radius", "thickness"); err != nil {
		return err
	}
	if err := extrudeProfile(b, func(sk *SketchContext) error {
		return sk.GroundedRoundedRectangle("width", "height", "outer_radius")
	}, "new"); err != nil {
		return err
	}
	return extrudeProfile(b, func(sk *SketchContext) error {
		return sk.GroundedRoundedRectangle("width - 2 * (thickness)", "height - 2 * (thickness)", "inner_radius")
	}, "cut")
}

// HollowRound generates a circular hollow structural section (EN 10219 CHS) — a pipe — as a tube:
// an `outer_dia` circle extruded to `length`, then a concentric inner circle inset by twice the
// wall `thickness` (on the diameter) cut through it.
type HollowRound struct{}

// Kind is the family `generator` binding for circular hollow sections.
func (HollowRound) Kind() string { return "hollow_round" }

// Build publishes the member's parameters and builds the pipe from an outer and an inset inner
// circle. It expects the family to expose `outer_dia`, `thickness`, and `length`.
func (HollowRound) Build(b *PartBuilder, rm ResolvedMember) error {
	return tubeExtrude(b, rm,
		func(sk *SketchContext) error {
			return sk.GroundedCircle(0, 0, "outer_dia")
		},
		func(sk *SketchContext) error {
			return sk.GroundedCircle(0, 0, "outer_dia - 2 * (thickness)")
		})
}
