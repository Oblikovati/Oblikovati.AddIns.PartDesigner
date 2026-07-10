# Geometry brief — #53 representational bearing detail (4 problems)

Advisory only: parameter-expression formulas + degeneracy guards for four detail
features in the PartDesigner Content-Center bearings. All bodies stay INDEPENDENT
(no boolean); every guard below is a *provable non-intersection* argument evaluated
across every tabulated ISO 15 member. Convention reminder (matches existing code):
meridian in XZ, X = radius, Z = axial; a **crest DIAMETER** = pitch_dia ± element_dia
because doubling the pitch-radius ± element-radius, `2·(PD/2 ± rd/2) = PD ± rd`; an
axial **half-span** = element_dia/2 (a radius). Roller/ball centre sits on the pitch
circle at radius PD/2, element radius = element_dia/2.

Shared clearance policy: because there is no boolean, two independent bodies read as
"not touching" only if they are *provably disjoint*. The cheapest disjointness proof
is a **separating gap in one coordinate** (axial or radial or angular) — I lean on
that everywhere rather than on volume subtraction. Set the absolute clearance floor
`ε_clr` to a few × the tessellation chord error, e.g. `ε_clr = max(0.10 mm, 2·chord_tol)`;
1.5–2 tessellation facets. Every gap below is ≥ `ε_clr` for the worst member.

---

## PROBLEM 1 — Cylindrical-roller guide flanges (NU2xx, two integral ribs on OUTER ring)

### Problem framing
Turn the plain outer ring into an inward-opening channel (⊐ in meridian): a raceway
band at `outer_race_dia` over the middle, and at each axial end a rib projecting
radially INWARD to axially locate the roller ends. Sought: the flange axial band
(two |z| stations), the flange bore diameter (radial reach), proven disjoint from
(a) the rollers except as a projected "locating" overlap, and (b) the plain inner
ring (spans `bore..inner_race_dia` over full width B).

### Derived parameters (DeriveParam name / expr)
```
flange_bore_dia   = pitch_dia                              # mid of roller end annulus
raceway_dia       = outer_race_dia                          # existing
flange_axial_clr  = max(0.10, 0.02*roller_length)          # gap roller-end -> flange face
flange_inner_z    = roller_length/2 + flange_axial_clr      # |z| of flange inner face
flange_outer_z    = width/2                                 # |z| of flange outer face (ring end)
flange_band       = flange_outer_z - flange_inner_z         # rib axial thickness
flange_land       = (pitch_dia - inner_race_dia)/2          # radial land above inner ring
flange_overlap    = (pitch_dia + roller_dia)/2 - pitch_dia/2 = roller_dia/2   # projected locate depth
```
`flange_bore_dia = pitch_dia` is the load-bearing choice: it puts the rib bore exactly
mid-annulus of the roller end (annulus in diameter = `PD−rd .. PD+rd`), so the rib
dips `roller_dia/2` below the roller outer crest (reads as "locating") yet leaves a
`(PD−inner_race_dia)/2` land above the inner ring.

### Meridian point layout (outer-ring ⊐ channel, revolve 360° about Z; z-symmetric)
```
1 (D/2,               -width/2)      2 (D/2,               +width/2)     # OD cylinder
3 (pitch_dia/2,       +width/2)      4 (pitch_dia/2,       +flange_inner_z)  # +flange: face in, bore down
5 (outer_race_dia/2,  +flange_inner_z)                                  # flange inner face out to raceway
6 (outer_race_dia/2,  -flange_inner_z)                                  # raceway cylinder across mid
7 (pitch_dia/2,       -flange_inner_z) 8 (pitch_dia/2, -width/2)        # -flange mirror
close 8->1 (-end face)
```
Solver: all radial coords via `Offset(origin, Z-axis, expr/2)`; all z via
`Offset(origin, X-axis, expr)`; `Fix` origin; profile is Horizontal/Vertical edges only,
DOF-0. No arcs → no center-pinning needed.

