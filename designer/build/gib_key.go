// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// gibNoseFraction sizes the axial length of the gib nose as a fraction of its height. DIN 6887
// tabulates the nose HEIGHT (h2) but the reduced dimension source carries no nose-length column, so
// the projection is a representative proportion: a gib nose is drawn roughly as long as it is tall.
const gibNoseFraction = "0.9"

// GibHeadKey generates a gib-head taper key (DIN 6887) — a parallel-key body with a raised "gib"
// nose at one end that a fitter can drive a wedge behind to extract the key. It is modelled from the
// LENGTH silhouette (a pentagon: the body rectangle plus the taller nose sloping back to the body
// top) extruded across the key width, so `head_height` (the nose) rises above `height` (the body).
// The fit-critical width/height and the grounded nose height h2 drive the shape; the 1:100 body
// taper and the exact nose length/radius are a tracked refinement (the body is modelled at constant
// height and the nose length is a representative proportion of h2).
type GibHeadKey struct{}

// Kind is the family `generator` binding for gib-head keys.
func (GibHeadKey) Kind() string { return "gib_head_key" }

// Build publishes the member's parameters, derives the nose length, and extrudes the gib silhouette
// across the width. It expects the family to expose `width`, `height`, `head_height` and `length`.
func (GibHeadKey) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	if err := b.DeriveParam("nose_length", "head_height * "+gibNoseFraction); err != nil {
		return err
	}
	sk, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := sk.GroundedGibProfile("length", "height", "head_height", "nose_length"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.Extrude(sk, "width", "new")
}

// GroundedGibProfile draws the gib-head key's length silhouette on the XY plane and fully constrains
// it (DOF 0): a pentagon with the body's bottom on the X axis and its back (nose) face on the Y
// axis. Points, clockwise from the origin corner: A(0,0) bottom-back, B(length,0) bottom-front,
// C(length,height) body top-front, D(nose_length,height) where the body top meets the nose, and
// E(0,head_height) the nose top. Edge DE is the nose's sloped top; every other edge is axis-aligned.
// lengthExpr/heightExpr/headHeightExpr/noseLenExpr are parameter names or formulas that re-drive
// the key when the size changes.
func (s *SketchContext) GroundedGibProfile(lengthExpr, heightExpr, headHeightExpr, noseLenExpr string) error {
	// Seeds set only the branch/topology; the constraints below drive the real size (cm-scale).
	pts := [][]float64{{0, 0}, {3, 0}, {3, 0.7}, {1, 0.7}, {0, 1.1}}
	points, _, err := s.closedPolyline(pts)
	if err != nil {
		return err
	}
	return s.constrainGibProfile(points, lengthExpr, heightExpr, headHeightExpr, noseLenExpr)
}

// constrainGibProfile pins the five silhouette points to DOF 0. The back corner A is fixed at the
// origin; the bottom (A-B) and body-top (C-D) run horizontal, the front (B-C) and nose-back (E-A)
// run vertical; the four distances then size the body length/height, the nose height (A-E) and the
// body-top length that places the nose foot D (C-D = length − nose_length).
func (s *SketchContext) constrainGibProfile(p []uint64, lengthExpr, heightExpr, headHeightExpr, noseLenExpr string) error {
	a, b2, c, d, e := p[0], p[1], p[2], p[3], p[4]
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	if _, err := con.Fix(a); err != nil {
		return fmt.Errorf("fix gib origin corner: %w", err)
	}
	for _, h := range [][2]uint64{{a, b2}, {c, d}} {
		if _, err := con.Horizontal(h[0], h[1]); err != nil {
			return fmt.Errorf("gib horizontal edge %d-%d: %w", h[0], h[1], err)
		}
	}
	for _, v := range [][2]uint64{{b2, c}, {e, a}} {
		if _, err := con.Vertical(v[0], v[1]); err != nil {
			return fmt.Errorf("gib vertical edge %d-%d: %w", v[0], v[1], err)
		}
	}
	dims := []struct {
		p, q uint64
		expr string
	}{
		{a, b2, lengthExpr},    // body length
		{b2, c, heightExpr},    // body height
		{a, e, headHeightExpr}, // nose height
		{c, d, "(" + lengthExpr + ") - (" + noseLenExpr + ")"}, // body-top run → nose foot at nose_length
	}
	for _, dd := range dims {
		if _, err := dim.Distance(dd.p, dd.q, dd.expr); err != nil {
			return fmt.Errorf("gib dimension %d-%d %q: %w", dd.p, dd.q, dd.expr, err)
		}
	}
	return nil
}
