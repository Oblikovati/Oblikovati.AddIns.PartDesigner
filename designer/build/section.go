// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"fmt"

	"oblikovati.org/api/client"
	"oblikovati.org/api/wire"
)

// edgeOffset pins one axis-aligned edge of a section a fixed perpendicular distance from the
// grounded origin — the parameter-driven half-dimension that both sizes and centres the section.
type edgeOffset struct {
	edge uint64
	expr string
}

// closedPolyline adds a closed outline through pts and returns its corner point ids and edge
// (line) ids, both in point order (edge i joins point i to point i+1, the last wrapping to the
// first). The seed coordinates only set the topology/branch the solver starts from; the
// constraints that follow drive the real size.
func (s *SketchContext) closedPolyline(pts [][]float64) (points, edges []uint64, err error) {
	res, err := s.b.api.Sketch().AddEntity(wire.AddSketchEntityArgs{
		SketchIndex: s.index, Kind: "polyline", Points: pts, Closed: true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("add closed polyline (%d points): %w", len(pts), err)
	}
	if len(res.PointIDs) != len(pts) || len(res.EntityIDs) != len(pts) {
		return nil, nil, fmt.Errorf("polyline reply: %d points / %d edges, want %d each",
			len(res.PointIDs), len(res.EntityIDs), len(pts))
	}
	return res.PointIDs, res.EntityIDs, nil
}

// groundedOrigin adds a point at the sketch origin and fixes it, giving the constrained-profile
// helpers a pinned reference to offset the section's edges from (which both sizes and centres
// the section on the origin).
func (s *SketchContext) groundedOrigin() (uint64, error) {
	res, err := s.b.api.Sketch().AddPoint(s.index, []float64{0, 0})
	if err != nil {
		return 0, fmt.Errorf("add origin point: %w", err)
	}
	if len(res.PointIDs) < 1 {
		return 0, fmt.Errorf("origin point reply had no point id")
	}
	if _, err := s.b.api.Sketch().Constrain(s.index).Fix(res.PointIDs[0]); err != nil {
		return 0, fmt.Errorf("fix origin point: %w", err)
	}
	return res.PointIDs[0], nil
}

// GroundedISection builds a doubly-symmetric I / wide-flange section (IPE, HE) centred on the
// sketch origin and fully constrains it to DOF 0. h/b/tw/tf are parameter expressions: total
// height, flange width, web thickness, flange thickness. The 12-vertex outline is aligned to the
// axes (every point sharing a coordinate is chained to one level) and pinned to the parameters by
// eight offset dimensions from a grounded origin — height and web at half, the flange-inner faces
// at h/2−tf — so the section stays centred (its neutral axis on the origin) and re-drives with the
// member. Root fillets are omitted: a faithful representational profile whose flanges and web
// carry the geometry.
func (s *SketchContext) GroundedISection(h, b, tw, tf string) error {
	const H, B, W, F = 10.0, 5.0, 0.3, 8.5 // seed half-dimensions (cm): H=h/2, B=b/2, W=tw/2, F=h/2−tf
	pts := [][]float64{
		{-B, H}, {B, H}, {B, F}, {W, F}, {W, -F}, {B, -F},
		{B, -H}, {-B, -H}, {-B, -F}, {-W, -F}, {-W, F}, {-B, F},
	}
	points, edges, err := s.closedPolyline(pts)
	if err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	return s.constrainISection(points, edges, o, h, b, tw, tf)
}

// constrainISection aligns the 12 outline points to their shared levels and pins those levels to
// the parameters. Points are in clockwise order from the top-left outer corner; edge i joins
// point i to point i+1 (see GroundedISection's seed order).
func (s *SketchContext) constrainISection(p, e []uint64, o uint64, h, b, tw, tf string) error {
	con := s.b.api.Sketch().Constrain(s.index)
	horiz := [][]uint64{{p[0], p[1]}, {p[2], p[3], p[10], p[11]}, {p[4], p[5], p[8], p[9]}, {p[6], p[7]}}
	vert := [][]uint64{{p[0], p[11], p[8], p[7]}, {p[1], p[2], p[5], p[6]}, {p[3], p[4]}, {p[9], p[10]}}
	if err := alignLevels(con, horiz, vert); err != nil {
		return err
	}
	fin := "(" + h + ") / 2 - (" + tf + ")"
	offsets := []edgeOffset{
		{e[0], half(h)}, {e[6], half(h)}, {e[2], fin}, {e[4], fin},
		{e[1], half(b)}, {e[11], half(b)}, {e[3], half(tw)}, {e[9], half(tw)},
	}
	return applyEdgeOffsets(s.b.api.Sketch().Dimension(s.index), o, offsets)
}

// GroundedChannelSection builds a channel / UPN section (symmetric about the X axis, web on the
// left) with its bounding box centred on the sketch origin and fully constrains it to DOF 0.
// h/b/tw/tf are parameter expressions: height, flange width, web thickness, flange thickness. The
// 8-vertex outline is axis-aligned and pinned by seven offset dimensions from a grounded origin.
// The flange is modelled at constant thickness (the UPN inner-flange taper and the toe/root radii
// are omitted) — a faithful representational profile.
func (s *SketchContext) GroundedChannelSection(h, b, tw, tf string) error {
	const H, B, TW, F = 10.0, 3.75, 0.85, 8.85 // seeds (cm): H=h/2, B=b/2, TW=tw, F=h/2−tf
	pts := [][]float64{
		{-B, H}, {B, H}, {B, F}, {-B + TW, F},
		{-B + TW, -F}, {B, -F}, {B, -H}, {-B, -H},
	}
	points, edges, err := s.closedPolyline(pts)
	if err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	return s.constrainChannelSection(points, edges, o, h, b, tw, tf)
}

// constrainChannelSection aligns the 8 outline points to their shared levels and pins those levels
// to the parameters. Points are clockwise from the top-left (web-back top) corner.
func (s *SketchContext) constrainChannelSection(p, e []uint64, o uint64, h, b, tw, tf string) error {
	con := s.b.api.Sketch().Constrain(s.index)
	horiz := [][]uint64{{p[0], p[1]}, {p[2], p[3]}, {p[4], p[5]}, {p[6], p[7]}}
	vert := [][]uint64{{p[1], p[2], p[5], p[6]}, {p[0], p[7]}, {p[3], p[4]}}
	if err := alignLevels(con, horiz, vert); err != nil {
		return err
	}
	fin := "(" + h + ") / 2 - (" + tf + ")"
	webFront := "(" + b + ") / 2 - (" + tw + ")"
	offsets := []edgeOffset{
		{e[0], half(h)}, {e[6], half(h)}, {e[2], fin}, {e[4], fin},
		{e[1], half(b)}, {e[7], half(b)}, {e[3], webFront},
	}
	return applyEdgeOffsets(s.b.api.Sketch().Dimension(s.index), o, offsets)
}

// half wraps an expression as its halved form for a centred half-dimension.
func half(expr string) string { return "(" + expr + ") / 2" }

// alignLevels chains a Horizontal constraint through every point group sharing a Y level and a
// Vertical constraint through every group sharing an X level, tying each group to a single
// coordinate so one offset dimension then pins the whole level.
func alignLevels(con client.Constrain, horiz, vert [][]uint64) error {
	for _, group := range horiz {
		if err := chainAlign(group, con.Horizontal); err != nil {
			return err
		}
	}
	for _, group := range vert {
		if err := chainAlign(group, con.Vertical); err != nil {
			return err
		}
	}
	return nil
}

// chainAlign applies rel between each consecutive pair of points, tying them all to one level.
func chainAlign(pts []uint64, rel func(a, b uint64) (wire.AddConstraintResult, error)) error {
	for i := 0; i+1 < len(pts); i++ {
		if _, err := rel(pts[i], pts[i+1]); err != nil {
			return fmt.Errorf("align points %d-%d: %w", pts[i], pts[i+1], err)
		}
	}
	return nil
}

// applyEdgeOffsets pins each edge a parameter-driven perpendicular distance from the grounded
// origin, sizing and centring the section in one pass.
func applyEdgeOffsets(dim client.Dimension, o uint64, offsets []edgeOffset) error {
	for _, off := range offsets {
		if _, err := dim.Offset(o, off.edge, off.expr); err != nil {
			return fmt.Errorf("offset edge %d to %q: %w", off.edge, off.expr, err)
		}
	}
	return nil
}
