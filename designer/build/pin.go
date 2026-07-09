// SPDX-License-Identifier: GPL-2.0-only

package build

// dowelChamferFraction sizes the pin's end lead-in chamfer as a fraction of its diameter. ISO 2338
// and ASME B18.8.2 dowels carry a small chamfer/crown at each end for start-in; the standards give
// a range, so a representative proportion is used.
const dowelChamferFraction = "0.1"

// Pin generates a cylindrical pin (ISO 2338 / ASME B18.8.2 dowel pin) — a `diameter` × `length`
// cylinder with a lead-in chamfer at each end, the precise pin that locates or fastens two parts
// through a reamed hole. It is revolved from a chamfered half-section about the Z axis; the diameter
// and length are exact and the chamfer is a representative proportion of the diameter. The clevis-pin
// head and cotter/split-pin forms are separate generators / a tracked refinement.
type Pin struct{}

// Kind is the family `generator` binding for cylindrical pins.
func (Pin) Kind() string { return "pin" }

// Build publishes the member's parameters, derives the end chamfer, and revolves the chamfered rod
// section about the Z axis. It expects the family to expose `diameter` and `length`.
func (Pin) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	if err := b.DeriveParam("end_chamfer", "diameter * "+dowelChamferFraction); err != nil {
		return err
	}
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedChamferedRodSection("diameter", "length", "end_chamfer"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}
