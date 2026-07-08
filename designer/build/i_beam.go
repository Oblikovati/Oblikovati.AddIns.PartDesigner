// SPDX-License-Identifier: GPL-2.0-only

package build

// IBeam generates a doubly-symmetric wide-flange / I-beam (EN IPE, HE A, HE B) by extruding a
// GroundedISection to `length`. It drives the section from four published parameters — `height`,
// `flange_width`, `web_thickness`, `flange_thickness` — so every member of every I-series shares
// one generator, differing only in its dimension table. Like the other structural shapes it rides
// the profile-extrude framework: publish params → centred DOF-0 section → extrude.
type IBeam struct{}

// Kind is the family `generator` binding for wide-flange / I sections.
func (IBeam) Kind() string { return "i_beam" }

// Build publishes the member's parameters and extrudes its I-section to `length`. It expects the
// family to expose `height`, `flange_width`, `web_thickness`, `flange_thickness`, and `length`.
func (IBeam) Build(b *PartBuilder, rm ResolvedMember) error {
	return profileExtrude(b, rm, func(sk *SketchContext) error {
		return sk.GroundedISection("height", "flange_width", "web_thickness", "flange_thickness")
	})
}
