# Standards data provenance

Every dimension table under `data/` must be grounded in an **official standard or an
authoritative standards-body / manufacturer table**, not estimated. This file records the
source of each family's numbers so they can be audited and re-verified. When adding or editing
a family, cite the source here (URL + what was taken from it) in the same change.

Values are the standard **nominal** dimensions (for toleranced fields, the nominal or the
mid-range of the published min/max), in millimetres.

## Size coverage (2026-07 expansion)

The fastener families were expanded from a 4–6-size seed to each standard's **full official
preferred series up to ~50 mm nominal** (metric) / ~2 in (inch). Provenance for the added sizes:

- **Metric bolts/screws/nuts/washers/studs/rod** — dimensions transcribed from the same
  fasteners.eu tables already cited per family below (ISO 4017/4032/4035/4762/10642/7089,
  DIN 127/939/976). The coarse **pitch `P` is the canonical ISO 261 coarse-thread series**
  (M1.6 = 0.35 … M48 = 5.0), authoritative and independent of the source page. Every added row's
  M6–M12 overlap was cross-checked against the previously-verified values and matched exactly.
- **Inch (ASME B18.2.1 / B18.2.2 / B18.3 / B18.22.1)** — across-flats/head/socket/washer
  dimensions from authoritative ASME reproductions (Boltport, Torqbolt, Fastenal, BHAM, Alma
  Bolt), each cross-checked against ≥2 concordant tables and the 6 pre-verified anchor rows.
- **Representative lengths** — `l` (and fully-threaded `b = l`) for bolts/screws/socket screws is
  a representative value ≈ 5·d (metric) / 4·D (inch) rounded to a preferred length; the standards
  fix a length *range* per diameter, not one length. `l` is user-overridable at placement.
- **Sizes deliberately dropped or capped** (official source did not confirm them — not guessed):
  ISO 10642 stops at **M20** (a suspect M24 row was discarded); ISO 7089 stops at **M36** (the
  page tabulates no M42/M48); DIN 939 stud starts at **M6** (no smaller size is tabulated);
  ASME B18.22.1 washer omits the **3/16** row (genuine cross-source disagreement).

## Fasteners / Washers

### `washer_din127.json` — DIN 127 Form B (spring lock washer)
- Source: fasteners.eu DIN 127 B — <https://www.fasteners.eu/standards/din/127-B/>
  (cross-checked against ananka fasteners' DIN 127 B chart).
- Columns: `d1` (inside dia, min), `d2` (outside dia, max), `s` (section thickness),
  `H` (free/unloaded height — nominal = mid-range of the published min/max).
- Verbatim (M6/M8/M10/M12): d1 = 6.4 / 8.1 / 10.2 / 12.2; d2 = 11.8 / 14.8 / 18.1 / 21.1;
  s = 1.6 / 2.0 / 2.2 / 2.5; H range = 3.2–3.8 / 4.0–4.7 / 4.4–5.2 / 5.0–5.9
  (modelled at the mid-range 3.5 / 4.35 / 4.8 / 5.45).

### `washer_iso7089.json` — ISO 7089 (plain washer, 200 HV, product grade A)
- Source: fasteners.eu ISO 7089 — <https://www.fasteners.eu/standards/ISO/7089/>.
- Columns: `d1` (inside dia), `d2` (outside dia), `h` (thickness).
- Verbatim (M6/M8/M10/M12): d1 = 6.4 / 8.4 / 10.5 / 13; d2 = 12 / 16 / 20 / 24;
  h = 1.6 / 1.6 / 2.0 / 2.5.

### `washer_din125.json` — DIN 125 A (plain washer)
- Dimensionally identical to ISO 7089 for these sizes (ISO 7089 supersedes DIN 125 A and
  fasteners.eu tabulates DIN 125 under the ISO 7089 entry). Same values as `washer_iso7089.json`.

## Fasteners / Bolts & screws

### `hex_bolt_iso4017.json` / `hex_bolt_din933.json` — ISO 4017 / DIN 933 (fully-threaded hex screw)
- Source: fasteners.eu ISO 4017 — <https://www.fasteners.eu/standards/ISO/4017/> (the site tabulates
  DIN 933 and ISO 4017 together; the two are dimensionally identical).
