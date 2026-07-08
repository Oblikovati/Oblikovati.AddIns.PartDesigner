// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"fmt"

	"oblikovati.org/api/client"
)

// pointDistance is a point-to-point distance dimension driven by a parameter expression — how the
// angle section, which is anchored at its heel rather than centred, sizes its legs and thickness.
type pointDistance struct {
	a, b uint64
	expr string
}

// GroundedAngleSection builds an L / angle section and fully constrains it to DOF 0 with the heel
// (the outer corner) at the sketch origin, the A leg running along +X and the B leg along +Y.
// legA/legB/thickness are parameter expressions. It serves both equal-leg (legA == legB) and
// unequal-leg angles from one outline. Unlike the doubly-symmetric shapes the angle has no axis
// symmetry, so it is anchored at the heel and sized by point-to-point distances rather than
// centred by offsets. Root and toe radii are omitted (sharp section).
func (s *SketchContext) GroundedAngleSection(legA, legB, thickness string) error {
	const A, B, T = 8.0, 6.0, 0.6 // seeds (cm): A leg along +X, B leg along +Y, thickness T
	pts := [][]float64{{0, 0}, {A, 0}, {A, T}, {T, T}, {T, B}, {0, B}}
	points, _, err := s.closedPolyline(pts)
	if err != nil {
		return err
	}
	return s.constrainAngle(points, legA, legB, thickness)
}

// constrainAngle aligns the six outline points to their shared levels, fixes the heel at the
// origin, and sizes the two legs and the two leg thicknesses. Points are counter-clockwise from
// the heel (see GroundedAngleSection's seed order).
func (s *SketchContext) constrainAngle(p []uint64, legA, legB, thickness string) error {
	con := s.b.api.Sketch().Constrain(s.index)
	horiz := [][]uint64{{p[0], p[1]}, {p[2], p[3]}, {p[4], p[5]}}
	vert := [][]uint64{{p[0], p[5]}, {p[3], p[4]}, {p[1], p[2]}}
	if err := alignLevels(con, horiz, vert); err != nil {
		return err
	}
	if _, err := con.Fix(p[0]); err != nil {
		return fmt.Errorf("fix angle heel: %w", err)
	}
	distances := []pointDistance{
		{p[0], p[1], legA}, {p[0], p[5], legB},
		{p[1], p[2], thickness}, {p[5], p[4], thickness},
	}
	return applyDistances(s.b.api.Sketch().Dimension(s.index), distances)
}

// GroundedTeeSection builds a T / tee section (symmetric about the Y axis, flange on top, stem
// hanging down) with its bounding box centred on the sketch origin and fully constrains it to
// DOF 0. h/b/tw/tf are parameter expressions: overall height, flange width, stem (web) thickness,
// flange thickness. The 8-vertex outline is axis-aligned and pinned by seven offset dimensions
// from a grounded origin. Root and toe radii are omitted (sharp section).
func (s *SketchContext) GroundedTeeSection(h, b, tw, tf string) error {
	const H, B, W, F = 8.0, 8.0, 0.6, 6.5 // seeds (cm): H=h/2, B=b/2, W=tw/2, F=h/2−tf
	pts := [][]float64{
		{-B, H}, {B, H}, {B, F}, {W, F},
		{W, -H}, {-W, -H}, {-W, F}, {-B, F},
	}
	points, edges, err := s.closedPolyline(pts)
	if err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	return s.constrainTeeSection(points, edges, o, h, b, tw, tf)
}

// constrainTeeSection aligns the 8 outline points to their shared levels and pins those levels to
// the parameters. Points are clockwise from the flange top-left corner.
func (s *SketchContext) constrainTeeSection(p, e []uint64, o uint64, h, b, tw, tf string) error {
	con := s.b.api.Sketch().Constrain(s.index)
	horiz := [][]uint64{{p[0], p[1]}, {p[2], p[3], p[6], p[7]}, {p[4], p[5]}}
	vert := [][]uint64{{p[0], p[7]}, {p[1], p[2]}, {p[3], p[4]}, {p[5], p[6]}}
	if err := alignLevels(con, horiz, vert); err != nil {
		return err
	}
	flangeInner := "(" + h + ") / 2 - (" + tf + ")"
	offsets := []edgeOffset{
		{e[0], half(h)}, {e[4], half(h)}, {e[2], flangeInner},
		{e[1], half(b)}, {e[7], half(b)}, {e[3], half(tw)}, {e[5], half(tw)},
	}
	return applyEdgeOffsets(s.b.api.Sketch().Dimension(s.index), o, offsets)
}

// applyDistances drives each point-to-point distance from its parameter expression.
func applyDistances(dim client.Dimension, distances []pointDistance) error {
	for _, d := range distances {
		if _, err := dim.Distance(d.a, d.b, d.expr); err != nil {
			return fmt.Errorf("distance %d-%d = %q: %w", d.a, d.b, d.expr, err)
		}
	}
	return nil
}
