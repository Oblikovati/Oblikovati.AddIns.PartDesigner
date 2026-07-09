// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// roundedRectArc is one corner arc of a rounded rectangle: its entity id plus its centre/start/end
// point ids (the order AddArcByCenterStartEnd returns).
type roundedRectArc struct {
	id               uint64
	centre, from, to uint64
}

// roundedRectLine is one straight edge: its entity id plus its start/end point ids.
type roundedRectLine struct {
	id       uint64
	from, to uint64
}

// GroundedRoundedRectangle builds an axis-aligned rounded rectangle centred on the sketch origin,
// fully constrained to DOF 0, so widthExpr/heightExpr (its extents) and radiusExpr (the corner
// radius) drive it — the profile of a cold-formed hollow section (EN 10219). It is built EXPLICITLY
// from four straight edges and four quarter arcs (not the sketch fillet primitive, which inserts
// free unconstrained arcs). The construction mirrors GroundedBallSection's proven pattern: each arc
// is pinned by orienting its centre to its endpoints (the quarter-arc's two radii are axis-aligned)
// plus one radius dimension shared across the four via EqualRadius; the four edges are positioned
// (and the section centred) by offset dimensions from the grounded origin.
func (s *SketchContext) GroundedRoundedRectangle(widthExpr, heightExpr, radiusExpr string) error {
	lines, arcs, err := s.addRoundedRectEntities()
	if err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	if err := s.closeRoundedRectLoop(lines, arcs); err != nil {
		return err
	}
	if err := s.orientRoundedRectArcs(arcs); err != nil {
		return err
	}
	return s.sizeRoundedRect(o, lines, arcs, widthExpr, heightExpr, radiusExpr)
}

// addRoundedRectEntities lays down the four edges (bottom, right, top, left) and four corner arcs
// (bottom-right, top-right, top-left, bottom-left, all CCW) at seed coordinates (cm). The seeds only
// set the branch the solver starts from; the constraints drive the real size.
func (s *SketchContext) addRoundedRectEntities() ([]roundedRectLine, []roundedRectArc, error) {
	const hw, hh, r = 2.0, 2.0, 0.6
	iw, ih := hw-r, hh-r
	lineSeeds := [][2][]float64{
		{{-iw, -hh}, {iw, -hh}}, {{hw, -ih}, {hw, ih}}, {{iw, hh}, {-iw, hh}}, {{-hw, ih}, {-hw, -ih}},
	}
	arcSeeds := [][3][]float64{
		{{iw, -ih}, {iw, -hh}, {hw, -ih}}, {{iw, ih}, {hw, ih}, {iw, hh}},
		{{-iw, ih}, {-iw, hh}, {-hw, ih}}, {{-iw, -ih}, {-hw, -ih}, {-iw, -hh}},
	}
	lines := make([]roundedRectLine, 0, 4)
	for _, p := range lineSeeds {
		res, err := s.b.api.Sketch().AddLine(s.index, p[0], p[1], false)
		if err != nil || len(res.PointIDs) < 2 {
			return nil, nil, fmt.Errorf("add rounded-rect edge: %w (points=%d)", err, len(res.PointIDs))
		}
		lines = append(lines, roundedRectLine{res.EntityID, res.PointIDs[0], res.PointIDs[1]})
	}
	arcs := make([]roundedRectArc, 0, 4)
	for _, p := range arcSeeds {
		res, err := s.b.api.Sketch().AddArcByCenterStartEnd(s.index, p[0], p[1], p[2], true, false)
		if err != nil || len(res.PointIDs) < 3 {
			return nil, nil, fmt.Errorf("add rounded-rect corner arc: %w (points=%d)", err, len(res.PointIDs))
		}
		arcs = append(arcs, roundedRectArc{res.EntityID, res.PointIDs[0], res.PointIDs[1], res.PointIDs[2]})
	}
	return lines, arcs, nil
}

// closeRoundedRectLoop joins the outline into a closed region: each edge's end coincides with the
// next arc's start and each arc's end with the following edge's start, walking bottom → BR → right →
// TR → top → TL → left → BL → bottom.
func (s *SketchContext) closeRoundedRectLoop(l []roundedRectLine, a []roundedRectArc) error {
	con := s.b.api.Sketch().Constrain(s.index)
	joins := [][2]uint64{
		{l[0].to, a[0].from}, {a[0].to, l[1].from}, {l[1].to, a[1].from}, {a[1].to, l[2].from},
		{l[2].to, a[2].from}, {a[2].to, l[3].from}, {l[3].to, a[3].from}, {a[3].to, l[0].from},
	}
	for _, j := range joins {
		if _, err := con.Coincident(j[0], j[1]); err != nil {
			return fmt.Errorf("close rounded-rect loop at %d-%d: %w", j[0], j[1], err)
		}
	}
	return nil
}

// orientRoundedRectArcs pins each quarter arc's shape: its two radii (centre→start, centre→end) are
// axis-aligned, one horizontal and one vertical, which — with the shared radius — makes it a proper
// quarter round tangent to the two edges it joins. This also implies the edges' own orientations, so
// no separate horizontal/vertical edge constraints are added (they would be redundant).
func (s *SketchContext) orientRoundedRectArcs(a []roundedRectArc) error {
	con := s.b.api.Sketch().Constrain(s.index)
	// Per corner (BR, TR, TL, BL) whether the centre→start radius is the vertical one; the CCW seed
	// layout alternates, so BR/TL have start below/above the centre and TR/BL have start beside it.
	startIsVertical := []bool{true, false, true, false}
	for i, arc := range a {
		vertPair, horizPair := [2]uint64{arc.centre, arc.from}, [2]uint64{arc.centre, arc.to}
		if !startIsVertical[i] {
			vertPair, horizPair = [2]uint64{arc.centre, arc.to}, [2]uint64{arc.centre, arc.from}
		}
		if _, err := con.Vertical(vertPair[0], vertPair[1]); err != nil {
			return fmt.Errorf("orient arc %d radius vertical: %w", i, err)
		}
		if _, err := con.Horizontal(horizPair[0], horizPair[1]); err != nil {
			return fmt.Errorf("orient arc %d radius horizontal: %w", i, err)
		}
	}
	return nil
}

// sizeRoundedRect drives the section's size and centres it: the four arcs share one radius
// (EqualRadius + a single radius dimension), and each edge is offset from the grounded origin by
// half the width (left/right) or half the height (top/bottom), which both sizes and centres it.
func (s *SketchContext) sizeRoundedRect(o uint64, l []roundedRectLine, a []roundedRectArc, w, h, r string) error {
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	for i := 0; i < 3; i++ {
		if _, err := con.EqualRadius(a[i].id, a[i+1].id); err != nil {
			return fmt.Errorf("equal corner radius %d: %w", i, err)
		}
	}
	if _, err := dim.Radius(a[0].id, r); err != nil {
		return fmt.Errorf("dimension corner radius %q: %w", r, err)
	}
	offsets := []edgeOffset{
		{l[0].id, half(h)}, {l[2].id, half(h)}, {l[1].id, half(w)}, {l[3].id, half(w)},
	}
	return applyEdgeOffsets(dim, o, offsets)
}