- Columns: `s` (across flats), `k` (head height), `P` (coarse pitch). Verified (M6/M8/M10/M12):
  s = 10/13/16/18, k = 4.0/5.3/6.4/7.5, P = 1.0/1.25/1.5/1.75. Matched — no change.
- `l` (length) and `b` (thread length) are representative example lengths (30/40/50/60), fully
  threaded so b = l; not fixed by the standard per diameter.

### `socket_screw_iso4762.json` / `socket_screw_din912.json` — ISO 4762 / DIN 912 (socket cap screw)
- Source: fasteners.eu ISO 4762 — <https://www.fasteners.eu/standards/ISO/4762/> (DIN 912 identical).
- Verified: dk = 10/13/16/18, k = 6/8/10/12, socket s = 5/6/8/10, socket depth t = 3/4/5/6. Matched.

### `socket_screw_iso10642.json` — ISO 10642 (countersunk socket cap screw)
- Source: fasteners.eu ISO 10642 — <https://www.fasteners.eu/standards/ISO/10642/>.
- Verified/**corrected**: dk (nominal=max) = 12/16/20/24 (was 12.6/17.3 for M6/M8 — those were the
  theoretical sharp-corner dk); k = 3.3/4.4/5.5/6.5; socket s = 4/5/6/8; socket depth t (nominal=max)
  = 2.5/3.5/4.4/4.6 (was 2.0/2.6/3.0/3.5).

## Fasteners / Nuts

### `hex_nut_iso4032.json` — ISO 4032 (hexagon nut, style 1)
- Source: fasteners.eu ISO 4032 — <https://www.fasteners.eu/standards/ISO/4032/>.
- Verified: s (max) = 10/13/16/18, m (max) = 5.2/6.8/8.4/10.8. Matched — no change.

### `hex_nut_din934.json` — DIN 934 (hexagon nut)
- Source: fasteners.eu DIN 934 — <https://www.fasteners.eu/standards/din/934/>.
- **Corrected** to the current harmonized values (modern DIN 934 = ISO 4032): s = 10/13/16/18
  (was 17/19 for M10/M12), m = 5.2/6.8/8.4/10.8 (was 5.0/6.5/8.0/10.0, the older DIN series).

### `hex_nut_iso4035.json` — ISO 4035 (hexagon thin nut, chamfered)
- Source: fasteners.eu ISO 4035 — <https://www.fasteners.eu/standards/ISO/4035/>.
- Across-flats `s` follows the harmonized ISO 272 hex series, **identical to ISO 4032** — a thin
  nut is a standard nut cut thinner, so it takes the same wrench size (M10 = 16, M12 = 18).
  **Reconciled** the 2026-07 size-coverage expansion to 16/18: fasteners.eu still tabulates the
  legacy DIN 439 widths (M10/M12 = 17/19) under ISO 4035, but ISO 4035:2012 uses the ISO 272
  series. `m` (max nut height, the thin value) = 3.2/4.0/5.0/6.0 for M6–M12, all below the
  ISO 4032 standard-nut heights.

## Fasteners / Studs & threaded rod

### `stud_din976.json` — DIN 976 (metric threaded rod / studding)
- Source: fasteners.eu DIN 976 — <https://www.fasteners.eu/standards/DIN/976/>.
- Continuous thread over the whole rod; `d`, `P` (coarse), `l` (stock length). Verbatim
  (M6/M8/M10/M12): P = 1.0 / 1.25 / 1.5 / 1.75. Length `l` = 1000 mm (the standard 1 m stock
  cut, as with `round_bar_iso1035`), not a per-size standard dimension.

### `stud_din939.json` — DIN 939 (double-end stud, metal end ≈ 1.25 d)
- Source: fasteners.eu DIN 939 — <https://www.fasteners.eu/standards/DIN/939/>, cross-checked
  against Fuller Fasteners / Aspen / TorqBolt DIN 939 tables.
- Columns: `d`, `P` (coarse), `l` (overall length), `b1` (metal-end thread length = 1.25 d),
  `b2` (nut-end thread length). Verbatim (M6/M8/M10/M12): b1 = 7.5 / 10 / **12.5** / 15
  (= 1.25 d; fasteners.eu tabulates M10 as a rounded 12, but the standard value is 12.5),
  b2 = 18 / 22 / 26 / 30 (= 2 d + 6, the value for nominal length l ≤ 125 mm).
- `l` = 40 / 50 / 60 / 70 mm are representative catalogue lengths (DIN 939 lists 12–200 mm per
  size); the plain shank between the two threaded ends is l − b1 − b2, which stays positive.

## Fasteners / ANSI inch (Unified UNC)

The ANSI inch families are authored in inches (`"units": "in"`, the loader converts) and carry an
explicit `thread` text column (the Unified designation, e.g. `1/4-20`) that the generators feed to
the host `ParseThreadDesignation` verbatim — the same generators as the metric families, driven from
the inch designation instead of a metric d/P pair.

### `hex_bolt_ansi_b18_2_1.json` — ASME B18.2.1 (hex bolt / hex cap screw, UNC)
- Width across flats F and head height H from ASME B18.2.1 (1/4 → F=7/16, H=11/64; 1/2 → F=3/4,
  H=11/32; 3/4 → F=1‑1/8, H=1/2). Nominal diameter D and UNC TPI (1/4‑20 … 3/4‑10) verbatim.
  Length l is a representative value. Fully threaded (cosmetic).

### `hex_nut_ansi_b18_2_2.json` — ASME B18.2.2 (finished hex nut, UNC)
- Width across flats F and thickness H from ASME B18.2.2 (1/4 → F=7/16, H=7/32; 1/2 → F=3/4,
  H=7/16). Bore is the nominal diameter; cosmetic UNC thread on the bore wall.

### `socket_screw_ansi_b18_3.json` — ASME B18.3 (socket head cap screw, UNC)
- Head diameter A, head height H (= D), hex socket size J and key engagement T from ASME B18.3
  (1/4 → A=3/8, J=3/16; 1/2 → A=3/4, J=3/8). Cylindrical head; cosmetic UNC thread on the shank.

### `washer_ansi_b18_22.json` — ASME B18.22.1 (Type A plain washer, narrow)
- Inside diameter, outside diameter and thickness from the SAE/ASME B18.22.1 narrow (Type A)
  series (1/2 → ID 0.531, OD 1.062, t 0.095). No thread.

## Structural

### `round_bar_iso1035.json` — ISO 1035 (hot-rolled round steel bar)
- Nominal preferred stock diameters (10/12/16/20/25 mm) per ISO 1035-1; length is a representative
  stock cut (1000 mm), not a per-size standard dimension.

### `flat_bar_en10058.json` — EN 10058 (hot-rolled flat steel bar for general purposes)
- Nominal width `b` and thickness `a` are drawn from the preferred nominal dimensions tabulated in
  EN 10058-1 (widths 10–150 mm, thicknesses 3–60 mm); the six members (20×5, 25×5, 30×6, 40×8,
  50×8, 60×10) are common commercially-stocked width×thickness combinations from that table.
- Length is a representative mill/stock length (6000 mm), user-overridable at placement — EN 10058
  does not fix a bar length, only the cross-section and its tolerances.

### `ipe_en10365.json`, `hea_en10365.json`, `heb_en10365.json` — EN 10365 (hot-rolled I / H sections)
- Section dimensions h (height), b (flange width), tw (web thickness), tf (flange thickness) and
  the web/flange **root radius r** are the nominal values tabulated in EN 10365:2017 for the IPE,
  HE A and HE B series (e.g. IPE 200 = 200/100/5.6/8.5, r 12; HE 200 B = 200/200/9.0/15.0, r 18).
  Root radii by nominal height: IPE 100 → 7, 200 → 12, 300 → 15, 400 → 21; HE 100 → 12, 200 → 18,
  300 → 27 (A and B share the same r per height). A representative subset of each series is included.
- **Root fillets modelled** (#51): the generator builds the I-outline with a real concave quarter-
  round fillet of radius `r` at each of the four web/flange junctions, so the extruded cross-section
  matches the EN tabulated sectional area (e.g. IPE 200 → 28.48 cm², reproduced to ~0.01 %). The
  overall depth, flange width and plate thicknesses are exact.
- Length is a representative stock length (6000 mm), user-overridable.

### `w_aisc.json`, `c_aisc.json` — AISC (US wide-flange W and American Standard channel C shapes)
- Section dimensions d (depth), bf (flange width), tw (web thickness), tf (flange thickness) are the
  nominal values from the **AISC Shapes Database v15.0** (Steel Construction Manual), in inches — e.g.
  W8×31 = 8.00/8.00/0.285/0.435; C10×15.3 = 10.0/2.60/0.240/0.436. A representative subset is included.
- **W shapes** have parallel flanges, so the four web-flange **root fillets are modelled** (#51,
  shared with the EN I-beam recipe). The fillet radius is derived from the AISC detailing dimension
  as `root_radius = kdes − tf` (kdes = design distance from the outer flange face to the web toe of
  the fillet), per the AISC Shapes Database v15.0 — e.g. W8×31 → kdes 0.829, tf 0.435 → r = 0.394 in.
  Cross-checked against the published area: the modelled filleted W8×31 area is 9.12 in² vs the
  tabulated 9.13 in². **C shapes** have a tapered inner flange face; tf is the AISC average flange
  thickness and the taper + root/toe radii are deferred (as with the EN UPN channel — modelling
  fillets on a sloped flange is incompatible with the axis-aligned centre-pinning fillet recipe).
- Length is a representative stock length (240 in = 20 ft), user-overridable.

### `upn_en10279.json` — EN 10279 (hot-rolled taper-flange channels, UPN)
- Section dimensions h, b, tw, tf are the nominal values tabulated in EN 10279:2000 for the UPN
  series (e.g. UPN 200 = 200/75/8.5/11.5, where tf is the standard reference flange thickness). A
  representative subset (UPN 80/100/160/200) is included.
- **Modelled sharp, constant flange thickness**: the ~5 % (≈2.9°) inner-flange taper and the
  root/toe radii of the real UPN section are deferred refinements. The section is symmetric about
  its X axis with the web on the left; overall height, flange reach, web and flange thicknesses are
  exact.
- Length is a representative stock length (6000 mm), user-overridable.

### `angle_equal_en10056.json`, `angle_unequal_en10056.json` — EN 10056 (hot-rolled angles, L)
- Leg lengths a, b and thickness t are the nominal values tabulated in EN 10056-1 for equal-leg
  (a = b, e.g. L 60×60×6) and unequal-leg (e.g. L 100×65×8) angles. A representative subset of each
  is included.
- **Root fillet r1 and toe radii r2 modelled** (#51): the concave root fillet at the inner heel and
  the convex radius at each leg tip are the EN 10056-1 tabulated values (e.g. L 40×40×4 → r1 = 6,
  r2 = 3), cross-checked against the published sectional area — the modelled filleted area for
  L 40×40×4 is 3.079 cm² vs the tabulated 3.08 cm². The heel (outer corner) stays sharp and sits at
  the part origin.
- Length is a representative stock length (6000 mm), user-overridable.

### `tee_en10055.json` — EN 10055 (hot-rolled equal-flange tees, T)
- Height h (= flange width b) and thickness (web = flange = s) are the nominal values tabulated in
  EN 10055 for the equal-flange tee series (e.g. T 50 = 50/50/6/6). A representative subset
  (T 40…T 80) is included.
- **Root fillet r1 modelled** (#51): the concave root fillet at each flange-stem junction is
  r1 = 2 mm, per the Montanstahl "Equal Flange Tees" datasheet (EN 10055:1995 geometry), which
  tabulates a 2 mm root radius across the T 20…T 100 range and whose published sectional areas are
  reproduced by the two root fillets alone (e.g. T 50×50×6 → sharp 5.64 cm² + 2·r1²(1−π/4) =
  5.66 cm², matching the tabulated 5.66 cm²). The flange **toes are left sharp**: EN 10055 rounds
  them, but the source's tabulated area shows no toe-area removal (the toe radii net out below
  tabulation precision), so modelling only the roots keeps the extruded area faithful to the
  standard. The section is symmetric about the Y axis (flange on top).
- Length is a representative stock length (6000 mm), user-overridable.

### `shs_en10219.json`, `rhs_en10219.json`, `chs_en10219.json` — EN 10219 (cold-formed hollow sections)
- Outer size (b×h for SHS/RHS, outer diameter d for CHS) and wall thickness t are the nominal
  values tabulated in EN 10219-2 for square (e.g. SHS 100×100×5), rectangular (e.g. RHS 120×60×5)
  and circular (e.g. CHS 88.9×4) hollow sections. A representative subset of each is included.
- Built as a **tube** (outer prism minus a concentric inner prism inset by the wall thickness); the
  external corner radii of the cold-formed section are deferred (sharp corners). Outer size, bore
  and wall thickness are exact. CHS outer diameters follow the standard steel-tube OD series.
- Length is a representative stock length (6000 mm), user-overridable.

## Shaft Parts

### `key_din6885.json` — DIN 6885 (parallel keys)
- Cross-section b × h from DIN 6885-1 (e.g. 8×7, 12×8, 20×12) with a representative length l from the
  standard length series. The section (width × height) and length are exact.
- **Modelled square-ended** (DIN 6885 Form B): the Form A round ends are a tracked refinement. The
  key's cross-section is centred on the part origin.

### `gib_head_key_din6887.json` — DIN 6887 (gib-head taper keys)
- Width b, body height h and gib-nose height h2 from the DIN 6887 dimension table (e.g. 8×7 → h2=11,
  16×10 → h2=16, 40×22 → h2=36); length l is a representative value from the standard length series.
  The width, body height and nose height are grounded in the standard.
- **Modelled from the length silhouette**: a body rectangle (length × h) with a raised nose rising to
  h2 at the back face, extruded across the width. The 1:100 body taper is a tracked refinement (the
  body is modelled at constant height h); DIN 6887's reduced table carries no nose-length column, so
  the nose projection is a representative proportion (≈ 0.9·h2) and the nose radius is omitted.

### `dowel_iso2338.json` — ISO 2338 (cylindrical dowel pins)
- Diameter d and length l from the ISO 2338 preferred sizes (e.g. 6 m6 × 30). Diameter and length
  are exact. Revolved from a chamfered rod section with a 45° lead-in chamfer at each end (a
  representative proportion, ≈ 0.1·d, since the standard gives a range).

### `clevis_pin_iso2341.json` — ISO 2341 (clevis pins)
- Shank d, head diameter dk, head height k, cotter-hole diameter d1 and the hole end distance le
  from the ISO 2341 table (e.g. d10 → dk18, k4, d1 3.2, le 4.5); length l is a representative value
  from the standard series. Shank/head/hole diameters, head height and the hole end distance are
  grounded in the standard.
- **Modelled** as a flat cylindrical head + shank (as the hex bolt builds head + shank) with the
  transverse cotter hole cut through the shank at le from the far end. The head-edge chamfer/dome is
  a tracked refinement. The cotter/split-pin (ISO 1234) form is a folded round-wire part that needs
  a part-level sweep-along-path (not yet in the API), tracked as a follow-up.

### `circlip_din471.json`, `circlip_din472.json` — DIN 471 / DIN 472 (retaining rings)
- Keyed by nominal shaft (DIN 471, external) or bore (DIN 472, internal) diameter, with ring
  thickness s from the standard (e.g. shaft 20 → s = 1.2). Modelled as a flat split ring — the
  rectangular radial section (inner_dia/2 → outer_dia/2 × thickness) revolved through 330°, leaving
  the split gap.
- **Representational** (per the milestone plan): the thickness and nominal size are grounded in the
  standard; the ring's radial width/outer diameter are representative proportions and the lug ears
  with their assembly-plier holes are not modelled.

### ANSI inch shaft parts

Authored in inches (`"units": "in"`) over the same `pin` / `key` / `circlip` generators as the
metric families.

- `dowel_pin_ansi_b18_8.json` — **ASME B18.8.2** hardened ground machine dowel pins; nominal
  diameter (1/8 … 1/2) and a representative length. Revolved with a 45° lead-in chamfer at each end,
  as for the ISO 2338 dowel.
- `square_key_ansi_b17.json` — **ASME B17.1** square parallel keys; square cross-section b × b
  (1/8 … 1/2) and a representative length. Square-ended (as DIN 6885 Form B).
- `circlip_ansi_external.json` — **ASME B27.7** external (shaft) retaining rings; nominal shaft
  diameter and ring thickness s grounded (1/2 → s ≈ 0.035). As with the DIN 471/472 rings the ring
  is representational — the radial width / free outer diameter are representative proportions and the
  lug ears are not modelled.

## Bearings

### `ball_bearing_iso15.json` — ISO 15 (deep-groove ball bearings, 60/62/63 series)
- Bore d, outer diameter D and width B are the **boundary dimensions** tabulated in ISO 15 for the
  6000 (light, 60), 6200 (medium, 62) and 6300 (heavy, 63) series (e.g. 6205 → 25 × 52 × 15). These
  three per member are exact.
- The **ball count Z** is a representative value (typical for the size in general-purpose catalogues,
  e.g. SKF), not an ISO 15 dimension — ISO 15 tabulates only the boundary dimensions. It drives the
  circular pattern of the ball complement.
- **Representational** (per the milestone plan): the pitch-circle diameter, ball diameter, groove
  radius and the two raceway shoulder diameters are **derived parameters** computed from
  bore/outer_dia by fixed fractions of the radial gap (ball ≈ 0.28·(D−d)). The whole bearing re-drives
  with the size.
- **Ground race grooves modelled** (#53): each ring's raceway carries a concave groove arc that seats
  the ball. The groove radius is a **conformity** multiple of the ball diameter (r_g = 0.52·ball_dia,
  in the standard 0.515–0.53 design band), centred on the pitch circle so it sits just outside the
  ball surface — the ball nests in the groove with a uniform clearance and the rings/balls stay
  independent bodies (no boolean). The raceway shoulders flank the groove at pitch_dia ± 1.1·r_g
  (shoulder factor k = 0.55), leaving a positive shoulder land across the whole size range (asserted
  in a unit test). ISO 15 does not tabulate groove geometry (it is proprietary/design-level), so the
  conformity-derived groove is the honest representational choice. Cage and seals/shields remain a
  tracked refinement.

### `roller_bearing_iso15.json` — ISO 15 (cylindrical roller bearings, NU 2-series)
- Bore d, outer diameter D and width B are the boundary dimensions tabulated in ISO 15 for the
  NU 2-series cylindrical roller bearings (e.g. NU205 → 25 × 52 × 15). These three per member are
  exact. The **roller count Z** is a representative value (ISO 15 tabulates only boundary
  dimensions), driving the circular pattern of the roller complement.
- **Representational**: the pitch-circle diameter, roller diameter (≈ 0.28·(D−d)), roller length
  (≈ 0.8·B) and the two race diameters (each set to clear the roller crest by ≈ 0.012·(D−d), as for
  the ball bearing) are derived parameters, so the bearing re-drives with the size. The rollers are
  straight cylinders standing on the pitch circle with their axes parallel to the bearing axis;
  roller-end chamfers, the cage and the ring guide flanges are a tracked refinement.

### `tapered_roller_iso355.json` — ISO 355 (tapered-roller bearings, 302xx/303xx series)
- Bore d, outer diameter D, assembled width T and the **contact angle α** are the ISO 355 boundary
  dimensions for the 302xx/303xx metric tapered-roller series (e.g. 30206 → 30 × 62 × 17.25, α = 14°).
  These four per member are exact; the **roller count Z** is a representative value (ISO 355 tabulates
  boundary dimensions and the angle, not the roller complement), driving the circular pattern.
- **Representational**: a cone (inner ring) and a cup (outer ring) with truncated-cone raceways and a
  circular pattern of tapered-frustum rollers between them, each roller tilted by the contact angle
  (its centre moves out by roller_axial·tan α from the small to the big end). The roller diameters,
  axial span (≈ 0.65·T) and the four race diameters are derived. The raceways are made **collinear
  with the roller surfaces** — the raceway line through the roller-end crest points is extrapolated
  from the roller axial span out to the full ring width, then offset by a small clearance (cone inward,
  cup outward) — so the tilted rollers sit just clear of the rings instead of poking through them.
  Roller-end sphere ends, the cage, the ribs and the exact on-apex geometry are a tracked refinement.

### `angular_contact_iso15.json` — ISO 15 (angular-contact ball bearings, 72xx-B series)
- Bore d, outer diameter D and width B are the ISO 15 boundary dimensions for the 72xx angular-contact
  series (shared with the 62xx deep-groove boundary plan; e.g. 7206 → 30 × 62 × 16). The **contact
  angle α** (40° for the B design) and the **ball count Z** are representative values; the boundary
  dimensions are exact.
- **Representational**: a plain inner ring and an outer ring whose inner raceway is **relieved to a low
  shoulder on the front face** — opened outward, halfway to the outside diameter — so the front clears
  the ball crest and exposes the balls (the counterbore that carries one-directional thrust and lets
  the bearing be assembled). It is distinguished from the deep-groove bearing by that relieved face.
  The ball and race diameters are derived as for the deep-groove bearing (races clear the ball crest).
  Ground raceway grooves and the true tilted contact line are a tracked refinement.

## Plain Bearings

### `plain_bush_iso4379.json` — ISO 4379 (cylindrical sleeve bushes)
- Bore d, outside diameter D and length L follow the ISO 4379 dimension series for cylindrical
  plain-bearing bushes (e.g. 20 × 26 × 20). Bore, outside diameter and length are exact; a
  representative subset of the series is included.
- Built as a plain concentric sleeve (the tube path: outside diameter as a new solid, bore cut
  through the full length). The flanged-bush variant and the wrapped-bush split seam are a tracked
  refinement — this is the solid cylindrical bush.

### `thrust_bearing_iso104.json` — ISO 104 (single-direction thrust ball bearings, 511xx)
- Bore d, outer diameter D and total height H are the ISO 104 boundary dimensions for the 511xx
  single-direction thrust ball bearing series (e.g. 51105 → 25 × 42 × 11). These three per member are
  exact; the **ball count Z** is a representative value (ISO 104 tabulates only boundary dimensions).
- **Representational**: a shaft washer and a housing washer with a ground groove on each ball-facing
  face (an outer land, a groove arc cradling the ball, an inner land) and a ball complement between
  them on the pitch circle, the stack centred on the mid-plane. The ball diameter (≈ 0.45·H), groove
  radius (≈ 0.53·ball) and land offset (≈ 0.7·groove) are derived so each ball nests in the groove
  with a small clearance rather than sitting tangent to a flat face.

### `thrust_self_aligning_iso104.json` — ISO 104 (self-aligning thrust ball bearings, 532xx)
- Bore d, outer diameter D and total height H are the ISO 104 boundary dimensions for the 532xx
  self-aligning thrust ball bearing series (532xx shares the 511xx boundary plan; e.g. 53206 →
  30 × 47 × 11). These three per member are exact; the **ball count Z** is a representative value
  (ISO 104 tabulates only boundary dimensions).
- **Representational**: the same grooved shaft washer and ball complement as the 511xx, but the
  housing washer's outboard **back is a shallow concave seat** and a separate **seat washer** (concave
  underside, flat top, larger OD) nests over it with a hair clearance, so the bearing can tilt to take
  up shaft misalignment. The boundary dimensions are exact ISO 104; **the sphered seat is a
  design-level derivation** — the seat sphere radius follows the classical relation R = (a² + s²)/2s
  from a cap-depth fraction s = 0.06·D over the OD radius a = D/2 (geometry-math-advisor #54), flat
  enough (large R) to fit the washer thickness rather than the catalog SR, its axis centre placed so
  the OD-back rim keeps the nominal +H/2 back level. Because that flatness puts the seat arc's sagitta
  below the tessellation floor (< 0.1 mm over the washer width), where a sketch arc is numerically
  degenerate, each concave back is built as the sphere's straight **chord** — a shallow cone < 0.1 mm
  off the true sphere, pinned by its rim stations (the advisor's sub-floor-sagitta fallback). The seat
  washer's underside is the chord of a slightly smaller sphere about the same centre (clearance
  ≈ 0.02·H), so the two seat faces do not z-fight.
