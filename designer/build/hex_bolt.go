// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"fmt"
	"strconv"
)

// HexBolt is the hex-head bolt / hex cap screw generator (ISO 4017, DIN 933 — fully threaded):
// a regular hexagonal head extruded up from the base plane, a cylindrical shank extruded down
// and joined to it, and a cosmetic thread over the shank. Every dimension is parameter-driven,
// so editing the published size (across_flats, head_height, nominal_dia, length) re-drives the
// whole part — the procedural analogue of picking another row from a Content-Center family
// table.
type HexBolt struct{}

// Kind is the family `generator` binding for hex bolts.
func (HexBolt) Kind() string { return "hex_bolt" }

// Build realizes the DOF-0 parametric bolt. It expects the family to expose the across_flats,
// head_height, nominal_dia and length parameters (via its columns), plus the d and P columns
// the thread designation is built from.
func (HexBolt) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	if err := buildHexHead(b); err != nil {
		return err
	}
	if err := buildShank(b); err != nil {
		return err
	}
	return threadShank(b, rm)
}

// buildHexHead extrudes the hexagonal head up from the XY base plane by head_height.
func buildHexHead(b *PartBuilder) error {
	sk, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := sk.GroundedHexagon("across_flats"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.Extrude(sk, "head_height", "new")
}

// buildShank extrudes the shank cylinder down from the XY base plane by length, joining it to
// the head (the head sits above the plane, the shank below, meeting and fusing at z = 0).
func buildShank(b *PartBuilder) error {
	sk, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := sk.GroundedCircle(0, 0, "nominal_dia"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.ExtrudeDirected(sk, "length", "join", "negative")
}

// threadShank tags the shank's (unique) cylindrical face with a cosmetic thread sized from the
// member's nominal diameter and pitch.
func threadShank(b *PartBuilder, rm ResolvedMember) error {
	face, err := b.CylinderFaceKey()
	if err != nil {
		return err
	}
	return b.CosmeticThread(face, metricThreadDesignation(rm))
}

// metricThreadDesignation renders the ISO metric thread designation for a member — "M8x1.25"
// from its nominal-diameter (d) and coarse-pitch (P) columns, the form ParseThreadDesignation
// resolves host-side.
func metricThreadDesignation(rm ResolvedMember) string {
	d := strconv.FormatFloat(rm.Value("d"), 'g', -1, 64)
	p := strconv.FormatFloat(rm.Value("P"), 'g', -1, 64)
	return fmt.Sprintf("M%sx%s", d, p)
}
