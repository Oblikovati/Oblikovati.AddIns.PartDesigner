// SPDX-License-Identifier: GPL-2.0-only

package build

// Pin generates a cylindrical pin (ISO 2338 dowel pin) — a `diameter` circle extruded to
// `length` — the precise cylinder that locates or fastens two parts through a reamed hole. The
// diameter and length are exact and the cylinder is centred on the origin. The end chamfers, and
// the clevis-pin head and cotter/split-pin forms, are tracked as a refinement.
type Pin struct{}

// Kind is the family `generator` binding for cylindrical pins.
func (Pin) Kind() string { return "pin" }

// Build publishes the member's parameters and extrudes the pin cylinder to `length`. It expects
// the family to expose `diameter` and `length`.
func (Pin) Build(b *PartBuilder, rm ResolvedMember) error {
	return profileExtrude(b, rm, func(sk *SketchContext) error {
		return sk.GroundedCircle(0, 0, "diameter")
	})
}