### Build-decision guard (scalar, per member)
Build flanges iff **all three** hold; else fall back to plain outer ring:
```
G1a  flange_land   = (roller_dia + 0.012*(D-d))/2  >= ε_clr        # positive land above inner ring
G1b  flange_overlap = roller_dia/2                 >= ε_clr        # actually locates the roller
G1c  flange_band    = width/2 - roller_length/2 - flange_axial_clr >= ε_clr   # visible rib
```
Non-intersection proofs: flange↔inner-ring is **radial** (`flange_bore/2 = PD/2 >
inner_race_dia/2`, gap = flange_land); flange↔roller is **axial** (roller |z| ≤ rl/2,
flange |z| ≥ rl/2 + flange_axial_clr). Both gaps > 0 ⇒ disjoint bodies.

### Degeneracies / detection
- *No radial room* (flange bore trapped between roller crest and inner-race land): detect
  by `G1a` or `G1b` ≤ ε_clr. Cannot occur here — land = (rd+0.012ΔD)/2 > 0 identically
  because rd = 0.28ΔD > 0.
- *Negative overhang* (roller longer than ring half): detect by `G1c`. Cannot occur —
  roller_length = 0.8·width ⇒ overhang = 0.1·width > 0 always.

### Worked worst case — NU202 (15/35/11, Z=11)
PD=25.0, rd=5.60, rl=8.80, inner_race=19.16, outer_race=30.84.
`flange_land = (25.0−19.16)/2 = 2.92 mm` (min over family). `flange_overlap = 2.80 mm`.
`flange_band = 5.5 − 4.4 − 0.176 = 0.924 mm` (overhang before clr = 1.10 mm, the family
minimum). All ≫ ε_clr ⇒ every NU2xx member gets flanges. Land grows monotonically to
5.84 mm (NU208-210); overhang grows to 2.00 mm (NU210).

Grounding: an NU-type outer ring is *physically* the two-integral-flange member (inner
ring plain) — Harris, *Rolling Bearing Analysis*, ch. 1 (bearing types). `flange_bore =
pitch_dia` is representational-arbitrary (a real rib bore is set by roller-end chamfer +
guide clearance); it is chosen to make the locate-overlap and inner-ring land both
provably positive for the whole family.

---

## PROBLEM 2 — Roller-end chamfers (revolve roller about its own centerline)

### Problem framing
Replace the symmetric extruded cylinder with a roller REVOLVED 360° about its own
centerline (the line X = pitch_dia/2, parallel to Z). Meridian = rectangle from the
centerline (ρ=0) out to ρ = roller_dia/2, spanning ±roller_length/2 in Z, with the two
OUTER corners (roller ends) cut by a 45° chamfer of leg `c`. Envelope must stay identical:
axial ±roller_length/2, radial pitch_dia ± roller_dia — so Problem-1 flange overlap and
race clearances are unchanged.

### Derived parameters
```
roller_axis_x   = pitch_dia/2                     # revolve axis (line parallel to Z)
roller_half_z   = roller_length/2
roller_rad      = roller_dia/2
chamfer_frac    = 0.10                             # k: leg as fraction of roller_dia
chamfer_leg     = chamfer_frac*roller_dia          # 45deg -> equal axial & radial leg
end_face_rad    = roller_dia/2 - chamfer_leg       # radius of remaining flat end disc
```

### Meridian point layout (local X = global radius; revolve about X = roller_axis_x)
```
P1 (roller_axis_x,                    -roller_half_z)          # bottom, on axis
P2 (roller_axis_x,                    +roller_half_z)          # top, on axis (revolve edge)
P3 (roller_axis_x + roller_rad - c,   +roller_half_z)          # +end face out to chamfer
P4 (roller_axis_x + roller_rad,       +roller_half_z - c)      # +chamfer (45deg)
P5 (roller_axis_x + roller_rad,       -roller_half_z + c)      # crest cylinder
P6 (roller_axis_x + roller_rad - c,   -roller_half_z)          # -chamfer
close P6->P1
```
c = `chamfer_leg`. Crest radius `roller_dia/2` preserved along P4→P5 (envelope intact);
end plane still at ±roller_length/2 (P3,P6) so flange projected-overlap unchanged; end
disc radius shrinks to `end_face_rad`. DOF-0 via Offset dims; the two 45° chamfer edges
are pinned by their two endpoints (no Tangent needed).

