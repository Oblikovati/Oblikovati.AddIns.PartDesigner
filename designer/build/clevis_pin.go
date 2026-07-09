// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// ClevisPin generates a clevis pin (ISO 2341) — a cylindrical shank with a flat cylindrical head at
// one end and a transverse cotter-pin hole near the other, the pin that carries a clevis joint and
// is retained by a split pin through the hole. The head extrudes up from the base plane, the shank
// down and joined to it (as the hex bolt builds head + shank), and the cotter hole is cut through
// the shank at the grounded end distance le. Shank/head/hole diameters, head height and the hole
// end distance are grounded in ISO 2341; the head-edge chamfer/dome is a tracked refinement.
type ClevisPin struct{}

// Kind is the family `generator` binding for clevis pins.
func (ClevisPin) Kind() string { return "clevis_pin" }

// Build publishes the member's parameters, derives the hole depth (length − hole_end_dist), extrudes
// the head and shank, and cuts the transverse cotter hole. It expects the family to expose
// `shank_dia`, `head_dia`, `head_height`, `hole_dia`, `hole_end_dist` and `length`.
func (ClevisPin) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	// The cotter hole axis sits hole_end_dist from the far (shank) end, i.e. this far below the head
	// plane where the shank was extruded to −length.
	if err := b.DeriveParam("hole_depth", "length - hole_end_dist"); err != nil {
		return err
	}
	if err := buildClevisHead(b); err != nil {
		return err
	}
	if err := buildClevisShank(b); err != nil {
		return err
	}
	return cutCotterHole(b)
}

// buildClevisHead extrudes the flat head disk up from the XY base plane by head_height.
func buildClevisHead(b *PartBuilder) error {
	sk, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := sk.GroundedCircle(0, 0, "head_dia"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.Extrude(sk, "head_height", "new")
}

// buildClevisShank extrudes the shank cylinder down from the base plane by length, joining it to the
// head (the head sits above the plane, the shank below, fusing at z = 0).
func buildClevisShank(b *PartBuilder) error {
	sk, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := sk.GroundedCircle(0, 0, "shank_dia"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.ExtrudeDirected(sk, "length", "join", "negative")
}

// cutCotterHole cuts the transverse cotter hole through the shank: a circle of hole_dia on the XZ
// plane (X radial, Z axial), placed hole_depth below the origin on the pin axis, extruded
// symmetrically through the shank along the plane normal as a cut.
func cutCotterHole(b *PartBuilder) error {
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedDepthCircle("hole_depth", "hole_dia"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	// Cut across the full shank (and clear it) symmetric about the axis plane; head_dia > shank_dia
	// guarantees the hole passes clean through.
	return b.ExtrudeDirected(sk, "head_dia", "cut", "symmetric")
}

// GroundedDepthCircle adds a circle whose centre sits on the sketch's −Y axis at depthExpr below the
// grounded origin, with diameter diameterExpr — the profile a transverse hole is cut from on a plane
// containing the pin axis (the axis running down −Y). It is fully constrained (DOF 0): the centre is
// aligned vertically under the origin and placed by a distance dimension, and the diameter dimension
// sizes it, so the hole tracks the pin length. It mirrors GroundedOffsetCircle onto the axial
// direction.
func (s *SketchContext) GroundedDepthCircle(depthExpr, diameterExpr string) error {
	const cy = -5.0 // seed depth (cm) below the origin; the distance dimension drives the real value
	sk := s.b.api.Sketch()
	res, err := sk.AddCircleByCenterRadius(s.index, []float64{0, cy}, "("+diameterExpr+")/2", false)
	if err != nil {
		return fmt.Errorf("add depth circle: %w", err)
	}
	if len(res.PointIDs) == 0 {
		return fmt.Errorf("depth circle returned no centre point (entity %d)", res.EntityID)
	}
	centre := res.PointIDs[0]
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	con, dim := sk.Constrain(s.index), sk.Dimension(s.index)
	if _, err := con.Vertical(o, centre); err != nil {
		return fmt.Errorf("align depth circle under axis: %w", err)
	}
	if _, err := dim.Distance(o, centre, depthExpr); err != nil {
		return fmt.Errorf("place depth circle at %q: %w", depthExpr, err)
	}
	if _, err := dim.Diameter(res.EntityID, diameterExpr); err != nil {
		return fmt.Errorf("dimension depth circle diameter %q: %w", diameterExpr, err)
	}
	return nil
}
