// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// GroundedFilletedAngleSection builds an L / angle section (EN 10056, equal or unequal leg) with
// its real hot-rolled radii — a concave root fillet `r1` at the inner heel and a convex toe radius
// `r2` at each leg tip — anchored with its heel at the sketch origin and fully constrained to DOF 0.
// legA/legB/thickness/r1/r2 are parameter expressions. The outline is 6 straight edges + 3 arcs
// (A-leg toe, root, B-leg toe); like the filleted I-section it disables host inference so only the
// explicit constraints apply, orients every edge, and centre-pins every arc (see section_fillet.go).
//
// The heel outer corner stays sharp (EN 10056 rounds only the root and the two toes). The root
// fillet adds material; each toe radius rounds off (removes) the tip corner — so the extruded area
// matches the tabulated sectional area (e.g. L 40×40×4 → 3.08 cm²).
func (s *SketchContext) GroundedFilletedAngleSection(legA, legB, thickness, r1, r2 string) error {
	if err := s.disableSketchInference(); err != nil {
		return err
	}
	lines, arcs, err := s.addFilletedAngleEntities()
	if err != nil {
		return err
	}
	if err := s.closeFilletedAngleLoop(lines, arcs); err != nil {
		return err
	}
	if err := s.orientFilletedAngle(lines, arcs); err != nil {
		return err
	}
	if err := s.sizeFilletedAngle(lines, arcs, legA, legB, thickness, r1, r2); err != nil {
		return err
	}
	return s.assertNoRedundancy("filleted angle section")
}

// angleFilletSeeds are the 6 edges and 3 arcs of the filleted-angle outline at seed coordinates
// (cm) for a representative unequal angle (a=6, b=4, t=0.5, r1=0.6, r2=0.3). Walking CCW from the
// heel: bottom → A-end → [A-toe] → A-top → [root] → B-inner → [B-toe] → B-end → B-outer. The seeds
// only pick the solver branch — the constraints drive the true size.
func angleFilletSeeds() (lineSeeds [][2][]float64, arcSeeds [][3][]float64, ccw []bool) {
	lineSeeds = [][2][]float64{
		{{0, 0}, {6, 0}},         // 0 bottom (A-leg outer), horizontal
		{{6, 0}, {6, 0.2}},       // 1 A-leg end face, vertical
		{{5.7, 0.5}, {1.1, 0.5}}, // 2 A-leg top face, horizontal
		{{0.5, 1.1}, {0.5, 3.7}}, // 3 B-leg inner face, vertical
		{{0.2, 4.0}, {0, 4}},     // 4 B-leg end face, horizontal
		{{0, 4}, {0, 0}},         // 5 B-leg outer face, vertical
	}
	arcSeeds = [][3][]float64{
		{{5.7, 0.2}, {6, 0.2}, {5.7, 0.5}},   // 0 A-toe: centre, start(on A-end), end(on A-top) — convex
		{{1.1, 1.1}, {1.1, 0.5}, {0.5, 1.1}}, // 1 root: centre, start(on A-top), end(on B-inner) — concave
		{{0.2, 3.7}, {0.5, 3.7}, {0.2, 4.0}}, // 2 B-toe: centre, start(on B-inner), end(on B-end) — convex
	}
	ccw = []bool{true, false, true} // toes sweep CCW into the tip; the root sweeps CW into the heel
	return lineSeeds, arcSeeds, ccw
}

// addFilletedAngleEntities lays down the 6 edges and 3 arcs at their seeds and returns their records.
func (s *SketchContext) addFilletedAngleEntities() ([]filletLine, []filletArc, error) {
	lineSeeds, arcSeeds, ccw := angleFilletSeeds()
	sk := s.b.api.Sketch()
	lines := make([]filletLine, 0, len(lineSeeds))
	for _, p := range lineSeeds {
		res, err := sk.AddLine(s.index, p[0], p[1], false)
		if err != nil || len(res.PointIDs) < 2 {
			return nil, nil, fmt.Errorf("add filleted-angle edge: %w (points=%d)", err, len(res.PointIDs))
		}
		lines = append(lines, filletLine{res.EntityID, res.PointIDs[0], res.PointIDs[1]})
	}
	arcs := make([]filletArc, 0, len(arcSeeds))
	for i, p := range arcSeeds {
		res, err := sk.AddArcByCenterStartEnd(s.index, p[0], p[1], p[2], ccw[i], false)
		if err != nil || len(res.PointIDs) < 3 {
			return nil, nil, fmt.Errorf("add filleted-angle arc %d: %w (points=%d)", i, err, len(res.PointIDs))
		}
		arcs = append(arcs, filletArc{res.EntityID, res.PointIDs[0], res.PointIDs[1], res.PointIDs[2]})
	}
	return lines, arcs, nil
}

