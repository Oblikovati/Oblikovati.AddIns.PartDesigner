// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// filletLine / filletArc are one straight edge / one root-fillet arc of a filleted open section:
// the entity id plus its endpoint ids (arcs also carry their centre). They mirror the roundedRect*
// records but are named for the open (I/channel) sections whose reentrant web-flange corners carry
// a concave root fillet.
type filletLine struct {
	id       uint64
	from, to uint64
}

type filletArc struct {
	id               uint64
	centre, from, to uint64
}

// GroundedFilletedISection builds a doubly-symmetric I / wide-flange section (IPE, HE) with a real
// concave root fillet at each of the four web-flange junctions, fully constrained to DOF 0. h/b/tw/
// tf/r are parameter expressions (height, flange width, web thickness, flange thickness, root
// radius). It replaces GroundedISection's sharp reentrant corners with quarter-round arcs tangent
// to the web face and the flange underside — the fillet a hot-rolled section actually carries,
// which improves the section-property fidelity of the extruded beam.
//
// Each fillet reuses the proven center-pinned arc recipe (see GroundedRoundedRectangle): the arc's
// centre sits in the air quadrant, pinned Vertical to its flange-side endpoint and Horizontal to
// its web-side endpoint; one radius dimension shared across the four arcs via EqualRadius sizes
// them. The radius together with the web (half-tw) and flange-underside (h/2−tf) offsets then fixes
// each tangent point at web±r / underside∓r automatically, so no extra fillet dimensions are
// needed and the section stays centred on the origin.
func (s *SketchContext) GroundedFilletedISection(h, b, tw, tf, r string) error {
	if err := s.disableSketchInference(); err != nil {
		return err
	}
	lines, arcs, err := s.addFilletedIEntities()
	if err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	if err := s.closeFilletedILoop(lines, arcs); err != nil {
		return err
	}
	if err := s.orientRootFillets(arcs); err != nil {
		return err
	}
	if err := s.orientFilletedIEdges(lines); err != nil {
		return err
	}
	if err := s.sizeFilletedISection(o, lines, arcs, h, b, tw, tf, r); err != nil {
		return err
	}
	return s.assertNoRedundancy("filleted I-section")
}

// iFilletSeeds are the 12 straight edges and 4 root-fillet arcs of the filleted I-outline at seed
// coordinates (cm) for a representative IPE 200 (H=10, B=5, W=0.28, F=9.15, r=1.2). Walking
// clockwise from the top-left flange corner; the arcs replace the four reentrant web-flange
// corners. The seeds only pick the solver branch — the constraints drive the true size.
func iFilletSeeds() (lineSeeds [][2][]float64, arcSeeds [][3][]float64) {
	lineSeeds = [][2][]float64{
		{{-5, 10}, {5, 10}},             // 0 top flange top
		{{5, 10}, {5, 9.15}},            // 1 top flange right edge
		{{5, 9.15}, {1.48, 9.15}},       // 2 top flange underside (right)
		{{0.28, 7.95}, {0.28, -7.95}},   // 3 web right face
		{{1.48, -9.15}, {5, -9.15}},     // 4 bottom flange underside (right)
		{{5, -9.15}, {5, -10}},          // 5 bottom flange right edge
		{{5, -10}, {-5, -10}},           // 6 bottom flange bottom
		{{-5, -10}, {-5, -9.15}},        // 7 bottom flange left edge
		{{-5, -9.15}, {-1.48, -9.15}},   // 8 bottom flange underside (left)
		{{-0.28, -7.95}, {-0.28, 7.95}}, // 9 web left face
		{{-1.48, 9.15}, {-5, 9.15}},     // 10 top flange underside (left)
		{{-5, 9.15}, {-5, 10}},          // 11 top flange left edge
	}
	arcSeeds = [][3][]float64{
		{{1.48, 7.95}, {1.48, 9.15}, {0.28, 7.95}},       // 0 top-right: centre, start(underside), end(web)
		{{1.48, -7.95}, {0.28, -7.95}, {1.48, -9.15}},    // 1 bottom-right: centre, start(web), end(underside)
		{{-1.48, -7.95}, {-1.48, -9.15}, {-0.28, -7.95}}, // 2 bottom-left: centre, start(underside), end(web)
		{{-1.48, 7.95}, {-0.28, 7.95}, {-1.48, 9.15}},    // 3 top-left: centre, start(web), end(underside)
	}
	return lineSeeds, arcSeeds
}

