# Circlip lug-ear geometry (#61) — derivation

Grounded against the live catalogues: `circlip_din471.json` (external, d10..d50),
`circlip_din472.json` (internal, d20..d62), `circlip_ansi_external.json` (ASME B27.7,
1/4"..1"), and the existing generator `designer/build/circlip.go` /
`designer/build/sketch.go` / `designer/build/builder.go` conventions (`GroundedCircle`,
`OffsetPlaneSketch`, `ExtrudeDirected(..., "symmetric")`, the `rollerCage*` guard-with-
fallback pattern in `roller_cage.go`).

## Problem framing

The ring band is a swept annulus: a rectangular meridian section (r ∈ [inner_dia/2,
outer_dia/2], z ∈ [0, thickness]) revolved about +Z from azimuth 0° through
`splitGapAngle = 330°`. The band's two end faces are flat half-planes through the axis
at azimuth 0° and 330°, each spanning r ∈ [inner_dia/2, outer_dia/2], z ∈ [0, thickness].
"Add lug ears" means: at each of those two azimuths, place a flat annular disc (an eye:
outer circle minus a concentric hole, both in one sketch — an annulus profile needs no
boolean) in the plane z = thickness/2 (the offset-plane convention `OffsetPlaneSketch`
already provides), extruded ±thickness/2 back onto z ∈ [0, thickness], and position its
centre so the disc's footprint overlaps the band's end-face rectangle (representational
contact) without the two ears — 30° apart — colliding with each other. This is placement
geometry (polar coordinates + a positive-clearance packing constraint), not curve/surface
intersection; "correct" here means: (1) every published DeriveParam expression evaluates
under the unit-strict evaluator, (2) the two ears provably do not overlap each other for
every catalogue member the guard passes, and (3) each eye provably overlaps its band end
by construction (not just by guard) so a passing member never needs a separate contact
check.

## Candidate methods for the centre placement

| Approach | How | Tradeoffs |
|---|---|---|
| **Polar dimension pair** (chosen) | Grounded origin + reference direction line; `Distance(origin,centre)=eye_radius_pos`, `Angle(ref,radial)=azimuth` | Matches the task's own spec; DOF-0 in 2 dimensions; azimuth is a single symbolic expression that can literally be `splitGapAngle`, so ear B's angle **is** the revolve's sweep angle — no duplicated magic number |
| Cartesian `(x,y)` from `radius*cos/sin(azimuth)` | `Distance`+`Distance` to two grounded reference points, or direct coordinate dimensions | Two coupled dimensions instead of one natural radius/angle pair; azimuth becomes implicit in two trig expressions instead of one readable constant — harder to re-derive when `splitGapAngle` changes |
| `AngledOrientedSketch(angleExpr)` (existing helper, used by the roller-cage bar) | Rotated work plane through the Z axis at `angleExpr`, sketch X=radial/Y=axial | **Wrong tool**: that helper builds a *meridian* plane (contains the Z axis) for revolve profiles. The ear is a *disc perpendicular to Z*, not a meridian section — it needs an XY-parallel offset plane, not a plane through the axis. Ruled out. |

The polar-dimension pair is what the task already specifies and is the right primitive:
it is exactly the constraint pattern `GroundedRadialSection` uses for the band section
itself (`Distance(o,bl) = inner_dia/2`), just rotated into the XY-parallel offset plane
and paired with an `Angle` dimension for the second DOF.

## Recommendation & derivation

### 1. Gap-edge azimuths

Confirmed against `circlip.go`: the band occupies azimuth [0°, 330°] (`splitGapAngle =
"330 deg"`, revolve from 0), so the gap — and the two ears that bracket it — spans
[330°, 360°]. Do **not** switch to a symmetric ±165° two-sided revolve: `splitGapAngle`
is already load-bearing (the band revolve, and any future groove/lug work keys off it),
and moving the meridian plane would just relabel the same two edges without simplifying
anything.

```
ear_a_azimuth = "0 deg"
ear_b_azimuth = splitGapAngle        // reuse the Go constant "330 deg" — do not redeclare
```

Angle-dimension value: Ear A's `Angle(ref, radial)` = `0 deg`; Ear B's = `330 deg`
(equivalently −30°, i.e. one gap-half-angle short of a full turn back to +X).