### Build-decision guard
```
G2a  chamfer_leg < roller_dia/2 - ε_clr     # end disc stays real (radius > 0)
G2b  chamfer_leg >= c_min                    # else invisible -> build PLAIN roller
     c_min = max(0.15 mm, 2*chord_tol)
```
With k=0.10, `chamfer_leg = 0.10·roller_dia` and `roller_dia/2 = 0.50·roller_dia`, so
G2a holds with 5× margin for *every* member — the 45° chamfer can never eat the end face.

### Degeneracies
- *End face vanishes*: c ≥ roller_dia/2 (radius binding, since rl ≫ rd here so the axial
  leg c < roller_length/2 is never binding — check: min roller_length/2 = 4.40 mm ≫ max
  chamfer_leg = 1.12 mm). Detect by G2a. Guarded by k = 0.10 < 0.5.
- *Sub-pixel chamfer*: fall back to the existing plain revolved cylinder when G2b fails.
  A rounded (arc) alternative would be gated on **sagitta** `s = r − √(r²−(c)²)` of the
  end radius vs chord_tol; we use a straight 45° chamfer so the visibility floor is just
  the leg length `c_min`.

### Worked worst case — NU202: `chamfer_leg = 0.10·5.60 = 0.56 mm`, end disc radius
`2.80 − 0.56 = 2.24 mm > 0`. This is the family MINIMUM leg; 0.56 mm > c_min = 0.15 mm ⇒
no member falls back. Leg range 0.56 mm (NU202) → 1.12 mm (NU208/9/10). Grounding: ISO
355 / ISO 15 define a roller end chamfer `r_s`; k=0.10 is a representational stand-in for
the tabulated chamfer (real `r_s` is ~0.3–0.6 mm absolute here) — physically motivated,
numerically arbitrary in the exact fraction.

---

## PROBLEM 3 — Cylindrical cage (phased bridge bars, one per inter-roller gap)

### Problem framing
A continuous pitch-radius band would pierce the rollers, so the representational cage =
one bar per gap at the pitch radius, axis ∥ Z, built as a 2nd body BEFORE the roller
circular pattern and phased by half a pitch (azimuth `π/roller_count`) so a single
`PatternCircular(count = roller_count)` arrays roller+bar together and each bar lands in
a gap. Each cylindrical roller subtends a CONSTANT half-angle about the axis
`φ = asin(roller_dia / pitch_dia)` (tangent half-angle of a circle of radius roller_dia/2
whose centre is at radius pitch_dia/2). Angular pitch = 2π/Z; bar centre is at π/Z from
each neighbour roller centre.

### Derived parameters
```
roller_subtend    = asin(roller_dia / pitch_dia)          # phi, radians (evaluator: asin)
half_pitch        = 3.14159265/roller_count               # pi/Z, bar azimuth offset
free_half_gap     = half_pitch - roller_subtend           # angular room from bar centre to roller edge
bar_half_angle    = 0.40*free_half_gap                    # psi: 60% clearance each side
bar_radial_thick  = 0.25*roller_dia                        # centred on pitch radius
bar_id            = pitch_dia - bar_radial_thick           # meridian inner radius*2
bar_od            = pitch_dia + bar_radial_thick
bar_axial_len     = 0.70*roller_length                     # short of the flanges
```

### Build geometry
Sketch plane tilted to azimuth `half_pitch`; meridian rectangle radius
`bar_id/2 .. bar_od/2`, axial `±bar_axial_len/2`; revolve about Z through
`±bar_half_angle` (total sweep `2·bar_half_angle`). Then the single circular pattern of
count `roller_count` steps by `2π/Z`, carrying roller (at azimuth 0) and bar (at
`π/Z`) together.

### Build-decision guard
```
G3a  free_half_gap = pi/Z - asin(roller_dia/pitch_dia) >= fh_min      # else NO bars
     fh_min = 0.020 rad  (~1.15 deg)  # = psi_min(0.5deg) + clearance(0.3deg) with headroom
G3b  bar_half_angle + roller_subtend < pi/Z    # (auto: psi = 0.4*free_half_gap < free_half_gap)
```
Non-intersection proof (bar↔roller): angular separation of centres = π/Z; roller occupies
±φ, bar occupies ±ψ; residual angular gap = π/Z − φ − ψ = 0.60·free_half_gap > 0 ⇒
disjoint. Bar↔flange is **axial**: `bar_axial_len/2 = 0.35·roller_length < roller_length/2
< flange_inner_z` ⇒ bar ends inboard of both roller ends and flanges.

