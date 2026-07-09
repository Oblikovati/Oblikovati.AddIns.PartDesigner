// SPDX-License-Identifier: GPL-2.0-only

package build

// Angle generates an L / angle section (EN 10056, equal or unequal leg) by extruding a filleted
// angle section to `length`. It drives the section from five published parameters — `leg_a`,
// `leg_b`, `thickness`, `root_radius`, `toe_radius` — so equal-leg (leg_a == leg_b) and unequal-leg
// angles share one generator, differing only in the data table. Its heel sits at the sketch origin.
type Angle struct{}

// Kind is the family `generator` binding for L / angle sections.
func (Angle) Kind() string { return "angle" }

// Build publishes the member's parameters and extrudes its filleted angle section to `length`. It
// expects the family to expose `leg_a`, `leg_b`, `thickness`, `root_radius`, `toe_radius`, and
// `length`. The section carries the EN 10056 inner-heel root fillet and the two convex toe radii.
func (Angle) Build(b *PartBuilder, rm ResolvedMember) error {
	return profileExtrude(b, rm, func(sk *SketchContext) error {
		return sk.GroundedFilletedAngleSection("leg_a", "leg_b", "thickness", "root_radius", "toe_radius")
	})
}
