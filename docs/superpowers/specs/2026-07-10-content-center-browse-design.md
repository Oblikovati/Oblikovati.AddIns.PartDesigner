# Content-Center tree + table browse (in-panel) — Design

- **Issue:** Oblikovati.AddIns.PartDesigner #48 (M2)
- **Date:** 2026-07-10
- **Status:** approved (brainstorming) — pending implementation plan
- **Repos touched:** `Oblikovati.API` (Apache-2.0), `Oblikovati` GPL head, `Oblikovati.AddIns.PartDesigner`

## Problem

The PartDesigner panel today is cascading Category → Standard → Search → Part →
Size dropdowns (`designer/panel.go`). The M1 plan explicitly deferred the real
**tree + parameter-table browse** (Inventor "Place from Content Center" flow) to
a follow-up API extension. The public panel vocabulary has **no tree and no
data-table** control kind (`PanelGrid` is a *layout* container of child controls,
not a selectable-row data grid), so the browse surface genuinely requires new API
primitives — this is the contract-first (ADR-0018) two-part shape the issue calls
for.

## Product decisions (confirmed)

1. **Surface:** in the existing right-dock panel. The tree (top) + member table
   (bottom) + Place button **replace** the Part/Size dropdowns. The **Search box
   and Standard/Category filters stay above the tree** and filter which families
   appear in it.
2. **API widgets:** two **generic, reusable** panel-control kinds —
   `PanelTree` + `PanelTable` — that the add-in composes. Other add-ins (CAM,
   FEA, exporters) inherit a tree and a data grid.
3. **Table content:** show **all** of a family's parameter columns (the full
   family-table view), with **horizontal scroll** for the narrow (~260px) dock.

## Key finding that sizes the work

The Vulkan head renders panels through a **Dear ImGui-style immediate-mode
wrapper** (`head/internal/native/imgui.go`), which already exposes:

- Tree: `TreeNode`, `TreeNodeSelectable`, `TreePop`, `SetNextItemOpen`,
  `CollapsingHeader`, `Indent/Unindent`, `IsItemClicked`.
- Table: `BeginTable`, `TableSetupColumn`, `TableSetupScrollFreeze`,
  `TableHeadersRow`, `TableNextRow`, `TableNextColumn`, `EndTable`.
- Scroll/selection: `BeginChild`, `Selectable`, `PushIDInt/PopID`.

So the head side is **wiring two new control kinds into the existing dispatch
`switch` and calling these primitives** — imgui owns expand/collapse, hit-testing
(twirl triangle vs label), selection highlight, and both vertical **and
horizontal** scroll natively. No hand-rolled Vulkan widgets.

## Architecture

Three concerns, three repos, dependency arrows pointing away from the domain:

```
Oblikovati.AddIns.PartDesigner  (catalog browse feature)
        │ composes
        ▼
Oblikovati.API                  (generic PanelTree / PanelTable vocabulary)
        ▲ renders specs of
        │
Oblikovati (GPL head)           (imgui rendering of the new kinds)
```

- The **API** knows nothing of PartDesigner (generic mechanism).
- The **head** renders API control specs (no catalog knowledge).
- The **add-in** composes API controls and owns catalog→widget mapping and
  `Place()`.

### Interaction / data-flow model (declarative re-send)

Panels are declarative: the add-in sends the full `[]PanelControlSpec`, the head
renders it, and on an edit the head pushes a `PanelValueChangedEvent{WindowID,
ControlID, Value}`; the add-in mutates its own state and **re-sends the whole
spec**. Applied here:

- **Expand/collapse is host-side, NO round-trip.** ImGui owns twirl state keyed
  by stable node IDs; `TreeNode.Expanded` is only a first-render seed
  (`SetNextItemOpen(open, firstUse)`). Twirling a node does not notify the add-in
  and does not churn the spec — expansion stays latency-free.
