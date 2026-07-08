// SPDX-License-Identifier: GPL-2.0-only

package build

// RoundBar is the reference generator and the simplest real one: a cylindrical bar —
// a circle of diameter `nominal_dia` extruded to `length`. It exercises the whole framework
// path (publish parameters → constrained sketch → DOF-0 check → parameter-driven feature)
// and is a genuine structural round-stock part; the richer fastener/structural/shaft/bearing
// generators (B/C/D/E) follow the same shape.
type RoundBar struct{}

// Kind is the family `generator` binding for round stock.
func (RoundBar) Kind() string { return "round_bar" }

// Build publishes the member's parameters and builds the DOF-0 cylinder. It expects the
// family to expose the `nominal_dia` and `length` parameters (via its columns).
func (RoundBar) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	sk, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := sk.GroundedCircle(0, 0, "nominal_dia"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.Extrude(sk, "length", "new")
}
