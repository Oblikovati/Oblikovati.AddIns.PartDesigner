// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"fmt"

	"oblikovati.org/api/client"
)

// GroundedConeSection builds the trapezoidal cross-section of a tapered-roller bearing's inner ring
// (the "cone"): a straight bore (vertical inner edge at vertDia) and a sloped outer raceway running
// from slopeBottomDia at −width/2 to slopeTopDia at +width/2. Revolved about Z it yields a truncated
// cone whose outer surface is the angled raceway the rollers run on. It is fully constrained (DOF 0).
func (s *SketchContext) GroundedConeSection(vertDia, slopeBottomDia, slopeTopDia, width string) error {
	return s.groundedTaperRing(vertDia, slopeBottomDia, slopeTopDia, width, true)
}

// GroundedCupSection builds the trapezoidal cross-section of a tapered-roller bearing's outer ring
// (the "cup"): a straight outside diameter (vertical outer edge at vertDia) and a sloped inner
// raceway from slopeBottomDia at −width/2 to slopeTopDia at +width/2. Revolved about Z it yields a
// truncated-cone shell whose inner surface is the angled raceway. Fully constrained (DOF 0).
func (s *SketchContext) GroundedCupSection(vertDia, slopeBottomDia, slopeTopDia, width string) error {
	return s.groundedTaperRing(vertDia, slopeBottomDia, slopeTopDia, width, false)
}

// groundedTaperRing builds and fully constrains a 4-point ring cross-section with one axis-parallel
// edge (at vertDia) and one sloped edge (slopeBottomDia at the bottom, slopeTopDia at the top),
// centred axially on the origin. innerVertical picks whether the straight edge is the inner one (a
// cone: straight bore, sloped outer) or the outer one (a cup: straight OD, sloped inner). The
// scheme mirrors the rectangular ring section but replaces one vertical edge with the sloped
// raceway, whose two corners are pinned by their radial distance from the straight edge.
func (s *SketchContext) groundedTaperRing(vertDia, slopeBottomDia, slopeTopDia, width string, innerVertical bool) error {
	seeds := taperRingSeeds(innerVertical)
	pts, edges, err := s.closedPolyline(seeds)
	if err != nil {
		return err
	}
	bl, br, tr, tl := pts[0], pts[1], pts[2], pts[3]
	bottom, right, top, left := edges[0], edges[1], edges[2], edges[3]
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	if _, err := con.Horizontal(bl, br); err != nil {
		return fmt.Errorf("taper ring bottom horizontal: %w", err)
	}
	if _, err := con.Horizontal(tl, tr); err != nil {
		return fmt.Errorf("taper ring top horizontal: %w", err)
	}
	if _, err := dim.Offset(o, bottom, half(width)); err != nil {
		return fmt.Errorf("centre taper ring bottom: %w", err)
	}
	if _, err := dim.Offset(o, top, half(width)); err != nil {
		return fmt.Errorf("centre taper ring top: %w", err)
	}
	return s.pinTaperEdges(con, dim, o, taperEdgeRefs{
		bl: bl, br: br, tr: tr, tl: tl, right: right, left: left,
	}, vertDia, slopeBottomDia, slopeTopDia, innerVertical)
}

// taperEdgeRefs bundles the corner and side ids groundedTaperRing pins, so pinTaperEdges stays
// within the argument budget while keeping the roles named at the call site.
type taperEdgeRefs struct {
	bl, br, tr, tl uint64
	right, left    uint64
}

// pinTaperEdges makes the straight edge vertical + placed at its radius, then pins the two sloped
// corners by their radial width from the straight edge — completing the DOF-0 trapezoid.
func (s *SketchContext) pinTaperEdges(con client.Constrain, dim client.Dimension, o uint64, e taperEdgeRefs, vertDia, slopeBottomDia, slopeTopDia string, innerVertical bool) error {
	vertEdge, bottomWidth, topWidth := e.left, radialWidth(slopeBottomDia, vertDia), radialWidth(slopeTopDia, vertDia)
	if !innerVertical {
		vertEdge, bottomWidth, topWidth = e.right, radialWidth(vertDia, slopeBottomDia), radialWidth(vertDia, slopeTopDia)
	}
	if innerVertical {
		if _, err := con.Vertical(e.bl, e.tl); err != nil {
			return fmt.Errorf("taper ring inner edge vertical: %w", err)
		}
	} else if _, err := con.Vertical(e.br, e.tr); err != nil {
		return fmt.Errorf("taper ring outer edge vertical: %w", err)
	}
	if _, err := dim.Offset(o, vertEdge, half(vertDia)); err != nil {
		return fmt.Errorf("place taper ring straight edge: %w", err)
	}
	if _, err := dim.Distance(e.bl, e.br, bottomWidth); err != nil {
		return fmt.Errorf("dimension taper ring bottom width: %w", err)
	}
	if _, err := dim.Distance(e.tl, e.tr, topWidth); err != nil {
		return fmt.Errorf("dimension taper ring top width: %w", err)
	}
	return nil
}

// radialWidth is the half-difference of two diameters as an expression — the radial span of one
// horizontal edge of the trapezoid.
func radialWidth(outerDia, innerDia string) string {
	return "((" + outerDia + ") - (" + innerDia + ")) / 2"
}

