# Oblikovati Part Designer

Oblikovati's answer to Inventor's **Content Center**: a browsable library of standardized
machine parts — **Fasteners, Structural Shapes, Shaft Parts, Bearings** across ISO / DIN /
ANSI. Unlike a catalogue of template files, every part is **generated procedurally** from
curated per-standard dimension tables, so a placed part is a real, fully-constrained
(DOF-0) **parametric** Oblikovati part that exposes and is constrained by its parameters.

It ships as a c-shared library (`.so`/`.dll`/`.dylib`) that the host loads at runtime and
drives over the Apache-2.0 public API (`oblikovati.org/api`) across the C ABI (ADR-0016).

## Architecture

- **cgo C-ABI shell** (`export.go`, `hostcaller.go`, `manifest.go`) — the runtime boundary.
  It constructs the engine at Activate and forwards host RPC + events.
- **`designer/` engine** (cgo-free, unit-testable on every OS with a fake host):
  - `catalog/` — the standards **data library** (families = per-standard dimension tables,
    members = one size) + the category tree and queries. _(A2)_
  - `build/` — the **generator framework**: a `PartGenerator` builds a DOF-0 parametric part
    from a resolved member via a thin `PartBuilder` over `api/client`. _(A3)_
  - `placement.go` / `persist.go` — create a Part document, generate, stamp the family/member,
    and optionally place an occurrence into the active assembly; re-drive on Change-Size. _(A4)_
  - `panel.go` — the dockable window: cascading Category → Family → Standard → Size
    dropdowns + Place, following Inventor's "Place from Content Center" flow. _(A5)_

The **shipped** library links only `oblikovati.org/api` (Apache-2.0). A require on the GPL
host module is test-scope only (designer↔host integration tests); the `gplpurity` guard
enforces this.

## Status

Milestone **M1 — Content Center: Procedural Standard Parts (v0.1)** is complete. Framework
(A1–A5), every category (B–E), and the polish/validation pass (F1–F3) are done and
live-validated over the MCP bridge.

### Framework

| Area | State |
| --- | --- |
| A1 — scaffold + host load (ribbon button + dockable panel) | done — loads live; `PartDesigner.Show` + dockable panel confirmed |
| A2 — catalogue data model & standards loader | done |
| A3 — generator framework + PartBuilder | done |
| A4 — placement + Change-Size | done |
| A5 — panel UI + headless commands | done — cascading dropdowns + Place |
| F1 — Standard/Category filters + quick search | done — filter by standard/category + text search over family name & size |
| F2 — Change-Size UX (panel bound to active part) | done — activating a placed part binds it; a Size change re-drives it in place |
| F3 — live cross-category validation + docs | done — see [`livetest/`](livetest/) and [`SOURCES.md`](designer/catalog/data/SOURCES.md) |

### Parts (procedural generators)

| Category | Parts (standards) | Generators |
| --- | --- | --- |
| Fasteners (metric) | Hex bolt (ISO 4017/4014, DIN 933/931) · Socket head & countersunk screw (ISO 4762/10642, DIN 912) · Hex nut (ISO 4032/4035, DIN 934) · Washer plain & spring (ISO 7089, DIN 125/127) · Stud & threaded rod (DIN 939/976) | `hex_bolt` `socket_screw` `hex_nut` `washer` `stud` |
| Fasteners (ANSI inch) | Hex bolt (ASME B18.2.1, UNC) · Hex nut (ASME B18.2.2) · Socket head cap screw (ASME B18.3) · Plain washer (ASME B18.22.1) | `hex_bolt` `hex_nut` `socket_screw` `washer` |
| Structural | Round & flat bar (ISO 1035, EN 10058) · I-beams (EN IPE/HE A/HE B, AISC W) · Channels (EN UPN, AISC C) · Angles L (EN 10056) · Tees (EN 10055) · Hollow SHS/RHS/CHS (EN 10219) | `round_bar` `flat_bar` `i_beam` `channel` `angle` `tee` `hollow_rect` `hollow_round` |
| Shaft Parts | Parallel key (DIN 6885) · Gib-head key (DIN 6887) · Dowel pin (ISO 2338) · Clevis pin (ISO 2341) · Retaining ring (DIN 471/472) | `key` `gib_head_key` `pin` `clevis_pin` `circlip` |
| Bearings | Deep-groove ball bearing (ISO 15) · Cylindrical roller bearing (ISO 15) · Plain sleeve bush (ISO 4379) | `ball_bearing` `roller_bearing` `plain_bush` |

Every table's numbers are grounded in the cited standard — provenance in
[`designer/catalog/data/SOURCES.md`](designer/catalog/data/SOURCES.md). Tracked refinements
(root fillets, tapered/thrust bearings, gib-head keys, dowel chamfers, …) are noted there.

Milestone and issues: see the repository's GitHub tracker.

## Build & test

```sh
make sync-header   # vendor the C ABI header from the oblikovati.org/api module
make build         # build the c-shared add-in into build/
make install       # build + copy the library + manifest.json into ../Oblikovati/head/addins
make test          # cgo-free designer unit tests + the gpl-purity guard

# lint (matches CI)
golangci-lint run
python3 scripts/add-spdx-headers.py --check
```

Cross-repo dependencies resolve via a git-ignored `go.work` locally (sibling
`../Oblikovati.API` and `../Oblikovati` checkouts); CI injects the equivalent replaces via
`.github/actions/siblings`.

## Run in the live host

`make install`, launch the head with `OBK_ADDINS_DIR` pointing at the add-ins directory,
and the **Part Designer** ribbon button + right-docked window appear. A Go c-shared add-in
cannot be hot-swapped in-process — rebuild → reinstall → restart the head to pick up a new
build.

## License

GPL-2.0-only. See [LICENSE](LICENSE).