### Degeneracy — dense members
`free_half_gap → 0` as roller fills its pitch cell (large Z, fat roller). Detect by G3a;
below `fh_min` build NO bars for that member (cage omitted, representationally acceptable
for a very dense complement). Worst family member = **NU204** (Z=12): `pi/12 = 15.000°`,
`φ = asin(7.56/33.5) = 13.042°`, `free_half_gap = 1.958° = 0.0342 rad`. This is > fh_min
= 0.020 rad ⇒ NU204 (and every other NU2xx) DOES get bars, but NU204 is the tightest and
its bars are thin (`ψ = 0.784°`, bar arc ≈ `2ψ·pitch_dia/2 ≈ 0.46 mm`). All other members
have free_half_gap 2.6–3.7°. No listed member hits the floor.

### Bars-only vs bars + end ring — decision: BARS ONLY
The only roller-free axial band is |z| ∈ [roller_length/2, width/2], width = 0.1·width
(≤ 2.0 mm, min 1.10 mm at NU202). On the OUTER ring that band is fully claimed by the
Problem-1 guide flanges, whose bore dips to radius pitch_dia/2. A cage end ring living at
the pitch radius would need body straddling `pitch_dia/2` in that same axial band and
would therefore **overlap the flanges radially** (flange occupies pitch_dia/2 .. D/2
there) — a provable intersection, forbidden without boolean. An end ring pushed radially
inboard of the flange bore no longer connects to the bars at pitch radius. Hence: no room
for an end ring; use the phased bars ("ladder") only. Physical caveat: real NU cages
(pressed-steel or machined) *do* have side rings; bars-only is a defensible representation
here specifically because our flanges occupy the end bands. Grounding: Harris ch. 1 on
cage/separator types; the constant roller subtend φ is exact geometry, ψ = 0.4·free_gap
is representational.

---

## PROBLEM 4 — Ball seals / shields (2Z metal, both faces of deep-groove bearing)

### Problem framing
A flat annular shield revolved about Z on each face, spanning radially across the groove
mouth (above the inner-ring shoulder, below the outer-ring bore), sitting axially just
OUTSIDE the ball equator but INSIDE the ring end faces (±width/2). Balls: centre on pitch
circle, span ±ball_dia/2 axially, `pitch_dia ± ball_dia` radially. Groove shoulders:
`inner_shoulder = pitch_dia − 1.1·groove_radius`, `outer_shoulder = pitch_dia +
1.1·groove_radius`, `groove_radius = 0.52·ball_dia` ⇒ shoulders at `pitch_dia ∓ 0.572·ball_dia`.

### Derived parameters
```
ball_half_span   = ball_dia/2                                  # ball axial reach
axial_slack      = width/2 - ball_dia/2                        # s: room outboard of ball
shield_near_z    = ball_dia/2 + shield_clr_ax                  # near face (toward ball)
shield_clr_ax    = 0.20                                        # >= eps_clr, clears ball equator
shield_ring_inset= 0.20                                        # far face inside ring end
shield_thick     = min(0.12*width, axial_slack - shield_clr_ax - shield_ring_inset)
shield_far_z     = shield_near_z + shield_thick                # <= width/2 - shield_ring_inset (proven)
shield_id        = inner_shoulder_dia + 2*shield_clr_rad       # a hair above inner shoulder
shield_od        = outer_shoulder_dia - 2*shield_clr_rad       # a hair below outer bore
shield_clr_rad   = 0.15
```

### Meridian point layout (flat ring, revolve 360° about Z; mirror to both faces)
```
1 (shield_id/2, shield_near_z)  2 (shield_od/2, shield_near_z)
3 (shield_od/2, shield_far_z)   4 (shield_id/2, shield_far_z)   close
```
Radial dims via Offset(origin, Z-axis, ·/2); axial via Offset(origin, X-axis, ·); DOF-0.

