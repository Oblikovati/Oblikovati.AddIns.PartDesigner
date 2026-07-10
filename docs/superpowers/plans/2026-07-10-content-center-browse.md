# Content-Center tree + table browse Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the PartDesigner panel's Part/Size dropdowns with an in-panel category **tree** + member **parameter table** + Place button, via two new generic public panel controls (`PanelTree`, `PanelTable`).

**Architecture:** Contract-first (ADR-0018) across three repos. Phase 1 adds the two control kinds + DTOs + client builders to `Oblikovati.API` (Apache-2.0) and releases them. Phase 2 renders them in the `Oblikovati` GPL Vulkan head via its existing Dear-ImGui wrapper (tree = `TreeNodeSelectable`, table = a new `BeginTableScrollX` variant). Phase 3 composes them in the `Oblikovati.AddIns.PartDesigner` add-in: a tree of category→family nodes, a table of the selected family's members, driving the unchanged `Place()`.

**Tech Stack:** Go 1.x; `oblikovati.org/api` module; cgo + Dear ImGui in the head; `//go:embed` catalog JSON; fakeHost test double in the add-in.

## Global Constraints

- **SPDX header on every new `.go` file:** `Apache-2.0` in `Oblikovati.API`; `GPL-2.0-only` in `Oblikovati` and `Oblikovati.AddIns.PartDesigner`. Run `scripts/add-spdx-headers.py` where available.
- **Never re-declare a wire DTO or method-name string** in the GPL module or an add-in — import from `api/wire`.
- **Functions 4–20 lines; files < 500 lines; explicit types (no `any` in new logic, no untyped funcs); early returns; max 2 indent levels.**
- **Every new function gets a test; bug fixes get a regression test.** Coverage > 80%, duplication < 3% before any PR.
- **API versioning is automatic:** `release.yml` derives the version from the commit scope on merge to `develop`. NEVER hand-edit `version.go`/`CHANGELOG.md`. Use a `feat:` scope so it bumps the MINOR version.
- **Ship order (REVISED per user):** do NOT submit or merge the API PR until the whole host implementation is built and **live-verified end-to-end**. Develop the API changes on a local branch *inside the `../Oblikovati.API` sibling checkout*; the `go.work` replace resolves `oblikovati.org/api` to that directory, so the head and add-in compile + run against the un-merged branch locally (no release needed to build). Only after the live MCP-bridge screenshot verification passes: submit + merge + release the API PR, fast-forward the sibling to the release commit, then submit the head PR and the add-in PR. This keeps a broken/unproven API surface from ever being released.
- **Full local suite + golangci-lint + markdownlint + SPDX check + cross-platform build** before each PR; `Closes #48` only in the final (add-in) PR body; the API/head PRs reference it with `Part of #48`.
- **Do not squash commits** (they carry granular context); merge PRs with `--merge --delete-branch`.
- **Repo paths (absolute):** API = `/home/vmiguel/git/oblikovati-workspace/Oblikovati.API`; head = `/home/vmiguel/git/oblikovati-workspace/Oblikovati/head`; add-in = `/home/vmiguel/git/oblikovati-workspace/Oblikovati.AddIns.PartDesigner`.

---

## Phase 1 — Oblikovati.API: PanelTree + PanelTable vocabulary (PR 1)

Branch: `feat/panel-tree-table` off `origin/develop` in `Oblikovati.API`.
Files touched: `types/panel_control_kind.go`, `types/ui_enums_test.go`, new `wire/panel_browse.go` + test, `client/panel_controls.go` + test.

### Task 1.1: Add the two `PanelControlKind` ordinals

**Files:**
- Modify: `Oblikovati.API/types/panel_control_kind.go` (const block after `PanelReferenceList = 12`; the `panelControlKindNames` map)
- Test: `Oblikovati.API/types/ui_enums_test.go`

**Interfaces:**
- Produces: `types.PanelTree PanelControlKind = 13`, `types.PanelTable PanelControlKind = 14`; `PanelTree.String() == "tree"`, `PanelTable.String() == "table"`.

- [ ] **Step 1: Write the failing test.** Append to `types/ui_enums_test.go`:

```go
func TestPanelTreeTableKinds(t *testing.T) {
	if PanelTree != 13 || PanelTable != 14 {
		t.Fatalf("ordinals: PanelTree=%d PanelTable=%d, want 13,14 (appended, stable)", PanelTree, PanelTable)
	}
	if PanelTree.String() != "tree" || PanelTable.String() != "table" {
		t.Fatalf("names: %q, %q, want \"tree\", \"table\"", PanelTree.String(), PanelTable.String())
	}
}
```

- [ ] **Step 2: Run it, expect FAIL** (undefined `PanelTree`/`PanelTable`).

Run: `cd Oblikovati.API && go test ./types/ -run TestPanelTreeTableKinds`
Expected: FAIL, `undefined: PanelTree`.

- [ ] **Step 3: Implement.** In `types/panel_control_kind.go`, after the `PanelReferenceList` const add:

```go
	// PanelTree is a hierarchical, selectable, expandable set of nodes (a category browser).
	// The disclosure arrow toggles a node open (handled host-side, no round-trip); a click on a
	// node's label selects it and pushes a [PanelValueChangedEvent] with Value = the node's ID.
	PanelTree PanelControlKind = 13
	// PanelTable is a data grid: a header of column names over selectable rows. Clicking a row
	// pushes a [PanelValueChangedEvent] with Value = the row's Key. Scrolls in both axes.
	PanelTable PanelControlKind = 14
```

And in `panelControlKindNames` add: `PanelTree: "tree", PanelTable: "table",`.

- [ ] **Step 4: Run it, expect PASS.**

Run: `cd Oblikovati.API && go test ./types/ -run TestPanelTreeTableKinds`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
cd Oblikovati.API && git add types/panel_control_kind.go types/ui_enums_test.go
git commit -m "feat(types): add PanelTree/PanelTable control kinds (Part of #48)"
```

### Task 1.2: Add `TreeNode` / `TableRow` DTOs + `PanelControlSpec` fields

**Files:**
- Create: `Oblikovati.API/wire/panel_browse.go`
- Create: `Oblikovati.API/wire/panel_browse_test.go`
- Modify: `Oblikovati.API/wire/docking.go` (`PanelControlSpec` struct, after the `Rows`/`Accepts` fields)

**Interfaces:**
- Produces: `wire.TreeNode{ID, Label string; Children []TreeNode; Expanded bool}`; `wire.TableRow{Key string; Cells []string}`; new `PanelControlSpec` fields `Nodes []TreeNode`, `TableColumns []string`, `TableRows []TableRow`.
- Consumes: existing `wire.PanelControlSpec`.

- [ ] **Step 1: Write the failing test.** Create `wire/panel_browse_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"encoding/json"
	"testing"
)

func TestTreeNodeRoundTrip(t *testing.T) {
	in := PanelControlSpec{
		Nodes: []TreeNode{{ID: "bearings", Label: "Bearings", Expanded: true, Children: []TreeNode{
			{ID: "iso15-6200", Label: "6200 series"},
		}}},
	}
	var out PanelControlSpec
	mustReJSON(t, in, &out)
	if out.Nodes[0].ID != "bearings" || !out.Nodes[0].Expanded ||
		out.Nodes[0].Children[0].ID != "iso15-6200" {
		t.Fatalf("tree round-trip lost data: %+v", out.Nodes)
	}
}

