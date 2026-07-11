// SPDX-License-Identifier: GPL-2.0-only

package build

// Channel generates a channel / U-section (EN UPN) by extruding a GroundedChannelSection to
// `length`. It drives the section from four published parameters — `height`, `flange_width`,
// `web_thickness`, `flange_thickness` — sharing one generator across the channel series. Like the
// other structural shapes it rides the profile-extrude framework: publish params → centred DOF-0
// section → extrude. The section is symmetric about the X axis with the web on the left, its
// bounding box centred on the origin.
type Channel struct{}

// Kind is the family `generator` binding for channel / U sections.
func (Channel) Kind() string { return "channel" }

// Build publishes the member's parameters and extrudes its channel section to `length`. It expects
// the family to expose `height`, `flange_width`, `web_thickness`, `flange_thickness`, and `length`.
func (Channel) Build(b *PartBuilder, rm ResolvedMember) error {
	return profileExtrude(b, rm, func(sk *SketchContext) error {
		return sk.GroundedChannelSection("height", "flange_width", "web_thickness", "flange_thickness")
	})
}

// TaperedChannel generates a taper-flange channel (UPN per DIN 1026-1 / EN 10279, AISC C): the
// inner flange faces slope inward at `flange_taper` (an angle) so the flange is thicker at the web
// than the toe. It shares the section parameters with Channel plus `flange_taper`, and extrudes to
// `length` (#69).
type TaperedChannel struct{}

// Kind is the family `generator` binding for taper-flange channels.
func (TaperedChannel) Kind() string { return "channel_taper" }

// Build publishes the member's parameters and extrudes its taper-flange channel section to
// `length`. It expects `height`, `flange_width`, `web_thickness`, `flange_thickness`,
// `flange_taper`, and `length`.
func (TaperedChannel) Build(b *PartBuilder, rm ResolvedMember) error {
	seeds := taperedChannelSeeds(rm)
	return profileExtrude(b, rm, func(sk *SketchContext) error {
		// r1 (root fillet) = tf and r2 (toe fillet) = tf/2 per the EN 10279 fillet convention.
		return sk.GroundedTaperedChannelSection(seeds, "height", "flange_width", "web_thickness",
			"flange_thickness", "flange_taper", "flange_thickness", "(flange_thickness) / 2")
	})
}
