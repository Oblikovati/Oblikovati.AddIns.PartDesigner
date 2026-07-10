// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// shieldSeedFar/shieldSeedNear are the seed |z| coordinates (cm) for a 2Z shield's two axial
// faces — branch-picking only; the offset dimensions in constrainCageRing drive the real size.
const shieldSeedFar, shieldSeedNear = 0.9, 0.4

// GroundedShieldSection builds the rectangular cross-section of one 2Z metal shield on the XZ
// plane (X radial, Z axial) and fully constrains it to DOF 0: a flat annulus spanning idDia/2 to
// odDia/2, positioned axially by its distance from the origin — nearZ (|z| of the face toward the
// ball equator) and farZ (|z| of the face toward the ring end), nearZ < farZ, both expressed as
// positive magnitudes. negZ picks which bearing face the shield sits on: false seeds the rectangle
// on the +Z side, true mirrors the seed onto the −Z side — the SAME nearZ/farZ parameters drive
// both, so revolveShields builds the two shields (one per face) from one pair of dimensions. The
// offset dimension measures an unsigned distance from the origin (AddOffsetDim, dimension_advanced.go),
// so nearZ/farZ must stay positive; only the seed's sign steers the solver onto the intended side.
//
// This deliberately reuses constrainCageRing (section_cage.go): despite its cage-specific name,
// positioning a rectangular ring by its near/far |z| offset from a grounded origin is generic — a
// shield needs exactly that idiom, not a duplicate of it.
func (s *SketchContext) GroundedShieldSection(idDia, odDia, nearZ, farZ string, negZ bool) error {
	sign := 1.0
	if negZ {
		sign = -1.0
	}
	res, err := s.b.api.Sketch().AddRectangle(s.index,
		[]float64{2, sign * shieldSeedFar}, []float64{3, sign * shieldSeedNear}, false)
	if err != nil {
		return fmt.Errorf("add shield section: %w", err)
	}
	if len(res.PointIDs) < 4 || len(res.EntityIDs) < 4 {
		return fmt.Errorf("shield section reply short: corners=%d, edges=%d", len(res.PointIDs), len(res.EntityIDs))
	}
	return s.constrainCageRing(res.PointIDs, res.EntityIDs, idDia, odDia, nearZ, farZ)
}