// addFilletedIEntities lays down the 12 edges and 4 fillet arcs at their seeds and returns their
// records. Arcs are wound CCW (start→end sweeps the quarter into the air quadrant).
func (s *SketchContext) addFilletedIEntities() ([]filletLine, []filletArc, error) {
	lineSeeds, arcSeeds := iFilletSeeds()
	sk := s.b.api.Sketch()
	lines := make([]filletLine, 0, len(lineSeeds))
	for _, p := range lineSeeds {
		res, err := sk.AddLine(s.index, p[0], p[1], false)
		if err != nil || len(res.PointIDs) < 2 {
			return nil, nil, fmt.Errorf("add filleted-I edge: %w (points=%d)", err, len(res.PointIDs))
		}
		lines = append(lines, filletLine{res.EntityID, res.PointIDs[0], res.PointIDs[1]})
	}
	arcs := make([]filletArc, 0, len(arcSeeds))
	for _, p := range arcSeeds {
		res, err := sk.AddArcByCenterStartEnd(s.index, p[0], p[1], p[2], true, false)
		if err != nil || len(res.PointIDs) < 3 {
			return nil, nil, fmt.Errorf("add filleted-I root fillet: %w (points=%d)", err, len(res.PointIDs))
		}
		arcs = append(arcs, filletArc{res.EntityID, res.PointIDs[0], res.PointIDs[1], res.PointIDs[2]})
	}
	return lines, arcs, nil
}

// closeFilletedILoop joins the outline into one closed region, walking the 16 line/arc junctions
// clockwise: line0→line1→line2→arc0→line3→arc1→line4…→line11→line0.
func (s *SketchContext) closeFilletedILoop(l []filletLine, a []filletArc) error {
	con := s.b.api.Sketch().Constrain(s.index)
	joins := [][2]uint64{
		{l[0].to, l[1].from}, {l[1].to, l[2].from}, {l[2].to, a[0].from}, {a[0].to, l[3].from},
		{l[3].to, a[1].from}, {a[1].to, l[4].from}, {l[4].to, l[5].from}, {l[5].to, l[6].from},
		{l[6].to, l[7].from}, {l[7].to, l[8].from}, {l[8].to, a[2].from}, {a[2].to, l[9].from},
		{l[9].to, a[3].from}, {a[3].to, l[10].from}, {l[10].to, l[11].from}, {l[11].to, l[0].from},
	}
	for _, j := range joins {
		if _, err := con.Coincident(j[0], j[1]); err != nil {
			return fmt.Errorf("close filleted-I loop at %d-%d: %w", j[0], j[1], err)
		}
	}
	return nil
}

// orientRootFillets pins each fillet arc's two radii axis-aligned: the flange-side radius vertical
// and the web-side radius horizontal (the arc's centre sits diagonally out in the air quadrant).
// Arcs 0 and 2 have their start on the flange underside (vertical radius); arcs 1 and 3 have their
// start on the web face (horizontal radius) — see iFilletSeeds.
func (s *SketchContext) orientRootFillets(a []filletArc) error {
	con := s.b.api.Sketch().Constrain(s.index)
	startOnFlange := []bool{true, false, true, false}
	for i, arc := range a {
		vertPair, horizPair := [2]uint64{arc.centre, arc.from}, [2]uint64{arc.centre, arc.to}
		if !startOnFlange[i] {
			vertPair, horizPair = [2]uint64{arc.centre, arc.to}, [2]uint64{arc.centre, arc.from}
		}
		if _, err := con.Vertical(vertPair[0], vertPair[1]); err != nil {
			return fmt.Errorf("orient root fillet %d flange radius vertical: %w", i, err)
		}
		if _, err := con.Horizontal(horizPair[0], horizPair[1]); err != nil {
			return fmt.Errorf("orient root fillet %d web radius horizontal: %w", i, err)
		}
	}
	return nil
}

