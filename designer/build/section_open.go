// SPDX-License-Identifier: GPL-2.0-only

package build

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