- **Selection round-trips** via the scalar `PanelValueChangedEvent`:
  - tree node clicked → `Value = nodeID` → add-in sets the selected family,
    populates the table from `family.Columns` / `family.Members`, re-sends.
  - table row clicked → `Value = rowKey` → add-in sets the selected member (arms
    Place), re-sends (highlight).
- **Place** uses the existing `Place(familyID, memberKey)` path unchanged.

## API contract (Oblikovati.API — PR 1)

New `PanelControlKind`s appended with stable ordinals in
`types/panel_control_kind.go`:

```go
PanelTree  PanelControlKind = 13 // hierarchical, selectable, expandable nodes
PanelTable PanelControlKind = 14 // columns + selectable rows (data grid)
```

New DTOs in a new file `wire/panel_browse.go` (SRP — keep `task_panel.go`
focused):

```go
type TreeNode struct {
    ID       string     `json:"id"`
    Label    string     `json:"label"`
    Children []TreeNode `json:"children,omitempty"`
    Expanded bool       `json:"expanded,omitempty"` // first-render hint only
}

type TableRow struct {
    Key   string   `json:"key"`
    Cells []string `json:"cells"`
}
```

Fields added to `PanelControlSpec`:

```go
Nodes        []TreeNode `json:"nodes,omitempty"`        // PanelTree
TableColumns []string   `json:"tableColumns,omitempty"` // PanelTable header (Columns is already []GridTrack for PanelGrid)
TableRows    []TableRow `json:"tableRows,omitempty"`    // PanelTable body (Rows is already PanelReferenceList's)
// existing Value string is reused as the selected nodeID / rowKey (drives highlight)
```

Note: `PanelControlSpec.Columns` already exists as `[]types.GridTrack` (grid tracks)
and `Rows` as `[]PanelReferenceRow`, so the table's header/body fields are named
`TableColumns` / `TableRows` to avoid collision.

Design choices and why:

- **Nested `TreeNode` (not flat + ParentID):** the catalog is small, fully
  embedded, eager (no lazy loading); nesting renders by natural recursion and
  matches the re-send model. Reconstructing a hierarchy from a flat list every
  send would be pure overhead.
- **Reuse `PanelValueChangedEvent`, no bespoke event:** each control's value is a
  single string (selected id / key). `PanelReferenceList` needed its own event
  only because its value is a *set*; that reason does not apply here.
- **`TableRow` has a stable `Key`** distinct from display cells so the selected
  row survives re-sends and filtering regardless of visible text.
- **v1 keeps it minimal** (all columns left-aligned; no per-column alignment /
  units row). Alignment can be added later without breaking the DTO (YAGNI).

Client builders in `client/panel_controls.go` (existing home of `PanelDropdown`
etc.):

```go
func PanelTree(id string, nodes []wire.TreeNode, selected string) wire.PanelControlSpec
func PanelTable(id string, columns []string, rows []wire.TableRow, selected string) wire.PanelControlSpec
```

## Head rendering (Oblikovati GPL — PR 2)

Two new cases in the `addin_panels.go` dispatch `switch`, delegating to new files
so every file stays < 500 lines:

- `head/ui/addin_tree.go` — `drawPanelTree`: recurse `control.Nodes`; per node
  `SetNextItemOpen(node.Expanded, firstUseOnly)`, then `TreeNodeSelectable(label,
  node.ID == control.Value)` for branches (`TreePop` on close) and `Selectable`
  for leaves; on `IsItemClicked` emit `PanelValueChangedEvent{ControlID,
  Value: node.ID}` via the session. ImGui provides indent, twirl hit-testing,
  selection highlight, and vertical scroll.
