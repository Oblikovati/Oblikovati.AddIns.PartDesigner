// SPDX-License-Identifier: GPL-2.0-only

package build

// PlainBush generates a cylindrical plain sleeve bushing (ISO 4379) — a plain bearing with no
// rolling elements: a concentric tube of bore `bore`, outside diameter `outer_dia` and axial
// `length`. It is the hollow-cylinder path (outer circle as a new solid, bore circle cut through),
// so the id/od/length come straight from the member and re-drive with it.
type PlainBush struct{}

// Kind is the family `generator` binding for plain sleeve bushings.
func (PlainBush) Kind() string { return "plain_bush" }

// Build publishes bore/outer_dia/length and extrudes the sleeve: the outside diameter as a new
// solid, the bore as a cut through the full length.
func (PlainBush) Build(b *PartBuilder, rm ResolvedMember) error {
	return tubeExtrude(b, rm,
		func(sk *SketchContext) error { return sk.GroundedCircle(0, 0, "outer_dia") },
		func(sk *SketchContext) error { return sk.GroundedCircle(0, 0, "bore") },
	)
}
