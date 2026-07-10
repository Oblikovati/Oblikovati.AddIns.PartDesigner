// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"fmt"

	"oblikovati.org/api/client"
	"oblikovati.org/api/wire"
)

// domedRollerSpec is the parametric + seed description of one tapered roller's meridian, revolved
// about the roller's own tilted axis (Method C, geometry-math-advisor #54). The roller cone shares
// the bearing's apex O on the axis; its big end is a spherical cap centred at O (radius
// roller_sphere_r), so it sweeps flat against the cone's guide rib. The small end is flat (real
// tapered rollers dome only the big end). Every distance is a parameter EXPRESSION (so the roller
// re-drives with the member); the seeds are approximate cm coordinates that pick the solver branch.
//
// Anchors: O = apex on the bearing axis at (0, −apex_arm); C = the pitch point at (pitch_dia/2, 0).
// The two fix the roller axis (the O→C centerline) parametrically. On-axis points ride the
// centerline (PointOnLine) at a distance from O; off-axis points are triangulated by their distance
// to both O and C. Validated DOF-0 in the kernel diagnostic (#54).
type domedRollerSpec struct {
	oSeed, cSeed   [2]float64 // apex O, pitch point C (cm)
	p1Seed, p2Seed [2]float64 // small-end axis, small-end rim (cm)
	p3Seed, p4Seed [2]float64 // big-end rim, dome pole (cm)
	p3axSeed       [2]float64 // flat-case big-end axis point (cm); unused when domed
	apexArm        string     // |origin−O| = apex_arm
	halfPitch      string     // |origin−C| = pitch_dia/2
	oP1            string     // |O−P1| = zeta_small
	oP2, cP2       string     // small-rim distances to O and C
	oP3, cP3       string     // big-rim distances to O and C (oP3 = roller_sphere_r)
	zetaBig        string     // flat-case |O−P3ax|
	domed          bool       // false ⇒ flat big end (dome sagitta below the tessellation floor)
}

// GroundedDomedRollerSection builds and fully constrains (DOF 0) the meridian of one tapered
// roller on the XZ plane, ready to revolve about its own centerline. The straight edges are one
// open polyline; the big-end dome is an arc centred at the apex O, welded onto the polyline's rim
// and pole points. A single centerline O→C carries the revolve axis. When spec.domed is false the
// big end is closed flat instead (a degenerate-shallow-dome fallback).
func (s *SketchContext) GroundedDomedRollerSection(spec domedRollerSpec) error {
	sk := s.b.api.Sketch()
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	oID, err := s.addSeedPoint(spec.oSeed)
	if err != nil {
		return err
	}
	cID, err := s.addSeedPoint(spec.cSeed)
	if err != nil {
		return err
	}
	clRes, err := sk.AddCenterline(s.index, xy(spec.oSeed), xy(spec.cSeed))
	if err != nil {
		return fmt.Errorf("add roller centerline: %w", err)
	}
	con, dim := sk.Constrain(s.index), sk.Dimension(s.index)
	if err := s.anchorRollerAxis(con, dim, o, oID, cID, spec); err != nil {
		return err
	}
	if spec.domed {
		return s.buildDomedRoller(con, dim, oID, cID, clRes.EntityID, spec)
	}
	return s.buildFlatRoller(con, dim, oID, cID, clRes.EntityID, spec)
}

// anchorRollerAxis fixes the apex O on the Z axis at −apex_arm and the pitch point C on the
// mid-plane at pitch_dia/2 — the two anchors pin the tilted roller axis (the O→C line) to the
// parameters, so the whole meridian re-drives with the member.
func (s *SketchContext) anchorRollerAxis(con client.Constrain, dim client.Dimension, o, oID, cID uint64, spec domedRollerSpec) error {
	if _, err := con.Vertical(o, oID); err != nil {
		return fmt.Errorf("apex on Z axis: %w", err)
	}
	if _, err := dim.Distance(o, oID, spec.apexArm); err != nil {
		return fmt.Errorf("apex axial position: %w", err)
	}
	if _, err := con.Horizontal(o, cID); err != nil {
		return fmt.Errorf("pitch point on mid-plane: %w", err)
	}
	if _, err := dim.Distance(o, cID, spec.halfPitch); err != nil {
		return fmt.Errorf("pitch point radius: %w", err)
	}
	return nil
}

// buildDomedRoller lays down the domed meridian: open polyline P4→P1→P2→P3 (pole, small-axis,
// small-rim, big-rim) closed by the dome arc P3→P4 centred at the apex. P1/P4 ride the centerline;
// P2/P3 are triangulated off O and C; the arc's own points are welded onto P3/P4 and O.
func (s *SketchContext) buildDomedRoller(con client.Constrain, dim client.Dimension, o, c, centerline uint64, spec domedRollerSpec) error {
	pts, err := s.openPolyline([][]float64{xy(spec.p4Seed), xy(spec.p1Seed), xy(spec.p2Seed), xy(spec.p3Seed)})
	if err != nil {
		return err
	}
	p4, p1, p2, p3 := pts[0], pts[1], pts[2], pts[3]
	if err := s.rideCenterline(con, dim, centerline, o, p1, spec.oP1); err != nil {
		return err
	}
	if _, err := con.PointOnLine(p4, centerline); err != nil { // pole radius pinned by the arc
		return fmt.Errorf("pole on centerline: %w", err)
	}
	if err := s.triangulateRim(dim, o, c, p2, spec.oP2, spec.cP2); err != nil {
		return err
	}
	if err := s.triangulateRim(dim, o, c, p3, spec.oP3, spec.cP3); err != nil {
		return err
	}
	return s.weldDomeArc(con, o, p3, p4, spec)
}

