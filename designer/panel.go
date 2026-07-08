// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"sort"

	"oblikovati.org/api/client"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// PanelID is the stable dockable-window id the add-in owns.
const PanelID = "com.oblikovati.part-designer.panel"

// Panel control ids. Dropdown edits arrive as panel.valueChanged for these; the Place button
// carries the PartDesigner.Place command, so it arrives as command.started instead.
const (
	categoryControlID = "category"
	standardControlID = "standard"
	familyControlID   = "family"
	sizeControlID     = "size"
)

// ShowPanel creates (or replaces) the Part Designer dockable window, following Inventor's
// "Place from Content Center" flow with the host's current control vocabulary: cascading
// Category → Standard → Part → Size dropdowns plus a Place button. A richer tree+table
// browse dialog is deferred to a follow-up API extension.
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

// browserControls builds the cascading dropdowns + Place button for a selection.
func (e *Engine) browserControls(sel panelState) []wire.PanelControlSpec {
	fam, _ := e.family(sel.familyID)
	return []wire.PanelControlSpec{
		client.PanelLabel("hdr", "— Standard Parts —"),
		client.PanelDropdown(categoryControlID, "Category", e.categoryOptions(), orAll(sel.category)),
		client.PanelDropdown(standardControlID, "Standard", e.standardOptions(), orAll(sel.standard)),
		client.PanelDropdown(familyControlID, "Part", e.familyOptions(sel), labelOf(fam)),
		client.PanelDropdown(sizeControlID, "Size", sizeOptions(fam), sizeLabelOf(fam, sel.memberKey)),
		{Kind: types.PanelSeparator},
		client.PanelButton("place", "Place", PlaceCommandID),
	}
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

// familyOptions lists the labels of the families matching the current filters.
func (e *Engine) familyOptions(sel panelState) []string {
	var opts []string
	for _, f := range e.filteredFamilies(sel.category, sel.standard) {
		opts = append(opts, familyLabel(f))
	}
	return opts
}

// sizeOptions lists the size labels of a family's members.
func sizeOptions(fam *catalog.Family) []string {
	if fam == nil {
		return nil
	}
	opts := make([]string, 0, len(fam.Members))
	for _, m := range fam.Members {
		opts = append(opts, sizeLabel(fam, m))
	}
	return opts
}

// labelOf is a family's dropdown label, or "" for none.
func labelOf(fam *catalog.Family) string {
	if fam == nil {
		return ""
	}
	return familyLabel(fam)
}

// sizeLabelOf is the size label of a family member by key, or "" when absent.
func sizeLabelOf(fam *catalog.Family, memberKey string) string {
	if fam == nil {
		return ""
	}
	if m, ok := fam.Member(memberKey); ok {
		return sizeLabel(fam, m)
	}
	return ""
}

// orAll renders an empty filter as the "All" sentinel for the dropdown's current value.
func orAll(value string) string {
	if value == "" {
		return allOption
	}
	return value
}
