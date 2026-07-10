// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// GroundedFlangedRingSection builds the cylindrical-roller outer ring's revolved meridian section
// (XZ plane, X = radius, Z = axial): an inward-opening ⊐ channel carrying two integral guide
// flanges that axially locate the rollers. edgeDia is the ring's outer cylindrical edge
// (outer_dia); raceDia is the raceway land diameter the two flanges' inner shoulders step out to
// (outer_race_dia, where the rollers actually ride); flangeBoreDia is the flange rim's bore
// (flange_bore_dia = pitch_dia, so the rib clears the plain inner ring with a land while
// overlapping the roller crest); innerZ is the flange's inner-shoulder |z| (flange_inner_z, the
// roller-end clearance band's inner edge); width is the ring's axial span. The 8-point outline is
// axis-aligned throughout (see the geometry-math-advisor derivation, #53):
//
//	P1 (D/2,-w/2) P2 (D/2,+w/2) P3 (boreDia/2,+w/2) P4 (boreDia/2,+innerZ)
//	P5 (raceDia/2,+innerZ) P6 (raceDia/2,-innerZ) P7 (boreDia/2,-innerZ) P8 (boreDia/2,-w/2)
//
// P1-P2 is the outer wall (full width); P2-P3/P8-P1 are the flange end faces (outer wall down to
// the flange bore, at the ring's true ends); P3-P4/P7-P8 are the flange ribs' inner bore faces;
// P4-P5/P6-P7 are the flange inner shoulders (stepping out to the raceway); P5-P6 is the raceway
// the rollers ride on. Fully constrained to DOF 0 and centred axially on the origin.
func (s *SketchContext) GroundedFlangedRingSection(edgeDia, raceDia, flangeBoreDia, innerZ, width string) error {
	if err := s.disableSketchInference(); err != nil {
		return err
	}
	const D, W, B, Z, R = 3.0, 0.9, 2.1, 0.35, 2.4 // seeds (cm): D=edge/2, W=width/2, B=boreDia/2, Z=flange_inner_z, R=raceDia/2
	pts := [][]float64{
		{D, -W}, {D, W}, {B, W}, {B, Z},
		{R, Z}, {R, -Z}, {B, -Z}, {B, -W},
	}
	points, edges, err := s.closedPolyline(pts)
	if err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	return s.constrainFlangedRing(points, edges, o, edgeDia, raceDia, flangeBoreDia, innerZ, width)
}

// constrainFlangedRing orients the 8-edge ⊐ outline axis-aligned and pins its three distinct radii
// and two distinct z-magnitudes to the parameters.
func (s *SketchContext) constrainFlangedRing(p, e []uint64, o uint64, edgeDia, raceDia, flangeBoreDia, innerZ, width string) error {
	if err := s.orientFlangedRing(p); err != nil {
		return err
	}
	return s.sizeFlangedRing(e, o, edgeDia, raceDia, flangeBoreDia, innerZ, width)
}

// orientFlangedRing makes every edge of the ⊐ outline Horizontal or Vertical. The two flange-rib
// bore edges (P3-P4, P7-P8) sit at the same radius (flangeBoreDia/2) though not adjacent, so their
// four corners are chained into one Vertical group — that single shared level then takes one
// dimension for both ribs, mirroring alignLevels' cross-outline level idiom (see GroundedISection).
func (s *SketchContext) orientFlangedRing(p []uint64) error {
	con := s.b.api.Sketch().Constrain(s.index)
	horiz := [][]uint64{{p[1], p[2]}, {p[3], p[4]}, {p[5], p[6]}, {p[7], p[0]}}
	vert := [][]uint64{{p[0], p[1]}, {p[2], p[3], p[6], p[7]}, {p[4], p[5]}}
	return alignLevels(con, horiz, vert)
}

// sizeFlangedRing pins the three distinct radii (outer edge, flange bore ×2 ribs tied as one
// level, raceway land) and the two distinct z-magnitudes (±width/2 flange ends, ±flange_inner_z
// shoulders) from the grounded origin, driving the ⊐ outline to DOF 0.
func (s *SketchContext) sizeFlangedRing(e []uint64, o uint64, edgeDia, raceDia, flangeBoreDia, innerZ, width string) error {
	dim := s.b.api.Sketch().Dimension(s.index)
	radii := []edgeOffset{
		{e[0], half(edgeDia)}, {e[2], half(flangeBoreDia)}, {e[4], half(raceDia)},
	}
	if err := applyEdgeOffsets(dim, o, radii); err != nil {
		return fmt.Errorf("dimension flanged-ring radii: %w", err)
	}
	levels := []edgeOffset{
		{e[1], half(width)}, {e[7], half(width)}, {e[3], innerZ}, {e[5], innerZ},
	}
	if err := applyEdgeOffsets(dim, o, levels); err != nil {
		return fmt.Errorf("dimension flanged-ring z-levels: %w", err)
	}
	return nil
}
