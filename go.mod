// The oblikovati-part-designer add-in: a c-shared library (.so/.dll/.dylib) loaded by the
// host at runtime. It is Oblikovati's procedural "Content Center" — a browsable library of
// standardized machine parts (fasteners, structural shapes, shaft parts, bearings) that,
// unlike Inventor's template-file catalogue, GENERATES each part procedurally from curated
// per-standard dimension tables so every placed part is a real, fully-constrained (DOF-0)
// parametric Oblikovati part.
//
// Its own module so the add-in deps stay independent of the host — the runtime boundary is
// the C ABI, not Go (see ./include/oblikovati_addin.h).
//
// The SHIPPED library links only the Apache-2.0 contract (oblikovati.org/api). A require on
// the GPL application module (oblikovati) is added later, TEST-SCOPE ONLY, for the
// designer<->real-host integration tests. Both modules are sibling repos resolved by the
// go.work workspace at this repo's root (no committed replace); CI injects the equivalent
// replaces via .github/actions/siblings.
module oblikovati.org/part-designer

go 1.24.0

require oblikovati.org/api v0.110.0
