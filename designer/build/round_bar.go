// SPDX-License-Identifier: GPL-2.0-only

package build

// RoundBar is the circular case of the profile-extrude framework and the simplest real
// generator: a cylindrical bar — a circle of diameter `nominal_dia` extruded to `length`. It is
// a genuine structural round-stock part; the richer profile shapes (flat bar, and the I-beams /
// channels / angles of C2–C3) share the same publish → constrained-profile → DOF-0 → extrude
// spine via profileExtrude, differing only in the cross-section they draw.
type RoundBar struct{}

// Kind is the family `generator` binding for round stock.
func (RoundBar) Kind() string { return "round_bar" }

// Build publishes the member's parameters and extrudes a grounded circle to `length`. It expects
// the family to expose the `nominal_dia` and `length` parameters (via its columns).
func (RoundBar) Build(b *PartBuilder, rm ResolvedMember) error {
	return profileExtrude(b, rm, func(sk *SketchContext) error {
		return sk.GroundedCircle(0, 0, "nominal_dia")
	})
}
