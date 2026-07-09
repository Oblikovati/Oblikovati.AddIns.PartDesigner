// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// GroundedAngularRingSection builds one angular-contact ball-bearing ring's revolved meridian
// section (XZ plane, X = radius, Z = axial) with a ground race groove whose centre is offset
// AXIALLY off the mid-plane, so the ball-to-race contact normal is tilted at the contact angle α
// from the radial plane — the geometry that lets the bearing carry a one-directional thrust load.
// It differs from GroundedGroovedRingSection in two ways: the groove centre is off-axis (pinned via
// a helper point, since Horizontal-to-origin would force it onto the mid-plane), and the two raceway
// shoulders are ASYMMETRIC — a tall retaining shoulder on the contact side and a relieved low
// shoulder on the counterbore side that exposes the balls (the visible angular-contact signature).
//
// farDia is the ring's far cylindrical edge (bore for the inner ring, outer_dia for the outer);
// highShoulderDia / reliefShoulderDia are the retaining and relieved raceway lands; grooveDia locates
// the groove-arc centre radially; grooveAxial is the axial offset magnitude of that centre off the
// mid-plane (positive; the ring picks the side); grooveRadius is the groove arc radius. See the
// geometry-math-advisor derivation (#54, Method C hybrid).
//
// The outline is 5 straight edges + 1 groove arc: far edge, two axial faces, high + relief shoulder
// lands, and the groove arc dipping between the shoulders. As with the deep-groove section the arc
// is NOT tangent to the shoulders (Tangent is a solver no-op); its endpoints ride Coincident on the
// axis-aligned shoulder lines and its centre is pinned once (radial + axial), so the endpoints'
// axial positions follow from the arc radius.
func (s *SketchContext) GroundedAngularRingSection(farDia, highShoulderDia, reliefShoulderDia, grooveDia, grooveAxial, grooveRadius, width string, innerRing bool) error {
	if err := s.disableSketchInference(); err != nil {
		return err
	}
	lines, arc, err := s.addAngularRingEntities(innerRing)
	if err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	helper, err := s.groundedGrooveHelper(o, grooveAxial, innerRing)
	if err != nil {
		return err
	}
	if err := s.closeGroovedRingLoop(lines, arc); err != nil {
		return err
	}
	if err := s.orientGroovedRing(lines); err != nil {
		return err
	}
	if err := s.sizeAngularRing(o, helper, lines, arc, farDia, highShoulderDia, reliefShoulderDia, grooveDia, grooveRadius, width); err != nil {
		return err
	}
	return s.assertNoRedundancy("angular-contact ring section")
}

// angularRingSeeds returns the 5 edges, the groove arc, and the helper point of the angular-contact
// ring outline at seed coordinates (cm), sized on the 7200-B (10×30×9, α=40°). The seeds only pick
// the solver branch — the constraints drive the true size. The outer ring's contact side (tall
// shoulder) faces +Z and its groove centre sits above the mid-plane; the inner ring mirrors both to
// −Z, so the two rings' contact normals converge on the ball at ±α. Order matches the deep-groove
// section (far edge, +face, high shoulder, [arc], relief shoulder, −face) so the shared loop/orient
// helpers apply: edges 0/2/3 vertical, 1/4 horizontal.
func angularRingSeeds(inner bool) (lineSeeds [][2][]float64, arcSeed [3][]float64, helperSeed []float64, ccw bool) {
	// Outer-ring seeds: far edge = OD radius; high/relief shoulder radii flank the off-axis groove
	// centre by 0.55·r_g / 0.85·r_g; arc endpoints ride the shoulder lines at ±sqrt(r_g²−Δ²).
	edge, hiSh, reSh := 1.5, 1.169, 1.256
	gcX, gcZ := 1.0086, 0.0072
	hiEndZ, reEndZ := 0.250, -0.146
	ccw = false
	if inner {
		edge, hiSh, reSh = 0.5, 0.831, 0.744
		gcX, gcZ = 0.9914, -0.0072
		hiEndZ, reEndZ = -0.250, 0.146
	}
	// contactZ is the tall-shoulder (contact) side: +width/2 for the outer ring, −width/2 for the
	// inner ring, so the two rings' groove centres and tall shoulders sit on opposite axial faces and
	// their contact normals converge on the ball from opposite sides at the contact angle.
	const w2 = 0.45
	contactZ := w2
	if inner {
		contactZ = -w2
	}
	lineSeeds = [][2][]float64{
		{{edge, -contactZ}, {edge, contactZ}},  // 0 far edge (bore / outer_dia)
		{{edge, contactZ}, {hiSh, contactZ}},   // 1 contact-side axial face
		{{hiSh, contactZ}, {hiSh, hiEndZ}},     // 2 high (retaining) shoulder land
		{{reSh, reEndZ}, {reSh, -contactZ}},    // 3 relief (low) shoulder land
		{{reSh, -contactZ}, {edge, -contactZ}}, // 4 relief-side axial face
	}
	arcSeed = [3][]float64{{gcX, gcZ}, {hiSh, hiEndZ}, {reSh, reEndZ}} // centre, start(high), end(relief)
	helperSeed = []float64{0, gcZ}                                     // on-axis helper at the groove-centre axial
	return lineSeeds, arcSeed, helperSeed, ccw
}

