// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// GroundedCageRingSection builds the rectangular cross-section of a tapered-roller cage's
// small-end rim ring on the XZ plane (X radial, Z axial) and fully constrains it to DOF 0. Unlike
// GroundedRingSection (centred on the mid-plane) this ring sits in the SMALL-END axial overhang —
// the band of ring width beyond the roller small ends, before the small face — where no roller
// exists, so a continuous revolved frustum there cannot pierce the roller complement. Its two
// faces are pinned by their axial distance from the origin: nearZ (|z| of the face toward the
// mid-plane) and farZ (|z| of the face toward the small end), nearZ < farZ, both on the −Z side.
// innerDia/outerDia bound it radially inside the cone→cup window at that station. All expressions.
func (s *SketchContext) GroundedCageRingSection(innerDia, outerDia, nearZ, farZ string) error {
	// Seed a radial box on the −Z side; the dimensions below drive it to true size and position.
	res, err := s.b.api.Sketch().AddRectangle(s.index, []float64{2, -0.9}, []float64{3, -0.4}, false)
	if err != nil {
		return fmt.Errorf("add cage ring section: %w", err)
	}
	if len(res.PointIDs) < 4 || len(res.EntityIDs) < 4 {
		return fmt.Errorf("cage ring section reply short: corners=%d, edges=%d", len(res.PointIDs), len(res.EntityIDs))
	}
	return s.constrainCageRing(res.PointIDs, res.EntityIDs, innerDia, outerDia, nearZ, farZ)
}

// constrainCageRing pins the small-end ring box (corners BL,BR,TR,TL; edges bottom,right,top,left)
// to DOF 0: axis-aligned, inner (left) edge at innerDia/2, radial width set, and the two axial
// faces at their |z| from the grounded origin — the far face (bottom, larger |z|) at farZ and the
// near face (top) at nearZ.
func (s *SketchContext) constrainCageRing(p, e []uint64, innerDia, outerDia, nearZ, farZ string) error {
	bl, br := p[0], p[1]
	bottom, top, left := e[0], e[2], e[3]
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	if err := rectAxisConstraints(con, p[0], p[1], p[2], p[3]); err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	if _, err := dim.Offset(o, left, half(innerDia)); err != nil {
		return fmt.Errorf("place cage ring at inner radius: %w", err)
	}
	if _, err := dim.Distance(bl, br, radialWidth(outerDia, innerDia)); err != nil {
		return fmt.Errorf("dimension cage ring radial width: %w", err)
	}
	if _, err := dim.Offset(o, bottom, farZ); err != nil {
		return fmt.Errorf("place cage ring far face at %q: %w", farZ, err)
	}
	if _, err := dim.Offset(o, top, nearZ); err != nil {
		return fmt.Errorf("place cage ring near face at %q: %w", nearZ, err)
	}
	return nil
}
