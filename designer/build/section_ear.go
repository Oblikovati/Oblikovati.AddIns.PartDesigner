// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// eyeCentreSeedX/eyeCentreSeedYMag is the arbitrary non-degenerate polar guess (cm) the eye centre
// (the radial line's far endpoint) is drawn at before Distance+Angle drive it to its true
// position — a seed the solver rotates/scales, exactly as GroundedCircle's literal centre is a
// seed the diameter dimension overrides. eyeCentreSeedYMag is a MAGNITUDE: its sign is chosen per
// call by eyeSeedY, which also picks the Angle dimension's solution branch — see pinEyeCentre.
const (
	eyeCentreSeedX    = 0.75
	eyeCentreSeedYMag = 0.3
)

// GroundedEyeSection builds one plier-lug eye — a flat annulus (the eye outline and the
// concentric plier hole, two circles, no boolean) — on the CURRENT sketch (an offset-plane sketch
// at the ring's axial mid-level), fully constrained to DOF 0. The eye centre is placed in POLAR
// coordinates from a grounded origin: a fixed +X reference line the Angle dimension measures from,
// and a radial line whose Distance/Angle pin the centre at (centreRadius, azimuth). circleAtPoint
// then hangs each circle's centre on that shared point — never Fix, see its doc — so both circles
// stay tied to the parametric polar centre and re-drive with eye_radius_pos/azimuth.
func (s *SketchContext) GroundedEyeSection(centreRadius, azimuth, eyeDia, holeDia string) error {
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	ref, err := s.addFixedReferenceLine(o)
	if err != nil {
		return err
	}
	seedY, err := eyeSeedY(azimuth)
	if err != nil {
		return err
	}
	radial, err := s.addRadialLine(o, seedY)
	if err != nil {
		return err
	}
	if err := s.pinEyeCentre(ref.id, radial.id, o, radial.to, centreRadius, azimuth); err != nil {
		return err
	}
	if err := s.circleAtPoint(radial.to, eyeDia); err != nil {
		return err
	}
	return s.circleAtPoint(radial.to, holeDia)
}

// eyeSeedY picks the radial line's construction seed Y (magnitude eyeCentreSeedYMag, sign from
// azimuth): pinEyeCentre's Angle dimension measures the UNSIGNED angle between two lines (host
// model/sketch, bounded to [0°,180°] — Cross().Abs().Atan2()), so an azimuth beyond 180° (e.g.
// splitGapAngle=330°=-30°) is folded there to its [0°,180°] magnitude, and the SEED is what decides
// which of the two symmetric branches (+folded or −folded) the solver lands on — mirroring
// section_angular.go's groundedGrooveHelper ("the seed sign ... picks the correct-tilt branch").
// Confirmed against the real kernel solver, not just the fakeHost DOF=0 stub (#61 Task 2 Step 5): a
// raw "330 deg" Angle target never converges (residual saturates at the 180° boundary); folded +
// seeded, both 0° and 330° converge to the exact intended polar position.
func eyeSeedY(azimuth string) (float64, error) {
	var deg float64
	if _, err := fmt.Sscanf(azimuth, "%g deg", &deg); err != nil {
		return 0, fmt.Errorf("eye azimuth %q is not a plain \"N deg\" literal (need a numeric seed branch): %w", azimuth, err)
	}
	if deg > 180 {
		return -eyeCentreSeedYMag, nil
	}
	return eyeCentreSeedYMag, nil
}

// addFixedReferenceLine adds the construction +X reference line (origin→(1,0)) the Angle
// dimension measures the eye's azimuth from, and rigidly fixes it: the near end Coincident onto
// the grounded origin (so it shares the origin's ground rather than adding its own free DOF), the
// far end Fixed absolutely — see GroundedEyeSection's DOF-0 accounting.
func (s *SketchContext) addFixedReferenceLine(o uint64) (filletLine, error) {
	sk := s.b.api.Sketch()
	res, err := sk.AddLine(s.index, []float64{0, 0}, []float64{1, 0}, true)
	if err != nil || len(res.PointIDs) < 2 {
		return filletLine{}, fmt.Errorf("add eye reference line: %w (points=%d)", err, len(res.PointIDs))
	}
	line := filletLine{res.EntityID, res.PointIDs[0], res.PointIDs[1]}
	con := sk.Constrain(s.index)
	if _, err := con.Coincident(o, line.from); err != nil {
		return filletLine{}, fmt.Errorf("anchor eye reference line at origin: %w", err)
	}
	if _, err := con.Fix(line.to); err != nil {
		return filletLine{}, fmt.Errorf("fix eye reference line far end: %w", err)
	}
	return line, nil
}

