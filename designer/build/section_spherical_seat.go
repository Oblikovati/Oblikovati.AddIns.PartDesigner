// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"fmt"

	"oblikovati.org/api/client"
)

// Self-aligning thrust seat (532xx) sections. The self-aligning variant replaces the housing
// washer's FLAT back with a shallow concave SEAT (a spherical zone, centre on the axis, well
// outboard) and adds a separate SEAT washer whose concave underside nests over it with a hair
// clearance, so the bearing can tilt to take up shaft misalignment. The seat sphere is derived from a
// cap-depth fraction of the outer diameter (geometry-math-advisor #54), flat enough to fit the washer
// thickness. That flatness (sagitta < 0.1 mm over the washer width) makes the seat arc numerically
// degenerate as a sketch entity, so each concave back is built as the sphere's straight CHORD — a
// shallow cone < 0.1 mm off the true sphere — pinned by its two rim STATIONS (√(r²+z²) from the
// origin), which is well-conditioned (the advisor's sub-floor-sagitta fallback). All expressions.
//
// The rim stations lie on z_back(r) = z_c − sqrt(R²−r²), which rises with r, so the OD-back corner is
// the higher endpoint and the bore-back corner the lower; the chord between them is the built edge.

// GroundedSphericalBackWasherSection builds the housing washer of a 532xx self-aligning thrust
// bearing: a grooved ball-facing front (outer land, ground groove arc, inner land — the 511xx race)
// with a concave conical SEAT back instead of a flat one. Six-entity closed loop (bore edge, back
// edge, OD edge, outer land, groove arc, inner land), fully constrained to DOF 0. The groove arc is
// centre-pinned (the only DOF-0-reliable arc idiom); the near-flat seat back is a straight edge
// pinned by its two rim STATIONS (its arc would be sub-floor-degenerate — see addSphericalBackEntities).
func (s *SketchContext) GroundedSphericalBackWasherSection(bore, outerDia, pitchDia, grooveRadius, landOffset, odBackDist, boreBackDist string) error {
	if err := s.disableSketchInference(); err != nil {
		return err
	}
	lines, groove, err := s.addSphericalBackEntities()
	if err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	if err := s.closeSphericalBackLoop(lines, groove); err != nil {
		return err
	}
	if err := s.orientSphericalBack(lines); err != nil {
		return err
	}
	if err := s.sizeSphericalBack(o, lines, groove, sphericalBackDims{
		bore: bore, outerDia: outerDia, pitchDia: pitchDia, grooveRadius: grooveRadius,
		landOffset: landOffset, odBackDist: odBackDist, boreBackDist: boreBackDist,
	}); err != nil {
		return err
	}
	return s.assertNoRedundancy("spherical-back washer section")
}

// sphericalBackDims bundles the housing washer's seven parameter expressions. odBackDist/boreBackDist
// are each rim vertex's Euclidean distance from the origin √(r²+z²): pinning the near-flat back arc by
// its two rim STATIONS (not by centre+radius) keeps the DOF-0 solve well-conditioned.
type sphericalBackDims struct {
	bore, outerDia, pitchDia, grooveRadius, landOffset, odBackDist, boreBackDist string
}