func TestTableRoundTrip(t *testing.T) {
	in := PanelControlSpec{
		TableColumns: []string{"d", "D", "B"},
		TableRows:    []TableRow{{Key: "d=10,D=30", Cells: []string{"10", "30", "9"}}},
		Value:        "d=10,D=30",
	}
	var out PanelControlSpec
	mustReJSON(t, in, &out)
	if len(out.TableColumns) != 3 || out.TableRows[0].Key != "d=10,D=30" ||
		out.TableRows[0].Cells[1] != "30" || out.Value != "d=10,D=30" {
		t.Fatalf("table round-trip lost data: %+v", out)
	}
}

func mustReJSON(t *testing.T, in, out any) {
	t.Helper()
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}
```

- [ ] **Step 2: Run it, expect FAIL** (undefined `TreeNode`/`TableRow`, unknown fields).

Run: `cd Oblikovati.API && go test ./wire/ -run 'TestTreeNodeRoundTrip|TestTableRoundTrip'`
Expected: FAIL / build error `undefined: TreeNode`.

- [ ] **Step 3: Implement the DTOs.** Create `wire/panel_browse.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package wire

// TreeNode is one node of a PanelTree (a category-browser hierarchy). ID is the stable key the
// host echoes in a PanelValueChangedEvent when the node's label is clicked; Label is the display
// text; Children nests sub-nodes (empty = a leaf). Expanded is a FIRST-RENDER hint only —
// afterwards the host owns expand/collapse (twirling a node does not notify the add-in), so a
// re-sent spec does not fight the user's open/closed state.
type TreeNode struct {
	ID       string     `json:"id"`
	Label    string     `json:"label"`
	Children []TreeNode `json:"children,omitempty"`
	Expanded bool       `json:"expanded,omitempty"`
}

// TableRow is one row of a PanelTable. Key is the stable identifier the host echoes in a
// PanelValueChangedEvent when the row is clicked (kept distinct from the display Cells so the
// selection survives re-sends and filtering); Cells are the per-column display strings, in the
// column order declared by the control's TableColumns.
type TableRow struct {
	Key   string   `json:"key"`
	Cells []string `json:"cells"`
}
```

- [ ] **Step 4: Add the `PanelControlSpec` fields.** In `wire/docking.go`, immediately after the `Accepts []string` field inside `PanelControlSpec`:

```go
	// PanelTree fields.
	Nodes []TreeNode `json:"nodes,omitempty"` // tree: the root nodes

	// PanelTable fields (Columns is already []GridTrack and Rows []PanelReferenceRow, so the
	// data-grid header/body use distinct names).
	TableColumns []string   `json:"tableColumns,omitempty"` // table: header column names
	TableRows    []TableRow `json:"tableRows,omitempty"`    // table: data rows
```

- [ ] **Step 5: Run it, expect PASS.**

Run: `cd Oblikovati.API && go test ./wire/ -run 'TestTreeNodeRoundTrip|TestTableRoundTrip'`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
cd Oblikovati.API && git add wire/panel_browse.go wire/panel_browse_test.go wire/docking.go
git commit -m "feat(wire): add TreeNode/TableRow DTOs + PanelTree/PanelTable spec fields (Part of #48)"
```

### Task 1.3: Add `client.PanelTree` / `client.PanelTable` builders

**Files:**
- Modify: `Oblikovati.API/client/panel_controls.go` (after `PanelReferenceList`)
- Test: `Oblikovati.API/client/panel_controls_test.go`

**Interfaces:**
- Produces:
  - `func PanelTree(id string, nodes []wire.TreeNode, selected string) wire.PanelControlSpec`
  - `func PanelTable(id string, columns []string, rows []wire.TableRow, selected string) wire.PanelControlSpec`
- Consumes: `types.PanelTree`, `types.PanelTable`, `wire.TreeNode`, `wire.TableRow` (Tasks 1.1–1.2).

- [ ] **Step 1: Write the failing test.** Append to `client/panel_controls_test.go`:

```go
func TestPanelTreeBuilder(t *testing.T) {
	c := PanelTree("catalog", []wire.TreeNode{{ID: "b", Label: "Bearings"}}, "b")
	if c.Kind != types.PanelTree || c.ID != "catalog" || c.Value != "b" || c.Nodes[0].ID != "b" {
		t.Fatalf("PanelTree built wrong control: %+v", c)
	}
}

func TestPanelTableBuilder(t *testing.T) {
	rows := []wire.TableRow{{Key: "k", Cells: []string{"10"}}}
	c := PanelTable("members", []string{"d"}, rows, "k")
	if c.Kind != types.PanelTable || c.ID != "members" || c.Value != "k" ||
		c.TableColumns[0] != "d" || c.TableRows[0].Key != "k" {
		t.Fatalf("PanelTable built wrong control: %+v", c)
	}
}
```

- [ ] **Step 2: Run it, expect FAIL** (undefined `PanelTree`/`PanelTable`).

Run: `cd Oblikovati.API && go test ./client/ -run 'TestPanelTreeBuilder|TestPanelTableBuilder'`
Expected: FAIL, `undefined: PanelTree`.

- [ ] **Step 3: Implement.** Append to `client/panel_controls.go`:

```go
// PanelTree builds a hierarchical browser control: nodes are the root TreeNodes, selected is the
// ID of the currently-highlighted node. A label click pushes a wire.PanelValueChangedEvent whose
// Value is the clicked node's ID; expand/collapse is handled host-side (no event).
func PanelTree(id string, nodes []wire.TreeNode, selected string) wire.PanelControlSpec {
	return wire.PanelControlSpec{Kind: types.PanelTree, ID: id, Nodes: nodes, Value: selected}
}

// PanelTable builds a data-grid control: columns are the header names, rows the data rows,
// selected the Key of the currently-highlighted row. A row click pushes a
// wire.PanelValueChangedEvent whose Value is the row's Key.
func PanelTable(id string, columns []string, rows []wire.TableRow, selected string) wire.PanelControlSpec {
	return wire.PanelControlSpec{Kind: types.PanelTable, ID: id, TableColumns: columns, TableRows: rows, Value: selected}
}
```

- [ ] **Step 4: Run it, expect PASS.**

Run: `cd Oblikovati.API && go test ./client/ -run 'TestPanelTreeBuilder|TestPanelTableBuilder'`
Expected: PASS.

- [ ] **Step 5: Full API suite + lint + SPDX, then commit.**

Run: `cd Oblikovati.API && go test ./... && golangci-lint run && python3 ../Oblikovati/scripts/add-spdx-headers.py --check 2>/dev/null || true`
Expected: all pass.

```bash
cd Oblikovati.API && git add client/panel_controls.go client/panel_controls_test.go
git commit -m "feat(client): add PanelTree/PanelTable builders (Part of #48)"
```

### Task 1.4: Prepare the API branch locally (do NOT PR yet)

Per the revised ship order, the API PR is submitted only after live verification
(Phase 4). Here we just stage the branch so the head + add-in build against it.

