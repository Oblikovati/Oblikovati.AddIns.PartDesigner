// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"fmt"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/build"
	"oblikovati.org/part-designer/designer/catalog"
)

// PlaceResult reports the outcome of placing a standard part.
type PlaceResult struct {
	DocumentID uint64 // the generated part document
	Occurrence bool   // whether an occurrence was also placed into an active assembly
}

// Place generates and places one standard part: it resolves the family + member, remembers
// any active assembly, creates and activates a new part document, runs the family's
// generator to build the DOF-0 parametric part, stamps its identity, and — when an assembly
// was active — places an occurrence of it into that assembly. Mirrors Inventor's "place a
// standard part (and drop it into the open assembly)".
func (e *Engine) Place(familyID, memberKey string) (PlaceResult, error) {
	fam, member, err := e.resolve(familyID, memberKey)
	if err != nil {
		return PlaceResult{}, err
	}
	gen, ok := e.gens.Get(fam.Generator)
	if !ok {
		return PlaceResult{}, fmt.Errorf("family %q needs generator %q, which is not registered", familyID, fam.Generator)
	}
	// Capture the active assembly BEFORE creating the part (creating + activating the part
	// changes the active document).
	asmID, hasAsm, err := e.activeAssembly()
	if err != nil {
		return PlaceResult{}, err
	}

	docID, name, err := e.generatePart(fam, member, gen)
	if err != nil {
		return PlaceResult{DocumentID: docID}, err
	}
	if err := e.stampPart(docID, familyID, memberKey); err != nil {
		return PlaceResult{DocumentID: docID}, err
	}

	res := PlaceResult{DocumentID: docID}
	if hasAsm {
		if err := e.placeOccurrence(asmID, docID, name); err != nil {
			return res, err
		}
		res.Occurrence = true
	}
	return res, nil
}

// generatePart creates + activates a new part document and runs the generator on the member,
// returning the document id (non-zero once created, so the caller can report it even on a
// generate failure) and the part's name.
func (e *Engine) generatePart(fam *catalog.Family, member catalog.Member, gen build.PartGenerator) (uint64, string, error) {
	name := partName(fam, member)
	docID, err := e.createPart(name)
	if err != nil {
		return 0, name, err
	}
	builder := build.NewPartBuilder(e.api, fam.Units)
	if err := gen.Build(builder, build.ResolvedMember{Family: fam, Member: member}); err != nil {
		return docID, name, fmt.Errorf("generate %s %s: %w", fam.ID, member.Key, err)
	}
	return docID, name, nil
}

// ChangeSize re-drives the active Part Designer part to a different member of its own family,
// in place — the procedural analogue of Inventor's read-only "Change Size". Because the
// geometry is parameter-driven, re-publishing the new member's parameters and recomputing
// updates the existing part without rebuilding it.
func (e *Engine) ChangeSize(newMemberKey string) error {
	docID, isPart, err := e.activePart()
	if err != nil {
		return err
	}
	if !isPart {
		return fmt.Errorf("no active part document to resize")
	}
	familyID, _, stamped, err := e.readStamp(docID)
	if err != nil {
		return err
	}
	if !stamped {
		return fmt.Errorf("active document is not a Part Designer part")
	}
	fam, member, err := e.resolve(familyID, newMemberKey)
	if err != nil {
		return err
	}
	builder := build.NewPartBuilder(e.api, fam.Units)
	if err := builder.PublishParams(build.ResolvedMember{Family: fam, Member: member}); err != nil {
		return err
	}
	if _, err := e.api.Documents().Update(true); err != nil {
		return fmt.Errorf("recompute after resize: %w", err)
	}
	return e.stampPart(docID, familyID, newMemberKey)
}

// resolve looks up a family and one of its members, surfacing a catalogue-load failure.
func (e *Engine) resolve(familyID, memberKey string) (*catalog.Family, catalog.Member, error) {
	if e.catalog == nil {
		return nil, catalog.Member{}, fmt.Errorf("catalogue unavailable: %w", e.catErr)
	}
	fam, ok := e.catalog.Family(familyID)
	if !ok {
		return nil, catalog.Member{}, fmt.Errorf("unknown family %q", familyID)
	}
	member, ok := fam.Member(memberKey)
	if !ok {
		return nil, catalog.Member{}, fmt.Errorf("family %q has no member %q", familyID, memberKey)
	}
	return fam, member, nil
}

// createPart creates a new part document and activates it, so the generator's subsequent
// sketch/feature/parameter calls target it.
func (e *Engine) createPart(name string) (uint64, error) {
	doc, err := e.api.Documents().Create(wire.CreateDocumentArgs{Type: "part", Name: name})
	if err != nil {
		return 0, fmt.Errorf("create part %q: %w", name, err)
	}
	if _, err := e.api.Documents().Activate(doc.ID); err != nil {
		return 0, fmt.Errorf("activate part %q: %w", name, err)
	}
	return doc.ID, nil
}

// placeOccurrence activates the assembly and nests the part into it at the origin.
func (e *Engine) placeOccurrence(assemblyID, partID uint64, name string) error {
	if _, err := e.api.Documents().Activate(assemblyID); err != nil {
		return fmt.Errorf("activate assembly: %w", err)
	}
	_, err := e.api.Assembly().Place(wire.PlaceOccurrenceArgs{
		Document: partID, Name: name, Transform: types.IdentityMatrix(),
	})
	if err != nil {
		return fmt.Errorf("place occurrence %q: %w", name, err)
	}
	return nil
}

// activeAssembly returns the active document's id when it is an assembly.
func (e *Engine) activeAssembly() (uint64, bool, error) {
	return e.activeOfType("assembly")
}

// activePart returns the active document's id when it is a part.
func (e *Engine) activePart() (uint64, bool, error) {
	return e.activeOfType("part")
}

// activeOfType returns the active document's id when its type matches.
func (e *Engine) activeOfType(docType string) (uint64, bool, error) {
	list, err := e.api.Documents().List()
	if err != nil {
		return 0, false, fmt.Errorf("list documents: %w", err)
	}
	for _, d := range list.Documents {
		if d.Active && d.Type == docType {
			return d.ID, true, nil
		}
	}
	return 0, false, nil
}

// partName is the generated part document's name: the standard plus a compact size label
// (e.g. "ISO 4017 M8x40"). It is a human label; the stamped attributes carry the exact
// family + member for round-tripping.
func partName(fam *catalog.Family, m catalog.Member) string {
	if label := sizeLabel(fam, m); label != "" {
		return fam.Standard + " " + label
	}
	return fam.Standard
}

// sizeLabel joins the member's key-column values compactly (e.g. "8x40" for keys d,l), reusing
// the shared per-column formatter (memberCellValue) so the members table and this label format
// numbers identically.
func sizeLabel(fam *catalog.Family, m catalog.Member) string {
	parts := make([]string, 0, len(fam.KeyColumns))
	for _, col := range fam.KeyColumns {
		if s, ok := memberCellValue(m, col); ok {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "x")
}