// addRadialLine adds the construction radial line (origin→centre) whose far endpoint becomes the
// eye centre, seeded at the non-degenerate polar guess (eyeCentreSeedX, seedY) — pinEyeCentre's
// Distance+Angle dimensions drive the true position; seedY's SIGN (from eyeSeedY) picks which
// branch of the folded Angle target the solver converges to.
func (s *SketchContext) addRadialLine(o uint64, seedY float64) (filletLine, error) {
	sk := s.b.api.Sketch()
	res, err := sk.AddLine(s.index, []float64{0, 0}, []float64{eyeCentreSeedX, seedY}, true)
	if err != nil || len(res.PointIDs) < 2 {
		return filletLine{}, fmt.Errorf("add eye radial line: %w (points=%d)", err, len(res.PointIDs))
	}
	line := filletLine{res.EntityID, res.PointIDs[0], res.PointIDs[1]}
	if _, err := sk.Constrain(s.index).Coincident(o, line.from); err != nil {
		return filletLine{}, fmt.Errorf("anchor eye radial line at origin: %w", err)
	}
	return line, nil
}

// pinEyeCentre pins the eye centre (the radial line's far endpoint) to DOF 0 from the origin: a
// Distance dimension sets its reach (centreRadius) and an Angle dimension sets its bearing. Angle
// measures the UNSIGNED angle between two lines (host model/sketch: "the angle ... in [0,π] between
// two lines"), so the raw azimuth is folded to that reachable [0°,180°] magnitude via
// acos(cos(azimuth)) — both functions are supported by the host expression engine
// (model/param/expr_functions.go) — and addRadialLine's seed (eyeSeedY) resolves which side of the
// fold the solver lands on. See GroundedEyeSection's doc and eyeSeedY for the #61 Task 2 Step 5
// kernel-diagnostic evidence this needed folding (a raw "330 deg" target never converges).
func (s *SketchContext) pinEyeCentre(refLine, radialLine, o, centre uint64, centreRadius, azimuth string) error {
	dim := s.b.api.Sketch().Dimension(s.index)
	if _, err := dim.Distance(o, centre, centreRadius); err != nil {
		return fmt.Errorf("place eye centre at radius %q: %w", centreRadius, err)
	}
	folded := "acos(cos(" + azimuth + "))"
	if _, err := dim.Angle(refLine, radialLine, folded); err != nil {
		return fmt.Errorf("place eye centre at azimuth %q (folded %q): %w", azimuth, folded, err)
	}
	return nil
}

// circleAtPoint adds a circle seeded near the eye centre and Coincidents its own centre onto
// centre — NOT Fix. Fixing a literal seed (as GroundedCircle does for an independent circle) would
// decouple this circle from the shared parametric eye-centre point, so it would stay put instead of
// re-driving with eye_radius_pos/azimuth — the #61 circle-at-point trap. Diameter then dimensions
// it. Two calls over the same centre point (eye Ø, then plier-hole Ø) make the flat annulus.
func (s *SketchContext) circleAtPoint(centre uint64, diameterExpr string) error {
	sk := s.b.api.Sketch()
	res, err := sk.AddCircleByCenterRadius(s.index, []float64{eyeCentreSeedX, eyeCentreSeedYMag}, "("+diameterExpr+")/2", false)
	if err != nil || len(res.PointIDs) == 0 {
		return fmt.Errorf("add eye circle (diameter %q): %w (points=%d)", diameterExpr, err, len(res.PointIDs))
	}
	if _, err := sk.Constrain(s.index).Coincident(centre, res.PointIDs[0]); err != nil {
		return fmt.Errorf("coincide eye circle centre onto eye-centre point: %w", err)
	}
	if _, err := sk.Dimension(s.index).Diameter(res.EntityID, diameterExpr); err != nil {
		return fmt.Errorf("dimension eye circle diameter %q: %w", diameterExpr, err)
	}
	return nil
}