// addAngularRingEntities lays down the 5 edges and the off-axis groove arc at their seeds.
func (s *SketchContext) addAngularRingEntities(inner bool) ([]filletLine, filletArc, error) {
	lineSeeds, arcSeed, _, ccw := angularRingSeeds(inner)
	sk := s.b.api.Sketch()
	lines := make([]filletLine, 0, len(lineSeeds))
	for _, p := range lineSeeds {
		res, err := sk.AddLine(s.index, p[0], p[1], false)
		if err != nil || len(res.PointIDs) < 2 {
			return nil, filletArc{}, fmt.Errorf("add angular-ring edge: %w (points=%d)", err, len(res.PointIDs))
		}
		lines = append(lines, filletLine{res.EntityID, res.PointIDs[0], res.PointIDs[1]})
	}
	res, err := sk.AddArcByCenterStartEnd(s.index, arcSeed[0], arcSeed[1], arcSeed[2], ccw, false)
	if err != nil || len(res.PointIDs) < 3 {
		return nil, filletArc{}, fmt.Errorf("add angular-ring groove arc: %w (points=%d)", err, len(res.PointIDs))
	}
	return lines, filletArc{res.EntityID, res.PointIDs[0], res.PointIDs[1], res.PointIDs[2]}, nil
}

// groundedGrooveHelper adds an on-axis helper point at the groove centre's axial level and pins it
// to the origin: Vertical(origin,helper) puts it on the axis (X=0) and a Distance dimension sets its
// axial offset to grooveAxial. The helper then anchors the off-axis groove centre — Horizontal to
// the helper carries the axial level and a Distance from the helper carries the radial reach — which
// is the DOF-0-reliable way to place a point at (r, ±z_off) with only Vertical/Horizontal/Distance
// (Horizontal-to-origin would force the groove centre onto the mid-plane). The seed sign of the
// helper picks the axial side (+Z outer / −Z inner) so the solver keeps the correct-tilt branch.
func (s *SketchContext) groundedGrooveHelper(o uint64, grooveAxial string, inner bool) (uint64, error) {
	_, _, helperSeed, _ := angularRingSeeds(inner)
	res, err := s.b.api.Sketch().AddPoint(s.index, helperSeed)
	if err != nil || len(res.PointIDs) < 1 {
		return 0, fmt.Errorf("add groove-centre helper point: %w (points=%d)", err, len(res.PointIDs))
	}
	helper := res.PointIDs[0]
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	if _, err := con.Vertical(o, helper); err != nil {
		return 0, fmt.Errorf("put groove helper on the axis: %w", err)
	}
	if _, err := dim.Distance(o, helper, grooveAxial); err != nil {
		return 0, fmt.Errorf("offset groove helper axially by %q: %w", grooveAxial, err)
	}
	return helper, nil
}

// sizeAngularRing pins the far edge to its radius, the two shoulders to their (asymmetric) land
// radii, the axial faces to ±width/2, and the off-axis groove arc by anchoring its centre to the
// helper (Horizontal carries the axial level, Distance carries the radial reach) plus its radius.
// The arc endpoints then land on the shoulder lines through the loop coincidences, so their axial
// positions are derived, not dimensioned — the same center-pin recipe as the deep-groove section,
// shifted onto the helper so the groove sits off the mid-plane.
func (s *SketchContext) sizeAngularRing(o, helper uint64, l []filletLine, a filletArc, farDia, highShoulderDia, reliefShoulderDia, grooveDia, grooveRadius, width string) error {
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	if _, err := dim.Offset(o, l[0].id, half(farDia)); err != nil {
		return fmt.Errorf("place angular ring far edge: %w", err)
	}
	if _, err := dim.Offset(o, l[2].id, half(highShoulderDia)); err != nil {
		return fmt.Errorf("place angular-ring high shoulder: %w", err)
	}
	if _, err := dim.Offset(o, l[3].id, half(reliefShoulderDia)); err != nil {
		return fmt.Errorf("place angular-ring relief shoulder: %w", err)
	}
	for _, i := range []int{1, 4} {
		if _, err := dim.Offset(o, l[i].id, half(width)); err != nil {
			return fmt.Errorf("centre angular-ring face %d axially: %w", i, err)
		}
	}
	if _, err := con.Horizontal(helper, a.centre); err != nil {
		return fmt.Errorf("carry groove-centre axial level from helper: %w", err)
	}
	if _, err := dim.Distance(helper, a.centre, half(grooveDia)); err != nil {
		return fmt.Errorf("place groove-arc centre radially: %w", err)
	}
	if _, err := dim.Radius(a.id, grooveRadius); err != nil {
		return fmt.Errorf("dimension angular groove radius %q: %w", grooveRadius, err)
	}
	return nil
}
