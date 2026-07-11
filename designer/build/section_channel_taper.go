// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"fmt"
	"math"

	"oblikovati.org/api/client"
	"oblikovati.org/part-designer/designer/catalog"
)

// pointDistance is a point-to-point distance dimension driven by an expression — how the tapered
// channel sizes its toe thickness and sloped-edge lengths.
type pointDistance struct {
	a, b uint64
	expr string
}

// applyDistances drives each point-to-point distance from its expression.
func applyDistances(dim client.Dimension, ds []pointDistance) error {
	for _, d := range ds {
		if _, err := dim.Distance(d.a, d.b, d.expr); err != nil {
			return fmt.Errorf("distance %d-%d = %q: %w", d.a, d.b, d.expr, err)
		}
	}
	return nil
}

// taperedChannelSeeds returns the 8 outline corner seeds (cm, the sketch's model unit) at the
// member's true tapered geometry, so the branch-sensitive sloped-edge distance constraints start
// on the correct solution for every size — not a fixed shape the solver must scale (and can flip)
// into. Toe thinner than root ⇒ the toe-inner corner sits higher than the root-inner corner.
// taperRefFrac is the fraction of the flange overhang, measured from the root, at which the
// reference flange thickness tf is taken (so the toe is tf − tan(taper)·overhang·taperRefFrac
// thick, the root tf + tan(taper)·overhang·(1−taperRefFrac)). A pure mid-overhang reference (0.5)
// is area-neutral; EN 10279 measures tf slightly toe-ward, which removes the small amount of
// flange material the root/toe fillets add back — calibrated so the extruded section area matches
// the published EN 10279 A to <0.5% across UPN 80…200 (the DIN 1026 measuring line is not
// published, so it is fixed from the authoritative tabulated area; see verify_upn.py) (#69).
const taperRefFrac = 0.565

func taperedChannelSeeds(rm ResolvedMember) [][]float64 {
	scale := 0.1 // mm → cm
	if rm.Family.Units == catalog.UnitsInch {
		scale = 2.54 // in → cm
	}
	H := rm.Value("h") * scale / 2
	B := rm.Value("b") * scale / 2
	tw := rm.Value("tw") * scale
	tf := rm.Value("tf") * scale
	slope := math.Tan(rm.Value("taper") * math.Pi / 180)
	overhang := 2*B - tw
	ftoe := H - (tf - slope*overhang*taperRefFrac)      // toe-inner y (toe flange thickness)
	froot := H - (tf + slope*overhang*(1-taperRefFrac)) // root-inner y (root flange thickness)
	x := -B + tw                                        // web-inner face x
	return [][]float64{
		{-B, H}, {B, H}, {B, ftoe}, {x, froot},
		{x, -froot}, {B, -ftoe}, {B, -H}, {-B, -H},
	}
}

// GroundedTaperedChannelSection builds a taper-flange channel (UPN per DIN 1026-1 / EN 10279,
// AISC C) centred on the sketch origin and fully constrains it to DOF 0. Unlike the parallel
// GroundedChannelSection, the inner flange faces slope inward at `taper` (an angle from the
// horizontal, 4.57° for the UPN 8% grade) so the flange is thicker at the root (web) than the toe,
// and the two web/flange root corners carry a fillet r1 (= tf) while the two flange toes carry a
// fillet r2. h/b/tw/tf are the usual section parameters; `seeds` are the member's true corner
// positions (see taperedChannelSeeds).
//
// It lays the sharp 8-vertex tapered outline, fully constrains it, rounds the four inner corners
// (the host consumes each corner + pins the arc tangent — Oblikovati#1943), then re-pins the
// trimmed faces to restore DOF 0 (#69).
func (s *SketchContext) GroundedTaperedChannelSection(seeds [][]float64, h, b, tw, tf, taper, r1, r2 string) error {
	points, edges, err := s.closedPolyline(seeds)
	if err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	if err := s.constrainTaperedChannel(points, edges, o, h, b, tw, tf, taper); err != nil {
		return err
	}
	blends := []struct {
		a, b uint64
		r    string
	}{
		{edges[1], edges[2], r2}, {edges[2], edges[3], r1},
		{edges[3], edges[4], r1}, {edges[4], edges[5], r2},
	}
	for _, bl := range blends {
		if _, err := s.Fillet(bl.a, bl.b, bl.r); err != nil {
			return err
		}
	}
	return s.repinTrimmedFaces(points, edges)
}