- `head/ui/addin_table.go` — `drawPanelTable`: a **new** `native.BeginTableScrollX`
  wrapper (backed by a new `obk_ig_begin_table_scrollx` C shim that ORs
  `ImGuiTableFlags_ScrollX` onto the existing `Borders|RowBg|Resizable|ScrollY`) —
  a *new variant* rather than changing the shared `obk_ig_begin_table`, because six
  existing tables (keymap, parameters, history, bom, derived, file-dialog) use it and
  `ScrollX` changes column auto-sizing. Then `TableSetupColumn` per column,
  `TableSetupScrollFreeze(0,1)` to pin the header row, `TableHeadersRow`, then per
  row `PushIDInt(i)` + a row-spanning `Selectable` keyed on `row.Key`; on
  `IsItemClicked` emit `PanelValueChanged(windowID, control.ID, row.Key)`.
  **Horizontal scroll is imgui-native** — the correct answer for the narrow dock.

Hot-path discipline: the spec's strings are drawn as-is; **no per-row string
building per frame**, no `make()` on the render path (holds the ≥60fps / zero-
alloc bar established by the prior perf effort).

## Add-in composition (PartDesigner — PR 3)

- `designer/panel.go` `browserControls`: keep the header, Category, Standard, and
  Search controls; **replace** the Part + Size dropdowns with
  `client.PanelTree("catalog", …)` + `client.PanelTable("members", …)` + the
  Place button.
- `designer/catalog/tree.go`: add a builder that emits `[]wire.TreeNode` from the
  filtered families, with **family leaves** (`ID = familyID`) under their
  category path; respects the existing Category/Standard/Search filters.
- `designer/selection.go`: route `PanelValueChangedEvent`:
  - `ControlID == "catalog"` → set `sel.familyID = Value`, clear member, re-send.
  - `ControlID == "members"` → set `sel.memberKey = Value`, re-send.
- Table model from the selected family: `Columns = family.Columns[*].Name`,
  one `TableRow{Key: memberKey, Cells: member values in column order}` per member.
- `Place` and Change-Size binding (`binding.go`, `persist.go`) unchanged.

## Slice / PR sequence (contract-first, ADR-0018)

1. **PR 1 — Oblikovati.API:** kinds + DTOs + client builders + tests. Merge →
   auto-release (minor bump) → **`git pull` the API sibling** (avoids the known
   stale-sibling `make run` break).
2. **PR 2 — Oblikovati GPL head:** render both kinds; needs the released API
   version.
3. **PR 3 — PartDesigner add-in:** compose the browse surface; unit tests
   (fakeHost) need only the released API, so PR3 can be developed in parallel with
   PR2, but its **live screenshot** validation needs PR2 merged into the head.

Merge order: **PR1 → PR2 → PR3.**

## Testing

- **API:** `TreeNode`/`TableRow`/`PanelControlSpec` JSON round-trip; client-builder
  unit tests; kind ordinal + `String()` tests.
- **Head:** the draw funcs call live imgui and cannot run headless (existing head
  tests only cover pure helpers/logic, never native draw calls), so automated
  coverage is limited to any pure helper (e.g. `firstUseSeed`, table-model
  shaping); rendering correctness is validated **live via MCP screenshot** below.
- **Add-in (fakeHost):** tree carries family leaves under the correct category
  path and honors filters; selecting a family populates the members table with
  all columns; selecting a row then Place calls `Place(familyID, memberKey)`.
  Coverage > 80%, duplication < 3%.
- **Live (MCP bridge, per the Live-tests rule):** expand Bearings → Deep-groove,
  confirm the table shows d/D/B/... rows, select 6202, Place, confirm the part
  appears in the viewport; screenshot-verify.

## Out of scope (v1)

- Per-column alignment / a dedicated units row (DTO leaves room to add later).
- A separate modal "Place from Content Center" dialog (decided: in-panel).
- Lazy / paged tree loading (catalog is small and fully embedded).
- Multi-select in the table.
```

Rejected surface alternatives: a separate modal dialog (Inventor-faithful but
heavier, no always-visible browse) and a tabbed panel (extra control-flow for
little gain); rejected widget shape: one combined `PanelBrowser` control (less
reusable) and faking the table from `PanelGrid` (no real row-selection/scroll).
