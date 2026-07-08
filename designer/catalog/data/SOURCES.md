# Standards data provenance

Every dimension table under `data/` must be grounded in an **official standard or an
authoritative standards-body / manufacturer table**, not estimated. This file records the
source of each family's numbers so they can be audited and re-verified. When adding or editing
a family, cite the source here (URL + what was taken from it) in the same change.

Values are the standard **nominal** dimensions (for toleranced fields, the nominal or the
mid-range of the published min/max), in millimetres.

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
- Verified/**corrected**: s (max) = 10/13/**17/19** (thin nuts keep the wider across-flats for
  M10/M12, unlike ISO 4032's 16/18; was 16/18), m (max) = 3.2/4.0/5.0/6.0 (matched).

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

## Structural

### `round_bar_iso1035.json` — ISO 1035 (hot-rolled round steel bar)
- Nominal preferred stock diameters (10/12/16/20/25 mm) per ISO 1035-1; length is a representative
  stock cut (1000 mm), not a per-size standard dimension.
