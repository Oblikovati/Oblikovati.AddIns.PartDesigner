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
