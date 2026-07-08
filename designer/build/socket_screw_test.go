// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"strings"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/part-designer/designer/catalog"
)

// socketMember builds a synthetic resolved socket-screw member with the columns the SocketScrew
// generator drives geometry against, tagged with the head-style variant.
func socketMember(variant string, d, pitch, dk, k, s, t, l, b float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-socket", Generator: "socket_screw", Variant: variant, Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"d", "l"},
		Columns: []catalog.Column{
			{Name: "d", Param: "nominal_dia", Type: catalog.ColumnLength},
			{Name: "P", Param: "thread_pitch", Type: catalog.ColumnLength},
			{Name: "dk", Param: "head_dia", Type: catalog.ColumnLength},
			{Name: "k", Param: "head_height", Type: catalog.ColumnLength},
			{Name: "s", Param: "socket_across_flats", Type: catalog.ColumnLength},
			{Name: "t", Param: "socket_depth", Type: catalog.ColumnLength},
			{Name: "l", Param: "length", Type: catalog.ColumnLength},
			{Name: "b", Param: "thread_length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{Key: "d=8,l=40", Values: map[string]float64{
			"d": d, "P": pitch, "dk": dk, "k": k, "s": s, "t": t, "l": l, "b": b,
		}}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestSocketScrewBuildsCylindricalCapScrew is the B2 acceptance check for ISO 4762 / DIN 912: a
// cylindrical head (extruded down, untapered) + a joined shank + a blind hex socket cut into the
// top + a cosmetic thread, every dimension bound to a published parameter. The head is itself a
// cylinder, so the thread must target the *deepest* cylinder (the shank), not the head.
func TestSocketScrewBuildsCylindricalCapScrew(t *testing.T) {
	h := &fakeHost{dof: 0, headCylinder: true}
	rm := socketMember("cylindrical", 8, 1.25, 13, 8, 6, 4, 40, 40)
	if err := (SocketScrew{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	assertParam(t, h.added, "head_dia", "13 mm")
	assertParam(t, h.added, "head_height", "8 mm")
	assertParam(t, h.added, "nominal_dia", "8 mm")
	assertParam(t, h.added, "socket_across_flats", "6 mm")
	assertParam(t, h.added, "socket_depth", "4 mm")

	// head circle fixed, then the socket hexagon fixed + rotation-pinned, then the shank circle
	// (on the head-underside offset plane) fixed.
	wantCon := []string{"fix", "fix", "vertical", "fix"}
	if strings.Join(h.constraints, ",") != strings.Join(wantCon, ",") {
		t.Errorf("constraints = %v, want %v", h.constraints, wantCon)
	}
	assertSocketDimensions(t, h.dimensions)
	assertCylindricalExtrudes(t, h.extrudes)

	// The shank is built on a work plane offset from XY by −head_height (the head underside),
	// created hidden so the construction datum does not clutter the placed part.
	if len(h.workPlanes) != 1 || h.workPlanes[0].Offset != "-head_height" ||
		h.workPlanes[0].Kind != "plane-offset" ||
		h.workPlanes[0].Visible == nil || *h.workPlanes[0].Visible {
		t.Errorf("work planes = %+v, want one hidden XY-offset plane at -head_height", h.workPlanes)
	}
	if len(h.threads) != 1 || h.threads[0].FaceRef != "shank-cyl" ||
		h.threads[0].Designation != "M8x1.25" || h.threads[0].Cut {
		t.Errorf("threads = %+v, want one cosmetic M8x1.25 on the deepest (shank) cylinder", h.threads)
	}
}

// assertSocketDimensions checks all three size dimensions are parameter expressions (never
// literals), in build order: the head diameter, the socket across-corners span, the shank
// diameter.
func assertSocketDimensions(t *testing.T, dims []wire.AddDimensionArgs) {
	t.Helper()
	if len(dims) != 3 ||
		dims[0].Kind != "diameter" || dims[0].Expression != "head_dia" ||
		dims[1].Kind != "distance" || dims[1].Expression != "(socket_across_flats) / cos(30 deg)" ||
		dims[2].Kind != "diameter" || dims[2].Expression != "nominal_dia" {
		t.Errorf("dimensions = %+v, want head_dia, socket across-corners, nominal_dia — all parameter-driven", dims)
	}
}

// assertCylindricalExtrudes checks the cylindrical cap screw's three extrudes in build order:
// head down (new, untapered), socket down (cut), shank down over its length (join).
func assertCylindricalExtrudes(t *testing.T, ex []featureargs.Extrude) {
	t.Helper()
	if len(ex) != 3 {
		t.Fatalf("extrudes = %d, want 3 (head + socket + shank)", len(ex))
	}
	if ex[0].Distance != "head_height" || ex[0].Operation != "new" || ex[0].Direction != "negative" || ex[0].Taper != "" {
		t.Errorf("head extrude = %+v, want head_height/new/negative/no-taper", ex[0])
	}
	if ex[1].Distance != "socket_depth" || ex[1].Operation != "cut" || ex[1].Direction != "negative" {
		t.Errorf("socket extrude = %+v, want socket_depth/cut/negative", ex[1])
	}
	if ex[2].Distance != "length" || ex[2].Operation != "join" || ex[2].Direction != "negative" {
		t.Errorf("shank extrude = %+v, want length/join/negative", ex[2])
	}
}

// TestSocketScrewBuildsCountersunkScrew covers ISO 10642: the head is a loft from the head-dia
// circle (top, on XY) to the shank-dia circle (bottom, on the head-underside plane), giving the
// standard cone; the socket is cut; and the shank spans length−head_height (a countersunk screw
// is measured overall). No extrude taper is used — the host cannot express it as a formula.
func TestSocketScrewBuildsCountersunkScrew(t *testing.T) {
	h := &fakeHost{dof: 0}
	rm := socketMember("countersunk", 8, 1.25, 17.3, 4.4, 5, 2.6, 40, 40)
	if err := (SocketScrew{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	// The head is a loft (not an extrude); the socket cut and the shank join are the two extrudes.
	if len(h.lofts) != 1 || h.lofts[0].Operation != "new" || len(h.lofts[0].Sections) != 2 {
		t.Fatalf("lofts = %+v, want one two-section new loft for the cone head", h.lofts)
	}
	for _, e := range h.extrudes {
		if e.Taper != "" {
			t.Errorf("extrude %+v carries a taper; the countersink is a loft, not a tapered extrude", e)
		}
	}
	if len(h.extrudes) != 2 || h.extrudes[0].Operation != "cut" ||
		h.extrudes[1].Distance != "length - head_height" || h.extrudes[1].Operation != "join" {
		t.Errorf("extrudes = %+v, want the socket cut then a (length - head_height)/join shank", h.extrudes)
	}
	// Two offset planes: the loft's bottom circle and the shank, both on the head-underside plane.
	if len(h.workPlanes) != 2 {
		t.Errorf("work planes = %d, want 2 (loft bottom + shank), both at the head underside", len(h.workPlanes))
	}
	if len(h.threads) != 1 || h.threads[0].FaceRef != "shank-cyl" {
		t.Errorf("threads = %+v, want one on the shank cylinder", h.threads)
	}
}

// TestSocketScrewUnknownVariant rejects a family whose head-style variant the generator does not
// recognise, naming the offending value rather than silently building the default head.
func TestSocketScrewUnknownVariant(t *testing.T) {
	h := &fakeHost{dof: 0}
	err := (SocketScrew{}).Build(newBuilder(h, catalog.UnitsMillimetre), socketMember("button", 8, 1.25, 13, 8, 6, 4, 40, 40))
	if err == nil || !strings.Contains(err.Error(), "\"button\"") || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("Build error = %v, want it to reject the unknown variant \"button\"", err)
	}
	if h.hasMethod(wire.MethodFeaturesAdd) {
		t.Error("a feature ran for an unknown variant; the variant check must gate the build")
	}
}

// TestSocketScrewDefaultsToCylindrical covers an empty Variant taking the cylindrical head path.
func TestSocketScrewDefaultsToCylindrical(t *testing.T) {
	h := &fakeHost{dof: 0, headCylinder: true}
	if err := (SocketScrew{}).Build(newBuilder(h, catalog.UnitsMillimetre), socketMember("", 8, 1.25, 13, 8, 6, 4, 40, 40)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if h.extrudes[0].Taper != "" {
		t.Errorf("default head taper = %q, want an untapered cylindrical head", h.extrudes[0].Taper)
	}
}

// TestSocketScrewThreadNeedsCylinder covers the guard: no cylindrical face means the thread step
// fails loudly rather than threading the wrong surface.
func TestSocketScrewThreadNeedsCylinder(t *testing.T) {
	h := &fakeHost{dof: 0, noCylinder: true}
	err := (SocketScrew{}).Build(newBuilder(h, catalog.UnitsMillimetre), socketMember("cylindrical", 8, 1.25, 13, 8, 6, 4, 40, 40))
	if err == nil || !strings.Contains(err.Error(), "cylindrical face") {
		t.Fatalf("Build error = %v, want it to mention the missing cylindrical face", err)
	}
	if len(h.threads) != 0 {
		t.Errorf("threads = %+v, want none when the shank face is missing", h.threads)
	}
}

// TestSocketScrewUnderConstrainedFails ensures a non-zero DOF aborts the build before any solid
// is extruded.
func TestSocketScrewUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 3}
	err := (SocketScrew{}).Build(newBuilder(h, catalog.UnitsMillimetre), socketMember("cylindrical", 8, 1.25, 13, 8, 6, 4, 40, 40))
	if err == nil {
		t.Fatal("Build accepted an under-constrained sketch; want an error")
	}
	if h.hasMethod(wire.MethodFeaturesAdd) {
		t.Error("a feature ran on an under-constrained sketch; the DOF check must gate it")
	}
}

// TestDefaultRegistryHasSocketScrew checks the generator is wired into the built-in set.
func TestDefaultRegistryHasSocketScrew(t *testing.T) {
	g, ok := DefaultRegistry().Get("socket_screw")
	if !ok || g.Kind() != "socket_screw" {
		t.Fatalf("DefaultRegistry socket_screw = (%v,%v), want the SocketScrew generator", g, ok)
	}
}