// closeFilletedAngleLoop joins the outline into one closed region (9 junctions), walking
// bottom → A-end → A-toe → A-top → root → B-inner → B-toe → B-end → B-outer → back to the heel.
func (s *SketchContext) closeFilletedAngleLoop(l []filletLine, a []filletArc) error {
	con := s.b.api.Sketch().Constrain(s.index)
	joins := [][2]uint64{
		{l[0].to, l[1].from}, {l[1].to, a[0].from}, {a[0].to, l[2].from}, {l[2].to, a[1].from},
		{a[1].to, l[3].from}, {l[3].to, a[2].from}, {a[2].to, l[4].from}, {l[4].to, l[5].from},
		{l[5].to, l[0].from},
	}
	for _, j := range joins {
		if _, err := con.Coincident(j[0], j[1]); err != nil {
			return fmt.Errorf("close filleted-angle loop at %d-%d: %w", j[0], j[1], err)
		}
	}
	return nil
}

// orientFilletedAngle fixes the heel at the origin, orients the six edges (bottom/A-top/B-end
// horizontal; A-end/B-inner/B-outer vertical), and centre-pins the three arcs axis-aligned: each
// toe's centre horizontal to its start and vertical to its end; the root's centre vertical to its
// start and horizontal to its end (see angleFilletSeeds for which endpoint sits where).
func (s *SketchContext) orientFilletedAngle(l []filletLine, a []filletArc) error {
	con := s.b.api.Sketch().Constrain(s.index)
	if _, err := con.Fix(l[0].from); err != nil {
		return fmt.Errorf("fix angle heel: %w", err)
	}
	for _, i := range []int{0, 2, 4} {
		if _, err := con.Horizontal(l[i].from, l[i].to); err != nil {
			return fmt.Errorf("orient angle edge %d horizontal: %w", i, err)
		}
	}
	for _, i := range []int{1, 3, 5} {
		if _, err := con.Vertical(l[i].from, l[i].to); err != nil {
			return fmt.Errorf("orient angle edge %d vertical: %w", i, err)
		}
	}
	pins := []struct {
		vertCentre, vertPt, horizCentre, horizPt uint64
	}{
		{a[0].centre, a[0].to, a[0].centre, a[0].from}, // A-toe: vertical to end, horizontal to start
		{a[1].centre, a[1].from, a[1].centre, a[1].to}, // root: vertical to start, horizontal to end
		{a[2].centre, a[2].to, a[2].centre, a[2].from}, // B-toe: vertical to end, horizontal to start
	}
	for i, p := range pins {
		if _, err := con.Vertical(p.vertCentre, p.vertPt); err != nil {
			return fmt.Errorf("pin angle arc %d vertical: %w", i, err)
		}
		if _, err := con.Horizontal(p.horizCentre, p.horizPt); err != nil {
			return fmt.Errorf("pin angle arc %d horizontal: %w", i, err)
		}
	}
	return nil
}

// sizeFilletedAngle shares the toe radius across the two toes (EqualRadius) and dimensions the root
// radius, a toe radius, and — as offsets of the leg faces from the fixed heel — the two leg lengths
// and the leg thickness (A-top and B-inner faces both at `thickness`).
func (s *SketchContext) sizeFilletedAngle(l []filletLine, a []filletArc, legA, legB, thickness, r1, r2 string) error {
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	if _, err := con.EqualRadius(a[0].id, a[2].id); err != nil {
		return fmt.Errorf("equal toe radius: %w", err)
	}
	if _, err := dim.Radius(a[1].id, r1); err != nil {
		return fmt.Errorf("dimension root radius %q: %w", r1, err)
	}
	if _, err := dim.Radius(a[0].id, r2); err != nil {
		return fmt.Errorf("dimension toe radius %q: %w", r2, err)
	}
	offsets := []edgeOffset{
		{l[1].id, legA},      // A-leg end face at x = legA
		{l[4].id, legB},      // B-leg end face at y = legB
		{l[2].id, thickness}, // A-leg top face at y = thickness
		{l[3].id, thickness}, // B-leg inner face at x = thickness
	}
	return applyEdgeOffsets(dim, l[0].from, offsets)
}