- [ ] **Step 1: Full API suite green on the branch.**

Run: `cd Oblikovati.API && go test ./...`
Expected: PASS.

- [ ] **Step 2: Keep the sibling on this branch so `go.work` resolves the new symbols.**

Run: `cd Oblikovati.API && git status -sb`
Expected: on `feat/panel-tree-table` with the three commits (Tasks 1.1–1.3); working tree clean. Do NOT push or open a PR yet — Phase 4 does that after live verification.

- [ ] **Step 3: Sanity-check downstream sees the symbols.**

Run: `cd Oblikovati/head && go build ./internal/native/ 2>&1 | head` and `cd Oblikovati.AddIns.PartDesigner && go build ./designer/... 2>&1 | head`
Expected: no `undefined: types.PanelTree` / `wire.TreeNode` errors (they resolve to the local sibling branch).

---

## Phase 2 — Oblikovati GPL head: render the two kinds (PR 2)

Branch: `feat/panel-tree-table-render` off `origin/develop` in `Oblikovati`.
Prereq: Phase 1 branch staged locally (sibling on `feat/panel-tree-table`). Requires the head to build (`cd Oblikovati/head && make build`) against the local API branch.

### Task 2.1: Add the `BeginTableScrollX` wrapper (horizontal scroll, non-regressing)

**Files:**
- Modify: `Oblikovati/head/internal/native/imgui_wrap.cpp` (near `obk_ig_begin_table`, ~line 520)
- Modify: `Oblikovati/head/internal/native/imgui.go` (near `BeginTable`, ~line 603)
- Modify: the C header that declares `obk_ig_begin_table` (find with `grep -rn "obk_ig_begin_table" Oblikovati/head/internal/native/*.h`)

**Interfaces:**
- Produces: `func BeginTableScrollX(id string, columns int, w, h float32) bool` — same as `BeginTable` but the underlying table also has `ImGuiTableFlags_ScrollX`. Pair with the existing `native.EndTable`.

- [ ] **Step 1: Add the C shim.** In `imgui_wrap.cpp`, after `obk_ig_begin_table`:

```cpp
// begin_table_scrollx is begin_table plus horizontal scrolling, for wide data grids in a narrow
// dock (a content-center member table). Kept separate from begin_table because ScrollX changes
// column auto-sizing and the existing tables rely on the stretch default.
int obk_ig_begin_table_scrollx(const char* id, int columns, float outer_w, float outer_h) {
    ImGuiTableFlags flags = ImGuiTableFlags_Borders | ImGuiTableFlags_RowBg |
                            ImGuiTableFlags_Resizable | ImGuiTableFlags_ScrollY |
                            ImGuiTableFlags_ScrollX;
    return ImGui::BeginTable(id, columns, flags, ImVec2(outer_w, outer_h)) ? 1 : 0;
}
```

- [ ] **Step 2: Declare it in the header** (mirror the `obk_ig_begin_table` line):

```c
int obk_ig_begin_table_scrollx(const char* id, int columns, float outer_w, float outer_h);
```

- [ ] **Step 3: Add the Go wrapper.** In `imgui.go`, after `BeginTable`:

```go
// BeginTableScrollX is BeginTable plus horizontal scrolling — for a wide grid (many columns) in a
// narrow docked panel. Pair every true return with EndTable.
func BeginTableScrollX(id string, columns int, w, h float32) bool {
	c, free := cstr(id)
	defer free()
	return C.obk_ig_begin_table_scrollx(c, C.int(columns), C.float(w), C.float(h)) != 0
}
```

- [ ] **Step 4: Build to verify it compiles.**

Run: `cd Oblikovati/head && go build ./internal/native/`
Expected: exit 0.

- [ ] **Step 5: Commit.**

```bash
cd Oblikovati && git add head/internal/native/imgui_wrap.cpp head/internal/native/imgui.go head/internal/native/*.h
git commit -m "feat(head/native): add BeginTableScrollX for wide docked grids (Part of #48)"
```

### Task 2.2: Render `PanelTree` in the head

**Files:**
- Create: `Oblikovati/head/ui/addin_tree.go`
- Create: `Oblikovati/head/ui/addin_tree_test.go` (pure-helper coverage only)
- Modify: `Oblikovati/head/ui/addin_panels.go` (`drawEditableControl` switch — add a `case types.PanelTree`)

**Interfaces:**
- Consumes: `wire.PanelControlSpec.Nodes` (Task 1.2), `native.TreeNodeSelectable/TreePop/SetNextItemOpen/Selectable/IsItemClicked/PushIDInt/PopID`, `app.Session.PanelValueChanged(windowID, controlID, value)`.
- Produces: `func drawPanelTree(s *app.Session, windowID string, control wire.PanelControlSpec)`; pure helper `func treeFirstUse(id string) bool` (reports first render of a control id, for the `Expanded` seed).

- [ ] **Step 1: Write the pure-helper failing test.** Create `ui/addin_tree_test.go`:

```go
// SPDX-License-Identifier: GPL-2.0-only

package ui

import "testing"

// treeFirstUse must report true the first time a control id is seen and false thereafter, so the
// Expanded seed only applies on first render (afterwards imgui owns open/closed state).
func TestTreeFirstUse(t *testing.T) {
	delete(treeSeeded, "w/catalog")
	if !treeFirstUse("w/catalog") {
		t.Fatal("first call = false, want true")
	}
	if treeFirstUse("w/catalog") {
		t.Fatal("second call = true, want false")
	}
}
```

- [ ] **Step 2: Run it, expect FAIL** (undefined `treeSeeded`/`treeFirstUse`).

Run: `cd Oblikovati/head && go test ./ui/ -run TestTreeFirstUse`
Expected: FAIL, `undefined: treeFirstUse`.

- [ ] **Step 3: Implement `addin_tree.go`.**

```go
// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// treeSeeded tracks which tree controls have rendered once, so a node's Expanded flag seeds the
// disclosure state only on first render; afterwards imgui owns open/collapsed (a re-sent spec must
// not re-collapse what the user opened). Keyed windowID+"/"+controlID.
var treeSeeded = map[string]bool{}

// treeFirstUse reports (and records) whether key is rendering for the first time.
func treeFirstUse(key string) bool {
	if treeSeeded[key] {
		return false
	}
	treeSeeded[key] = true
	return true
}

// drawPanelTree renders a PanelTree: a hierarchy of selectable, expandable nodes. The disclosure
// arrow toggles a node (host-side, no event); a click on a node's label selects it and pushes the
// node ID to the add-in, which re-sends the spec (e.g. to populate a members table).
func drawPanelTree(s *app.Session, windowID string, control wire.PanelControlSpec) {
	firstUse := treeFirstUse(windowID + "/" + control.ID)
	for i := range control.Nodes {
		drawTreeNode(s, windowID, control.ID, control.Value, control.Nodes[i], firstUse)
	}
}

// drawTreeNode renders one node and recurses into its children. selected is the currently-selected
// node ID (for highlight). A leaf uses Selectable; a branch uses TreeNodeSelectable so the arrow
// expands while a label click selects.
func drawTreeNode(s *app.Session, windowID, controlID, selected string, node wire.TreeNode, firstUse bool) {
	if len(node.Children) == 0 {
		if native.Selectable(node.Label, node.ID == selected) {
			s.PanelValueChanged(windowID, controlID, node.ID)
		}
		return
	}
	if firstUse {
		native.SetNextItemOpen(node.Expanded, true)
	}
	open := native.TreeNodeSelectable(node.Label, node.ID == selected)
	if native.IsItemClicked(0) {
		s.PanelValueChanged(windowID, controlID, node.ID)
	}
	if !open {
		return
	}
	for i := range node.Children {
		drawTreeNode(s, windowID, controlID, selected, node.Children[i], firstUse)
	}
	native.TreePop()
}
```

