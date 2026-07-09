// SPDX-License-Identifier: GPL-2.0-only

package build

// Tee generates a T / tee section (EN 10055 hot-rolled equal-flange tees) by extruding a filleted
// tee section to `length`. It drives the section from five published parameters — `height`,
// `flange_width`, `web_thickness`, `flange_thickness`, `root_radius` — the same parameter vocabulary
// as the I-beam, so the section builders stay uniform. The section is symmetric about the Y axis
// with its bounding box centred on the origin, and carries the EN 10055 concave root fillet at each
// flange-stem junction.
type Tee struct{}

// Kind is the family `generator` binding for T / tee sections.
func (Tee) Kind() string { return "tee" }

// Build publishes the member's parameters and extrudes its filleted tee section to `length`. It
// expects the family to expose `height`, `flange_width`, `web_thickness`, `flange_thickness`,
// `root_radius`, and `length`.
func (Tee) Build(b *PartBuilder, rm ResolvedMember) error {
	return profileExtrude(b, rm, func(sk *SketchContext) error {
		return sk.GroundedFilletedTeeSection("height", "flange_width", "web_thickness",
			"flange_thickness", "root_radius")
	})
}
