# Catalog size-coverage expansion — design

## Goal

Every standard-parts family currently ships only a handful of sizes (most metric fasteners
carry just M6–M12, 4 rows). Expand **all 48 families** so each spans its standard's full
**official preferred-size series up to ~50 mm nominal**, with every value grounded in an
official standard / authoritative table (never interpolated), and provenance recorded in
`SOURCES.md`.

Decisions confirmed with the user:
- **Size rule:** each family's real official preferred series ≤ ~50 mm (not a forced uniform
  1–50 mm axis — ranges legitimately differ by standard).
- **Scope:** all categories in one sweep (fasteners, shaft parts, bearings, structural).

## The hard constraints (non-negotiable)

1. **Officially sourced, every value.** `SOURCES.md` already requires each dimension be grounded
   in an official standard / standards-body / manufacturer table. New rows extend the existing
   sources (fasteners.eu for metric fasteners/washers; ISO/SKF series for bearings; EN 10365 /
   EN 10056 / EN 10219 / AISC for structural; ISO/DIN/ASME tables for shaft parts). Each family's
   `SOURCES.md` entry gains the source URL and the added sizes, in the same change.
2. **Every column filled, every row.** `catalog/load.go` rejects a member missing any column
   ("missing cell for column %q"). A new size is a *complete* record across the family's columns
   (e.g. DIN 933 needs `d,P,s,k,l,b`; ISO 4762 needs `d,P,dk,k,s,t,l,b`).
3. **Representative lengths keep their existing convention.** For bolts/screws/studs/rod, `l`
   (length) and `b` (thread length) are representative — the standard fixes a *range* per
   diameter, not one length. Continue the current convention: one representative standard length
   per diameter (~5·d rounded to a preferred length), fully-threaded `b=l` where the family is
   fully threaded (DIN 933/ISO 4017), else the standard's thread-length formula. Fixed
   per-diameter dimensions (`s,k,P,dk,t,…`) are taken verbatim from the standard.
4. **No degenerate solids.** The kernel's unit-strict engine turns a wrong/blank cell into a
   collapsed body. A mis-sourced row is invisible to fakeHost tests, so new sizes are gated by a
   **live per-body volume check** (the #53/#61 discipline), not unit tests alone.

## Preferred series per category (targets, ≤50 mm)

- **Metric fastener thread ⌀:** M1.6, M2, M2.5, M3, M4, M5, M6, M8, M10, M12, M16, M20, M24, M30,
  M36, M42, M48 (each family takes the subset its standard actually tabulates; e.g. socket screws
  start at M1.6/M2, some nuts at M2).
- **Inch fasteners (ASME):** the standard's UNC series up to ~2 in ⌀ where the family is inch-based.
- **Washers:** the size series matching their bolt standard.
- **Circlips DIN 471 / 472:** the shaft/bore ⌀ series (471: ~3–50 mm shaft; 472: ~8–50 mm bore).
  ASME B27.7 retaining ring: its inch shaft series.
- **Pins/keys:** ISO 2338/2341/1234, DIN 6885/6887, ASME B17.1/B18.8.2 — each standard's ⌀ (or
  key width) series across the range.
- **Bearings:** ISO 15 / ISO 104 / ISO 355 designation series (e.g. deep-groove 60/62/63/64,
  bore 3–50 mm) with `d,D,B/H/T,Z` per the standard/SKF table; keep `Z` (rolling-element count)
  from the manufacturer table.
- **Structural:** EN 10365 IPE/HEA/HEB (up to ~200 mm height so the profile stays ≤~50 mm-class
  where the user's ⌀ intuition applies — confirm ceiling per series), EN 10056 angles, EN 10219
  SHS/RHS/CHS, AISC W/C — each series' standard section list across the small-to-mid range.

(The exact per-family list is fixed by the standard's own table at execution time — the finder
subagent reads the official source and transcribes the tabulated sizes, it does not invent them.)

## Architecture / execution

Data-only change under `designer/catalog/data/**` plus `SOURCES.md`; no schema, generator, or API
change. Executed via subagent-driven development, **one family per work item**:

1. **Source** — the finder subagent WebFetches the family's official table, extracts the preferred
   sizes ≤50 mm and every column's value per size (verbatim, nominal per the SOURCES.md rule).
2. **Fill** — append the complete member rows to the family JSON in size order; add the source URL
   + added-size note to `SOURCES.md`.
3. **Verify (data)** — a review subagent independently re-fetches the source and checks each added
   value against it (adversarial: default to "wrong" on any mismatch), plus completeness (every
   column present) and monotonic-ish sanity (across-flats grows with ⌀, etc.).
4. **Verify (geometry, live)** — after each category, place a *sampling* of new sizes (smallest,
   a mid, largest per family) in a live head over the MCP bridge and assert each body's volume is
   non-degenerate and within tolerance of the analytic estimate — catches unit-strict collapses.

## Tests

- **Existing:** `load` already validates completeness + key columns; keep green.
- **New coverage test** (`catalog`): assert each family now carries ≥ N sizes and that its size
  column spans the expected min→max for its standard (a regression guard against silent shrink).
- **Live:** a `livetest/verify_size_coverage.py` that places the sampled new sizes per category
  and gates on 3 bodies-worth of non-degenerate volume + a screenshot per category.

## PR structure

Executed in one sweep but shipped as **four category PRs** (fasteners, shaft, bearings,
structural) so each stays reviewable and independently verifiable; each cites its `SOURCES.md`
additions and the live-validation result. Granular commits per family within each PR.

## Risks

- **Sourcing accuracy** is the dominant risk. Mitigation: adversarial re-fetch verification +
  live geometry gate; any size whose source can't be confirmed is dropped, not guessed, and the
  gap is logged in `SOURCES.md`.
- **Structural "≤50 mm" is ambiguous** (a beam's height isn't a bolt ⌀). Mitigation: fill each
  structural series' standard small-to-mid sections and note the ceiling chosen per series.
- **Length convention** for bolts/screws is representative, not standard-fixed — documented above
  and in `SOURCES.md` so it's auditable.