- [ ] **Step 4: Wire the dispatch.** In `addin_panels.go` `drawEditableControl`, add before `default:`:

```go
	case types.PanelTree:
		drawPanelTree(s, windowID, control)
```

- [ ] **Step 5: Run helper test + build, expect PASS + compile.**

Run: `cd Oblikovati/head && go test ./ui/ -run TestTreeFirstUse && go build ./...`
Expected: PASS, build exit 0.

- [ ] **Step 6: Commit.**

```bash
cd Oblikovati && git add head/ui/addin_tree.go head/ui/addin_tree_test.go head/ui/addin_panels.go
git commit -m "feat(head/ui): render PanelTree control (Part of #48)"
```

### Task 2.3: Render `PanelTable` in the head

**Files:**
- Create: `Oblikovati/head/ui/addin_table.go`
- Create: `Oblikovati/head/ui/addin_table_test.go` (pure-helper coverage only)
- Modify: `Oblikovati/head/ui/addin_panels.go` (`drawEditableControl` switch — add `case types.PanelTable`)

**Interfaces:**
- Consumes: `wire.PanelControlSpec.TableColumns`/`TableRows` (Task 1.2), `native.BeginTableScrollX/EndTable/TableSetupColumn/TableSetupScrollFreeze/TableHeadersRow/TableNextRow/TableNextColumn/Selectable/Text/PushIDInt/PopID` (Tasks 2.1 + existing).
- Produces: `func drawPanelTable(s *app.Session, windowID string, control wire.PanelControlSpec)`; pure helper `func cellAt(row wire.TableRow, col int) string` (safe indexing — "" when a row has fewer cells than columns).

- [ ] **Step 1: Write the pure-helper failing test.** Create `ui/addin_table_test.go`:

```go
// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/api/wire"
)

// cellAt must tolerate a row shorter than the column count (a ragged member row) by returning ""
// rather than panicking, so a malformed catalog row can't crash the head.
func TestCellAt(t *testing.T) {
	row := wire.TableRow{Key: "k", Cells: []string{"10", "30"}}
	if got := cellAt(row, 1); got != "30" {
		t.Fatalf("cellAt col1 = %q, want 30", got)
	}
	if got := cellAt(row, 5); got != "" {
		t.Fatalf("cellAt out-of-range = %q, want empty", got)
	}
}
```

- [ ] **Step 2: Run it, expect FAIL** (undefined `cellAt`).

Run: `cd Oblikovati/head && go test ./ui/ -run TestCellAt`
Expected: FAIL, `undefined: cellAt`.

- [ ] **Step 3: Implement `addin_table.go`.**

```go
// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// addInTableHeight caps the member table's height so it shares the docked panel with the tree above
// and the Place button below; the table scrolls internally past this many rows.
const addInTableRows = 8

// cellAt returns row's cell for column col, or "" when the row is shorter than the header (a
// ragged catalog row must not panic the render loop).
func cellAt(row wire.TableRow, col int) string {
	if col < 0 || col >= len(row.Cells) {
		return ""
	}
	return row.Cells[col]
}

// drawPanelTable renders a PanelTable: a scrolling, horizontally-scrolling data grid with a pinned
// header. A row click pushes the row Key to the add-in (which arms Place). No per-frame allocation:
// the spec's strings are drawn as-is.
func drawPanelTable(s *app.Session, windowID string, control wire.PanelControlSpec) {
	cols := len(control.TableColumns)
	if cols == 0 {
		return
	}
	h := float32(rowHeight() * addInTableRows)
	if !native.BeginTableScrollX("##"+control.ID, cols, -1, h) {
		return
	}
	defer native.EndTable()
	for _, name := range control.TableColumns {
		native.TableSetupColumn(name)
	}
	native.TableSetupScrollFreeze(0, 1)
	native.TableHeadersRow()
	for i := range control.TableRows {
		drawTableRow(s, windowID, control, i)
	}
}

// drawTableRow draws one selectable row spanning all columns; the first column carries a
// row-spanning Selectable so a click anywhere on the row selects it.
func drawTableRow(s *app.Session, windowID string, control wire.PanelControlSpec, i int) {
	row := control.TableRows[i]
	native.PushIDInt(i)
	defer native.PopID()
	native.TableNextRow()
	for c := 0; c < len(control.TableColumns); c++ {
		if !native.TableNextColumn() {
			continue
		}
		if c == 0 {
			if native.Selectable(cellAt(row, 0), row.Key == control.Value) {
				s.PanelValueChanged(windowID, control.ID, row.Key)
			}
			continue
		}
		native.Text(cellAt(row, c))
	}
}
```

Note: `rowHeight()` — reuse the head's existing row-height helper if one exists (grep `func rowHeight`/`TextLineHeight`); otherwise use `native.TextLineHeight() + 6`. Pick whichever the codebase already uses for table sizing (`parameters_window.go` `paramTableHeight`).

- [ ] **Step 4: Wire the dispatch.** In `addin_panels.go` `drawEditableControl`, after the `PanelTree` case:

```go
	case types.PanelTable:
		drawPanelTable(s, windowID, control)
```

- [ ] **Step 5: Run helper test + build, expect PASS + compile.**

Run: `cd Oblikovati/head && go test ./ui/ -run TestCellAt && go build ./...`
Expected: PASS, build exit 0.

- [ ] **Step 6: Commit.**

```bash
cd Oblikovati && git add head/ui/addin_table.go head/ui/addin_table_test.go head/ui/addin_panels.go
git commit -m "feat(head/ui): render PanelTable control (Part of #48)"
```

### Task 2.4: Head build + smoke (commit locally, no PR yet)

- [ ] **Step 1: Full head build + smoke** (against the local API branch).

Run: `cd Oblikovati/head && make build && make smoke`
Expected: build exit 0; smoke renders 5 frames exit 0.

- [ ] **Step 2: Full repo suite + lint.**

Run: `cd Oblikovati && go test ./... && (cd head && golangci-lint run)`
Expected: pass. Commit any remaining changes on `feat/panel-tree-table-render`. Do NOT push/PR — Phase 4 ships after live verification.

---

## Phase 3 — PartDesigner add-in: compose the browse surface (PR 3)

Branch: `feat/48-content-center-browse` (already created; the design doc lives here). Rebase on `origin/main` first.
Prereq: Phase 1 released (add-in `go.work` sibling `../Oblikovati.API` pulled to the release). Phase 2 merged is required only for the live screenshot (Task 3.5), not for unit tests.

### Task 3.1: Build the tree model — catalog → `[]wire.TreeNode` with family leaves

