// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"reflect"
	"strings"
	"testing"

	"oblikovati.org/part-designer/designer/catalog"
)

// stubGen is a minimal PartGenerator for registry tests.
type stubGen struct{ kind string }

func (g stubGen) Kind() string                           { return g.kind }
func (stubGen) Build(*PartBuilder, ResolvedMember) error { return nil }

func TestRegistryRegisterGetKinds(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(stubGen{"alpha"}); err != nil {
		t.Fatalf("Register(alpha) error = %v", err)
	}
	if err := r.Register(stubGen{"beta"}); err != nil {
		t.Fatalf("Register(beta) error = %v", err)
	}
	if err := r.Register(stubGen{"alpha"}); err == nil {
		t.Error("Register accepted a duplicate kind; want an error")
	}
	if err := r.Register(stubGen{""}); err == nil {
		t.Error("Register accepted an empty kind; want an error")
	}
	if _, ok := r.Get("alpha"); !ok {
		t.Error("Get(alpha) missing after Register")
	}
	if _, ok := r.Get("nope"); ok {
		t.Error("Get(nope) resolved an unregistered kind")
	}
	if got := r.Kinds(); !reflect.DeepEqual(got, []string{"alpha", "beta"}) {
		t.Errorf("Kinds() = %v, want [alpha beta] (sorted)", got)
	}
}

func TestDefaultRegistryHasRoundBar(t *testing.T) {
	r := DefaultRegistry()
	g, ok := r.Get("round_bar")
	if !ok {
		t.Fatal("DefaultRegistry missing round_bar")
	}
	if g.Kind() != "round_bar" {
		t.Errorf("kind = %q, want round_bar", g.Kind())
	}
}

func TestResolvedMemberValue(t *testing.T) {
	rm := roundBarMember(8, 40)
	if rm.Value("d") != 8 || rm.Value("l") != 40 {
		t.Errorf("Value = (%v,%v), want (8,40)", rm.Value("d"), rm.Value("l"))
	}
	if rm.Value("absent") != 0 {
		t.Errorf("Value(absent) = %v, want 0", rm.Value("absent"))
	}
}

// TestBuilderAccessors covers API() and a sketch handle's Index().
func TestBuilderAccessors(t *testing.T) {
	b := newBuilder(&fakeHost{}, catalog.UnitsMillimetre)
	if b.API() == nil {
		t.Fatal("API() = nil")
	}
	sk, err := b.Sketch("XY")
	if err != nil {
		t.Fatalf("Sketch error = %v", err)
	}
	if sk.Index() != 1 {
		t.Errorf("sketch index = %d, want 1", sk.Index())
	}
}

// TestGroundedCircleNoCentre exercises the guard that catches a circle the host returned
// with no centre point — pinning it would panic on an empty PointIDs slice.
func TestGroundedCircleNoCentre(t *testing.T) {
	err := (RoundBar{}).Build(newBuilder(&fakeHost{noPoints: true}, catalog.UnitsMillimetre), roundBarMember(8, 40))
	if err == nil || !strings.Contains(err.Error(), "no centre point") {
		t.Fatalf("Build error = %v, want it to mention the missing centre point", err)
	}
}

// TestPublishParamsListError propagates a host failure while listing parameters.
func TestPublishParamsListError(t *testing.T) {
	h := &fakeHost{failMethod: "parameters.list"}
	if err := newBuilder(h, catalog.UnitsMillimetre).PublishParams(roundBarMember(8, 40)); err == nil {
		t.Fatal("PublishParams ignored a host list failure; want an error")
	}
}
