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

## To verify (grounded from memory in earlier PRs — re-audit against official sources)

The fastener families below were populated before this provenance rule and should be
re-verified against official standard tables:

- `hex_bolt_*.json` (ISO 4017 / DIN 933)
- `socket_screw_*.json` (ISO 4762 / DIN 912 / ISO 10642)
- `hex_nut_*.json` (ISO 4032 / DIN 934 / ISO 4035)
- `round_bar_*.json` (ISO 1035 stock)
