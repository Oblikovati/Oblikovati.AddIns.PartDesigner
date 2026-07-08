// SPDX-License-Identifier: GPL-2.0-only

package build

// profileExtrude is the shared path every prismatic structural shape follows: publish the
// member's parameters, build a fully-constrained (DOF-0) profile on the XY plane via
// drawProfile, then extrude that profile along +Z to the `length` parameter as a new solid.
// A generator supplies only its cross-section (a circle for round bar, a centred rectangle for
// flat bar; an I/channel/angle outline later) — the framework owns the publish → constrain →
// extrude spine, so a new structural shape is one drawProfile away and every one is guaranteed
// to publish its parameters, reach DOF 0, and length-drive identically.
//
// The profile MUST leave the sketch at DOF 0; AssertFullyConstrained fails the build otherwise,
// catching an under-constrained (floppy) section before it becomes a solid. Every family bound
// to a profile-extrude generator therefore has to expose a `length` column.
func profileExtrude(b *PartBuilder, rm ResolvedMember, drawProfile func(*SketchContext) error) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	sk, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := drawProfile(sk); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.Extrude(sk, "length", "new")
}
