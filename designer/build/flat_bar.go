// SPDX-License-Identifier: GPL-2.0-only

package build

// FlatBar is the reference structural profile-extrude generator: a hot-rolled flat steel bar
// (EN 10058) — a `width` × `thickness` rectangular section extruded to `length`. It is the
// simplest rectangular member of the profile-extrude framework and the template the richer
// structural shapes (I-beams, channels, angles) follow: publish the section parameters, draw a
// centred DOF-0 profile, extrude to length. The section is centred on the sketch origin so the
// extruded solid's neutral axis is the origin — the convention symmetric structural shapes build
// on.
type FlatBar struct{}

// Kind is the family `generator` binding for rectangular flat stock.
func (FlatBar) Kind() string { return "flat_bar" }

// Build publishes the member's parameters and extrudes a centred `width`×`thickness` rectangle to
// `length`. It expects the family to expose the `width`, `thickness`, and `length` parameters.
func (FlatBar) Build(b *PartBuilder, rm ResolvedMember) error {
	return profileExtrude(b, rm, func(sk *SketchContext) error {
		return sk.GroundedRectangle("width", "thickness")
	})
}
