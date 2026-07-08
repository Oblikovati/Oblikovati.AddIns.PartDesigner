// SPDX-License-Identifier: GPL-2.0-only

package catalog

import (
	"strings"
	"testing"
)

// TestLoadEmbeddedCatalog is the A2 acceptance check: the embedded seed families load,
// validate, and expose the expected shape.
func TestLoadEmbeddedCatalog(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if c.Len() < 2 {
		t.Fatalf("loaded %d families, want at least 2 (the two seed fasteners)", c.Len())
	}
	bolt, ok := c.Family("iso4017-hex-bolt")
	if !ok {
		t.Fatal("iso4017-hex-bolt not loaded")
	}
	if bolt.Generator != "hex_bolt" {
		t.Errorf("generator = %q, want hex_bolt", bolt.Generator)
	}
	if bolt.Units != UnitsMillimetre {
		t.Errorf("units = %q, want mm", bolt.Units)
	}
	if got := bolt.Category.String(); got != "Fasteners/Bolts/Hex Head" {
		t.Errorf("category = %q, want Fasteners/Bolts/Hex Head", got)
	}
	if len(bolt.Members) != 4 {
		t.Fatalf("bolt members = %d, want 4", len(bolt.Members))
	}
}

// TestMemberRoundTrip loads a family, recomputes a member's key, and resolves it back — the
// stable-key contract that placement + Change-Size rely on.
func TestMemberRoundTrip(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	bolt, _ := c.Family("iso4017-hex-bolt")

	m := bolt.Members[1] // M8×40
	if m.Key != "d=8,l=40" {
		t.Fatalf("member key = %q, want d=8,l=40", m.Key)
	}
	got, ok := bolt.Member(m.Key)
	if !ok {
		t.Fatalf("Member(%q) not found", m.Key)
	}
	for _, want := range []struct {
		col string
		val float64
	}{{"d", 8}, {"P", 1.25}, {"s", 13}, {"k", 5.3}, {"l", 40}, {"b", 40}} {
		if got.Values[want.col] != want.val {
			t.Errorf("member[%q] = %v, want %v", want.col, got.Values[want.col], want.val)
		}
	}
}

// TestParseFamilyValidation exercises the guarantees the loader makes about a family table,
// each via a minimally-broken family. The messages must name the offending value.
func TestParseFamilyValidation(t *testing.T) {
	cases := []struct {
		name string
		json string
		want string
	}{
		{"duplicate param", famJSON(`"columns":[
			{"name":"d","param":"dia","type":"length"},
			{"name":"e","param":"dia","type":"length"}],
			"keyColumns":["d"],"members":[{"d":6,"e":7}]`), "duplicate param name"},
		{"missing key column", famJSON(`"columns":[{"name":"d","param":"dia","type":"length"}],
			"keyColumns":["x"],"members":[{"d":6}]`), "key column \"x\" is not a declared column"},
		{"malformed category", strings.Replace(famJSON(`"columns":[{"name":"d","param":"dia","type":"length"}],
			"keyColumns":["d"],"members":[{"d":6}]`), `"Fasteners/Bolts"`, `"Fasteners//Bolts"`, 1), "empty segment"},
		{"missing cell", famJSON(`"columns":[
			{"name":"d","param":"dia","type":"length"},{"name":"s","param":"af","type":"length"}],
			"keyColumns":["d"],"members":[{"d":6}]`), "missing cell for column \"s\""},
		{"duplicate member key", famJSON(`"columns":[{"name":"d","param":"dia","type":"length"}],
			"keyColumns":["d"],"members":[{"d":6},{"d":6}]`), "duplicate key"},
		{"bad units", strings.Replace(famJSON(`"columns":[{"name":"d","param":"dia","type":"length"}],
			"keyColumns":["d"],"members":[{"d":6}]`), `"units":"mm"`, `"units":"furlong"`, 1), "units \"furlong\" invalid"},
		{"non-numeric cell", famJSON(`"columns":[{"name":"d","param":"dia","type":"length"}],
			"keyColumns":["d"],"members":[{"d":"big"}]`), "is not a number"},
		{"empty members", famJSON(`"columns":[{"name":"d","param":"dia","type":"length"}],
			"keyColumns":["d"],"members":[]`), "no members"},
		{"no columns", famJSON(`"columns":[],"keyColumns":["d"],"members":[{"d":6}]`), "no columns"},
		{"empty column name", famJSON(`"columns":[{"name":"","param":"dia","type":"length"}],
			"keyColumns":["d"],"members":[{"d":6}]`), "empty name"},
		{"empty param", famJSON(`"columns":[{"name":"d","param":"","type":"length"}],
			"keyColumns":["d"],"members":[{"d":6}]`), "empty param"},
		{"invalid type", famJSON(`"columns":[{"name":"d","param":"dia","type":"weight"}],
			"keyColumns":["d"],"members":[{"d":6}]`), "invalid type \"weight\""},
		{"duplicate column name", famJSON(`"columns":[
			{"name":"d","param":"dia","type":"length"},{"name":"d","param":"two","type":"length"}],
			"keyColumns":["d"],"members":[{"d":6}]`), "duplicate column name \"d\""},
		{"text cell not text", famJSON(`"columns":[{"name":"g","param":"grade","type":"text"}],
			"keyColumns":["g"],"members":[{"g":8}]`), "is not a text value"},
		{"missing id", replaceOnce(famJSON(colBody), `"id":"t"`, `"id":""`), "missing id"},
		{"missing standard", replaceOnce(famJSON(colBody), `"standard":"ISO 1"`, `"standard":""`), "missing standard"},
		{"missing generator", replaceOnce(famJSON(colBody), `"generator":"g"`, `"generator":""`), "missing generator"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseFamily("test.json", []byte(tc.json))
			if err == nil {
				t.Fatalf("parseFamily accepted an invalid family; want error %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tc.want)
			}
		})
	}
}

