// SPDX-License-Identifier: GPL-2.0-only

package catalog

import "testing"

// TestSearchByFamilyNameAndSize checks the free-text search over family name and size: a standard
// token, a category token and a member designation each find the expected families, and an empty
// query returns everything.
func TestSearchByFamilyNameAndSize(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	cases := []struct {
		query   string
		wantID  string
		wantHit bool
	}{
		{"hex", "iso4017-hex-bolt", true},                   // category segment "Hex Head"
		{"ISO 15", "iso15-deep-groove-ball-bearing", true},  // standard
		{"6205", "iso15-deep-groove-ball-bearing", true},    // a ball-bearing designation (size)
		{"NU205", "iso15-cylindrical-roller-bearing", true}, // a roller designation (size)
		{"circlip", "din471-external-circlip", true},        // family id token
		{"zzz-no-such-part", "", false},
	}
	for _, tc := range cases {
		got := c.Search(tc.query)
		if !tc.wantHit {
			if len(got) != 0 {
				t.Errorf("Search(%q) = %v, want no matches", tc.query, ids(got))
			}
			continue
		}
		if !containsID(got, tc.wantID) {
			t.Errorf("Search(%q) = %v, want to contain %s", tc.query, ids(got), tc.wantID)
		}
	}
	if got := c.Search("   "); len(got) != c.Len() {
		t.Errorf("blank search matched %d, want all %d", len(got), c.Len())
	}
}

// TestSearchIsCaseInsensitive checks a lower-cased query finds an upper-cased designation.
func TestSearchIsCaseInsensitive(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !containsID(c.Search("nu205"), "iso15-cylindrical-roller-bearing") {
		t.Errorf("case-insensitive search for nu205 missed the roller bearing")
	}
}