**Verify the winding sign live** (Numerical pitfalls, below) before trusting that "330
deg" measured by the sketch's `Angle` dimension lands the same physical edge the revolve
already terminates at — the two APIs (revolve sweep vs. two-line angle dimension) are not
guaranteed to share a sign convention without a screenshot check.

### 2. `eye_radius_pos`

Let `ear_band_width = (outer_dia − inner_dia) / 2` (the ring's own radial width — the
natural size scale, already implicit in `GroundedRadialSection`'s "radial width" term).

External (eye projects outward, centred beyond OD):
```
eye_radius_pos = outer_dia / 2 + eye_outer_dia * 0.3
```
Internal (eye projects inward, centred inside ID):
```
eye_radius_pos = inner_dia / 2 - eye_outer_dia * 0.3
```
The `0.3` (`eyeOutwardFrac`) means the eye centre sits only 30% of the eye's *own*
diameter past the band edge, so **60% of the eye disc's diameter overlaps back into the
band** by construction — the "touches/overlaps the ring band end" requirement is an
identity of this formula, not a separate check, and holds at every scale automatically.

### 3. `eye_outer_dia` / `plier_hole_dia`

```
eye_outer_dia    = ear_band_width * kEye        // kEye = 1.0 external, 0.9 internal
plier_hole_dia   = eye_outer_dia * 0.45
```
`kEye = 1.0` reads as "the eye disc is about as wide as the ring's own cross-section" —
a clean, size-invariant proportion that tracks DIN471/472 practice for large panel
retaining rings with lug tabs (representational choice; the milestone catalogue has no
lug-hole columns of its own, see Numerical pitfalls). `kHole = 0.45` leaves a 27.5%-of-
eye-diameter rim on each side. `kEye` is asymmetric between external (1.0) and internal
(0.9) — derived, not guessed, from the collision guard below: internal ears sit at a
*smaller absolute radius* than external ears for the same nominal size (they're pulled
toward the axis, not away from it), so an equal-size eye collides with its 30°-neighbour
sooner going inward. Trimming `kEye` 10% for internal rings restores the same order of
collision margin external rings get "for free" from their larger radius.

Computed sizes (mm), all within a sane visible-but-not-dwarfing band:

| Member | band width | eye Ø | hole Ø |
|---|---|---|---|
| DIN471 d10 (ext) | 3.5 | 3.50 | 1.58 |
| DIN471 d50 (ext) | 7.0 | 7.00 | 3.15 |
| ASME 1/4" (ext)  | 2.22 | 2.22 | 1.00 |
| DIN472 d20 (int) | 3.0 | 2.70 | 1.22 |
| DIN472 d62 (int) | 5.9 | 5.31 | 2.39 |

### 4. Guards

Let `w = ear_band_width`, `R = eye_radius_pos`, `e = eye_outer_dia`, `h =
plier_hole_dia`, half-gap-edge separation `Δ = 30°` (⇒ half-angle 15°, chord
`c = 2R·sin15°`).

**(a) Rim positive**: `h + 2·earMinRim ≤ e`, `earMinRim = "0.3 mm"`. Since `h = 0.45e`,
this reduces to a floor on the eye size itself: `e ≥ 2·earMinRim / 0.55 ≈ 1.09 mm`.
Worst case in the catalogue: **ASME 1/4"** (`e = 2.22 mm`) — ~2× the floor, comfortable.

**(b) Non-collision of the two ears** (30° apart, same radius by symmetry — both ears
of one ring use the same `eye_radius_pos` formula):
```
2 · eye_radius_pos · sin(15°) ≥ eye_outer_dia + earMinClearance     // earMinClearance = "0.3 mm"
```
This is the binding guard. Substituting the external/internal `R` formulas:
- External: `2·(outer_dia/2 + 0.3e)·sin15° ≥ e + clr`
- Internal: `2·(inner_dia/2 − 0.3e)·sin15° ≥ e + clr`
Internal is structurally tighter (subtracting instead of adding shrinks `R`). Worst case
in the catalogue: **DIN472 d20** (`di=15, w=3.0` → with `kEye=0.9`, `e=2.7`,
`R = 7.5 − 0.81 = 6.69`, `c = 2·6.69·sin15° = 3.463 mm` vs. required `e + clr = 3.0 mm`
→ **15.4% margin**). Before the internal `kEye` trim (using `kEye=1.0` uniformly) this
same member only cleared by **3.5%** — the guard is not decorative; it is already close
to binding at the smallest published internal size, and any future smaller/thicker
internal member must be re-checked, not assumed safe.

**(c) Internal eye stays at positive radius**: `eye_radius_pos − eye_outer_dia/2 > 0`.
Never binding in this catalogue (smallest margin ≈ 5.34 mm at DIN472 d20) because
`eyeOutwardFrac = 0.3 < 0.5` already keeps the centre well clear of the axis for any
plausible `di ≫ w`; keep it as a defensive guard for a hypothetical thick-band/small-bore
custom member where `w` approaches `di/2`.

**Fallback**: if any guard fails for a resolved member, skip building the ear pair
entirely (ring still builds as today) — mirror `rollerCageBarsFit(rm)` in
`roller_cage.go`: a same-package `circlipEarsFit(rm ResolvedMember) bool` recomputing (a)
and (b) in float64 from `rm.Value("di")/("do")/("s")` (or whatever the family calls its
columns), called before `deriveCirclipEarParams`/`buildCirclipEar`, so a future
undersized custom member degrades to "ring only" rather than emitting overlapping or
degenerate lugs.

### 5. Sketch layout (per ear — two ears ⇒ two independent instances of this)

1. `OffsetPlaneSketch("thickness/2")` — a hidden work plane parallel to XY at the ring's
   axial mid-level, and a sketch on it. (Not `AngledOrientedSketch`: that plane contains
   the Z axis, this one is perpendicular to it.)
2. Ground the sketch origin point (`Ground`).
3. A construction reference line from the origin to a fixed seed point on local +X
   (`Horizontal`, pins the "+X axis" reference the `Angle` dimension measures from — the
   offset plane's local X/Y already align with global X/Y by the `OffsetHidden(XY,...)`
   convention, matching the ring's own azimuth-0 meridian).
4. A construction "radial" line from the origin to the eye-centre point (a fresh sketch
   point, not yet dimensioned).
5. `Dimension.Distance(origin, centre, eye_radius_pos)` — pins the radius.
6. `Dimension.Angle(refLine, radialLine, ear_a_azimuth | ear_b_azimuth)` — pins the
   azimuth. Together with the grounded origin and the two direction constraints this is
   DOF-0 for the centre (2 positional DOF removed by one distance + one angle dimension,
   same counting as `GroundedRadialSection`'s distance-pair).
7. Two circles **centred at that same point** (not two independently-seeded-and-fixed
   circles as `GroundedCircle` does today — that helper fixes a literal `(cx,cy)` seed,
   which would decouple the circles from the parametric centre. This needs either a
   `Coincident` constraint tying each circle's centre to the eye-centre point, or a new
   `CircleAtPoint(pointID, diameterExpr)` sketch helper — an implementation note for the
   builder layer, not a math question): one dimensioned to `eye_outer_dia`, one to
   `plier_hole_dia`. Two concentric circles in one sketch is inherently an annulus
   profile (outer boundary, inner hole) — no boolean subtraction needed, consistent with
   the task's framing.
8. `ExtrudeDirected(sk, "thickness", "new", "symmetric")` — per the confirmed convention
   in `roller_bearing.go` (`ExtrudeNamed(sk, "roller_length", "new", "symmetric")`), the
   **symmetric distance expression is the total span**, split ±half around the sketch
   plane. `"thickness"` (not `"thickness/2"`) on a plane already offset to
   `thickness/2` reproduces exactly z ∈ [0, thickness] — the ring's own axial span.
   Operation `"new"`: a separate body, touching/overlapping the band, no boolean — matches
   the existing multi-body idiom (`roller_cage.go`'s bridge bar is also a second body
   built before patterning).

### Paste-ready DeriveParam list (dependency order, after `PublishParams`)

```
ear_band_width  = "(outer_dia - inner_dia) / 2"
eye_outer_dia   = "ear_band_width * 1.0"     // external; "ear_band_width * 0.9" for internal
plier_hole_dia  = "eye_outer_dia * 0.45"
eye_radius_pos  = "outer_dia / 2 + eye_outer_dia * 0.3"   // external
                = "inner_dia / 2 - eye_outer_dia * 0.3"   // internal
ear_a_azimuth   = "0 deg"
ear_b_azimuth   = splitGapAngle               // "330 deg" — reuse the Go const, don't re-litералize
```
(External vs. internal branch is a Go-level decision — key off `rm.Family.Category`,
same signal `circlip_din471.json`/`circlip_din472.json` already encode as
"External"/"Internal" — not a math question.)

## Numerical pitfalls

- **Winding-sign ambiguity between the revolve sweep and the sketch `Angle` dimension.**
  `Dimension.Angle(l1,l2,expr)` is a two-line angle constraint; nothing here confirms it
  measures the same rotational sense (CCW about +Z from +X) as the host's `Revolve`
  sweep. If it doesn't, ear B lands at the *wrong* azimuth (mirrored across +X, i.e. at
  +30° instead of −30°/330°) while ear A (0°) is unaffected by the ambiguity (0° is its
  own mirror). **Guard**: live-test with a screenshot before trusting `ear_b_azimuth =
  330 deg` — check the eye sits on the same side as the band's actual terminus, not the
  gap's empty side.
- **Unit-strict evaluator + mixed-unit families.** ASME members are `"in"`-unit
  (`circlip_ansi_external.json`); `earMinRim`/`earMinClearance` are written here as
  `"0.3 mm"`. This assumes the evaluator cross-unit-converts a `mm` literal against an
  `in`-family document (the task states literals need *a* unit, not necessarily the
  family's own unit) — **verify this once** against a live ASME member; if conversion
  isn't supported, express the guard constants per-family unit instead (a Go-level
  concern, not a formula change).
- **`kEye`/`eyeOutwardFrac` are representational-arbitrary, not standard.** The five-
  column catalogues here (`d`, `di`, `do`, `s`) carry no lug/hole geometry of their own —
  real DIN 471/472 hole-and-lug dimensions (where they exist, mainly on larger panel
  rings) are a separate, unpublished table this milestone doesn't ingest. Flag this
  plainly in code comments (as `circlip.go`'s existing "representational" language
  already does for `splitGapAngle`) so nobody later treats `eye_outer_dia`'s `1.0`/`0.9`
  factors as sourced from a standard.
- **Guard (b) is scale-sensitive, not scale-invariant.** Because `kEye`, `kHole`,
  `eyeOutwardFrac` are all bare fractions, the *ratio* `c/e` is asymptotically constant
  as size grows — but the additive `earMinClearance` (must carry a unit) means small
  absolute sizes are disproportionately at risk. DIN472 d20 already shows only 15.4%
  headroom after the internal `kEye` correction; a hypothetical smaller/thinner internal
  member (small bore, thin band ⇒ small `w` and small `di`) could fail outright. This is
  exactly why guard (b) must be a per-member Go check with a skip-ears fallback, never a
  one-time "does the formula work" sanity check.
- **Circle-at-a-parametric-point isn't an existing primitive.** `GroundedCircle(cx, cy,
  diameterExpr)` grounds a *literal* seed point — reusing it verbatim for the eye would
  silently decouple the circles from `eye_radius_pos`/azimuth (they'd sit at whatever
  float seed was written, never re-driving). This is a builder-layer gap the
  implementation must close (`Coincident` constraint or a new point-centred circle
  helper) before these formulas do anything — noted here because it would otherwise look
  like a passing guard with a broken result.

## References

This is placement/packing geometry (polar coordinates, a chord-vs-disc-diameter
clearance condition) rather than curve/surface numerics, so the load-bearing references
are practical rather than algorithmic:
- Standard polar-to-Cartesian dimensioning and DOF counting for 2D sketch constraint
  solving follows the same pattern as **Sutherland (1963), Sketchpad**, and the general
  treatment in **Bettig & Hoffmann (2011), "Geometric constraint solving in parametric
  computer-aided design"** — grounding + distance + angle as a minimal DOF-0 placement
  for a point is a standard decomposition in that literature, not a novel derivation here.
- The chord/disc non-overlap condition (`2R sin(θ/2) ≥ d1/2 + d2/2` for two discs of
  diameter `d1,d2` at angular separation `θ` on a common circle of radius `R`) is
  elementary circle geometry — no specialized citation needed beyond noting it is the
  same "packing on a circle" condition used for bolt-circle / pitch-circle minimum-spacing
  checks throughout mechanical CAD practice (cf. the existing `rollerCageBarGapFloor`
  angular-clearance guard in `roller_cage.go` for the same family of check).
- DIN 471 / DIN 472 dimensional data: as already sourced in `circlip_din471.json` /
  `circlip_din472.json` (fasteners.eu-grounded per the project's `SOURCES.md`
  provenance convention — this brief does not introduce new standard data, only
  representational lug proportions layered on top).
