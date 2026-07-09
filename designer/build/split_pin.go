// SPDX-License-Identifier: GPL-2.0-only

package build

import "math"

// SplitPin generates a split / cotter pin (ISO 1234) — a folded round-wire hairpin: two legs joined
// by a semicircular eye at the top, retaining a clevis/castle nut through a transverse hole. A single
// wire circle is swept along the fold centreline (leg 1 up, the eye over the top, leg 2 back down).
//
// It is a REPRESENTATIONAL model: the real pin is folded half-round stock, approximated here as round
// wire of diameter d/2 (the two prongs together span the nominal size d). The eye is a semicircle of
// radius c/2 grounded in the ISO 1234 "width of eye" c; the leg length is the standard length l. Per
// issue #58; the part sweep uses featureargs.Sweep.PathPoints (a computed rail).
type SplitPin struct{}

// Kind is the family `generator` binding for split pins.
func (SplitPin) Kind() string { return "split_pin" }

// Build publishes the member's parameters, derives the wire diameter, authors the wire-circle
// profile perpendicular to the fold's first leg, and sweeps it along the computed hairpin path. It
// expects the family to expose `d` (nominal size), `c` (eye width) and `length`.
func (SplitPin) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	// The round-wire approximation of the two half-round prongs: each ~d/2, so a single swept circle
	// of d/2 gives two legs that together span the nominal size d.
	if err := b.DeriveParam("wire_dia", "d / 2"); err != nil {
		return err
	}
	// Profile: the wire circle on XY (normal +Z, matching the path's first tangent), centred at the
	// origin where the fold path starts (the bottom of leg 1).
	sk, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := sk.GroundedCircle(0, 0, "wire_dia"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	if _, err := b.Sweep(sk, splitPinPath(rm), "new"); err != nil {
		return err
	}
	return nil
}

// splitPinPath builds the fold centreline as a model-space polyline in CM, in the XZ plane (Y=0):
// leg 1 straight up +Z by the length l, a semicircular eye of radius c/2 over the top, then leg 2
// back down. The two leg centrelines sit the eye width c apart (the eye's chord), so the round legs
// read as a hairpin with a clean, open eye — the eye radius c/2 exceeds the wire radius d/4, so the
// swept tube keeps an open loop rather than pinching shut. Dims come from the member (mm → cm).
func splitPinPath(rm ResolvedMember) [][]float64 {
	const mmToCm = 0.1
	l := rm.Value("l") * mmToCm // leg length (standard length)
	a := rm.Value("a") * mmToCm // offset end: one prong extends this far past the other (ISO 1234 a)
	c := rm.Value("c") * mmToCm // eye width → the eye semicircle's diameter / leg separation
	if c <= 0 {
		c = rm.Value("d") * mmToCm // fall back to the nominal size if no eye width is tabulated
	}
	r := c / 2 // eye radius
	pts := [][]float64{
		{0, 0, 0}, // leg 1 (short prong) bottom = path start = profile centre
		{0, 0, l}, // up leg 1 to the eye
	}
	// Eye: a semicircle centred at (c/2, 0, l), from θ=π (leg-1 top) to θ=0 (leg-2 top), bulging +Z.
	const seg = 16
	for i := 1; i < seg; i++ {
		th := math.Pi * (1 - float64(i)/float64(seg))
		pts = append(pts, []float64{r + r*math.Cos(th), 0, l + r*math.Sin(th)})
	}
	pts = append(pts,
		[]float64{c, 0, l},  // leg 2 top
		[]float64{c, 0, -a}, // down leg 2, extending the offset end `a` past leg 1's foot
	)
	return pts
}
