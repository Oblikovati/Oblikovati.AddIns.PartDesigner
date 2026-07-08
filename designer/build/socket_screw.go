// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// SocketScrew is the hex-socket-drive screw generator (ISO 4762 / DIN 912 cylindrical-head cap
// screws and ISO 10642 countersunk-head screws). Both share a threaded shank and a hex socket
// recess; they differ only in the head, which the family's Variant selects: a "cylindrical"
// head is a plain cylinder, a "countersunk" one a cone that shrinks from the head diameter to
// the shank diameter over the head height. Every dimension is parameter-driven, so editing the
// published size re-drives the whole part.
//
// The whole part is built from the XY plane growing downward (−Z): the head top sits at z=0 and
// the socket is cut down into it. Crucially the socket is cut BEFORE the shank exists, and the
// shank is then built from the head-underside plane (a fresh solid). This is forced by two
// kernel behaviours: the socket's boolean cut re-facets whatever curved faces it touches (so it
// must not touch the shank the thread needs analytic), and a cosmetic thread must be the final
// feature (a boolean cut placed after one silently no-ops). Cutting the socket first — before
// the shank — and threading last satisfies both.
type SocketScrew struct{}

// Kind is the family `generator` binding for socket-drive screws.
func (SocketScrew) Kind() string { return "socket_screw" }

// Build realizes the DOF-0 parametric screw. It expects the family to expose head_dia,
// head_height, nominal_dia, socket_across_flats, socket_depth and length parameters (via its
// columns), plus the d and P columns the thread designation is built from, and a Variant of
// "cylindrical" (default) or "countersunk".
func (SocketScrew) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	style, err := styleForVariant(rm.Family.Variant)
	if err != nil {
		return err
	}
	if err := buildScrewHead(b, style.countersunk); err != nil {
		return err
	}
	if err := cutHexSocket(b); err != nil {
		return err
	}
	if err := buildScrewShank(b, style.shankSpan); err != nil {
		return err
	}
	return threadScrewShank(b, rm)
}

// socketHeadStyle is the geometry a Variant selects: the head shape and the shank's downward
// span (measured from the head-underside plane).
type socketHeadStyle struct {
	countersunk bool   // true ⇒ conical (ISO 10642) head; false ⇒ cylindrical (ISO 4762/DIN 912)
	shankSpan   string // downward extrude distance of the shank, from the head underside
}

// styleForVariant resolves the family Variant. The shank is built from the head-underside plane
// (z = −head_height). A cylindrical cap screw measures its length under the head, so the shank
// spans the full `length`; a countersunk screw is measured overall (the flush head is part of
// the length), so the shank spans length−head_height.
func styleForVariant(variant string) (socketHeadStyle, error) {
	switch variant {
	case "", "cylindrical":
		return socketHeadStyle{countersunk: false, shankSpan: "length"}, nil
	case "countersunk":
		return socketHeadStyle{countersunk: true, shankSpan: "length - head_height"}, nil
	default:
		return socketHeadStyle{}, fmt.Errorf("socket screw variant %q unknown; want \"cylindrical\" or \"countersunk\"", variant)
	}
}

// buildScrewHead builds the head down from the XY top plane. The cylindrical (ISO 4762/DIN 912)
// head is a plain circle extruded by head_height; the countersunk (ISO 10642) head is a cone.
func buildScrewHead(b *PartBuilder, countersunk bool) error {
	if countersunk {
		return buildCountersunkHead(b)
	}
	return buildCylindricalHead(b)
}

// buildCylindricalHead extrudes the head_dia circle down by head_height (a plain cylinder).
func buildCylindricalHead(b *PartBuilder) error {
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
	return b.ExtrudeDirected(sk, "head_height", "new", "negative")
}

// buildCountersunkHead lofts the head_dia circle (top, on XY) to the nominal_dia circle (bottom,
// on the head-underside plane), giving the standard flush cone. The cone angle is implied by the
// two parameter-driven diameters over head_height — the host cannot express an extrude taper as
// a formula, so a loft (not a tapered extrude) is what keeps the countersink parametric.
func buildCountersunkHead(b *PartBuilder) error {
	top, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := top.GroundedCircle(0, 0, "head_dia"); err != nil {
		return err
	}
	if err := top.AssertFullyConstrained(); err != nil {
		return err
	}
	bottom, err := b.OffsetPlaneSketch("-head_height")
	if err != nil {
		return err
	}
	if err := bottom.GroundedCircle(0, 0, "nominal_dia"); err != nil {
		return err
	}
	if err := bottom.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.Loft(bottom, top, "new")
}

// cutHexSocket cuts the blind hex-key recess down into the head from the top plane by
// socket_depth, its wrench size driven by socket_across_flats. It runs before the shank so the
// cut's re-faceting never reaches the shank cylinder the thread needs.
func cutHexSocket(b *PartBuilder) error {
	sk, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := sk.GroundedHexagon("socket_across_flats"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.ExtrudeDirected(sk, "socket_depth", "cut", "negative")
}

// buildScrewShank extrudes the shank cylinder down from the head-underside plane
// (z = −head_height) by spanExpr, joining it to the head. Building it here — after the socket is
// cut — keeps the shank a fresh analytic cylinder the thread can bind to, and starting at the
// head underside means the shank never fills the socket recess above it.
func buildScrewShank(b *PartBuilder, spanExpr string) error {
	sk, err := b.OffsetPlaneSketch("-head_height")
	if err != nil {
		return err
	}
	if err := sk.GroundedCircle(0, 0, "nominal_dia"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.ExtrudeDirected(sk, spanExpr, "join", "negative")
}

// threadScrewShank tags the shank (the deepest cylindrical face) with a cosmetic thread sized
// from the member's nominal diameter and pitch.
func threadScrewShank(b *PartBuilder, rm ResolvedMember) error {
	face, err := b.ShankCylinderFaceKey()
	if err != nil {
		return err
	}
	return b.CosmeticThread(face, metricThreadDesignation(rm))
}
