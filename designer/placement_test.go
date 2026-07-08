// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"testing"

	"oblikovati.org/api/client"
	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/build"
	"oblikovati.org/part-designer/designer/catalog"
)

// recordingGen is a test PartGenerator bound to the seed families' generator kinds. It
// publishes the member's parameters (so the fake host records them) and remembers what it
// built, standing in for the real geometry generators until B-phase.
type recordingGen struct {
	kind  string
	built []build.ResolvedMember
}

func (g *recordingGen) Kind() string { return g.kind }

func (g *recordingGen) Build(b *build.PartBuilder, rm build.ResolvedMember) error {
	g.built = append(g.built, rm)
	return b.PublishParams(rm)
}

// engineWith builds an engine over the real embedded catalogue and a registry that binds the
// seed families' generator to a recording stub.
func engineWith(t *testing.T, h *fakeHost, kinds ...string) (*Engine, map[string]*recordingGen) {
	t.Helper()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("catalog.Load: %v", err)
	}
	reg := build.NewRegistry()
	gens := map[string]*recordingGen{}
	for _, k := range kinds {
		g := &recordingGen{kind: k}
		if err := reg.Register(g); err != nil {
			t.Fatalf("register %q: %v", k, err)
		}
		gens[k] = g
	}
	return &Engine{host: h, api: client.New(h), catalog: cat, gens: reg}, gens
}

func TestPlaceCreatesStampedPart(t *testing.T) {
	h := newFakeHost()
	e, gens := engineWith(t, h, "hex_bolt")

	res, err := e.Place("iso4017-hex-bolt", "d=8,l=40")
	if err != nil {
		t.Fatalf("Place error = %v", err)
	}
	if res.DocumentID == 0 || res.Occurrence {
		t.Errorf("result = %+v, want a part id and no occurrence (no assembly active)", res)
	}
	// A part document was created and the generator ran on the resolved member.
	if len(h.docs) != 1 || h.docs[0].Type != "part" {
		t.Fatalf("docs = %+v, want one part", h.docs)
	}
	if len(gens["hex_bolt"].built) != 1 || gens["hex_bolt"].built[0].Member.Key != "d=8,l=40" {
		t.Errorf("generator built = %+v, want member d=8,l=40", gens["hex_bolt"].built)
	}
	// Parameters were published from the member's columns.
	if !hasParam(h.added, "across_flats", "13 mm") || !hasParam(h.added, "length", "40 mm") {
		t.Errorf("published params = %+v, want across_flats=13 mm and length=40 mm", h.added)
	}
	// The part is stamped with its family + member for Change-Size.
	if got := h.attrs[attrKey(res.DocumentID, attrSet, familyAttr)]; got != "iso4017-hex-bolt" {
		t.Errorf("family stamp = %q, want iso4017-hex-bolt", got)
	}
	if got := h.attrs[attrKey(res.DocumentID, attrSet, memberAttr)]; got != "d=8,l=40" {
		t.Errorf("member stamp = %q, want d=8,l=40", got)
	}
}

func TestPlaceIntoActiveAssembly(t *testing.T) {
	h := newFakeHost()
	asmID := h.seedActiveAssembly("Assembly1")
	e, _ := engineWith(t, h, "hex_bolt")

	res, err := e.Place("iso4017-hex-bolt", "d=6,l=30")
	if err != nil {
		t.Fatalf("Place error = %v", err)
	}
	if !res.Occurrence {
		t.Error("Occurrence = false, want an occurrence placed into the active assembly")
	}
	if len(h.placed) != 1 || h.placed[0].Document != res.DocumentID {
		t.Errorf("placed = %+v, want one occurrence of part %d", h.placed, res.DocumentID)
	}
	if res.DocumentID == asmID {
		t.Error("placed the assembly as its own occurrence")
	}
}