// addSphericalBackEntities lays down the 5 straight edges (bore, OD, outer land, inner land, back) and
// the ground groove arc at seed coordinates (cm, 53206 boundary). The self-aligning seat sphere is so
// flat (R ≈ 99 mm over an 8.5 mm-wide washer → arc sagitta ≈ 0.09 mm, well below the coincidence floor)
// that a sketch arc is numerically degenerate — its radius-equality is near-singular and the DOF-0 solve
// collapses (#54). Per the geometry-math-advisor's sub-floor-sagitta fallback the concave back is a
// straight OBLIQUE edge (revolving to a shallow cone that matches the sphere to < 0.1 mm), pinned by its
// two rim STATIONS, which is well-conditioned. Order of lines: [0] bore, [1] OD, [2] outer land,
// [3] inner land, [4] back (OD-back → bore-back).
func (s *SketchContext) addSphericalBackEntities() ([]filletLine, filletArc, error) {
	sk := s.b.api.Sketch()
	const bore2, od2, pit, land, w, odBack, boreBack = 1.5, 2.35, 1.925, 0.184, 0.187, 0.55, 0.382
	lineSeeds := [][2][]float64{
		{{bore2, boreBack}, {bore2, land}}, // 0 bore edge (back corner → front land)
		{{od2, odBack}, {od2, land}},       // 1 OD edge (back corner → front land)
		{{od2, land}, {pit + w, land}},     // 2 outer land
		{{pit - w, land}, {bore2, land}},   // 3 inner land
		{{od2, odBack}, {bore2, boreBack}}, // 4 conical back (OD-back → bore-back)
	}
	lines := make([]filletLine, 0, len(lineSeeds))
	for _, p := range lineSeeds {
		res, err := sk.AddLine(s.index, p[0], p[1], false)
		if err != nil || len(res.PointIDs) < 2 {
			return nil, filletArc{}, fmt.Errorf("add spherical-back edge: %w (points=%d)", err, len(res.PointIDs))
		}
		lines = append(lines, filletLine{res.EntityID, res.PointIDs[0], res.PointIDs[1]})
	}
	groove, err := s.addSeatArc([3][]float64{{pit, 0}, {pit + w, land}, {pit - w, land}}, true) // groove on mid-plane
	if err != nil {
		return nil, filletArc{}, err
	}
	return lines, groove, nil
}

// addSeatArc adds one centre/start/end arc and returns it as a filletArc (centre, start, end ids).
func (s *SketchContext) addSeatArc(seed [3][]float64, ccw bool) (filletArc, error) {
	res, err := s.b.api.Sketch().AddArcByCenterStartEnd(s.index, seed[0], seed[1], seed[2], ccw, false)
	if err != nil || len(res.PointIDs) < 3 {
		return filletArc{}, fmt.Errorf("add seat arc: %w (points=%d)", err, len(res.PointIDs))
	}
	return filletArc{res.EntityID, res.PointIDs[0], res.PointIDs[1], res.PointIDs[2]}, nil
}

// closeSphericalBackLoop joins the outline into one closed region: bore-back → back edge → OD-back,
// OD edge → outer land → groove arc → inner land → bore edge.
func (s *SketchContext) closeSphericalBackLoop(l []filletLine, groove filletArc) error {
	con := s.b.api.Sketch().Constrain(s.index)
	joins := [][2]uint64{
		{l[0].from, l[4].to},   // bore back corner ↔ back edge end (bore-back)
		{l[4].from, l[1].from}, // back edge start (OD-back) ↔ OD back corner
		{l[1].to, l[2].from},   // OD front ↔ outer land
		{l[2].to, groove.from}, // outer land ↔ groove arc
		{groove.to, l[3].from}, // groove arc ↔ inner land
		{l[3].to, l[0].to},     // inner land ↔ bore front
	}
	for _, j := range joins {
		if _, err := con.Coincident(j[0], j[1]); err != nil {
			return fmt.Errorf("close spherical-back loop at %d-%d: %w", j[0], j[1], err)
		}
	}
	return nil
}

// orientSphericalBack makes the two lands horizontal and the bore/OD edges vertical (the back and
// groove arcs are pinned by their centres, not orientation).
func (s *SketchContext) orientSphericalBack(l []filletLine) error {
	con := s.b.api.Sketch().Constrain(s.index)
	for _, i := range []int{2, 3} { // lands
		if _, err := con.Horizontal(l[i].from, l[i].to); err != nil {
			return fmt.Errorf("orient spherical-back land %d: %w", i, err)
		}
	}
	for _, i := range []int{0, 1} { // bore, OD edges
		if _, err := con.Vertical(l[i].from, l[i].to); err != nil {
			return fmt.Errorf("orient spherical-back edge %d vertical: %w", i, err)
		}
	}
	return nil
}