### Build-decision guard
```
G4  axial_slack = width/2 - ball_dia/2 >= s_min
    s_min = shield_clr_ax + shield_ring_inset + t_min,  t_min = 0.30 mm  -> s_min = 0.70 mm
    equivalently:  ball_dia <= width - 1.4 mm   i.e.  0.28*(D-d) <= width - 1.4
```
Non-intersection proofs:
- *Shield ↔ ball* — **axial**: shield near face |z| = ball_dia/2 + shield_clr_ax >
  ball_dia/2 = the ball's max |z|. The shield is entirely axially outboard of the ball's
  axial extent ⇒ cannot intersect regardless of radial overlap. (This is why the shield may
  sit radially *inside* the ball's `pitch±ball_dia` envelope — real 2Z shields do.)
- *Shield ↔ rings* — **radial**: `shield_id/2 > inner_shoulder/2` (gap shield_clr_rad) and
  `shield_od/2 < outer_shoulder/2` (gap shield_clr_rad); annulus width =
  `0.572·ball_dia − 2·shield_clr_rad > 0` always (ball_dia large).
- *Shield inside ring* — **axial**: `shield_far_z ≤ width/2 − shield_ring_inset` holds by
  construction because `shield_thick ≤ axial_slack − clr − inset`.

### Degeneracy — ball too fat for the width
`axial_slack ≤ 0` ⇔ ball_dia ≥ width (ball wider than ring) — then no shield. More
practically, `axial_slack < s_min` ⇒ FALL BACK to no shields. In ratio form the fat-ball
guard is `ball_dia/width = 0.28·(D−d)/width ≤ (width − 1.4)/width`.

### Member set assumed (ISO 15 deep-groove; ball_dia = 0.28·(D−d))
60 series 6000–6008, 62 series 6200–6208, 63 series 6300–6308. `ball_dia/width` ranges
0.504 → 0.636; `axial_slack` ranges 1.70 → 4.50 mm. **Worst = 6200** (10/30/9):
`ball_dia = 5.60`, `axial_slack = 4.5 − 2.8 = 1.70 mm`. Guard `5.60 ≤ 9 − 1.4 = 7.6` ✓;
`s_min = 0.70 < 1.70` ⇒ shields fit. Every listed 60/62/63 member passes (slack ≥ 1.70).
Chosen shield at 6200: near_z = 3.00, thick = min(1.08, 1.30) = 1.08, far_z = 4.08 ≤
4.50 − 0.20 = 4.30 ✓. Fat-ball fallback would trigger only for a hypothetical member with
`0.28(D−d) > width − 1.4` (none in this set; the wide 63 series stays safe because width
grows with the ball). Grounding: 2Z = two metal shields snapped into the outer-ring
groove, radial reach across the groove mouth between the shoulders — Harris ch. 1
(seals/shields). The 0.12·width thickness cap and 0.20 mm absolute clearances are
representational; the shoulder-to-shoulder radial span and the "axially outboard of the
ball equator" placement are physically grounded.

---

## References
- **Harris, T.A. & Kotzalas, M.N., _Rolling Bearing Analysis_, 5th ed. (2007)** — bearing
  types (NU integral flanges, deep-groove groove/shoulder geometry, cage/separator and
  shield types); the load-bearing source for all four features' physical grounding.
- **ISO 15 (boundary dimensions, radial bearings)** and **ISO 355 / ISO 15:2017** — the
  bore/OD/width tables and roller-end chamfer `r_s` referenced by Problem 2.
- **ISO 76 / ISO 281** — context for shoulder and raceway proportions (not directly used;
  our shoulder diameters come from the existing `1.1·groove_radius` derivation).
- Geometry of the constant roller subtend `φ = asin(r/R)` and the separating-gap
  disjointness arguments are elementary; no specialized citation. The tolerance-to-model-
  scale discipline (`ε_clr` tied to chord error, not a magic constant) follows standard
  robust-predicate practice — see e.g. Ericson, _Real-Time Collision Detection_ (2004),
  ch. 11 on tolerances and robustness.