func TestChangeSizeRedrivesActivePart(t *testing.T) {
	h := newFakeHost()
	e, _ := engineWith(t, h, "hex_bolt")
	if _, err := e.Place("iso4017-hex-bolt", "d=8,l=40"); err != nil {
		t.Fatalf("Place error = %v", err)
	}
	partID := h.docs[0].ID

	if err := e.ChangeSize("d=12,l=60"); err != nil {
		t.Fatalf("ChangeSize error = %v", err)
	}
	// The new member's parameters were re-driven via Set (they already existed), the document
	// recomputed, and the stamp updated to the new member.
	if !hasParam(h.set, "across_flats", "18 mm") || !hasParam(h.set, "length", "60 mm") {
		t.Errorf("re-driven params = %+v, want across_flats=18 mm and length=60 mm", h.set)
	}
	if h.updates == 0 {
		t.Error("document was not recomputed after resize")
	}
	if got := h.attrs[attrKey(partID, attrSet, memberAttr)]; got != "d=12,l=60" {
		t.Errorf("member stamp after resize = %q, want d=12,l=60", got)
	}
}

func TestPlaceErrors(t *testing.T) {
	h := newFakeHost()
	e, _ := engineWith(t, h, "hex_bolt") // hex_nut deliberately NOT registered

	if _, err := e.Place("nope", "d=8,l=40"); err == nil {
		t.Error("Place accepted an unknown family")
	}
	if _, err := e.Place("iso4017-hex-bolt", "d=99,l=99"); err == nil {
		t.Error("Place accepted an unknown member")
	}
	if _, err := e.Place("din934-hex-nut", "d=8"); err == nil {
		t.Error("Place accepted a family whose generator is not registered")
	}
}

func TestChangeSizeRequiresStampedPart(t *testing.T) {
	// Active document is an assembly, not a part.
	h := newFakeHost()
	h.seedActiveAssembly("Assembly1")
	e, _ := engineWith(t, h)
	if err := e.ChangeSize("d=8,l=40"); err == nil {
		t.Error("ChangeSize accepted a non-part active document")
	}
}

func TestChangeSizeUnstampedPart(t *testing.T) {
	h := newFakeHost()
	h.seedActivePart("Some Part") // active part, but not one this add-in placed
	e, _ := engineWith(t, h, "hex_bolt")
	if err := e.ChangeSize("d=8,l=40"); err == nil {
		t.Error("ChangeSize accepted an unstamped part")
	}
}

func TestPlaceHostFailurePropagates(t *testing.T) {
	h := newFakeHost()
	h.failMethod = "documents.create"
	e, _ := engineWith(t, h, "hex_bolt")
	if _, err := e.Place("iso4017-hex-bolt", "d=8,l=40"); err == nil {
		t.Error("Place ignored a host document-create failure")
	}
}

func TestEngineAccessors(t *testing.T) {
	e := NewEngine(newFakeHost())
	if e.Catalog() == nil {
		t.Error("Catalog() = nil, want the loaded catalogue")
	}
	if e.API() == nil {
		t.Error("API() = nil")
	}
}

func TestPartNameLabels(t *testing.T) {
	numeric := &catalog.Family{
		Standard: "ISO 4017", KeyColumns: []string{"d", "l"},
	}
	numMember := catalog.Member{Values: map[string]float64{"d": 8, "l": 40}}
	if got := partName(numeric, numMember); got != "ISO 4017 8x40" {
		t.Errorf("partName = %q, want ISO 4017 8x40", got)
	}

	text := &catalog.Family{Standard: "ANSI B18", KeyColumns: []string{"size"}}
	textMember := catalog.Member{Labels: map[string]string{"size": "1/4"}}
	if got := partName(text, textMember); got != "ANSI B18 1/4" {
		t.Errorf("partName = %q, want ANSI B18 1/4", got)
	}

	noKeys := &catalog.Family{Standard: "DIN 1"}
	if got := partName(noKeys, catalog.Member{}); got != "DIN 1" {
		t.Errorf("partName = %q, want DIN 1 (no size label)", got)
	}
}

// hasParam reports whether args contains a parameter with the given name + expression.
func hasParam(args []wire.ParameterSetArgs, name, expr string) bool {
	for _, a := range args {
		if a.Name == name && a.Expression == expr {
			return true
		}
	}
	return false
}