**Files:**
- Create: `Oblikovati.AddIns.PartDesigner/designer/browse_tree.go`
- Create: `Oblikovati.AddIns.PartDesigner/designer/browse_tree_test.go`

**Interfaces:**
- Consumes: `catalog.CategoryNode` (from `catalog.Catalog.Tree()`), `catalog.Family`, `familyLabel(*catalog.Family) string` (existing, `selection.go`), `wire.TreeNode`.
- Produces: `func familyTreeNodes(root *catalog.CategoryNode) []wire.TreeNode` — category interior nodes (ID = category path string, from `CategoryPath.String()`) with **family leaves** (ID = `family.ID`, Label = `familyLabel(fam)`); top two levels seeded `Expanded: true`.

- [ ] **Step 1: Write the failing test.** Create `designer/browse_tree_test.go`:

```go
// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"testing"

	"oblikovati.org/part-designer/designer/catalog"
)

// familyTreeNodes must turn the catalog category tree into wire TreeNodes whose LEAVES are
// families (ID = family ID) nested under their category path, so a tree-node click identifies a
// family directly (no label lookup).
func TestFamilyTreeNodesHasFamilyLeaves(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("catalog.Load: %v", err)
	}
	nodes := familyTreeNodes(cat.Tree())
	fam := findFamilyLeaf(nodes)
	if fam == nil {
		t.Fatal("no family leaf found in tree")
	}
	if _, ok := cat.Family(fam.ID); !ok {
		t.Fatalf("family leaf ID %q is not a real family", fam.ID)
	}
	if len(fam.Children) != 0 {
		t.Fatalf("family leaf %q has children, want leaf", fam.ID)
	}
}

// findFamilyLeaf returns the first depth-first node with no children (a family leaf).
func findFamilyLeaf(nodes []wire.TreeNode) *wire.TreeNode {
	for i := range nodes {
		if len(nodes[i].Children) == 0 {
			return &nodes[i]
		}
		if leaf := findFamilyLeaf(nodes[i].Children); leaf != nil {
			return leaf
		}
	}
	return nil
}
```

Add imports `oblikovati.org/api/wire` and `oblikovati.org/part-designer/designer/catalog`. (`catalog.Load()` is the real embedded-catalog loader — see `catalog/load.go:44`, used throughout `catalog/*_test.go`.)

- [ ] **Step 2: Run it, expect FAIL** (undefined `familyTreeNodes`).

Run: `cd Oblikovati.AddIns.PartDesigner && go test ./designer/ -run TestFamilyTreeNodesHasFamilyLeaves`
Expected: FAIL, `undefined: familyTreeNodes`.

- [ ] **Step 3: Implement `browse_tree.go`.**

```go
// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// familyTreeNodes converts the catalog's category tree into wire TreeNodes for a PanelTree, with
// FAMILY leaves: interior nodes are categories (ID = category path) and each family hangs as a
// childless leaf whose ID is the family ID, so a node-click identifies a family directly. The top
// two category levels are seeded open (Expanded) for a useful initial view; deeper levels start
// closed. The host owns expand/collapse after first render.
func familyTreeNodes(root *catalog.CategoryNode) []wire.TreeNode {
	return childNodes(root, 0)
}

// childNodes builds the wire nodes for one category node's children (sub-categories then families).
func childNodes(node *catalog.CategoryNode, depth int) []wire.TreeNode {
	out := make([]wire.TreeNode, 0, len(node.Children)+len(node.Families))
	for _, ch := range node.Children {
		out = append(out, wire.TreeNode{
			ID:       ch.Path.String(),
			Label:    ch.Name,
			Expanded: depth < 2,
			Children: childNodes(ch, depth+1),
		})
	}
	for _, fam := range node.Families {
		out = append(out, wire.TreeNode{ID: fam.ID, Label: familyLabel(fam)})
	}
	return out
}
```

- [ ] **Step 4: Run it, expect PASS.**

Run: `cd Oblikovati.AddIns.PartDesigner && go test ./designer/ -run TestFamilyTreeNodesHasFamilyLeaves`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
cd Oblikovati.AddIns.PartDesigner && git add designer/browse_tree.go designer/browse_tree_test.go
git commit -m "feat(#48): build category-tree wire model with family leaves"
```

### Task 3.2: Build the table model — selected family → columns + `[]wire.TableRow`

**Files:**
- Create: `Oblikovati.AddIns.PartDesigner/designer/browse_table.go`
- Create: `Oblikovati.AddIns.PartDesigner/designer/browse_table_test.go`

**Interfaces:**
- Consumes: `catalog.Family` (`Columns []catalog.Column`, `Members []catalog.Member`, `Member.Values map[string]float64`, `Member.Labels map[string]string`, `Member.Key`), `wire.TableRow`.
- Produces:
  - `func tableColumns(fam *catalog.Family) []string` — the family's column `Name`s in order (nil for a nil family).
  - `func tableRows(fam *catalog.Family) []wire.TableRow` — one row per member, `Key = member.Key`, `Cells` = each column's value formatted (numeric via `catalog`'s existing cell formatter, else the text label).

- [ ] **Step 1: Write the failing test.** Create `designer/browse_table_test.go`:

```go
// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"testing"
)

// tableRows must produce one row per member, keyed by the member Key, with a cell per declared
// column in column order — the full family-table view.
func TestTableModelForFamily(t *testing.T) {
	e, _ := engineWith(t, newFakeHost()) // real helper: placement_test.go:31
	fam := mustFamily(t, e)              // real helper: panel_test.go:126
	cols := tableColumns(fam)
	rows := tableRows(fam)
	if len(cols) != len(fam.Columns) {
		t.Fatalf("cols = %d, want %d", len(cols), len(fam.Columns))
	}
	if len(rows) != len(fam.Members) {
		t.Fatalf("rows = %d, want %d", len(rows), len(fam.Members))
	}
	if rows[0].Key != fam.Members[0].Key {
		t.Fatalf("row0 key = %q, want %q", rows[0].Key, fam.Members[0].Key)
	}
	if len(rows[0].Cells) != len(cols) {
		t.Fatalf("row0 has %d cells, want %d (one per column)", len(rows[0].Cells), len(cols))
	}
}
```

Helpers are real: `engineWith(t, h *fakeHost, kinds ...string) (*Engine, map[string]*recordingGen)` at `placement_test.go:31`, `newFakeHost()` at `host_fake_test.go`, `mustFamily(t, *Engine) *catalog.Family` at `panel_test.go:126`. Call `engineWith(t, newFakeHost())` (no generator kinds needed for a model-only test).

- [ ] **Step 2: Run it, expect FAIL** (undefined `tableColumns`/`tableRows`).

Run: `cd Oblikovati.AddIns.PartDesigner && go test ./designer/ -run TestTableModelForFamily`
Expected: FAIL, `undefined: tableColumns`.

- [ ] **Step 3: Implement `browse_table.go`.** (Use the catalog's existing cell formatter — grep `catalog` for how `sizeLabel`/`placement.go` formats a member value, e.g. a `formatCell`/`strconv.FormatFloat(v,'g',-1,64)`; reuse it, do NOT duplicate.)

```go
// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"strconv"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// tableColumns is the family's column names in declared order (the member table's header).
func tableColumns(fam *catalog.Family) []string {
	if fam == nil {
		return nil
	}
	names := make([]string, len(fam.Columns))
	for i, col := range fam.Columns {
		names[i] = col.Name
	}
	return names
}

