// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"strings"

	"oblikovati.org/part-designer/designer/catalog"
)

// panelState is the panel's current cascading selection: two filters (a top-level category
// and a standards body, empty = "All") narrow the Part list, and a chosen family + member
// (size) identify what Place builds.
type panelState struct {
	category  string // top-level category segment; "" = all
	standard  string // standards body (ISO/DIN/ANSI); "" = all
	familyID  string
	memberKey string
}

// allOption is the dropdown entry that clears a filter.
const allOption = "All"

// applySelection updates the selection for one dropdown edit, then reconciles so the
// downstream choices stay valid (changing a filter re-picks the family; changing the family
// re-picks the size). The caller holds the engine mutex.
func (e *Engine) applySelection(controlID, value string) {
	switch controlID {
	case categoryControlID:
		e.sel.category = clearAll(value)
	case standardControlID:
		e.sel.standard = clearAll(value)
	case familyControlID:
		if f := e.familyByLabel(e.sel, value); f != nil {
			e.sel.familyID = f.ID
			e.sel.memberKey = "" // force reconcile to the new family's first size
		}
	case sizeControlID:
		if fam, ok := e.family(e.sel.familyID); ok {
			if m, ok := memberByLabel(fam, value); ok {
				e.sel.memberKey = m.Key
			}
		}
	}
	e.sel = e.reconcile(e.sel)
}

// reconcile fixes a selection so its family is one of the currently-filtered families and its
// member is one of that family's members (falling back to the first of each), so the panel
// never shows a stale or impossible choice.
func (e *Engine) reconcile(sel panelState) panelState {
	wantMember := sel.memberKey // capture before the reset below
	fam := pickFamily(e.filteredFamilies(sel.category, sel.standard), sel.familyID)
	sel.familyID, sel.memberKey = "", ""
	if fam != nil {
		sel.familyID = fam.ID
		sel.memberKey = pickMember(fam, wantMember)
	}
	return sel
}

// defaultSelection is the initial selection: no filters, the first family + its first size.
func (e *Engine) defaultSelection() panelState { return e.reconcile(panelState{}) }

// filteredFamilies returns the families matching both filters (empty filter matches all), in
// catalogue order.
func (e *Engine) filteredFamilies(category, standard string) []*catalog.Family {
	if e.catalog == nil {
		return nil
	}
	var out []*catalog.Family
	for _, f := range e.catalog.Families() {
		if category != "" && (len(f.Category) == 0 || f.Category[0] != category) {
			continue
		}
		if standard != "" && f.Body() != standard {
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

// familyByLabel finds a filtered family by its display label.
func (e *Engine) familyByLabel(sel panelState, label string) *catalog.Family {
	for _, f := range e.filteredFamilies(sel.category, sel.standard) {
		if familyLabel(f) == label {
			return f
		}
	}
	return nil
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

// memberByLabel finds a member of fam by its size label.
func memberByLabel(fam *catalog.Family, label string) (catalog.Member, bool) {
	for _, m := range fam.Members {
		if sizeLabel(fam, m) == label {
			return m, true
		}
	}
	return catalog.Member{}, false
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
