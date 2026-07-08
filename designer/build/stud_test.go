// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"strings"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// studMember builds a synthetic resolved member with the stud columns the generator drives. The
// continuous variant (DIN 976 rod) needs only d/P/length; the double_end variant (DIN 939) adds
// the metal- and nut-end thread lengths.
func studMember(variant string, d, p, l, b1, b2 float64) ResolvedMember {
	cols := []catalog.Column{
		{Name: "d", Param: "nominal_dia", Type: catalog.ColumnLength},
		{Name: "P", Param: "thread_pitch", Type: catalog.ColumnLength},
		{Name: "l", Param: "length", Type: catalog.ColumnLength},
	}
	values := map[string]float64{"d": d, "P": p, "l": l}
	if variant == "double_end" {
		cols = append(cols,
			catalog.Column{Name: "b1", Param: "metal_thread_length", Type: catalog.ColumnLength},
			catalog.Column{Name: "b2", Param: "nut_thread_length", Type: catalog.ColumnLength})
		values["b1"] = b1
		values["b2"] = b2
	}
	fam := &catalog.Family{
		ID: "t-stud", Generator: "stud", Variant: variant, Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"d", "l"}, Columns: cols,
		Members: []catalog.Member{{Key: "d=8,l=50", Values: values}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestStudContinuousBuildsThreadedRod is the DIN 976 acceptance check: a single nominal_dia
// cylinder extruded to length, threaded over its whole (sole) cylindrical face, every dimension
// parameter-driven.
func TestStudContinuousBuildsThreadedRod(t *testing.T) {
	h := &fakeHost{dof: 0} // default reference keys report exactly one (shank) cylinder
	rm := studMember("continuous", 8, 1.25, 1000, 0, 0)
	if err := (Stud{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	assertParam(t, h.added, "nominal_dia", "8 mm")
	assertParam(t, h.added, "thread_pitch", "1.25 mm")
	assertParam(t, h.added, "length", "1000 mm")

	if len(h.extrudes) != 1 || h.extrudes[0].Distance != "length" || h.extrudes[0].Operation != "new" {
		t.Errorf("extrudes = %+v, want one length/new cylinder", h.extrudes)
	}
	// One full-length cosmetic thread over the sole cylinder: no span limits.
	if len(h.threads) != 1 || h.threads[0].Designation != "M8x1.25" || h.threads[0].Cut {
		t.Fatalf("threads = %+v, want one cosmetic M8x1.25 over the sole cylinder", h.threads)
	}
	if h.threads[0].FaceRef != "shank-cyl" || h.threads[0].Offset != "" || h.threads[0].Length != "" {
		t.Errorf("thread = %+v, want the sole cylinder shank-cyl, full length (no offset/length)", h.threads[0])
	}
}

// TestStudDoubleEndThreadsBothEnds covers DIN 939: a single analytic-cylinder rod whose ONE face
// carries two partial-length cosmetic threads — the metal end from the base and the nut end at the
// far end — with the shank between left bare. No boolean join (which would re-facet the cylinder).
func TestStudDoubleEndThreadsBothEnds(t *testing.T) {
	h := &fakeHost{dof: 0} // one analytic cylinder, exactly what CylinderFaceKey wants
	rm := studMember("double_end", 8, 1.25, 50, 10, 22)
	if err := (Stud{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	assertParam(t, h.added, "metal_thread_length", "10 mm")
	assertParam(t, h.added, "nut_thread_length", "22 mm")

	// One extrude — the whole rod, a single solid so its cylinder stays analytic.
	if len(h.extrudes) != 1 || h.extrudes[0].Distance != "length" || h.extrudes[0].Operation != "new" {
		t.Fatalf("extrudes = %+v, want one length/new rod (no segments to join)", h.extrudes)
	}
	if len(h.workPlanes) != 0 {
		t.Errorf("workPlanes = %d, want 0 (the stud is one extrude, ends set by thread spans)", len(h.workPlanes))
	}

	// Two cosmetic threads on the SAME face: metal end (offset 0, run metal_thread_length) and nut
	// end (offset nutEndOffset, run nut_thread_length), leaving the shank bare.
	if len(h.threads) != 2 {
		t.Fatalf("threads = %d, want 2 (metal + nut ends)", len(h.threads))
	}
	if h.threads[0].FaceRef != "shank-cyl" || h.threads[0].Offset != "" || h.threads[0].Length != "metal_thread_length" {
		t.Errorf("metal-end thread = %+v, want face shank-cyl, offset \"\", length metal_thread_length", h.threads[0])
	}
	if h.threads[1].FaceRef != "shank-cyl" || h.threads[1].Offset != nutEndOffset || h.threads[1].Length != "nut_thread_length" {
		t.Errorf("nut-end thread = %+v, want face shank-cyl, offset %q, length nut_thread_length", h.threads[1], nutEndOffset)
	}
	for _, th := range h.threads {
		if th.Designation != "M8x1.25" || th.Cut {
			t.Errorf("thread = %+v, want cosmetic M8x1.25", th)
		}
	}
}

// TestStudDoubleEndNeedsAnalyticCylinder guards the boolean-faceting failure mode: if the rod's
// cylinder is missing (no analytic face), CylinderFaceKey fails loudly rather than leaving the stud
// unthreaded.
func TestStudDoubleEndNeedsAnalyticCylinder(t *testing.T) {
	h := &fakeHost{dof: 0, noCylinder: true}
	rm := studMember("double_end", 8, 1.25, 50, 10, 22)
	err := (Stud{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm)
	if err == nil {
		t.Fatal("Build accepted a rod with no analytic cylinder; want an error")
	}
	if len(h.threads) != 0 {
		t.Errorf("threads = %d, want 0 when the cylinder face could not be resolved", len(h.threads))
	}
}

// TestStudRejectsUnknownVariant ensures a mis-typed variant fails loudly rather than silently
// picking a form.
func TestStudRejectsUnknownVariant(t *testing.T) {
	h := &fakeHost{dof: 0}
	err := (Stud{}).Build(newBuilder(h, catalog.UnitsMillimetre), studMember("tapered", 8, 1.25, 50, 10, 22))
	if err == nil || !strings.Contains(err.Error(), "variant") {
		t.Fatalf("Build error = %v, want it to reject the unknown variant", err)
	}
	if len(h.threads) != 0 {
		t.Errorf("threads = %d, want 0 when the variant is rejected", len(h.threads))
	}
}

// TestStudUnderConstrainedFails ensures a non-zero DOF aborts the build before any solid is made.
func TestStudUnderConstrainedFails(t *testing.T) {
	for _, variant := range []string{"continuous", "double_end"} {
		h := &fakeHost{dof: 2}
		err := (Stud{}).Build(newBuilder(h, catalog.UnitsMillimetre), studMember(variant, 8, 1.25, 50, 10, 22))
		if err == nil {
			t.Errorf("variant %s: Build accepted an under-constrained profile; want an error", variant)
		}
	}
}

// TestStudBuildErrorsPropagate injects a host failure at each wire method both build paths use and
// asserts the error surfaces rather than a half-built part.
func TestStudBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodParametersList, wire.MethodSketchCreate, wire.MethodSketchAddEntity,
		wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodModelReferenceKeys, wire.MethodFeaturesAdd,
	}
	for _, variant := range []string{"continuous", "double_end"} {
		for _, m := range methods {
			h := &fakeHost{dof: 0, failMethod: m}
			rm := studMember(variant, 8, 1.25, 50, 10, 22)
			if err := (Stud{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm); err == nil {
				t.Errorf("variant %s, failMethod %q: Build succeeded, want an error", variant, m)
			}
		}
	}
}

// TestDefaultRegistryHasStud checks the generator is wired into the built-in set so a stud family
// resolves at placement.
func TestDefaultRegistryHasStud(t *testing.T) {
	g, ok := DefaultRegistry().Get("stud")
	if !ok || g.Kind() != "stud" {
		t.Fatalf("DefaultRegistry stud = (%v,%v), want the Stud generator", g, ok)
	}
}
