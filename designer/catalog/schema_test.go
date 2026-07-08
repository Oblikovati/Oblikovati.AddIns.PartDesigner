// SPDX-License-Identifier: GPL-2.0-only

package catalog

import (
	"reflect"
	"testing"
)

func TestParseCategoryPath(t *testing.T) {
	got, err := ParseCategoryPath("Fasteners / Bolts / Hex Head")
	if err != nil {
		t.Fatalf("ParseCategoryPath error = %v", err)
	}
	if !reflect.DeepEqual(got, CategoryPath{"Fasteners", "Bolts", "Hex Head"}) {
		t.Errorf("path = %v, want [Fasteners Bolts Hex Head] (segments trimmed)", got)
	}
	for _, bad := range []string{"", "Fasteners//Bolts", "  ", "Bolts/ /Hex"} {
		if _, err := ParseCategoryPath(bad); err == nil {
			t.Errorf("ParseCategoryPath(%q) accepted a malformed path", bad)
		}
	}
}

func TestCategoryHasPrefix(t *testing.T) {
	p := CategoryPath{"Fasteners", "Bolts", "Hex Head"}
	for _, tc := range []struct {
		prefix CategoryPath
		want   bool
	}{
		{CategoryPath{"Fasteners"}, true},
		{CategoryPath{"Fasteners", "Bolts"}, true},
		{p, true},
		{nil, true},
		{CategoryPath{"Fasteners", "Nuts"}, false},
		{CategoryPath{"Fasteners", "Bolts", "Hex Head", "Extra"}, false},
	} {
		if got := p.HasPrefix(tc.prefix); got != tc.want {
			t.Errorf("%v.HasPrefix(%v) = %v, want %v", p, tc.prefix, got, tc.want)
		}
	}
}

func TestFamilyBody(t *testing.T) {
	for _, tc := range []struct{ standard, want string }{
		{"ISO 4017", "ISO"},
		{"DIN 934", "DIN"},
		{"ANSI B18.2.1", "ANSI"},
		{"", ""},
	} {
		f := &Family{Standard: tc.standard}
		if got := f.Body(); got != tc.want {
			t.Errorf("Body(%q) = %q, want %q", tc.standard, got, tc.want)
		}
	}
}

func TestColumnAndUnitsValidity(t *testing.T) {
	if !UnitsMillimetre.valid() || !UnitsInch.valid() || Units("furlong").valid() {
		t.Error("Units.valid() disagrees with the accepted set")
	}
	for _, ct := range []ColumnType{ColumnLength, ColumnAngle, ColumnCount, ColumnText} {
		if !ct.valid() {
			t.Errorf("%q should be valid", ct)
		}
	}
	if ColumnType("weight").valid() {
		t.Error("unknown column type reported valid")
	}
	if ColumnText.Numeric() || !ColumnLength.Numeric() {
		t.Error("numeric() misclassifies a column type")
	}
}
