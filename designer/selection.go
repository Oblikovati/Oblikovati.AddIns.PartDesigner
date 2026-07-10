// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"strings"

	"oblikovati.org/part-designer/designer/catalog"
)

// panelState is the panel's current browse selection: three filters (a top-level category, a
// standards body — empty = "All" — and a free-text search) narrow the category tree/members
// table, and a chosen family + member (size) identify what Place builds.
type panelState struct {
	category  string // top-level category segment; "" = all
	standard  string // standards body (ISO/DIN/ANSI); "" = all
	search    string // free-text query over family name and size; "" = no search
	familyID  string
	memberKey string
}

// allOption is the dropdown entry that clears a filter.
const allOption = "All"

// applySelection updates the selection for one control edit (filter dropdown, tree-node click,
// or table-row click), then reconciles so the downstream choices stay valid (changing a filter
// re-picks the family; changing the family re-picks the size). The caller holds the engine mutex.
func (e *Engine) applySelection(controlID, value string) {
	switch controlID {
	case categoryControlID:
		e.sel.category = clearAll(value)
	case standardControlID:
		e.sel.standard = clearAll(value)
	case searchControlID:
		e.sel.search = strings.TrimSpace(value)
	case catalogControlID:
		if _, ok := e.family(value); ok { // value is a family-leaf ID; category nodes are ignored
			e.sel.familyID = value
			e.sel.memberKey = "" // reconcile picks the new family's first size
		}
	case membersControlID:
		e.sel.memberKey = value
	}
	e.sel = e.reconcile(e.sel)
}

// reconcile fixes a selection so its family is one of the currently-filtered families and its
// member is one of that family's members (falling back to the first of each), so the panel
// never shows a stale or impossible choice.
func (e *Engine) reconcile(sel panelState) panelState {
	wantMember := sel.memberKey // capture before the reset below
	fam := pickFamily(e.filteredFamilies(sel), sel.familyID)
	sel.familyID, sel.memberKey = "", ""
	if fam != nil {
		sel.familyID = fam.ID
		sel.memberKey = pickMember(fam, wantMember)
	}
	return sel
}

// defaultSelection is the initial selection: no filters, the first family + its first size.
func (e *Engine) defaultSelection() panelState { return e.reconcile(panelState{}) }

// filteredFamilies returns the families matching all three filters — top-level category, standards
// body and free-text search (an empty filter matches all) — in catalogue order.
func (e *Engine) filteredFamilies(sel panelState) []*catalog.Family {
	if e.catalog == nil {
		return nil
	}
	query := strings.ToLower(sel.search)
	var out []*catalog.Family
	for _, f := range e.catalog.Families() {
		if sel.category != "" && (len(f.Category) == 0 || f.Category[0] != sel.category) {
			continue
		}
		if sel.standard != "" && f.Body() != sel.standard {
			continue
		}
		if !f.Matches(query) {
			continue
		}
		out = append(out, f)
	}
	return out
}

// family resolves a family by id.
func (e *Engine) family(id string) (*catalog.Family, bool) {
	if e.catalog == nil {
		return nil, false
	}
	return e.catalog.Family(id)
}

// pickFamily returns the family with id, else the first family, else nil.
func pickFamily(fams []*catalog.Family, id string) *catalog.Family {
	for _, f := range fams {
		if f.ID == id {
			return f
		}
	}
	if len(fams) > 0 {
		return fams[0]
	}
	return nil
}

// pickMember returns key when it is a member of fam, else the first member's key, else "".
func pickMember(fam *catalog.Family, key string) string {
	for _, m := range fam.Members {
		if m.Key == key {
			return key
		}
	}
	if len(fam.Members) > 0 {
		return fam.Members[0].Key
	}
	return ""
}

// familyLabel is a family's human name in the Part dropdown: standard + leaf category (e.g.
// "ISO 4017 Hex Head").
func familyLabel(f *catalog.Family) string {
	seg := ""
	if n := len(f.Category); n > 0 {
		seg = f.Category[n-1]
	}
	return strings.TrimSpace(f.Standard + " " + seg)
}

// clearAll maps the "All" sentinel to the empty (no-filter) value.
func clearAll(value string) string {
	if value == allOption {
		return ""
	}
	return value
}
