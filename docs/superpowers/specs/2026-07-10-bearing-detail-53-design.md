# Bearing detail (#53) — ball shields & cylindrical-roller detail — Design

**Issue:** Oblikovati.AddIns.PartDesigner #53 — *M2: Ball & cylindrical-roller bearing
detail (grooves, seals, cage, flanges)*. Ball race grooves already shipped (PR#70); this
spec covers the remaining four refinements.

**Goal:** Add ball seals/shields to the deep-groove ball bearing and guide flanges,
roller-end chamfers, and a cage to the cylindrical roller bearing — representationally,
each as an independent DOF-0 parametric body with a *provable* non-intersection guard,
no boolean.

**Delivery:** One PR closing #53, with granular commits (one feature per commit, no
squash). Add-in-only — no Oblikovati.API or host change (every primitive already exists:
`Revolve`, `RevolveAboutCenterline`, `RevolveTwoSided`, `AngledOrientedSketch`, grounded
sections, whole-body-pattern ordering).

## Global constraints

- **No boolean, independent bodies.** Two bodies read as "not touching" only if
  *provably disjoint*. Every guard below is a separating-gap proof in ONE coordinate
  (axial, radial, or angular). Clearance floor `ε_clr = max(0.10 mm, 2·chord_tol)`.
- **DOF-0 parametric.** Every dimension is a published-parameter *formula expression*
  (never a literal coordinate); every section asserts full constraint before the feature.
- **`disableSketchInference()` FIRST** on any section built from lines/polylines/rectangles
  via the router, else auto H/V/coincidence inference → redundant → the revolve silently
  makes NO body (symptom: element bodies present, ring/detail missing). See the M1
  interference lesson and `procedural-sketch-disable-inference`.
- **Whole-body-pattern ordering.** A circular pattern of a new-body source copies *every*
  body present at each occurrence. Build the rolling element (+ cage bar) BEFORE the single
  `PatternCircular`; add rings/shields AFTER so they are not replicated.
- **Fall-back-to-plain** on every guard failure (a degenerate member yields the current
  correct-but-plain body, never a broken one).
- SPDX `GPL-2.0-only` header on every new `.go`; functions 4–20 lines; files <500 lines;
  explicit types; tests mock host I/O with the existing `fakeHost`.

## Existing parameters (inputs)

**Cylindrical roller bearing** (`roller_bearing`, ISO 15 NU2xx): published `bore`,
`outer_dia`, `width`, `roller_count`. Derived: `pitch_dia=(bore+outer_dia)/2`,
`roller_dia=0.28*(outer_dia-bore)`, `roller_length=0.8*width`,
`inner_race_dia=pitch_dia-roller_dia-0.012*(outer_dia-bore)`,
`outer_race_dia=pitch_dia+roller_dia+0.012*(outer_dia-bore)`. Members NU202–NU210.

**Deep-groove ball bearing** (`ball_bearing`, ISO 15 60/62/63): published `bore`,
`outer_dia`, `width`, `ball_count`. Derived: `pitch_dia`, `ball_dia=0.28*(outer_dia-bore)`,
`groove_radius=0.52*ball_dia`, `inner_shoulder_dia=pitch_dia-1.1*groove_radius`,
`outer_shoulder_dia=pitch_dia+1.1*groove_radius`.

## Architecture

**`roller_bearing.go` new `Build` order:**
1. `PublishParams` + `deriveRollerParams` (existing) + `deriveFlangeParams` +
   `deriveRollerChamferParams` + `deriveRollerCageParams`.
2. `patternRollers` rebuilt: build the **chamfered roller** by revolve-about-centerline
   (replaces the plain symmetric extrude), then the **cage bar** (2nd body, if
   `cageBarsFit`), then the single `PatternCircular("roller_count")` over both.
3. `revolveRing("bore","inner_race_dia")` — inner ring stays a plain cylinder (NU design).
4. **Flanged outer ring** (⊐ channel) if `flangesFit`, else plain `revolveRing`.

**`bearing.go`:** after the two grooved rings, `revolveShields` builds two flat annular
shields (both faces) if `shieldsFit`.