// sizeSphericalBack pins the lands to +land_offset, the bore/OD edges to their radii, the groove arc
// (centre on the mid-plane at the pitch radius + its radius) and the back arc (centre on the axis at
// sphereCentreZ + the sphere radius). The back-arc endpoints ride the bore/OD edges by the loop
// coincidences, so their axial level is derived from the sphere — the concave zone.
func (s *SketchContext) sizeSphericalBack(o uint64, l []filletLine, groove filletArc, d sphericalBackDims) error {
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	for _, i := range []int{2, 3} {
		if _, err := dim.Offset(o, l[i].id, d.landOffset); err != nil {
			return fmt.Errorf("place spherical-back land %d: %w", i, err)
		}
	}
	if _, err := dim.Offset(o, l[0].id, half(d.bore)); err != nil {
		return fmt.Errorf("place spherical-back bore edge: %w", err)
	}
	if _, err := dim.Offset(o, l[1].id, half(d.outerDia)); err != nil {
		return fmt.Errorf("place spherical-back OD edge: %w", err)
	}
	if err := s.pinArcCentreOnAxis(con, dim, o, groove.centre, half(d.pitchDia), false); err != nil {
		return fmt.Errorf("groove arc: %w", err) // centre at the pitch RADIUS (not the diameter — else it detaches)
	}
	if _, err := dim.Radius(groove.id, d.grooveRadius); err != nil {
		return fmt.Errorf("dimension spherical-back groove radius: %w", err)
	}
	return s.pinConicalBack(dim, o, l[4], d.odBackDist, d.boreBackDist)
}

// pinConicalBack pins the oblique back edge by its two rim STATIONS — each rim's Euclidean distance from
// the origin. With the rim's radius already fixed by its bore/OD edge, the distance fixes its axial
// level; the two stations lie on the seat sphere, so the straight edge is the sphere's chord over the
// washer annulus (< 0.1 mm from the true arc). No arc entity means no near-singular radius-equality —
// the DOF-0 solve stays well-conditioned. backEdge.from is the OD rim, .to the bore rim.
func (s *SketchContext) pinConicalBack(dim client.Dimension, o uint64, backEdge filletLine, odDist, boreDist string) error {
	if _, err := dim.Distance(o, backEdge.from, odDist); err != nil { // OD rim station
		return fmt.Errorf("pin OD back rim: %w", err)
	}
	if _, err := dim.Distance(o, backEdge.to, boreDist); err != nil { // bore rim station
		return fmt.Errorf("pin bore back rim: %w", err)
	}
	return nil
}

