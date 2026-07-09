// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// Stud is the threaded-rod / double-ended-stud generator. Its Variant selects the form:
// "continuous" (DIN 976 metric studding) is a plain cylinder threaded over its whole length;
// "double_end" (DIN 939 studs, metal end ≈ 1.25 d) is a cylinder threaded only at each end with
// a plain shank between. Both are ONE analytic-cylinder extrude (a boolean join re-facets coaxial
// same-radius cylinders, destroying the analytic face a thread binds to); the double-ended stud
// gets its bare middle from two partial-length cosmetic threads on that single face rather than
// from separate segments. Every dimension is parameter-driven, so editing the published size
// re-drives the whole part.
type Stud struct{}

// Kind is the family `generator` binding for studs and threaded rod.
func (Stud) Kind() string { return "stud" }

// nutEndOffset is where the nut-end thread begins, measured from the base of the rod: the overall
// length less the nut-end thread run. A formula over the published parameters, so it re-drives.
const nutEndOffset = "length - nut_thread_length"

// Build realizes the DOF-0 parametric stud. It expects the family to expose nominal_dia,
// thread_pitch and length parameters (and, for the double_end variant, metal_thread_length and
// nut_thread_length), plus the d and P columns the thread designation is built from.
func (Stud) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	if err := buildRodCylinder(b); err != nil {
		return err
	}
	switch rm.Family.Variant {
	case "", "continuous":
		return threadSoleCylinder(b, rm)
	case "double_end":
		return threadStudEnds(b, rm)
	default:
		return fmt.Errorf("stud variant %q unknown; want \"continuous\" or \"double_end\"", rm.Family.Variant)
	}
}

// buildRodCylinder extrudes the nominal_dia rod to length as a single solid — the one analytic
// cylinder both variants thread. A single extrude (no boolean) keeps the face analytic; joining
// coaxial same-radius segments would re-facet it and leave nothing for the thread to bind.
func buildRodCylinder(b *PartBuilder) error {
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

// threadStudEnds tags the rod's single cylindrical face with two partial-length cosmetic threads —
// the metal end (from the base for metal_thread_length) and the nut end (the far nut_thread_length)
// — leaving the shank between them bare, which is what distinguishes a stud from continuous
// studding. Both threads target the same analytic face (a cosmetic thread does not alter topology).
func threadStudEnds(b *PartBuilder, rm ResolvedMember) error {
	face, err := b.CylinderFaceKey()
	if err != nil {
		return err
	}
	designation := threadDesignation(rm)
	if err := b.CosmeticThreadSpan(face, designation, "", "metal_thread_length"); err != nil {
		return err
	}
	return b.CosmeticThreadSpan(face, designation, nutEndOffset, "nut_thread_length")
}