// disableSketchInference turns off the host's constraint inference on the active document before
// the filleted section is built. The host auto-applies inferred horizontal/vertical/coincidence
// relations to each line as it is committed (the interactive-sketching convenience); for a
// procedurally, fully-explicitly constrained profile those inferred relations duplicate the ones
// the generator adds, producing redundant constraints that let the solver settle on a degenerate
// (self-intersecting) configuration. Disabling inference makes the build deterministic — only the
// generator's explicit constraints apply — so the section reaches DOF 0 with redundant 0. Document
// 0 is the active (just-created) part, so this is scoped to this generated part.
func (s *SketchContext) disableSketchInference() error {
	cur, err := s.b.api.Documents().GetSketchSettings(0)
	if err != nil {
		return fmt.Errorf("read sketch settings: %w", err)
	}
	next := cur.Settings
	next.InferConstraints = false
	next.AutoApplyConstraints = false
	if _, err := s.b.api.Documents().SetSketchSettings(0, next); err != nil {
		return fmt.Errorf("disable sketch inference: %w", err)
	}
	return nil
}

// orientFilletedIEdges gives every straight edge an explicit horizontal/vertical constraint: the
// six horizontal edges (flange tops/bottoms and the four flange undersides) and the six vertical
// edges (the four flange sides and the two web faces). With inference disabled, the arc centre-pins
// alone do NOT imply the arc-adjacent edges' orientation, so all twelve are stated explicitly
// (verified against the host solver: DOF 0, redundant 0).
func (s *SketchContext) orientFilletedIEdges(l []filletLine) error {
	con := s.b.api.Sketch().Constrain(s.index)
	for _, i := range []int{0, 2, 4, 6, 8, 10} { // flange tops/bottoms + the four undersides
		if _, err := con.Horizontal(l[i].from, l[i].to); err != nil {
			return fmt.Errorf("orient edge %d horizontal: %w", i, err)
		}
	}
	for _, i := range []int{1, 3, 5, 7, 9, 11} { // the four flange sides + the two web faces
		if _, err := con.Vertical(l[i].from, l[i].to); err != nil {
			return fmt.Errorf("orient edge %d vertical: %w", i, err)
		}
	}
	return nil
}

// sizeFilletedISection shares one radius across the four fillets, dimensions it, and pins EVERY
// straight edge by an absolute offset from the grounded origin — height and web at half, flange
// undersides at h/2−tf, flange width at half. Positioning every edge absolutely (never relative to
// another edge) is what avoids loop-closure redundancy; the offsets both size and centre the
// section. The two web faces and the two flange undersides get independent offsets (never chained
// left-to-right), and the fillet tangent points are left to follow from the arcs.
func (s *SketchContext) sizeFilletedISection(o uint64, l []filletLine, a []filletArc, h, b, tw, tf, r string) error {
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	for i := 0; i < 3; i++ {
		if _, err := con.EqualRadius(a[i].id, a[i+1].id); err != nil {
			return fmt.Errorf("equal root fillet radius %d: %w", i, err)
		}
	}
	if _, err := dim.Radius(a[0].id, r); err != nil {
		return fmt.Errorf("dimension root fillet radius %q: %w", r, err)
	}
	fin := "(" + h + ") / 2 - (" + tf + ")"
	offsets := []edgeOffset{
		{l[0].id, half(h)}, {l[6].id, half(h)}, // flange top / bottom
		{l[1].id, half(b)}, {l[5].id, half(b)}, {l[7].id, half(b)}, {l[11].id, half(b)}, // flange sides
		{l[2].id, fin}, {l[4].id, fin}, {l[8].id, fin}, {l[10].id, fin}, // flange undersides
		{l[3].id, half(tw)}, {l[9].id, half(tw)}, // web faces
	}
	return applyEdgeOffsets(dim, o, offsets)
}

// assertNoRedundancy fails a filleted-section build when the sketch carries redundant constraints.
// A redundant sketch can report DOF 0 yet let the solver settle on a degenerate (self-intersecting)
// configuration that extrudes to nothing, so the arc-based sections verify redundant == 0 at the
// source rather than discovering the empty solid downstream.
func (s *SketchContext) assertNoRedundancy(what string) error {
	st, err := s.b.api.Sketch().ConstraintStatus(s.index)
	if err != nil {
		return fmt.Errorf("constraint status of %s (sketch %d): %w", what, s.index, err)
	}
	if st.Redundant != 0 || st.DOF != 0 {
		return fmt.Errorf("%s not cleanly constrained: DOF=%d redundant=%d (want 0/0)",
			what, st.DOF, st.Redundant)
	}
	return nil
}
