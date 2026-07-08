// SPDX-License-Identifier: GPL-2.0-only

package build

// Key generates a parallel machine key (DIN 6885) — a `width` × `height` rectangular cross-section
// extruded to `length` — the block that seats in a shaft/hub keyway to transmit torque. The
// fit-critical section (width, height) and the length are exact; the section is centred on the
// origin so the key's axis is the origin. Square ends (DIN 6885 Form B); the Form A round ends and
// the gib head are tracked as a refinement.
type Key struct{}

// Kind is the family `generator` binding for parallel keys.
func (Key) Kind() string { return "key" }

// Build publishes the member's parameters and extrudes the key's cross-section to `length`. It
// expects the family to expose `width`, `height`, and `length`.
func (Key) Build(b *PartBuilder, rm ResolvedMember) error {
	return profileExtrude(b, rm, func(sk *SketchContext) error {
		return sk.GroundedRectangle("width", "height")
	})
}