// tableRows is one PanelTable row per family member — the full family table. Key is the member's
// canonical Key (what Place consumes); Cells are each column's value in header order (numeric
// columns formatted compactly, text columns via their label).
func tableRows(fam *catalog.Family) []wire.TableRow {
	if fam == nil {
		return nil
	}
	rows := make([]wire.TableRow, len(fam.Members))
	for i, m := range fam.Members {
		rows[i] = wire.TableRow{Key: m.Key, Cells: memberCells(fam, m)}
	}
	return rows
}

// memberCells formats one member's cells in the family's column order.
func memberCells(fam *catalog.Family, m catalog.Member) []string {
	cells := make([]string, len(fam.Columns))
	for i, col := range fam.Columns {
		if v, ok := m.Values[col.Name]; ok {
			cells[i] = strconv.FormatFloat(v, 'g', -1, 64)
			continue
		}
		cells[i] = m.Labels[col.Name]
	}
	return cells
}
```

- [ ] **Step 4: Run it, expect PASS.**

Run: `cd Oblikovati.AddIns.PartDesigner && go test ./designer/ -run TestTableModelForFamily`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
cd Oblikovati.AddIns.PartDesigner && git add designer/browse_table.go designer/browse_table_test.go
git commit -m "feat(#48): build member-table wire model (all columns) for a family"
```

### Task 3.3: Route tree/table selection events

**Files:**
- Modify: `Oblikovati.AddIns.PartDesigner/designer/panel.go` (add control-ID consts `catalogControlID`, `membersControlID`)
- Modify: `Oblikovati.AddIns.PartDesigner/designer/selection.go` (`applySelection` switch)
- Test: `Oblikovati.AddIns.PartDesigner/designer/selection_test.go` (create if absent, else append)

**Interfaces:**
- Consumes: `applySelection(controlID, value string)` (existing), `panelState{familyID, memberKey}`.
- Produces: two new `applySelection` cases — `catalogControlID` sets `sel.familyID = value` (when `value` is a real family ID) and clears `memberKey`; `membersControlID` sets `sel.memberKey = value`. Both then `reconcile`.

- [ ] **Step 1: Write the failing test.** Append/create `designer/selection_test.go`:

```go
// SPDX-License-Identifier: GPL-2.0-only

package designer

import "testing"

// A tree-node click (catalog control) selects that family by ID; a table-row click (members
// control) selects that member by Key. This replaces the old label-based Part/Size dropdowns.
func TestApplySelectionTreeAndTable(t *testing.T) {
	e, _ := engineWith(t, newFakeHost())
	fams := e.catalog.Families()
	fam := fams[0]

	e.applySelection(catalogControlID, fam.ID)
	if e.sel.familyID != fam.ID {
		t.Fatalf("after tree click, familyID = %q, want %q", e.sel.familyID, fam.ID)
	}
	key := fam.Members[0].Key
	e.applySelection(membersControlID, key)
	if e.sel.memberKey != key {
		t.Fatalf("after table click, memberKey = %q, want %q", e.sel.memberKey, key)
	}
}
```

- [ ] **Step 2: Run it, expect FAIL** (undefined `catalogControlID`/`membersControlID`).

Run: `cd Oblikovati.AddIns.PartDesigner && go test ./designer/ -run TestApplySelectionTreeAndTable`
Expected: FAIL, `undefined: catalogControlID`.

- [ ] **Step 3: Add the control-ID consts** in `panel.go`'s control-id `const (...)` block (replacing `familyControlID`/`sizeControlID` usage; keep `categoryControlID`/`standardControlID`/`searchControlID`):

```go
	catalogControlID = "catalog" // PanelTree of category→family
	membersControlID = "members" // PanelTable of the selected family's members
```

- [ ] **Step 4: Add the `applySelection` cases** in `selection.go` (replace the `familyControlID`/`sizeControlID` cases):

```go
	case catalogControlID:
		if _, ok := e.family(value); ok { // value is a family-leaf ID; category nodes are ignored
			e.sel.familyID = value
			e.sel.memberKey = "" // reconcile picks the new family's first size
		}
	case membersControlID:
		e.sel.memberKey = value
```

- [ ] **Step 5: Run it, expect PASS.**

Run: `cd Oblikovati.AddIns.PartDesigner && go test ./designer/ -run TestApplySelectionTreeAndTable`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
cd Oblikovati.AddIns.PartDesigner && git add designer/panel.go designer/selection.go designer/selection_test.go
git commit -m "feat(#48): route tree(family-ID) and table(member-Key) selection events"
```

### Task 3.4: Swap the panel controls to tree + table + Place

**Files:**
- Modify: `Oblikovati.AddIns.PartDesigner/designer/panel.go` (`browserControls`; remove now-dead `familyOptions`/`sizeOptions`/`labelOf`/`sizeLabelOf` if unused after the swap — grep first)
- Test: `Oblikovati.AddIns.PartDesigner/designer/panel_test.go` (append)

**Interfaces:**
- Consumes: `familyTreeNodes` (3.1), `tableColumns`/`tableRows` (3.2), `client.PanelTree`/`client.PanelTable` (1.3), existing `e.family(sel.familyID)`, `e.catalog.Tree()`, filter dropdowns.
- Produces: `browserControls` returns header + Category + Standard + Search + **PanelTree("catalog", …)** + **PanelTable("members", …)** + separator + Place button.

- [ ] **Step 1: Write the failing test.** Append to `panel_test.go`:

```go
// browserControls must expose a PanelTree and a PanelTable (the browse surface) plus the Place
// button, and keep the Category/Standard/Search filters. The old Part/Size dropdowns are gone.
func TestBrowserControlsHasTreeAndTable(t *testing.T) {
	e, _ := engineWith(t, newFakeHost())
	controls := e.browserControls(e.defaultSelection())
	kinds := map[types.PanelControlKind]int{}
	for _, c := range controls {
		kinds[c.Kind]++
	}
	if kinds[types.PanelTree] != 1 || kinds[types.PanelTable] != 1 {
		t.Fatalf("want exactly one tree and one table, got tree=%d table=%d", kinds[types.PanelTree], kinds[types.PanelTable])
	}
	if kinds[types.PanelDropdown] < 2 { // Category + Standard filters remain
		t.Fatalf("want the Category/Standard filter dropdowns retained, got %d dropdowns", kinds[types.PanelDropdown])
	}
}
```

Add imports `oblikovati.org/api/types` if not present.

- [ ] **Step 2: Run it, expect FAIL** (browserControls still builds dropdowns).

Run: `cd Oblikovati.AddIns.PartDesigner && go test ./designer/ -run TestBrowserControlsHasTreeAndTable`
Expected: FAIL (tree=0 table=0).

- [ ] **Step 3: Rewrite `browserControls`.**

```go
// browserControls builds the browse surface: the three filters, then a category TREE (leaves are
// families) over a member parameter TABLE, then Place. The tree/table replace the old cascading
// Part/Size dropdowns (issue #48). The catalogue is immutable, so it is read lock-free here.
func (e *Engine) browserControls(sel panelState) []wire.PanelControlSpec {
	fam, _ := e.family(sel.familyID)
	return []wire.PanelControlSpec{
		client.PanelLabel("hdr", "— Standard Parts —"),
		client.PanelDropdown(categoryControlID, "Category", e.categoryOptions(), orAll(sel.category)),
		client.PanelDropdown(standardControlID, "Standard", e.standardOptions(), orAll(sel.standard)),
		client.PanelTextBox(searchControlID, "Search", sel.search),
		client.PanelTree(catalogControlID, e.treeNodes(sel), sel.familyID),
		client.PanelTable(membersControlID, tableColumns(fam), tableRows(fam), sel.memberKey),
		{Kind: types.PanelSeparator},
		client.PanelButton("place", "Place", PlaceCommandID),
	}
}