// weldDomeArc adds the big-end dome arc by centre/start/end positions (new points) and welds them
// onto the apex O and the rim/pole points, so the arc becomes the spherical cap of radius
// roller_sphere_r centred at O (already pinned by |O−P3|).
func (s *SketchContext) weldDomeArc(con client.Constrain, o, p3, p4 uint64, spec domedRollerSpec) error {
	arc, err := s.b.api.Sketch().AddArcByCenterStartEnd(s.index, xy(spec.oSeed), xy(spec.p3Seed), xy(spec.p4Seed), true, false)
	if err != nil || len(arc.PointIDs) < 3 {
		return fmt.Errorf("add dome arc: %w (points=%d)", err, len(arc.PointIDs))
	}
	// AddArcByCenterStartEnd returns PointIDs in [centre, start, end] order (arcPointIDs in the
	// host router): the arc was created centre=O, start=P3(rim), end=P4(pole), so weld each back
	// onto its anchor. Getting this order wrong scrambles the arc into a garbage sphere.
	welds := [][2]uint64{{arc.PointIDs[0], o}, {arc.PointIDs[1], p3}, {arc.PointIDs[2], p4}}
	for _, w := range welds {
		if _, err := con.Coincident(w[0], w[1]); err != nil {
			return fmt.Errorf("weld dome arc point %d→%d: %w", w[0], w[1], err)
		}
	}
	return nil
}

// buildFlatRoller closes the big end with a flat radial face instead of the dome — used when the
// dome sagitta falls below the tessellation floor, where a near-chord arc is numerically fragile.
// Closed polyline P3ax→P1→P2→P3 (big-axis, small-axis, small-rim, big-rim).
func (s *SketchContext) buildFlatRoller(con client.Constrain, dim client.Dimension, o, c, centerline uint64, spec domedRollerSpec) error {
	pts, _, err := s.closedPolyline([][]float64{xy(spec.p3axSeed), xy(spec.p1Seed), xy(spec.p2Seed), xy(spec.p3Seed)})
	if err != nil {
		return err
	}
	p3ax, p1, p2, p3 := pts[0], pts[1], pts[2], pts[3]
	if err := s.rideCenterline(con, dim, centerline, o, p1, spec.oP1); err != nil {
		return err
	}
	if _, err := con.PointOnLine(p3ax, centerline); err != nil {
		return fmt.Errorf("flat big-end axis point on centerline: %w", err)
	}
	if _, err := dim.Distance(o, p3ax, spec.zetaBig); err != nil {
		return fmt.Errorf("flat big-end axial position: %w", err)
	}
	if err := s.triangulateRim(dim, o, c, p2, spec.oP2, spec.cP2); err != nil {
		return err
	}
	return s.triangulateRim(dim, o, c, p3, spec.oP3, spec.cP3)
}

// rideCenterline pins an on-axis point onto the centerline and at a fixed distance from the apex.
func (s *SketchContext) rideCenterline(con client.Constrain, dim client.Dimension, centerline, o, p uint64, distExpr string) error {
	if _, err := con.PointOnLine(p, centerline); err != nil {
		return fmt.Errorf("point on centerline: %w", err)
	}
	if _, err := dim.Distance(o, p, distExpr); err != nil {
		return fmt.Errorf("distance from apex %q: %w", distExpr, err)
	}
	return nil
}

// triangulateRim pins an off-axis point by its distance to both the apex O and the pitch point C —
// two circles whose intersection (seeded on the +radial side) fixes the point without an angle dim.
func (s *SketchContext) triangulateRim(dim client.Dimension, o, c, p uint64, oDist, cDist string) error {
	if _, err := dim.Distance(o, p, oDist); err != nil {
		return fmt.Errorf("rim distance to apex %q: %w", oDist, err)
	}
	if _, err := dim.Distance(c, p, cDist); err != nil {
		return fmt.Errorf("rim distance to pitch point %q: %w", cDist, err)
	}
	return nil
}

// openPolyline connects the seed points with shared-endpoint lines WITHOUT closing the loop (the
// dome arc closes it). Returns the shared point ids in input order.
func (s *SketchContext) openPolyline(pts [][]float64) ([]uint64, error) {
	res, err := s.b.api.Sketch().AddEntity(wire.AddSketchEntityArgs{
		SketchIndex: s.index, Kind: "polyline", Points: pts, Closed: false,
	})
	if err != nil {
		return nil, fmt.Errorf("add open polyline (%d points): %w", len(pts), err)
	}
	if len(res.PointIDs) != len(pts) {
		return nil, fmt.Errorf("open polyline reply: %d points, want %d", len(res.PointIDs), len(pts))
	}
	return res.PointIDs, nil
}

// addSeedPoint adds a standalone sketch point at a seed position (cm) and returns its id.
func (s *SketchContext) addSeedPoint(seed [2]float64) (uint64, error) {
	res, err := s.b.api.Sketch().AddPoint(s.index, xy(seed))
	if err != nil || len(res.PointIDs) < 1 {
		return 0, fmt.Errorf("add roller seed point %v: %w", seed, err)
	}
	return res.PointIDs[0], nil
}

// xy converts a [2]float64 seed to the []float64 the client entity constructors expect.
func xy(p [2]float64) []float64 { return []float64{p[0], p[1]} }
