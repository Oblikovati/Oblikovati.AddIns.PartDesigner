// SPDX-License-Identifier: GPL-2.0-only

package build

// HexNut is the hex-nut generator (ISO 4032 / DIN 934 standard, ISO 4035 thin/jam): a regular
// hexagonal prism extruded up from the base plane, a through-bore cut down its axis, and a
// cosmetic internal thread over the bore wall. Height (nut_height) drives the standard/thin
// distinction, so the same generator realizes every member — the procedural analogue of picking
// another row from a Content-Center family table.
//
// Build order mirrors the cut+thread rule proven by the socket-screw generator: the thread must
// be the TERMINAL feature (a boolean placed after a cosmetic thread silently no-ops), so the
// bore is cut BEFORE threading and nothing follows the thread. The bore cut itself creates the
// only cylindrical face (the hex prism is all planar), so the thread targets it unambiguously.
type HexNut struct{}

// Kind is the family `generator` binding for hex nuts.
func (HexNut) Kind() string { return "hex_nut" }

// Build realizes the DOF-0 parametric nut. It expects the family to expose the across_flats,
// nut_height and nominal_dia parameters (via its columns), plus the d and P columns the thread
// designation is built from.
func (HexNut) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	if err := buildNutPrism(b); err != nil {
		return err
	}
	if err := cutNutBore(b); err != nil {
		return err
	}
	// The bore cut leaves the part's sole cylindrical face; thread it LAST (a boolean after a
	// cosmetic thread silently no-ops), sharing the fastener terminal step with the hex bolt.
	return threadSoleCylinder(b, rm)
}

// buildNutPrism extrudes the hexagonal prism up from the XY base plane by nut_height. The
// across-flats size drives the hexagon, so the six flats re-derive with the member.
func buildNutPrism(b *PartBuilder) error {
	sk, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := sk.GroundedHexagon("across_flats"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.Extrude(sk, "nut_height", "new")
}

// cutNutBore bores the tapped-hole clearance through the prism: a nominal-diameter circle on the
// XY base plane extruded up by nut_height as a cut, spanning the full prism height to leave a
// through-hole. The cut's lateral wall is the part's single cylindrical face — the surface the
// cosmetic thread binds to.
func cutNutBore(b *PartBuilder) error {
	sk, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := sk.GroundedCircle(0, 0, "nominal_dia"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.Extrude(sk, "nut_height", "cut")
}
