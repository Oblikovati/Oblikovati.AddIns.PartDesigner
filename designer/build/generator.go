// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"fmt"
	"sort"

	"oblikovati.org/part-designer/designer/catalog"
)

// ResolvedMember is a catalogue member together with its family, handed to a generator. It
// is the single input a generator turns into a part: the family gives the column→parameter
// mapping and units, the member gives the concrete sizes.
type ResolvedMember struct {
	Family *catalog.Family
	Member catalog.Member
}

// Value returns a numeric cell of the member by column name (0 if absent).
func (rm ResolvedMember) Value(column string) float64 { return rm.Member.Values[column] }

// PartGenerator builds one kind of standard part procedurally. Build assumes an active part
// document (the placement service creates it) and drives the PartBuilder to publish the
// member's parameters and realize a DOF-0 parametric part. Kind is the stable string a
// family's `generator` field binds to.
type PartGenerator interface {
	Kind() string
	Build(b *PartBuilder, rm ResolvedMember) error
}

// Registry maps a generator kind to its implementation, so placing a family looks up its
// generator by the family's `generator` string.
type Registry struct {
	gens map[string]PartGenerator
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry { return &Registry{gens: map[string]PartGenerator{}} }

// Register adds a generator, erroring on a duplicate kind so two generators can't silently
// claim the same family binding.
func (r *Registry) Register(g PartGenerator) error {
	kind := g.Kind()
	if kind == "" {
		return fmt.Errorf("generator has empty kind")
	}
	if _, dup := r.gens[kind]; dup {
		return fmt.Errorf("generator kind %q already registered", kind)
	}
	r.gens[kind] = g
	return nil
}

// Get returns the generator for a kind.
func (r *Registry) Get(kind string) (PartGenerator, bool) {
	g, ok := r.gens[kind]
	return g, ok
}

// Kinds lists the registered generator kinds, sorted.
func (r *Registry) Kinds() []string {
	out := make([]string, 0, len(r.gens))
	for k := range r.gens {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// DefaultRegistry returns a registry with every built-in generator registered. Category
// PBIs (B/C/D/E) add their generators here as they land.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	mustRegister(r, RoundBar{})
	mustRegister(r, HexBolt{})
	mustRegister(r, SocketScrew{})
	mustRegister(r, HexNut{})
	mustRegister(r, Washer{})
	return r
}

// mustRegister panics on a duplicate-kind programming error at startup (the built-in set is
// fixed at compile time, so a collision is a bug, not a runtime condition).
func mustRegister(r *Registry, g PartGenerator) {
	if err := r.Register(g); err != nil {
		panic("build: " + err.Error())
	}
}
