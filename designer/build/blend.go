// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"fmt"

	"oblikovati.org/api/wire"
)

// Fillet rounds the corner shared by two sketch edges with a tangent arc of radiusExpr (a
// parameter name or formula), then dimensions the arc's radius so it stays parameter-driven.
// The host fillet consumes the shared corner vertex and pins the arc tangent to each edge, so a
// filleted corner reaches DOF 0 with just this radius dimension plus the edges' own direction
// constraints (Oblikovati#1943). It returns the arc's entity id. Apply fillets before other
// constraints reference the filleted corner, since that vertex is removed.
func (s *SketchContext) Fillet(edge1, edge2 uint64, radiusExpr string) (uint64, error) {
	res, err := s.b.api.Sketch().AddEntity(wire.AddSketchEntityArgs{
		SketchIndex: s.index, Kind: "fillet",
		EntityRefs: []uint64{edge1, edge2}, Radius: radiusExpr,
	})
	if err != nil {
		return 0, fmt.Errorf("add fillet (edges %d,%d) r=%q: %w", edge1, edge2, radiusExpr, err)
	}
	if _, err := s.b.api.Sketch().Dimension(s.index).Radius(res.EntityID, radiusExpr); err != nil {
		return 0, fmt.Errorf("dimension fillet radius %q: %w", radiusExpr, err)
	}
	return res.EntityID, nil
}