// GroundedSeatWasherSection builds the separate SEAT (aligning) washer that the housing washer rests
// in: a dished ring with a flat top and a concave underside that matches the housing back. Like the
// housing back the underside sphere is sub-floor-flat, so it is a straight OBLIQUE edge (a shallow cone
// chord of the sphere, < 0.1 mm off) rather than a numerically degenerate arc — see
// addSphericalBackEntities. Four straight edges (bore, flat top, OD, underside), DOF 0. bore is shared
// with the housing; seatOuterDia is the (larger) seat OD; topOffset is the |z| of the flat top;
// odUnderDist/boreUnderDist are the underside rim stations (√(r²+z²) on the seat sphere).
func (s *SketchContext) GroundedSeatWasherSection(bore, seatOuterDia, topOffset, odUnderDist, boreUnderDist string) error {
	if err := s.disableSketchInference(); err != nil {
		return err
	}
	sk := s.b.api.Sketch()
	const bore2, od2, top, bBore, bOd = 1.5, 2.63, 0.935, 0.404, 0.646
	lineSeeds := [][2][]float64{
		{{bore2, bBore}, {bore2, top}}, // 0 bore edge (bottom-bore → top-bore)
		{{bore2, top}, {od2, top}},     // 1 flat top
		{{od2, top}, {od2, bOd}},       // 2 OD edge (top-OD → bottom-OD)
		{{od2, bOd}, {bore2, bBore}},   // 3 conical underside (OD-bottom → bore-bottom)
	}
	lines := make([]filletLine, 0, len(lineSeeds))
	for _, p := range lineSeeds {
		res, err := sk.AddLine(s.index, p[0], p[1], false)
		if err != nil || len(res.PointIDs) < 2 {
			return fmt.Errorf("add seat-washer edge: %w (points=%d)", err, len(res.PointIDs))
		}
		lines = append(lines, filletLine{res.EntityID, res.PointIDs[0], res.PointIDs[1]})
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	if err := s.closeSeatWasherLoop(lines); err != nil {
		return err
	}
	if err := s.sizeSeatWasher(o, lines, bore, seatOuterDia, topOffset, odUnderDist, boreUnderDist); err != nil {
		return err
	}
	return s.assertNoRedundancy("seat washer section")
}

// closeSeatWasherLoop joins the seat washer outline: bore-bottom → underside edge → OD-bottom, OD edge →
// flat top → bore edge.
func (s *SketchContext) closeSeatWasherLoop(l []filletLine) error {
	con := s.b.api.Sketch().Constrain(s.index)
	joins := [][2]uint64{
		{l[0].from, l[3].to}, // bore bottom ↔ underside edge end (bore-bottom)
		{l[3].from, l[2].to}, // underside edge start (OD-bottom) ↔ OD bottom
		{l[2].from, l[1].to}, // OD top ↔ flat-top OD end
		{l[1].from, l[0].to}, // flat-top bore end ↔ bore top
	}
	for _, j := range joins {
		if _, err := con.Coincident(j[0], j[1]); err != nil {
			return fmt.Errorf("close seat-washer loop at %d-%d: %w", j[0], j[1], err)
		}
	}
	return nil
}

// sizeSeatWasher orients the edges and pins the seat washer to DOF 0: bore/OD edges vertical at their
// radii, the flat top at topOffset, and the concave underside (a shallow conical chord of the seat
// sphere) pinned by its two rim STATIONS (not by centre+radius, which is ill-conditioned for the
// near-flat seat sphere — see pinConicalBack).
func (s *SketchContext) sizeSeatWasher(o uint64, l []filletLine, bore, seatOD, topOffset, odUnderDist, boreUnderDist string) error {
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	if _, err := con.Horizontal(l[1].from, l[1].to); err != nil {
		return fmt.Errorf("orient seat-washer top: %w", err)
	}
	for _, i := range []int{0, 2} {
		if _, err := con.Vertical(l[i].from, l[i].to); err != nil {
			return fmt.Errorf("orient seat-washer edge %d vertical: %w", i, err)
		}
	}
	if _, err := dim.Offset(o, l[0].id, half(bore)); err != nil {
		return fmt.Errorf("place seat-washer bore edge: %w", err)
	}
	if _, err := dim.Offset(o, l[2].id, half(seatOD)); err != nil {
		return fmt.Errorf("place seat-washer OD edge: %w", err)
	}
	if _, err := dim.Offset(o, l[1].id, topOffset); err != nil {
		return fmt.Errorf("place seat-washer top: %w", err)
	}
	return s.pinConicalBack(dim, o, l[3], odUnderDist, boreUnderDist)
}

// pinArcCentreOnAxis locates an arc centre either on the axis (onAxis: Vertical to the origin →
// radius 0, then a Distance sets its axial level) or on the mid-plane (Horizontal to the origin,
// then a Distance sets its radius). distExpr is the axial level (onAxis) or the radius (mid-plane).
func (s *SketchContext) pinArcCentreOnAxis(con client.Constrain, dim client.Dimension, o, centre uint64, distExpr string, onAxis bool) error {
	var err error
	if onAxis { // on-axis centre: same radius as the origin (X=0), offset axially
		_, err = con.Vertical(o, centre)
	} else { // mid-plane centre: same axial level as the origin, offset radially
		_, err = con.Horizontal(o, centre)
	}
	if err != nil {
		return fmt.Errorf("align arc centre: %w", err)
	}
	if _, err := dim.Distance(o, centre, distExpr); err != nil {
		return fmt.Errorf("place arc centre at %q: %w", distExpr, err)
	}
	return nil
}
