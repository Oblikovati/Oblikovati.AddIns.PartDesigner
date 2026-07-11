// SPDX-License-Identifier: GPL-2.0-only

package build

// profileExtrude is the shared path every solid prismatic structural shape follows: publish the
// member's parameters, then build a fully-constrained (DOF-0) profile on XY and extrude it along
// +Z to the `length` parameter as a new solid. A generator supplies only its cross-section (a
// circle for round bar, a centred rectangle for flat bar, an I/channel/angle/tee outline) — the
// framework owns the publish → constrain → extrude spine, so a new structural shape is one
// drawProfile away and every one is guaranteed to publish its parameters, reach DOF 0, and
// length-drive identically.
func profileExtrude(b *PartBuilder, rm ResolvedMember, drawProfile func(*SketchContext) error) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	return extrudeProfile(b, drawProfile, "new")
}

// tubeExtrude is profileExtrude's hollow variant: publish the parameters, extrude the outer
// profile as a new solid, then extrude the inner profile as a cut through the whole length —
// leaving the wall of a hollow structural section (SHS/RHS/CHS). The inner profile is concentric
// with and smaller than the outer, so the cut removes exactly the bore and the two planar
// prisms leave clean planar tube faces.
func tubeExtrude(b *PartBuilder, rm ResolvedMember, outer, inner func(*SketchContext) error) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	if err := extrudeProfile(b, outer, "new"); err != nil {
		return err
	}
	return extrudeProfile(b, inner, "cut")
}

// extrudeProfile builds one DOF-0 profile on a fresh XY sketch and extrudes it to `length` with
// the given operation. The profile MUST leave the sketch at DOF 0; AssertFullyConstrained fails
// the build otherwise, catching an under-constrained (floppy) section before it becomes a solid.
func extrudeProfile(b *PartBuilder, drawProfile func(*SketchContext) error, operation string) error {
	sk, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := drawProfile(sk); err != nil {
		return err
	}
	sk.DumpGeometry("profile") // dev-only (OBK_PD_DEBUG); traces DOF + coords before the DOF-0 gate
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.Extrude(sk, "length", operation)
}
