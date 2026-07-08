// SPDX-License-Identifier: GPL-2.0-only

package catalog

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// dataFS embeds the curated standards tables. Each family is one JSON file under data/,
// grouped by category folder (fasteners/, structural/, shaft/, bearings/).
//
//go:embed data
var dataFS embed.FS

// Catalog is the loaded, validated set of families, addressable by id and ordered for a
// stable UI. Build one with Load.
type Catalog struct {
	families map[string]*Family
	order    []string // family ids in load order (sorted), for deterministic listing
}

// familyFile is the on-disk JSON shape of a family. Members carry raw cells so each is
// routed by its column type into Member.Values / Member.Labels during validation.
type familyFile struct {
	ID         string                       `json:"id"`
	Category   string                       `json:"category"`
	Standard   string                       `json:"standard"`
	Generator  string                       `json:"generator"`
	Units      string                       `json:"units"`
	KeyColumns []string                     `json:"keyColumns"`
	Columns    []Column                     `json:"columns"`
	Members    []map[string]json.RawMessage `json:"members"`
}

// Load parses and validates every embedded family table into an in-memory Catalog. It fails
// on the first malformed or inconsistent family so a bad table never reaches a generator.
func Load() (*Catalog, error) {
	c := &Catalog{families: map[string]*Family{}}
	files, err := dataFiles()
	if err != nil {
		return nil, err
	}
	for _, name := range files {
		raw, readErr := dataFS.ReadFile(name)
		if readErr != nil {
			return nil, fmt.Errorf("read %q: %w", name, readErr)
		}
		fam, parseErr := parseFamily(name, raw)
		if parseErr != nil {
			return nil, parseErr
		}
		if _, dup := c.families[fam.ID]; dup {
			return nil, fmt.Errorf("duplicate family id %q (in %q)", fam.ID, name)
		}
		c.families[fam.ID] = fam
		c.order = append(c.order, fam.ID)
	}
	sort.Strings(c.order)
	return c, nil
}

// dataFiles lists every *.json under the embedded data tree, sorted for determinism.
func dataFiles() ([]string, error) {
	var names []string
	err := fs.WalkDir(dataFS, "data", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() && strings.HasSuffix(p, ".json") {
			names = append(names, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk embedded data: %w", err)
	}
	sort.Strings(names)
	return names, nil
}

// parseFamily decodes and validates one family file into a Family.
func parseFamily(name string, raw []byte) (*Family, error) {
	var ff familyFile
	if err := json.Unmarshal(raw, &ff); err != nil {
		return nil, fmt.Errorf("family %q: invalid JSON: %w", name, err)
	}
	fam, err := ff.build()
	if err != nil {
		return nil, fmt.Errorf("family %q (%s): %w", path.Base(name), ff.ID, err)
	}
	return fam, nil
}

// build turns a validated familyFile into a Family, or returns the first validation error.
func (ff *familyFile) build() (*Family, error) {
	if err := ff.validateHeader(); err != nil {
		return nil, err
	}
	if err := ff.validateColumns(); err != nil {
		return nil, err
	}
	category, err := ParseCategoryPath(ff.Category)
	if err != nil {
		return nil, err
	}
	fam := &Family{
		ID: ff.ID, Category: category, Standard: ff.Standard, Generator: ff.Generator,
		Units: Units(ff.Units), KeyColumns: ff.KeyColumns, Columns: ff.Columns,
	}
	if err := fam.buildMembers(ff.Members); err != nil {
		return nil, err
	}
	return fam, nil
}

// validateHeader checks the scalar family fields.
func (ff *familyFile) validateHeader() error {
	switch {
	case strings.TrimSpace(ff.ID) == "":
		return fmt.Errorf("missing id")
	case strings.TrimSpace(ff.Standard) == "":
		return fmt.Errorf("missing standard")
	case strings.TrimSpace(ff.Generator) == "":
		return fmt.Errorf("missing generator")
	case !Units(ff.Units).valid():
		return fmt.Errorf("units %q invalid; want %q or %q", ff.Units, UnitsMillimetre, UnitsInch)
	case len(ff.Members) == 0:
		return fmt.Errorf("no members; a family table needs at least one size")
	}
	return nil
}

// validateColumns checks the column list, key columns, and the uniqueness the loader
// guarantees: no duplicate column names, no duplicate published parameter names, and every
// key column referencing a real column.
func (ff *familyFile) validateColumns() error {
	if len(ff.Columns) == 0 {
		return fmt.Errorf("no columns")
	}
	names, params := map[string]bool{}, map[string]bool{}
	for _, c := range ff.Columns {
		if err := validateColumn(c, names, params); err != nil {
			return err
		}
		names[c.Name], params[c.Param] = true, true
	}
	if len(ff.KeyColumns) == 0 {
		return fmt.Errorf("no key columns; at least one column must identify a member")
	}
	for _, kc := range ff.KeyColumns {
		if !names[kc] {
			return fmt.Errorf("key column %q is not a declared column", kc)
		}
	}
	return nil
}

// validateColumn checks one column's fields and its uniqueness against those already seen.
func validateColumn(c Column, names, params map[string]bool) error {
	switch {
	case strings.TrimSpace(c.Name) == "":
		return fmt.Errorf("column with empty name")
	case strings.TrimSpace(c.Param) == "":
		return fmt.Errorf("column %q has empty param", c.Name)
	case !c.Type.valid():
		return fmt.Errorf("column %q has invalid type %q; want length|angle|count|text", c.Name, c.Type)
	case names[c.Name]:
		return fmt.Errorf("duplicate column name %q", c.Name)
	case params[c.Param]:
		return fmt.Errorf("duplicate param name %q (columns must map to distinct parameters)", c.Param)
	}
	return nil
}

// buildMembers routes every member's raw cells into typed Values/Labels, requiring each
// declared column to be present and correctly typed, and rejecting duplicate member keys.
func (f *Family) buildMembers(rows []map[string]json.RawMessage) error {
	seen := map[string]bool{}
	for i, row := range rows {
		m, err := f.buildMember(row)
		if err != nil {
			return fmt.Errorf("member %d: %w", i, err)
		}
		if seen[m.Key] {
			return fmt.Errorf("member %d: duplicate key %q", i, m.Key)
		}
		seen[m.Key] = true
		f.Members = append(f.Members, m)
	}
	return nil
}

// buildMember parses one member row against the column schema.
func (f *Family) buildMember(row map[string]json.RawMessage) (Member, error) {
	values, labels := map[string]float64{}, map[string]string{}
	for _, col := range f.Columns {
		cell, ok := row[col.Name]
		if !ok {
			return Member{}, fmt.Errorf("missing cell for column %q", col.Name)
		}
		if err := assignCell(col, cell, values, labels); err != nil {
			return Member{}, err
		}
	}
	return Member{Key: f.memberKey(values, labels), Values: values, Labels: labels}, nil
}

// assignCell parses one raw cell into the right typed map per its column type.
func assignCell(col Column, cell json.RawMessage, values map[string]float64, labels map[string]string) error {
	if col.Type.Numeric() {
		var v float64
		if err := json.Unmarshal(cell, &v); err != nil {
			return fmt.Errorf("column %q: cell %s is not a number", col.Name, string(cell))
		}
		values[col.Name] = v
		return nil
	}
	var s string
	if err := json.Unmarshal(cell, &s); err != nil {
		return fmt.Errorf("column %q: cell %s is not a text value", col.Name, string(cell))
	}
	labels[col.Name] = s
	return nil
}