// repinTrimmedFaces re-pins the five faces the fillets trimmed — the fillet consumed each inner
// corner, orphaning the pre-fillet direction constraints that referenced it, leaving those edges
// free (8 DOF). The toe-ends and web-inner stay vertical (their far ends are still-pinned outer
// corners / positioned by their offset). Each sloped inner face is made collinear with its two
// inner corners (which the fillet KEPT, since the section's distance dimensions still reference
// them) — two points on the line fix both its position and direction. This restores DOF 0 on the
// already correctly-placed filleted geometry (#69).
func (s *SketchContext) repinTrimmedFaces(p, e []uint64) error {
	con := s.b.api.Sketch().Constrain(s.index)
	for _, ln := range []uint64{e[1], e[3], e[5]} { // toe-ends + web-inner run vertical
		if _, err := con.VerticalLine(ln); err != nil {
			return fmt.Errorf("re-pin vertical face %d: %w", ln, err)
		}
	}
	onLine := [][2]uint64{ // trimmed face ← its kept inner corners
		{p[2], e[2]}, {p[3], e[2]}, // top sloped through toe-inner + root-inner
		{p[5], e[4]}, {p[4], e[4]}, // bottom sloped through toe-inner + root-inner
		{p[3], e[3]}, // web-inner through the top root-inner (its offset + the p3/p4 chain pin the rest)
	}
	for _, ol := range onLine {
		if _, err := con.PointOnLine(ol[0], ol[1]); err != nil {
			return fmt.Errorf("re-pin sloped face %d through corner %d: %w", ol[1], ol[0], err)
		}
	}
	return nil
}

// constrainTaperedChannel pins the outer envelope (top/bottom horizontal, web-back and flange-tip
// verticals, web-inner vertical) with offset dimensions, then locates the two sloped inner faces
// by distances alone — the toe flange thickness and each sloped edge's length — which keeps the
// section symmetric about the X axis without an angle dimension. Points are clockwise from the
// web-back top corner (see GroundedTaperedChannelSection's seed order).
func (s *SketchContext) constrainTaperedChannel(p, e []uint64, o uint64, h, b, tw, tf, taper string) error {
	con := s.b.api.Sketch().Constrain(s.index)
	horiz := [][]uint64{{p[0], p[1]}, {p[6], p[7]}}
	vert := [][]uint64{{p[1], p[2], p[5], p[6]}, {p[0], p[7]}, {p[3], p[4]}}
	if err := alignLevels(con, horiz, vert); err != nil {
		return err
	}
	webFront := "(" + b + ") / 2 - (" + tw + ")"
	offsets := []edgeOffset{
		{e[0], half(h)}, {e[6], half(h)}, {e[1], half(b)}, {e[7], half(b)}, {e[3], webFront},
	}
	if err := applyEdgeOffsets(s.b.api.Sketch().Dimension(s.index), o, offsets); err != nil {
		return err
	}
	return s.pinTaperedFlanges(p, b, tw, tf, taper)
}

// pinTaperedFlanges locates the sloped inner faces: the toe-inner corner sits `toe` below its
// flange-tip corner (the toe flange thickness), and each sloped edge has length `slopedLen`
// (its overhang / cos(taper)), which fixes the root-inner corner on the already-placed web-inner
// line. toe = tf − tan(taper)·overhang·taperRefFrac (tf measured at the calibrated reference).
func (s *SketchContext) pinTaperedFlanges(p []uint64, b, tw, tf, taper string) error {
	dim := s.b.api.Sketch().Dimension(s.index)
	overhang := "((" + b + ") - (" + tw + "))"
	toe := fmt.Sprintf("(%s) - tan(%s) * %s * %g", tf, taper, overhang, taperRefFrac)
	slopedLen := overhang + " / cos(" + taper + ")"
	spans := []pointDistance{
		{p[1], p[2], toe}, {p[6], p[5], toe}, // toe flange thickness, top + bottom
		{p[2], p[3], slopedLen}, {p[5], p[4], slopedLen}, // sloped inner faces to the root corners
	}
	return applyDistances(dim, spans)
}