// treeNodes builds the family tree filtered by the current Category/Standard/Search selection so
// the tree honours the filters above it. It re-runs the catalog tree over the filtered families.
func (e *Engine) treeNodes(sel panelState) []wire.TreeNode {
	return familyTreeNodes(e.filteredTree(sel))
}
```

- [ ] **Step 4: Add `filteredTree`** — a catalog tree built from only the filtered families. In `browse_tree.go`, add a helper that rebuilds the tree from `e.filteredFamilies(sel)`. Simplest: add to `catalog` a `TreeOf(families []*Family) *CategoryNode` OR filter in the add-in. Prefer keeping catalog pure and add to the add-in:

```go
// filteredTree returns a category tree containing only the families that pass the current filters,
// so the browse tree shows the same set the old Part dropdown did.
func (e *Engine) filteredTree(sel panelState) *catalog.CategoryNode {
	return catalog.TreeOf(e.filteredFamilies(sel))
}
```

Add `catalog.TreeOf` in `catalog/tree.go` (refactor `Tree` to delegate) — see Task 3.4a below.

- [ ] **Step 5: Update obsolete dropdown tests.** Swapping the Part/Size dropdowns for the tree/table makes these `panel_test.go` tests stale — they assert `familyControlID`/`sizeControlID` dropdown behaviour: `TestPanelShowsCascadingBrowser` (:25), `TestSelectionCascade` (:74), `TestSearchNarrowsAndSelects` (:100), `TestSelectFamilyAndCategory` (:136). For each: if it tests a filter (Category/Standard/Search) that still exists, rewrite its assertions to drive `catalogControlID`/`membersControlID` (tree/table selection) instead of the removed dropdowns; if it tests only the removed cascade mechanics now covered by `TestApplySelectionTreeAndTable`/`TestBrowserControlsHasTreeAndTable`, delete it. Keep `controlByID` (:14) and `mustFamily` (:126) helpers. Run `go test ./designer/` and fix each failure.

- [ ] **Step 6: Remove dead code.** Grep for `familyOptions`, `sizeOptions`, `labelOf`, `sizeLabelOf`, `familyByLabel`, `memberByLabel`, `familyControlID`, `sizeControlID`; delete the ones no longer referenced (keep `sizeLabel` if `placement.go` still uses it; keep `familyLabel` — `browse_tree.go` uses it). Run `go vet ./...` to catch unused.

- [ ] **Step 7: Run test + build, expect PASS + compile.**

Run: `cd Oblikovati.AddIns.PartDesigner && go test ./designer/ && go build ./...`
Expected: PASS (including the rewritten tests), build exit 0.

- [ ] **Step 8: Commit.**

```bash
cd Oblikovati.AddIns.PartDesigner && git add designer/panel.go designer/browse_tree.go designer/panel_test.go
git commit -m "feat(#48): replace Part/Size dropdowns with tree+table browse surface"
```

### Task 3.4a: `catalog.TreeOf(families)` (pure refactor of `Tree`)

**Files:**
- Modify: `Oblikovati.AddIns.PartDesigner/designer/catalog/tree.go`
- Test: `Oblikovati.AddIns.PartDesigner/designer/catalog/tree_test.go` (append)

**Interfaces:**
- Produces: `func TreeOf(families []*Family) *CategoryNode` — the existing tree algorithm over an explicit family slice; `Catalog.Tree()` becomes `TreeOf(c.orderedFamilies())`.

- [ ] **Step 1: Write the failing test.** Append to `catalog/tree_test.go`:

```go
func TestTreeOfSubsetOmitsOthers(t *testing.T) {
	c := loadTestCatalog(t) // existing catalog-test loader
	all := c.Families()
	one := TreeOf(all[:1])
	if countFamilies(one) != 1 {
		t.Fatalf("TreeOf(1 family) has %d families, want 1", countFamilies(one))
	}
}

func countFamilies(n *CategoryNode) int {
	total := len(n.Families)
	for _, ch := range n.Children {
		total += countFamilies(ch)
	}
	return total
}
```

- [ ] **Step 2: Run it, expect FAIL** (undefined `TreeOf`).

Run: `cd Oblikovati.AddIns.PartDesigner && go test ./designer/catalog/ -run TestTreeOfSubsetOmitsOthers`
Expected: FAIL.

- [ ] **Step 3: Implement.** In `catalog/tree.go`, extract:

```go
// TreeOf assembles a category tree from an explicit set of families (used to show a filtered
// subtree). Children are sorted by name and families by id, so the tree is deterministic.
func TreeOf(families []*Family) *CategoryNode {
	root := &CategoryNode{}
	for _, fam := range families {
		root.descend(fam.Category).Families = append(root.descend(fam.Category).Families, fam)
	}
	root.sortRecursive()
	return root
}
```

And rewrite `Tree` to delegate:

```go
func (c *Catalog) Tree() *CategoryNode {
	fams := make([]*Family, 0, len(c.order))
	for _, id := range c.order {
		fams = append(fams, c.families[id])
	}
	return TreeOf(fams)
}
```

(Fix the double `descend` in the loop — call once: `node := root.descend(fam.Category); node.Families = append(node.Families, fam)`.)

- [ ] **Step 4: Run it + the existing tree tests, expect PASS.**

Run: `cd Oblikovati.AddIns.PartDesigner && go test ./designer/catalog/`
Expected: PASS (existing `Tree` tests still green).

- [ ] **Step 5: Commit.**

```bash
cd Oblikovati.AddIns.PartDesigner && git add designer/catalog/tree.go designer/catalog/tree_test.go
git commit -m "refactor(#48): extract catalog.TreeOf(families) for filtered subtrees"
```

### Task 3.5: Full add-in suite (commit locally, no PR yet)

**Files:** none (validation).

- [ ] **Step 1: Full add-in suite + coverage + lint.**

Run: `cd Oblikovati.AddIns.PartDesigner && CGO_ENABLED=0 go test ./designer/... -cover && golangci-lint run && markdownlint docs/`
Expected: all pass; coverage > 80% on `designer/`.

- [ ] **Step 2: Duplication check.** Run the repo's duplication tool (see `Makefile`/CI; e.g. `make dup` or the jscpd/`dupl` invocation) — must be < 3%. Commit remaining changes on `feat/48-content-center-browse`. Do NOT push/PR yet.

---

## Phase 4 — Live end-to-end verification, THEN ship all three PRs

Gate: the API PR is submitted only after this live verification passes (user
directive). Everything so far is committed locally on three branches, built
against the local API sibling branch.

### Task 4.1: Live MCP-bridge screenshot verification (MANDATORY)

**Files:** none (a throwaway live-test driver under the scratchpad).

- [ ] **Step 1: Install the add-in + launch the head against the local API branch.** `make install` the add-in into `../Oblikovati/head/addins`; launch the head (built in Task 2.4, which links the local `feat/panel-tree-table` API) with `OBK_ADDINS_DIR` + `DISPLAY=:1`; wait for the MCP bridge (`127.0.0.1:7800`). Follow the live-head launch recipe: Popen the head as a child of ONE foreground python; never background it (memory `live-head-launch-popen-pattern`).

- [ ] **Step 2: Drive + screenshot the browse flow** (memory `always-visual-confirmation`):
  - open the Part Designer panel → screenshot: a **tree** renders with category nodes (Fasteners/Bearings/Structural/Shaft), top levels expanded.
  - click Bearings → Deep-groove disclosure, then a family leaf → screenshot: the **member table** populates below with all parameter columns (d/D/B/…) and one row per size, header pinned, horizontal scroll present if columns overflow.
  - click a member row (e.g. 6202) → screenshot: the row highlights (selection round-trip).
  - click **Place** → screenshot the viewport: the bearing part appears; confirm it is a real DOF-0 parametric part (tree shows published parameters).

- [ ] **Step 3: If any screenshot is wrong, FIX before shipping.** A blank tree, an empty/misaligned table, no horizontal scroll, a click that doesn't populate the table, or Place not firing → return to the relevant Phase 2/3 task. This is the whole point of gating the API PR on live verification. Re-verify after the fix.

- [ ] **Step 4: Save the passing screenshots** to the scratchpad and reference them in the add-in PR body.

### Task 4.2: Ship PR 1 — Oblikovati.API (release)

- [ ] **Step 1: Push + open the API PR.**

```bash
cd Oblikovati.API && git push -u origin feat/panel-tree-table
gh pr create --base develop \
  --title "feat: PanelTree + PanelTable panel controls" \
  --body "Two generic panel-control kinds (tree, data-table) for content-center-style browse surfaces, verified live in the head + PartDesigner add-in before release. Part of Oblikovati.AddIns.PartDesigner#48."
