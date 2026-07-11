// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"strconv"
	"strings"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// tableColumns is the member table's header row: each column's human-readable name in declared
// order (see columnHeader — the cryptic standards symbol "s" reads as "Across Flats").
func tableColumns(fam *catalog.Family) []string {
	if fam == nil {
		return nil
	}
	headers := make([]string, len(fam.Columns))
	for i, col := range fam.Columns {
		headers[i] = columnHeader(col)
	}
	return headers
}

// headerAbbrevExpansions expands terse tokens in a column's param name to full words for the table
// header, so a column reads "Outer Diameter" instead of "outer_dia". Only genuinely cryptic
// abbreviations belong here — a token not listed is just title-cased.
var headerAbbrevExpansions = map[string]string{
	"dia":  "Diameter",
	"dist": "Distance",
}

// columnHeader is the human-readable member-table header for a column: the descriptive Param name
// (which the generator already carries, e.g. "across_flats", "head_height") turned into
// "Across Flats" / "Head Height" — underscores to spaces, each word title-cased, known
// abbreviations expanded. It falls back to the terse standards symbol Name only when a column has
// no Param. This gives full-word headers in place of the cryptic single letters (d, s, k) WITHOUT
// touching Name/Param, which key the member cells, lookup, and geometry-driving parameters.
func columnHeader(col catalog.Column) string {
	if col.Param == "" {
		return col.Name
	}
	words := strings.Split(col.Param, "_")
	for i, w := range words {
		if full, ok := headerAbbrevExpansions[w]; ok {
			words[i] = full
			continue
		}
		words[i] = titleWord(w)
	}
	return strings.Join(words, " ")
}

// titleWord upper-cases the first byte of an ASCII param token ("flats" -> "Flats", "a" -> "A") and
// leaves the rest untouched, so an already-cased or numeric token is preserved. Param tokens are
// ASCII snake_case, so byte indexing is safe here (unlike the deprecated strings.Title).
func titleWord(w string) string {
	if w == "" {
		return ""
	}
	return strings.ToUpper(w[:1]) + w[1:]
}

// tableRows is one PanelTable row per family member — the full family table. Key is the member's
// canonical Key (what Place consumes); Cells are each column's value in header order (numeric
// columns formatted compactly, text columns via their label).
func tableRows(fam *catalog.Family) []wire.TableRow {
	if fam == nil {
		return nil
	}
	rows := make([]wire.TableRow, len(fam.Members))
	for i, m := range fam.Members {
		rows[i] = wire.TableRow{Key: m.Key, Cells: memberCells(fam, m)}
	}
	return rows
}

// memberCells formats one member's cells in the family's column order.
func memberCells(fam *catalog.Family, m catalog.Member) []string {
	cells := make([]string, len(fam.Columns))
	for i, col := range fam.Columns {
		cells[i], _ = memberCellValue(m, col.Name)
	}
	return cells
}

// memberCellValue formats one member's value for a column (by name): a numeric value compactly,
// else its text label, reporting whether the column is present on the member at all. It is the
// single source of member-cell formatting, shared by the members table (memberCells) and the
// compact size label (sizeLabel in placement.go).
func memberCellValue(m catalog.Member, colName string) (string, bool) {
	if v, ok := m.Values[colName]; ok {
		return strconv.FormatFloat(v, 'g', -1, 64), true
	}
	if s, ok := m.Labels[colName]; ok {
		return s, true
	}
	return "", false
}
