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
