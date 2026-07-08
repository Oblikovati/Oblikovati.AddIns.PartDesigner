// SPDX-License-Identifier: GPL-2.0-only

// Package catalog is the standards data library of the Part Designer add-in: the
// procedural analogue of Inventor's Content Center family tables. A Family is a
// per-standard dimension table (e.g. ISO 4017 hex bolts); a Member is one row of that
// table (one size). The package is pure data — it has no host dependency — so a generator
// maps a resolved Member's columns onto published parameters and the add-in stays testable
// on every OS.
package catalog

import (
	"fmt"
	"strconv"
	"strings"
)

// Units is the length unit a family's numeric columns are expressed in.
type Units string

const (
	// UnitsMillimetre is the metric unit used by ISO/DIN families.
	UnitsMillimetre Units = "mm"
	// UnitsInch is the imperial unit used by ANSI inch families.
	UnitsInch Units = "in"
)

// valid reports whether u is a unit the loader accepts.
func (u Units) valid() bool { return u == UnitsMillimetre || u == UnitsInch }

// ColumnType classifies a family-table column so the loader knows how to parse a cell and a
// generator knows what parameter kind to publish.
type ColumnType string

const (
	// ColumnLength is a linear dimension in the family's Units.
	ColumnLength ColumnType = "length"
	// ColumnAngle is an angle in degrees.
	ColumnAngle ColumnType = "angle"
	// ColumnCount is a dimensionless integer count (e.g. number of balls).
	ColumnCount ColumnType = "count"
	// ColumnText is a non-numeric label (e.g. a designation or standard code).
	ColumnText ColumnType = "text"
)

// valid reports whether t is a column type the loader accepts.
func (t ColumnType) valid() bool {
	switch t {
	case ColumnLength, ColumnAngle, ColumnCount, ColumnText:
		return true
	default:
		return false
	}
}

// Numeric reports whether cells of this column carry a number (vs a text label). Generators
// publish numeric columns as dimensional parameters and skip text columns.
func (t ColumnType) Numeric() bool { return t != ColumnText }

// Column maps one family-table column to the parameter a generator drives from it. Name is
// the cell key in the member rows; Param is the published parameter name the generator's
// geometry dimensions reference; Type fixes how the cell is parsed.
type Column struct {
	Name  string     `json:"name"`
	Param string     `json:"param"`
	Type  ColumnType `json:"type"`
}

// CategoryPath is a family's place in the browse tree, root-first (e.g.
// {"Fasteners","Bolts","Hex Head"}). It mirrors Inventor's colon-delimited category path,
// stored here as segments so tree building and prefix filtering are straightforward.
type CategoryPath []string

// ParseCategoryPath splits a "/"-delimited category string into trimmed segments, rejecting
// empty input or any blank segment (a malformed path like "Fasteners//Bolts").
func ParseCategoryPath(s string) (CategoryPath, error) {
	parts := strings.Split(s, "/")
	path := make(CategoryPath, 0, len(parts))
	for _, p := range parts {
		seg := strings.TrimSpace(p)
		if seg == "" {
			return nil, fmt.Errorf("category %q: empty segment; want non-blank %q-separated segments", s, "/")
		}
		path = append(path, seg)
	}
	if len(path) == 0 {
		return nil, fmt.Errorf("category is empty; want at least one segment")
	}
	return path, nil
}

// String renders the path back to its "/"-delimited form.
func (c CategoryPath) String() string { return strings.Join(c, "/") }

// HasPrefix reports whether c starts with every segment of p (an empty p matches all).
func (c CategoryPath) HasPrefix(p CategoryPath) bool {
	if len(p) > len(c) {
		return false
	}
	for i := range p {
		if c[i] != p[i] {
			return false
		}
	}
	return true
}

// Member is one row of a family table — one concrete size. Values holds the numeric cells
// (length/angle/count) keyed by column name; Labels holds text cells. Key is the canonical,
// deterministic identifier built from the family's key columns (used for lookup + the
// attribute stamp a placed part carries).
type Member struct {
	Key    string
	Values map[string]float64
	Labels map[string]string
}

// Family is a per-standard dimension table bound to a procedural generator. Placing a
// member means: publish its columns as parameters, then run Generator to build the DOF-0
// part.
type Family struct {
	ID         string
	Category   CategoryPath
	Standard   string
	Generator  string
	Units      Units
	KeyColumns []string
	Columns    []Column
	Members    []Member
}

// Body is the standards body a family belongs to — the leading token of Standard uppercased
// (e.g. "ISO 4017" → "ISO", "DIN 934" → "DIN"). Used to filter the catalogue by standard.
func (f *Family) Body() string {
	fields := strings.Fields(f.Standard)
	if len(fields) == 0 {
		return ""
	}
	return strings.ToUpper(fields[0])
}

// Member resolves one member of the family by its canonical Key.
func (f *Family) Member(key string) (Member, bool) {
	for _, m := range f.Members {
		if m.Key == key {
			return m, true
		}
	}
	return Member{}, false
}

// memberKey builds the canonical key for a member from the family's key columns, so the same
// size always maps to the same string (for lookup and the placed-part attribute stamp). It
// is stable and unambiguous — "d=8,l=30" — not a display label (the panel derives those).
func (f *Family) memberKey(values map[string]float64, labels map[string]string) string {
	parts := make([]string, 0, len(f.KeyColumns))
	for _, name := range f.KeyColumns {
		if v, ok := values[name]; ok {
			parts = append(parts, name+"="+strconv.FormatFloat(v, 'g', -1, 64))
			continue
		}
		parts = append(parts, name+"="+labels[name])
	}
	return strings.Join(parts, ",")
}
