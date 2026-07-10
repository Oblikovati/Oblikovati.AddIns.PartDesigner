// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"sort"

	"oblikovati.org/api/client"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// PanelID is the stable dockable-window id the add-in owns.
const PanelID = "com.oblikovati.part-designer.panel"

// Panel control ids. Dropdown/tree/table edits arrive as panel.valueChanged for these; the
// Place button carries the PartDesigner.Place command, so it arrives as command.started instead.
const (
	categoryControlID = "category"
	standardControlID = "standard"
	searchControlID   = "search"
	catalogControlID  = "catalog" // PanelTree of category→family
	membersControlID  = "members" // PanelTable of the selected family's members
)

// ShowPanel creates (or replaces) the Part Designer dockable window, following Inventor's
// "Place from Content Center" flow: Category/Standard/Search filters over a category TREE
// (leaves are families) and a member parameter TABLE, plus a Place button (issue #48).
func (e *Engine) ShowPanel() (wire.OKResult, error) {
	return e.api.DockableWindows().Set(wire.DockableWindowSpec{
		ID:       PanelID,
		Title:    "Part Designer",
		Dock:     types.DockRight,
		Visible:  true,
		Controls: e.panelControls(),
	})
}

// panelControls renders the browser from a snapshot of the current selection (taken under the
// lock; the catalogue is immutable, so the controls are then built lock-free).
func (e *Engine) panelControls() []wire.PanelControlSpec {
	if e.catalog == nil {
		return catalogErrorControls(e.catErr)
	}
	e.mu.Lock()
	sel := e.sel
	e.mu.Unlock()
	return e.browserControls(sel)
}

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

// catalogErrorControls is shown when the embedded catalogue failed to load (a build-time
// bug), so the failure is visible rather than an empty panel.
func catalogErrorControls(err error) []wire.PanelControlSpec {
	msg := "Standard-parts catalogue failed to load."
	if err != nil {
		msg = "Catalogue error: " + err.Error()
	}
	return []wire.PanelControlSpec{
		client.PanelLabel("hdr", "— Part Designer —"),
		client.PanelLabel("err", msg),
	}
}

// categoryOptions lists "All" plus the distinct top-level categories, sorted.
func (e *Engine) categoryOptions() []string {
	seen := map[string]bool{}
	var segs []string
	for _, f := range e.catalog.Families() {
		if len(f.Category) > 0 && !seen[f.Category[0]] {
			seen[f.Category[0]] = true
			segs = append(segs, f.Category[0])
		}
	}
	sort.Strings(segs)
	return append([]string{allOption}, segs...)
}

// standardOptions lists "All" plus the catalogue's standards bodies.
func (e *Engine) standardOptions() []string {
	return append([]string{allOption}, e.catalog.Standards()...)
}

// orAll renders an empty filter as the "All" sentinel for the dropdown's current value.
func orAll(value string) string {
	if value == "" {
		return allOption
	}
	return value
}
