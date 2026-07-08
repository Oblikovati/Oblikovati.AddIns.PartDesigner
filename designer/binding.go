// SPDX-License-Identifier: GPL-2.0-only

package designer

// bindActiveDocument reconciles the panel with whatever document just became active: if it is a
// stamped Part Designer part, the panel switches to that part's family + size and binds, so the
// panel shows the active part's current size and a Size change re-drives it in place (Change-Size);
// otherwise the panel unbinds and keeps the user's browse selection. It runs OFF the session
// goroutine (dispatched from Notify) because it makes host calls (list documents, read attributes,
// re-show the panel).
func (e *Engine) bindActiveDocument() {
	familyID, memberKey, ok := e.activeStampedPart()
	e.mu.Lock()
	if ok {
		e.sel = panelState{familyID: familyID, memberKey: memberKey}
	}
	e.bound = ok
	e.mu.Unlock()
	_, _ = e.ShowPanel()
}

// activeStampedPart returns the family + member of the active document when it is a Part Designer
// part whose stamped family is still in the catalogue, else ok=false. A host or catalogue error is
// treated as "not a bound part" (the panel simply stays in browse mode rather than surfacing it).
func (e *Engine) activeStampedPart() (familyID, memberKey string, ok bool) {
	docID, isPart, err := e.activePart()
	if err != nil || !isPart {
		return "", "", false
	}
	familyID, memberKey, stamped, err := e.readStamp(docID)
	if err != nil || !stamped {
		return "", "", false
	}
	if _, _, err := e.resolve(familyID, memberKey); err != nil {
		return "", "", false // the stamped family is no longer in the catalogue
	}
	return familyID, memberKey, true
}
