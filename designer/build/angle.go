// SPDX-License-Identifier: GPL-2.0-only

package build

// Angle generates an L / angle section (EN 10056, equal or unequal leg) by extruding a
// GroundedAngleSection to `length`. It drives the section from three published parameters —
// `leg_a`, `leg_b`, `thickness` — so equal-leg (leg_a == leg_b) and unequal-leg angles share one
// generator, differing only in the data table. Its heel sits at the sketch origin.
type Angle struct{}

// Kind is the family `generator` binding for L / angle sections.
func (Angle) Kind() string { return "angle" }

// Build publishes the member's parameters and extrudes its angle section to `length`. It expects
// the family to expose `leg_a`, `leg_b`, `thickness`, and `length`.
func (Angle) Build(b *PartBuilder, rm ResolvedMember) error {
	return profileExtrude(b, rm, func(sk *SketchContext) error {
		return sk.GroundedAngleSection("leg_a", "leg_b", "thickness")
	})
}
