// SPDX-License-Identifier: GPL-2.0-only

package catalog

import (
	"sort"
	"strings"
)

// Families returns every family in deterministic (id-sorted) order.
func (c *Catalog) Families() []*Family {
	out := make([]*Family, 0, len(c.order))
	for _, id := range c.order {
		out = append(out, c.families[id])
	}
	return out
}

// Family returns the family with the given id.
func (c *Catalog) Family(id string) (*Family, bool) {
	f, ok := c.families[id]
	return f, ok
}

// Len is the number of loaded families.
func (c *Catalog) Len() int { return len(c.order) }

// Standards lists the distinct standards bodies present (e.g. ISO, DIN, ANSI), sorted — the
// options for the panel's standard filter.
func (c *Catalog) Standards() []string {
	seen := map[string]bool{}
	var bodies []string
	for _, id := range c.order {
		if b := c.families[id].Body(); b != "" && !seen[b] {
			seen[b] = true
			bodies = append(bodies, b)
		}
	}
	sort.Strings(bodies)
	return bodies
}

// ByStandardBody returns the families of one standards body (case-insensitive, e.g. "iso"),
// in id order.
func (c *Catalog) ByStandardBody(body string) []*Family {
	want := strings.ToUpper(strings.TrimSpace(body))
	var out []*Family
	for _, id := range c.order {
		if f := c.families[id]; f.Body() == want {
			out = append(out, f)
		}
	}
	return out
}

// Search returns the families matching a free-text query (case-insensitive; an empty query
// returns all), in id order — backing the panel's quick search over family name and size.
func (c *Catalog) Search(query string) []*Family {
	q := strings.ToLower(strings.TrimSpace(query))
	var out []*Family
	for _, id := range c.order {
		if f := c.families[id]; f.Matches(q) {
			out = append(out, f)
		}
	}
	return out
}

// Matches reports whether the family answers a lower-cased search query. An empty query always
// matches. It searches the family's name components (standard + every category segment), its id,
// and each member's size (key columns and text labels), so a query like "hex", "6205" or "ipe"
// finds the family by name or by any of its sizes.
func (f *Family) Matches(loweredQuery string) bool {
	if loweredQuery == "" {
		return true
	}
	if containsFold(f.Standard, loweredQuery) || containsFold(f.ID, loweredQuery) {
		return true
	}
	for _, seg := range f.Category {
		if containsFold(seg, loweredQuery) {
			return true
		}
	}
	return f.matchesMember(loweredQuery)
}

// matchesMember reports whether any member's key or text labels contain the query.
func (f *Family) matchesMember(loweredQuery string) bool {
	for _, m := range f.Members {
		if containsFold(m.Key, loweredQuery) {
			return true
		}
		for _, label := range m.Labels {
			if containsFold(label, loweredQuery) {
				return true
			}
		}
	}
	return false
}

// containsFold reports whether s contains the already-lower-cased substring q.
func containsFold(s, q string) bool {
	return strings.Contains(strings.ToLower(s), q)
}

// ByCategory returns the families whose category is at or under prefix (an empty prefix
// returns all), in id order — backing the tree's subtree filtering.
func (c *Catalog) ByCategory(prefix CategoryPath) []*Family {
	var out []*Family
	for _, id := range c.order {
		if f := c.families[id]; f.Category.HasPrefix(prefix) {
			out = append(out, f)
		}
	}
	return out
}