**New files** (one responsibility each, `section_*.go` convention):

| File | Contents |
|------|----------|
| `section_flange.go` | `GroundedFlangedRingSection` — 8-point ⊐ outer-ring channel |
| `section_roller_chamfer.go` | `GroundedChamferedRollerSection` — 6-point roller meridian, revolve about own centerline |
| `roller_cage.go` | cage param derivations + `buildRollerCageBar` + `cageBarsFit` (adapts `tapered_cage.go`) |
| `section_shield.go` | `GroundedShieldSection` — 4-point flat annulus + shield derivations + `shieldsFit` |

## Feature 1 — Cylindrical-roller guide flanges (outer ring ⊐ channel)

NU series = two integral flanges on the OUTER ring; inner ring plain. Derived params:

```
flange_axial_clr = max(0.10, 0.02*roller_length)
flange_inner_z   = roller_length/2 + flange_axial_clr     # |z| of flange inner face
flange_bore_dia  = pitch_dia                               # rib reach (mid roller-end annulus)
```

Outer-ring meridian (8 points, revolve 360° about Z, z-symmetric), radii via
`Offset(origin, Z-axis, expr/2)`, axial via `Offset(origin, X-axis, expr)`, all
Horizontal/Vertical edges (no arcs):

```
1 (D/2, -width/2)  2 (D/2, +width/2)  3 (pitch_dia/2, +width/2)  4 (pitch_dia/2, +flange_inner_z)
5 (outer_race_dia/2, +flange_inner_z)  6 (outer_race_dia/2, -flange_inner_z)
7 (pitch_dia/2, -flange_inner_z)  8 (pitch_dia/2, -width/2)  close 8->1
```

**Guard `flangesFit`** (all three ≥ ε_clr; else plain outer ring):
`flange_land=(roller_dia+0.012*(D-d))/2`, `flange_overlap=roller_dia/2`,
`flange_band=width/2-roller_length/2-flange_axial_clr`. Proofs: flange↔inner-ring radial
(land>0), flange↔roller axial (roller |z|≤rl/2 < flange |z|). Worst NU202: land 2.92 mm,
overlap 2.80 mm, band 0.92 mm — never degenerates.

## Feature 2 — Roller-end chamfers (revolve about own centerline)

Replace the plain extruded cylinder with a roller revolved 360° about its own centerline
(`X = pitch_dia/2`, parallel to Z). Envelope (±roller_length/2, pitch_dia±roller_dia)
**unchanged** so Feature-1 overlap/clearances hold.

```
chamfer_leg  = 0.10*roller_dia          # 45deg, equal axial & radial leg
end_face_rad = roller_dia/2 - chamfer_leg
```

Meridian (6 points; local X = global radius; revolve about the centerline at
`roller_axis_x = pitch_dia/2`):

```
P1 (axis, -rl/2)  P2 (axis, +rl/2)  P3 (axis+rd/2-c, +rl/2)  P4 (axis+rd/2, +rl/2-c)
P5 (axis+rd/2, -rl/2+c)  P6 (axis+rd/2-c, -rl/2)  close P6->P1
```

(`axis=roller_axis_x`, `rd=roller_dia`, `rl=roller_length`, `c=chamfer_leg`; the two 45°
chamfer edges pinned by endpoints, no Tangent.) **Guard `rollerChamferFits`**:
`chamfer_leg < roller_dia/2 - ε_clr` and `chamfer_leg ≥ 0.15 mm`; else build the plain
revolved cylinder. k=0.10 gives 5× margin; no member falls back (min leg 0.56 mm at NU202).

## Feature 3 — Cylindrical cage (phased bridge bars only)

A continuous pitch band would pierce the rollers → one bar per inter-roller gap at the
pitch radius, axis ∥ Z, built as a 2nd body at half-pitch azimuth BEFORE the roller
pattern so the single pattern arrays roller+bar together. Derived:

```
roller_subtend   = asin(roller_dia / pitch_dia)     # phi (constant, radians)
cage_half_pitch  = 180 deg / roller_count           # bar azimuth offset
bar_half_angle   = 0.40 * (cage_half_pitch - roller_subtend)
bar_radial_thick = 0.25 * roller_dia
bar_id           = pitch_dia - bar_radial_thick
bar_od           = pitch_dia + bar_radial_thick
bar_axial_len    = 0.70 * roller_length
```

Build: `AngledOrientedSketch("cage_half_pitch")` → `GroundedRingSection(bar_id, bar_od,
bar_axial_len)` → `RevolveTwoSided(z, "bar_half_angle", "new")`. **Guard `cageBarsFit`**
(Go, mirrors the param derivation): `free_half_gap = π/Z - asin(roller_dia/pitch_dia) ≥
0.020 rad`; else no bars (dense-member fallback). NU204 tightest at 0.0342 rad — every
member gets bars. Proofs: bar↔roller angular (residual gap 0.6·free_half_gap>0), bar↔flange
axial (`bar_axial_len/2 = 0.35·rl < rl/2 < flange_inner_z`). **No end ring** — the only
roller-free axial band is claimed by the flanges; an end ring at the pitch radius would
provably intersect them.

## Feature 4 — Ball shields (2Z, both faces)

A flat annular shield revolved about Z on each face, axially outboard of the ball equator,
radially between the shoulders. Derived:

```
axial_slack   = width/2 - ball_dia/2
shield_near_z = ball_dia/2 + 0.20                                   # near face (toward ball)
shield_thick  = min(0.12*width, axial_slack - 0.20 - 0.20)          # clr + ring inset
shield_far_z  = shield_near_z + shield_thick
shield_id     = inner_shoulder_dia + 0.30                           # 2*0.15 above inner shoulder
shield_od     = outer_shoulder_dia - 0.30                           # below outer bore
```

Meridian (4-point flat annulus, revolve 360° about Z, mirrored to both faces):
`1 (shield_id/2, shield_near_z)  2 (shield_od/2, shield_near_z)  3 (shield_od/2,
shield_far_z)  4 (shield_id/2, shield_far_z)` close. **Guard `shieldsFit`**:
`ball_dia ≤ width - 1.4` (i.e. `axial_slack ≥ 0.70`); else no shields. Proofs: shield↔ball
axial (near face |z| > ball_dia/2 = ball max |z|), shield↔ring radial (between shoulders),
shield inside ring axial (`shield_far_z ≤ width/2 - 0.20` by construction). All 60/62/63
pass (worst 6200, slack 1.70 mm).

## Testing

**Per feature — fakeHost unit tests** (in the existing `*_bearing_test.go` style):
- assert each new derived param's exact expression string;
- assert added revolve/pattern **count and ordering** (roller+bar before the single
  pattern; inner ring plain; outer ring flanged; shields after the ball rings);
- assert the DOF-0 `AssertFullyConstrained` path is reached;
- assert each `…Fit` guard returns true for a representative member and false for a
  synthetic degenerate (drives the fall-back-to-plain branch).

**Kernel diagnostic** (host `model/sketch`, pure Go, no router): confirm each new section
solves `DOF=0, Redundant=0` and the solver converges, before trusting the router path.

**Live MCP test** (before the PR): place NU206 (cylindrical) and 6206 (ball) via the panel;
capture screenshots confirming flanges/chamfers/bars/shields read correctly; compare placed
volume to an analytic model within ~0.3%; edit a driver parameter and confirm re-drive.
Exposed/section capture to visually confirm bars sit in the azimuthal gaps and shields clear
the balls (volume alone cannot prove non-intersection without boolean).

## Coverage / hygiene

Coverage >80%, duplication <3%; `make build/test/lint` + markdownlint green on
Linux/macOS/Windows; `Closes #53` in the PR body; auto-merge when green; delete branch
after merge.

## Rejected alternatives

- **Chamfers via an edge Chamfer feature** — fragile topological edge references in a
  procedural build; the profile-built chamfer is DOF-0 and reference-free.
- **Cage with side rings** — physically real, but provably intersects the guide flanges
  here (both want the roller-free end band); bars-only is the correct representation given
  our flanges.
- **Single shield (Z)** — user chose 2Z (both faces).
