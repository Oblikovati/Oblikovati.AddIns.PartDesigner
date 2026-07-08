// SPDX-License-Identifier: GPL-2.0-only

package build

// HollowRect generates a square or rectangular hollow structural section (EN 10219 SHS/RHS) as a
// tube: a `width`×`height` outer rectangle extruded to `length`, then a concentric inner rectangle
// inset by the wall `thickness` on every side cut through it. One generator serves both SHS
// (width == height) and RHS. Corner radii are omitted (sharp section).
type HollowRect struct{}

// Kind is the family `generator` binding for rectangular/square hollow sections.
func (HollowRect) Kind() string { return "hollow_rect" }

// Build publishes the member's parameters and builds the tube from an outer and an inset inner
// rectangle. It expects the family to expose `width`, `height`, `thickness`, and `length`.
func (HollowRect) Build(b *PartBuilder, rm ResolvedMember) error {
	return tubeExtrude(b, rm,
		func(sk *SketchContext) error {
			return sk.GroundedRectangle("width", "height")
		},
		func(sk *SketchContext) error {
			return sk.GroundedRectangle("width - 2 * (thickness)", "height - 2 * (thickness)")
		})
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