// taperRingSeeds returns non-degenerate seed corners (cm) for the trapezoid — inner edge nearer the
// axis for a cone, outer edge for a cup — ordered CCW (BL, BR, TR, TL); the dimensions drive the
// real geometry.
func taperRingSeeds(innerVertical bool) [][]float64 {
	if innerVertical { // straight bore at 2.0, sloped outer opening from 2.5 to 3.0
		return [][]float64{{2.0, -0.5}, {2.5, -0.5}, {3.0, 0.5}, {2.0, 0.5}}
	}
	// straight OD at 3.0, sloped inner from 2.5 to 2.0
	return [][]float64{{2.5, -0.5}, {3.0, -0.5}, {3.0, 0.5}, {2.0, 0.5}}
}

// GroundedRibbedConeSection builds the inner ring ("cone") of a tapered-roller bearing with a
// big-end retaining rib: a 6-vertex L-profile with a straight bore, a sloped raceway running from
// the small-end face up to the rib foot, and a rib that rises to the crest and fills out to the
// big-end face. Revolved about Z it yields a cone ring whose raised big-end flange guides the roller
// big ends. bore is the straight bore diameter; raceSmallDia / raceRibDia are the raceway diameters
// at the small-end face and at the rib foot; ribCrestDia is the flange crest diameter; ribInnerZ is
// the rib-foot axial position (the roller big ends stop just short of it). Fully constrained (DOF 0).
//
// The rib foot sits axially beyond the roller big end, so the revolved rib solid stays disjoint from
// the patterned rollers (see the geometry-math-advisor rib derivation, #54).
func (s *SketchContext) GroundedRibbedConeSection(bore, raceSmallDia, raceRibDia, ribCrestDia, ribInnerZ, width string) error {
	pts, edges, err := s.closedPolyline(ribbedConeSeeds())
	if err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	if err := s.orientRibbedCone(pts); err != nil {
		return err
	}
	return s.pinRibbedCone(o, pts, edges, ribbedConeDims{
		bore: bore, raceSmall: raceSmallDia, raceRib: raceRibDia,
		ribCrest: ribCrestDia, ribInnerZ: ribInnerZ, width: width,
	})
}

// ribbedConeDims bundles the six parameter expressions the ribbed cone section pins, keeping
// pinRibbedCone within the argument budget while naming each role at the call site.
type ribbedConeDims struct {
	bore, raceSmall, raceRib, ribCrest, ribInnerZ, width string
}

// ribbedConeSeeds returns the 6 outline corners (cm), CCW from the small-end bore corner: bore/small
// face, raceway small, raceway at the rib foot, rib crest foot, rib crest / big face, big-end bore.
// Seeds only pick the branch; the dimensions drive the real geometry.
func ribbedConeSeeds() [][]float64 {
	return [][]float64{
		{1.5, -0.86}, // A bore, small-end face
		{1.8, -0.86}, // B raceway, small-end face
		{2.0, 0.63},  // C raceway at the rib foot
		{2.67, 0.63}, // D rib crest foot
		{2.67, 0.86}, // E rib crest, big-end face
		{1.5, 0.86},  // F bore, big-end face
	}
}

// orientRibbedCone aligns the axis-parallel and radial edges: the small-end face, rib inner face and
// big-end face share their axial level (Horizontal); the rib crest and the bore share their radius
// (Vertical). The sloped raceway (B→C) is left free, pinned by its endpoint radii.
func (s *SketchContext) orientRibbedCone(p []uint64) error {
	con := s.b.api.Sketch().Constrain(s.index)
	horiz := [][2]uint64{{p[0], p[1]}, {p[2], p[3]}, {p[4], p[5]}} // small face, rib inner face, big face
	vert := [][2]uint64{{p[3], p[4]}, {p[5], p[0]}}                // rib crest, bore
	for _, h := range horiz {
		if _, err := con.Horizontal(h[0], h[1]); err != nil {
			return fmt.Errorf("orient ribbed cone edge %d-%d horizontal: %w", h[0], h[1], err)
		}
	}
	for _, v := range vert {
		if _, err := con.Vertical(v[0], v[1]); err != nil {
			return fmt.Errorf("orient ribbed cone edge %d-%d vertical: %w", v[0], v[1], err)
		}
	}
	return nil
}

// pinRibbedCone pins every edge to its parameter: the three faces to their axial levels, the bore
// and rib crest to their radii, and the two raceway corners by their radial width from the bore (B)
// and from the rib crest (C). Edge order matches ribbedConeSeeds (edge i joins point i to i+1).
func (s *SketchContext) pinRibbedCone(o uint64, p, e []uint64, d ribbedConeDims) error {
	dim := s.b.api.Sketch().Dimension(s.index)
	offsets := []edgeOffset{
		{e[0], half(d.width)},    // small-end face at −width/2
		{e[4], half(d.width)},    // big-end face at +width/2
		{e[2], d.ribInnerZ},      // rib inner face at +rib_inner_z
		{e[5], half(d.bore)},     // bore at bore/2
		{e[3], half(d.ribCrest)}, // rib crest at rib_crest/2
	}
	if err := applyEdgeOffsets(dim, o, offsets); err != nil {
		return err
	}
	if _, err := dim.Distance(p[0], p[1], radialWidth(d.raceSmall, d.bore)); err != nil {
		return fmt.Errorf("dimension ribbed cone raceway small end: %w", err)
	}
	if _, err := dim.Distance(p[3], p[2], radialWidth(d.ribCrest, d.raceRib)); err != nil {
		return fmt.Errorf("dimension ribbed cone raceway at rib foot: %w", err)
	}
	return nil
}