```

- [ ] **Step 2: Wait CI green** (build, race, SonarCloud `new_coverage` ≥ 80, duplication < 3%), then `gh pr merge <N> --merge --delete-branch`.

- [ ] **Step 3: Confirm auto-release** bumped the MINOR version (new tag on `develop`); fast-forward the sibling: `cd Oblikovati.API && git checkout develop && git pull --ff-only origin develop`. Record the released version.

### Task 4.3: Ship PR 2 — Oblikovati head

- [ ] **Step 1:** Rebase `feat/panel-tree-table-render` onto the updated `origin/develop` (now carrying the released API `go.mod` bump if any); rebuild `make build && make smoke` to confirm it still links against the released API.
- [ ] **Step 2: Push, open PR, wait green, merge.**

```bash
cd Oblikovati && git push -u origin feat/panel-tree-table-render
gh pr create --base develop --title "feat(head): render PanelTree + PanelTable controls" \
  --body "Renders the two new panel-control kinds in the docked-panel UI (tree = TreeNodeSelectable; table = new BeginTableScrollX). Live-verified with the PartDesigner browse surface. Part of Oblikovati.AddIns.PartDesigner#48."
gh pr merge <N> --merge --delete-branch
git checkout develop && git pull --ff-only origin develop
```

### Task 4.4: Ship PR 3 — PartDesigner add-in (Closes #48)

- [ ] **Step 1:** Rebase `feat/48-content-center-browse` onto `origin/main`; ensure the add-in `go.mod`/`go.work` reference the released API version; full suite green.
- [ ] **Step 2: Push, open PR (Closes #48), attach the Task 4.1 screenshots, wait green, merge.**

```bash
cd Oblikovati.AddIns.PartDesigner && git push -u origin feat/48-content-center-browse
gh pr create --base main --title "feat: content-center tree + table browse (in-panel)" \
  --body "Replaces the Part/Size dropdowns with an in-panel category tree + member parameter table + Place, via the new PanelTree/PanelTable API controls. Live screenshots attached.

Closes #48"
gh pr merge <N> --merge --delete-branch
git checkout main && git pull --ff-only origin main
```

- [ ] **Step 3: Update memory** — note #48 done in the PartDesigner memory (`partdesigner-content-center-milestone` / `partdesigner-addin`), and that PanelTree/PanelTable now exist in the API for reuse.

---

## Self-Review

**Spec coverage:**
- Surface = in-panel (tree top, table bottom, Place) → Task 3.4. ✓
- Two generic controls PanelTree/PanelTable → Tasks 1.1–1.3 (API), 2.2–2.3 (head). ✓
- All columns + horizontal scroll → Task 3.2 (all columns) + Task 2.1/2.3 (BeginTableScrollX). ✓
- Filters stay above the tree → Task 3.4 (`browserControls` keeps Category/Standard/Search). ✓
- Nested TreeNode, reuse PanelValueChangedEvent, host-side expand/collapse (Expanded = seed) → Task 1.2 (DTO doc), 2.2 (`treeFirstUse`/`SetNextItemOpen(...,true)`). ✓
- TableRow.Key distinct from cells → Task 1.2 + 3.2 (Key = member.Key). ✓
- Contract-first PR1→PR2→PR3, sibling pull between → Task 1.4, Phase 2/3 prereqs. ✓
- Change-Size/persist unchanged → not touched (binding.go/persist.go absent from file lists). ✓
- Testing: API DTO/builder unit tests, head pure-helper tests + live screenshot, add-in fakeHost + live → Tasks 1.2/1.3, 2.2/2.3/3.5, 3.1–3.4. ✓

**Placeholder scan:** No TBD/TODO. Two tasks (3.1, 3.2) reference an existing test loader/engine helper whose exact name must be grepped (`loadTestCatalog`/`newTestEngine`) — flagged inline with the grep target and a fallback, not left vague.

**Type consistency:** `familyTreeNodes(root)` / `childNodes(node, depth)` / `treeNodes(sel)` / `filteredTree(sel)` / `TreeOf(families)` consistent across 3.1/3.4/3.4a. `tableColumns`/`tableRows`/`memberCells` consistent 3.2/3.4. Control IDs `catalogControlID`/`membersControlID` consistent 3.3/3.4. `BeginTableScrollX` consistent 2.1/2.3. `treeFirstUse`/`treeSeeded` consistent 2.2. `cellAt` consistent 2.3. Field names `Nodes`/`TableColumns`/`TableRows` consistent 1.2/1.3/2.2/2.3/3.4.

**Open item for the implementer:** confirm the add-in/catalog test-helper names (`loadTestCatalog`, `newTestEngine`, `firstFamily`) by grepping the existing `*_test.go` before writing Tasks 3.1/3.2 tests; substitute the real constructor. This is the only unresolved naming and is bounded to test scaffolding.
