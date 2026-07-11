# Circlip plier-hole lug ears (#61) — Design

**Issue:** Oblikovati.AddIns.PartDesigner #61 — *M2: Circlip plier-hole lug ears
(DIN 471/472, ASME B27.7)*. Split from #55 (dowel chamfers done).

**Goal:** Add the two assembly-plier **lug ears** at the split-gap edges of the retaining
ring — each a flat annular "eye" with a plier hole — so the representational circlip reads
as a real, plier-installable ring instead of a plain split band.

**Delivery:** One PR closing #61, add-in-only (no `oblikovati.org/api` or host change).
Granular commits. The ring itself (`circlip.go`, a flat radial section revolved `330°`
about Z leaving the split gap) is unchanged; the ears are added after the revolve.

## Global constraints

- **No boolean; independent bodies.** Each ear is a flat **annulus** (outer eye circle
  minus a concentric plier hole in one sketch) — the hole needs no boolean. The eye body
  just overlaps the ring band's end face at the gap edge (representational contact).
- **DOF-0 parametric.** Every dimension is a published-parameter formula expression; the
  eye sketch reaches `AssertFullyConstrained()` and solves DOF-0.
- **Unit-strict evaluator.** A bare literal added to / compared with a length evaluates to
  0 (see the #53 lesson). Every additive/clearance constant carries a unit (`"0.3 mm"`).
  ASME rings are `in`-unit — cross-unit conversion of an `mm` constant against an `in`
  family must be verified live; if unsupported, express the guard constants per-family unit.
- **Fall-back-to-plain.** `circlipEarsFit(rm)` gates the ears; on failure the ring builds
  as today with no ears (never overlapping/degenerate lugs).
- SPDX `GPL-2.0-only` header on new `.go`; funcs 4–20 lines; explicit types; `fakeHost`
  tests; coverage >80%, duplication <3%.

## Existing inputs

`Circlip.Build` (`circlip.go`) publishes `inner_dia`, `outer_dia`, `thickness`
(`nominal_dia` too) and revolves `GroundedRadialSection("inner_dia","outer_dia","thickness")`
about Z through `splitGapAngle = "330 deg"` (one-sided from azimuth 0 → band occupies
`[0°, 330°]`, gap `[330°, 360°]`). Families: `circlip_din471.json` (External),
`circlip_din472.json` (Internal), `circlip_ansi_external.json` (ASME External). `rm.Family.Category`
(a `CategoryPath`) ends in `External`/`Internal`.

## Architecture

`circlip.go` `Build` gains, after the ring revolve:
1. `if circlipEarsFit(rm)` → `deriveCirclipEarParams(b, external)` then `buildCirclipEar`
   twice (one per gap-edge azimuth). External/internal is decided in Go from the category.

New file `section_ear.go`:
- `GroundedEyeSection(centreRadius, azimuth, eyeDia, holeDia string) error` — on an
  `OffsetPlaneSketch("thickness/2")`, place the eye centre by `Distance(origin,centre)=centreRadius`
  + `Angle(refLine, radialLine)=azimuth`, then two concentric circles (`eyeDia`, `holeDia`)
  **centred on that parametric point** (Coincident-tied, *not* `GroundedCircle`'s literal
  seed), forming the annulus. DOF-0.
- A builder helper to centre a circle on an existing sketch point (`Coincident` the circle
  centre to the eye-centre point), since `GroundedCircle(cx,cy,dia)` grounds a literal seed
  and would decouple the circles from the parametric centre.

## Geometry (from the geometry-math derivation; representational proportions)

```
ear_band_width = "(outer_dia - inner_dia) / 2"
eye_outer_dia  = "ear_band_width * 1.0"   # external;  "* 0.9" internal (collision-driven trim)
plier_hole_dia = "eye_outer_dia * 0.45"   # leaves a 27.5%-of-Ø rim each side
eye_radius_pos = "outer_dia / 2 + eye_outer_dia * 0.3"   # external (eye centre just beyond OD)
               = "inner_dia / 2 - eye_outer_dia * 0.3"   # internal (just inside ID)
ear_a_azimuth  = "0 deg"
ear_b_azimuth  = splitGapAngle            # reuse the "330 deg" constant, do not redeclare
```

The `0.3` (`eyeOutwardFrac`) puts the eye centre only 30% of its own diameter past the band
edge, so 60% of the disc overlaps the band by construction — the "touches the ring" property
is an identity, not a separate check.

Ear build: `OffsetPlaneSketch("thickness/2")` → `GroundedEyeSection(eye_radius_pos, azimuth,
eye_outer_dia, plier_hole_dia)` → `AssertFullyConstrained()` → `ExtrudeDirected(sk,
"thickness","new","symmetric")` (symmetric span is the *total*, per `roller_bearing.go`;
`"thickness"` on the `thickness/2`-offset plane reproduces `z∈[0,thickness]`).

## Guard `circlipEarsFit(rm ResolvedMember) bool`

Float math on `rm.Value("di")/("do")/("s")` mirroring the parametric formulas; external vs
internal from the category. All three must hold, else skip ears:

- **(a) Rim positive:** `plier_hole_dia + 2·0.3mm ≤ eye_outer_dia` (⇒ `eye_outer_dia ≥ ~1.09mm`).
  Worst ASME 1/4" `e=2.22mm` (2× margin).
- **(b) Two-ear non-collision (binding):** `2·eye_radius_pos·sin15° ≥ eye_outer_dia + 0.3mm`
  (the two ears are 30° apart on `eye_radius_pos`). Internal is tighter (smaller R); worst
  DIN 472 d20 clears 15.4% *after* the internal `kEye`=0.9 trim (3.5% without it).
- **(c) Internal positive radius:** `eye_radius_pos − eye_outer_dia/2 > 0`. Defensive.

## Testing

- **fakeHost unit tests:** derived-ear-param expressions (exact strings), external vs internal
  branch (eye_radius_pos + kEye differ), two ear extrudes emitted (`thickness/new/symmetric`)
  after the ring revolve, DOF-0 reached, `circlipEarsFit` true for a representative member /
  false for a synthetic undersized internal member (drives skip-ears).
- **Kernel diagnostic:** the eye sketch (Distance+Angle centre + two Coincident circles) solves
  `DOF=0, Redundant=0, Converged` for a DIN 471 member.
- **Live MCP (the real gate):**
  1. Place DIN 471 d30 (external) → screenshot: two holed lug eyes bracketing the gap, both on
     the **band-terminus side** (verifies the winding sign — ear B at the actual 330° edge, not
     mirrored to +30° on the empty gap side).
  2. Place DIN 472 (internal) → eyes project inward, clear each other and the axis.
  3. Place an **ASME** member → verifies the `mm` clearance constant converts against an `in`
     family (else switch guard constants to per-family unit).
  4. Per-body volume: 2 ears present as non-degenerate bodies (annulus volume ≈
     `π·(eye_r² − hole_r²)·thickness`); driver re-drive recomputes DOF-0.

## Rejected alternatives

- **Eye on a short neck / simple nub** — user chose the flush lug (faithful DIN look; the
  issue mandates the hole, ruling out the nub).
- **`AngledOrientedSketch`** — builds a *meridian* plane (through Z) for revolves; the eye is a
  disc *perpendicular* to Z, so it needs the XY-parallel offset plane + polar dimensions.
- **Symmetric ±165° revolve** — `splitGapAngle` is already load-bearing; recentring only
  relabels the same two edges. Keep the one-sided 330° revolve.
- **Cutting the hole with a boolean** — the concentric-circle annulus gives the hole for free.
