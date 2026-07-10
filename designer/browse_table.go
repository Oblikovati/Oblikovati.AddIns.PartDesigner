// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"strconv"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// tableColumns is the family's column names in declared order (the member table's header).
func tableColumns(fam *catalog.Family) []string {
	if fam == nil {
		return nil
	}
	names := make([]string, len(fam.Columns))
	for i, col := range fam.Columns {
		names[i] = col.Name
	}
	return names
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
