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

| Area | State |
| --- | --- |
| A1 — scaffold + host load (ribbon button + dockable panel) | done — loads live; `PartDesigner.Show` command + dockable panel confirmed on the running head |
| A2 — catalogue data model & standards loader | planned |
| A3 — generator framework + PartBuilder | planned |
| A4 — placement + Change-Size | planned |
| A5 — panel UI + headless commands | planned |
| B/C/D/E — Fasteners / Structural / Shaft / Bearings generators | planned |

Milestone and issues: see the repository's GitHub tracker
(_M1 — Content Center: Procedural Standard Parts (v0.1)_).

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
