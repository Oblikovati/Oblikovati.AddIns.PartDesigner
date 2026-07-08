// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"fmt"

	"oblikovati.org/api/types"
)

// attrSet is the attribute-set name the add-in stamps on every part it places, marking it a
// Part Designer part and carrying which family + member it is — so the add-in can recognise
// an activated part and re-drive it to another size (Change-Size).
const attrSet = "com.oblikovati.part-designer"

// familyAttr / memberAttr are the stamped attribute names: the catalogue family id and the
// member's canonical key. Together they identify exactly which standard size a part is.
const (
	familyAttr = "family"
	memberAttr = "member"
)

// stampPart records the family + member a placed part was generated from, so activating it
// later recovers its identity for Change-Size.
func (e *Engine) stampPart(documentID uint64, familyID, memberKey string) error {
	if _, err := e.api.Attributes().Set(documentID, attrSet, familyAttr, types.StringVariant(familyID)); err != nil {
		return fmt.Errorf("stamp family %q: %w", familyID, err)
	}
	if _, err := e.api.Attributes().Set(documentID, attrSet, memberAttr, types.StringVariant(memberKey)); err != nil {
		return fmt.Errorf("stamp member %q: %w", memberKey, err)
	}
	return nil
}

// readStamp reads the family + member a document was stamped with, returning ok=false when
// the document is not a Part Designer part (no stamp).
func (e *Engine) readStamp(documentID uint64) (familyID, memberKey string, ok bool, err error) {
	familyID, ok, err = e.readAttr(documentID, familyAttr)
	if err != nil || !ok {
		return "", "", false, err
	}
	memberKey, ok, err = e.readAttr(documentID, memberAttr)
	if err != nil || !ok {
		return "", "", false, err
	}
	return familyID, memberKey, true, nil
}

// readAttr reads one string attribute from the add-in's set, reporting ok=false when absent.
func (e *Engine) readAttr(documentID uint64, name string) (string, bool, error) {
	res, err := e.api.Attributes().Get(documentID, attrSet, name)
	if err != nil {
		return "", false, fmt.Errorf("read attribute %q: %w", name, err)
	}
	if !res.Found {
		return "", false, nil
	}
	s, isStr := res.Attribute.Value.Str()
	if !isStr {
		return "", false, fmt.Errorf("attribute %q is not a string (got %s)", name, res.Attribute.Value.Type())
	}
	return s, true, nil
}