// colBody is a minimal valid columns/keyColumns/members body, for header-only breakages.
const colBody = `"columns":[{"name":"d","param":"dia","type":"length"}],
	"keyColumns":["d"],"members":[{"d":6}]`

// famJSON wraps a columns/keyColumns/members body in the common valid header so each
// validation case only states what it breaks.
func famJSON(body string) string {
	return `{"id":"t","standard":"ISO 1","generator":"g","units":"mm",
		"category":"Fasteners/Bolts",` + body + `}`
}

// replaceOnce swaps the first occurrence of old for new, for header-field breakages.
func replaceOnce(s, old, new string) string { return strings.Replace(s, old, new, 1) }

// TestTextColumnRoundTrip covers a family with a text key column (e.g. a grade/designation):
// the cell lands in Labels, and the member key is built from the text value.
func TestTextColumnRoundTrip(t *testing.T) {
	fam, err := parseFamily("t.json", []byte(famJSON(`"columns":[
		{"name":"grade","param":"steel_grade","type":"text"},
		{"name":"d","param":"dia","type":"length"},
		{"name":"n","param":"count","type":"count"}],
		"keyColumns":["grade","d"],"members":[{"grade":"8.8","d":8,"n":6}]`)))
	if err != nil {
		t.Fatalf("parseFamily error = %v", err)
	}
	m := fam.Members[0]
	if m.Labels["grade"] != "8.8" {
		t.Errorf("grade label = %q, want 8.8", m.Labels["grade"])
	}
	if m.Values["d"] != 8 || m.Values["n"] != 6 {
		t.Errorf("values = %v, want d=8 n=6", m.Values)
	}
	if m.Key != "grade=8.8,d=8" {
		t.Errorf("member key = %q, want grade=8.8,d=8", m.Key)
	}
	if _, ok := fam.Member("nope"); ok {
		t.Error("Member(nope) resolved a nonexistent key")
	}
}

// TestFamiliesOrdered checks Families() returns every family in deterministic id order.
func TestFamiliesOrdered(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	got := ids(c.Families())
	if len(got) < 2 || got[0] != "din125-washer" {
		t.Errorf("Families() = %v, want id-sorted with din125-washer first", got)
	}
}
