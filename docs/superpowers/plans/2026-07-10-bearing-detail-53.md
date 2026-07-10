# Bearing detail (#53) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add ball 2Z shields, and cylindrical-roller guide flanges + roller-end chamfers + a phased bridge-bar cage, to the representational PartDesigner bearings — each an independent DOF-0 parametric body with a provable non-intersection guard.

**Architecture:** Add-in-only (no Oblikovati.API/host change). New `section_*.go` grounded-section helpers, wired into the existing `roller_bearing.go` and `bearing.go` generators, preserving the whole-body-pattern ordering (rolling element + cage bar built before the single `PatternCircular`; rings/shields added after). Every feature has a Go `…Fit` guard that falls back to the current plain body.

**Tech Stack:** Go, `oblikovati.org/api/client` over a `HostCaller` seam, `fakeHost` test double, `//go:embed` catalog data. Kernel sketch solver reached via `client.Sketch()` (Constrain/Dimension/AddRectangle/AddPoint/AddCenterline).

## Global Constraints

- **No boolean; independent bodies.** Every non-intersection guard is a separating-gap proof in ONE coordinate (axial/radial/angular). Clearance floor `ε_clr = max(0.10 mm, 2·chord_tol)`; use `0.10` mm absolute where the spec gives an absolute.
- **DOF-0 parametric.** Every dimension is a published-parameter formula expression; every section calls `AssertFullyConstrained()` before the feature (the roller/bar/shield sections that already do so inside `patternRollers`/wiring keep doing so).
- **`disableSketchInference()` FIRST** in every section built from lines/polylines/rectangles (`GroundedFlangedRingSection`, `GroundedChamferedRollerSection`, `GroundedShieldSection`; the cage bar reuses `GroundedRingSection` which is called on an angled sketch — add the guard there too). Omitting it silently yields NO body.
- **Whole-body-pattern ordering.** In `roller_bearing.go`: build roller, then cage bar, then ONE `PatternCircular("roller_count")`; add rings after.
- **Fall-back-to-plain** on guard failure. Guards are Go predicates on the `ResolvedMember`, mirroring the parametric formulas.
- SPDX `GPL-2.0-only` header on every new `.go`; funcs 4–20 lines; files <500 lines; explicit types; `fakeHost` for host I/O; coverage >80%, duplication <3%.
- Derived-param expressions are appended in dependency order via `b.DeriveParam(name, expr)`. Angles: the evaluator stores angles in radians; `asin`/`atan`/`sin`/`cos`/`tan`/`sqrt`/`min` available; `180 deg / roller_count` yields radians.

**Existing helpers to reuse (do not reinvent):**
- `(*SketchContext).GroundedRingSection(innerDia, outerDia, width string) error` — z-centred rectangular ring section (model the flange/shield sections on it and on `GroundedCageRingSection`).
- `(*SketchContext).GroundedChamferedRodSection(diameterExpr, lengthExpr, chamferExpr string) error` — 6-point chamfered rod revolved about the sketch axis; the roller chamfer section is this pattern but offset to a centerline at the pitch radius.
- `(*SketchContext).disableSketchInference() error`, `(*SketchContext).groundedOrigin() (uint64,error)`, `(*SketchContext).closedPolyline(pts) (points,edges []uint64,err error)`, `(*SketchContext).addSeedPoint(seed [2]float64) (uint64,error)`, `xy(p [2]float64) []float64`, `half(expr string) string`, `radialWidth(outerDia,innerDia string) string`, `rectAxisConstraints(con, bl,br,tr,tl)`, `alignLevels(con, horiz, vert)`.
- `(*PartBuilder).Sketch("XZ"|"XY") (*SketchContext,error)`, `.Revolve(sk, "origin/axis/z", "360 deg", "new") (string,error)`, `.RevolveAboutCenterline(sk, "360 deg", "new") (string,error)`, `.RevolveTwoSided(sk, "origin/axis/z", halfAngleExpr, "new") (string,error)`, `.AngledOrientedSketch(angleExpr string) (*SketchContext,error)`, `.PatternCircular(sourceFeature, countExpr string) error`, `.DeriveParam(name, expr string) error`.
- `client.Sketch().AddCenterline(index int, a, b []float64) (…EntityID uint64…, error)` — see `GroundedDomedRollerSection` in `section_roller.go` for the centerline idiom.
- `ResolvedMember.Value(col string) float64` — tabulated column value (mm) for a member, used by the Go `…Fit` guards.

**Existing `fakeHost` recorder fields (assert against these):** `h.added []wire.ParameterSetArgs` (derived+published params), `h.revolves []featureargs.Revolve`, `h.extrudes []featureargs.Extrude`, `h.patterns []wire.CircularPatternFeatureArgs`, `h.dof int` (set to force `AssertFullyConstrained` fail). `assertParam(t, h.added, name, expr)` asserts a param's exact expression string.

---

### Task 1: Cylindrical-roller guide flanges (outer ring ⊐ channel)

**Files:**
- Create: `designer/build/section_flange.go`
- Modify: `designer/build/roller_bearing.go` (add `deriveFlangeParams`; replace the outer-ring build with a flanged-or-plain choice)
- Test: `designer/build/roller_bearing_test.go` (update the ring-count assertion; add flange tests)

**Interfaces:**
- Produces: `deriveFlangeParams(b *PartBuilder) error`; `flangesFit(rm ResolvedMember) bool`; `(*SketchContext).GroundedFlangedRingSection(edgeDia, raceDia, flangeBoreDia, innerZ, width string) error`; `(*PartBuilder).revolveFlangedOuterRing() error`.
- Consumes: existing `derivePitchDia`, `deriveRacesClearing`, `revolveRing`.

- [ ] **Step 1: Write failing tests** in `roller_bearing_test.go`:

```go
// deriveFlangeParams publishes the flange axial band; flangesFit gates on positive land/overlap/band.
func TestRollerFlangeParams(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rollerMember("NU206", 30, 62, 16, 13)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "flange_axial_clr", "max(0.1, roller_length * 0.02)")
	assertParam(t, h.added, "flange_inner_z", "roller_length / 2 + flange_axial_clr")
	assertParam(t, h.added, "flange_bore_dia", "pitch_dia")
}

func TestRollerFlangedOuterRingRevolved(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rollerMember("NU206", 30, 62, 16, 13)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	// inner ring + flanged outer ring = 2 revolves about Z, all 360deg/new.
	if len(h.revolves) < 2 {
		t.Fatalf("revolves = %d, want >=2 (inner + flanged outer)", len(h.revolves))
	}
	last := h.revolves[len(h.revolves)-1]
	if last.AxisRef != "origin/axis/z" || last.Angle != "360 deg" || last.Operation != "new" {
		t.Errorf("outer ring revolve = %+v, want z/360 deg/new", last)
	}
}

func TestFlangesFitAcrossFamily(t *testing.T) {
	members := [][4]float64{{15, 35, 11, 11}, {30, 62, 16, 13}, {50, 90, 20, 15}} // NU202, NU206, NU210
	for _, m := range members {
		if !flangesFit(rollerMember("x", m[0], m[1], m[2], m[3])) {
			t.Errorf("flangesFit false for d=%v D=%v B=%v; every NU2xx must get flanges", m[0], m[1], m[2])
		}
	}
	// Degenerate: a roller nearly as long as the ring leaves no overhang band → no flanges.
	if flangesFit(rollerMember("x", 30, 62, 1.0, 13)) {
		t.Error("flangesFit true for a member with no axial overhang; want plain-ring fallback")
	}
}
```

- [ ] **Step 2: Run — expect FAIL** (`flangesFit`/`deriveFlangeParams` undefined, ring assertions):
`cd designer && go test ./build/ -run 'RollerFlange|FlangesFit' -v` → FAIL (undefined).

- [ ] **Step 3: Create `section_flange.go`.** `GroundedFlangedRingSection` draws the 8-point ⊐ meridian (spec Feature 1), modelled on `GroundedCageRingSection`'s rectangle idiom but as a `closedPolyline` of 8 seed points; call `disableSketchInference()` first, then `groundedOrigin()`. Point layout (X=radius, Z=axial), seeds cm-scale to set topology only:

```
P1 (D/2, -width/2)  P2 (D/2, +width/2)  P3 (pitch_dia/2, +width/2)  P4 (pitch_dia/2, +flange_inner_z)
P5 (outer_race_dia/2, +flange_inner_z)  P6 (outer_race_dia/2, -flange_inner_z)
P7 (pitch_dia/2, -flange_inner_z)  P8 (pitch_dia/2, -width/2)  close P8->P1
```

Constrain DOF-0 with axis-aligned edges only (each edge Horizontal or Vertical via `con.Horizontal/Vertical` on its two endpoints — all edges are axis-aligned in this ⊐), and dimension the distinct radii and z-levels with `dim.Offset(origin, edge, expr)`: radii `D/2`, `pitch_dia/2`, `outer_race_dia/2` (as `half("outer_dia")`, `half("pitch_dia")`, `half("outer_race_dia")`); z-levels `width/2` (`half("width")`), `flange_inner_z`. Reuse `alignLevels`. Assert full constraint via the caller.

- [ ] **Step 4: Add to `roller_bearing.go`:**

```go
// Flange proportions: the axial clearance from a roller end to the flange inner face, as an
// absolute floor or a fraction of the roller length — whichever is larger.
const flangeAxialClrFraction = "0.02"

// deriveFlangeParams adds the outer-ring guide-flange band: the roller-end→flange clearance, the
// flange inner-face |z|, and the flange bore diameter (pitch_dia = mid roller-end annulus, so the
// rib dips roller_dia/2 below the roller crest yet keeps a land above the plain inner ring).
func deriveFlangeParams(b *PartBuilder) error {
	if err := b.DeriveParam("flange_axial_clr", "max(0.1, roller_length * "+flangeAxialClrFraction+")"); err != nil {
		return err
	}
	if err := b.DeriveParam("flange_inner_z", "roller_length / 2 + flange_axial_clr"); err != nil {
		return err
	}
	return b.DeriveParam("flange_bore_dia", "pitch_dia")
}

// flangesFit reports whether the outer ring can carry integral guide flanges: a positive land above
// the inner ring, a real locating overlap, and a visible axial overhang band. Mirrors the parametric
// guard so the Go build decision matches the geometry. Units: mm (tabulated columns).
func flangesFit(rm ResolvedMember) bool {
	d, D, B := rm.Value("d"), rm.Value("D"), rm.Value("B")
	gap := D - d
	rollerDia, rollerLen := 0.28*gap, 0.8*B
	land := (rollerDia + 0.012*gap) / 2
	overlap := rollerDia / 2
	axialClr := math.Max(0.10, 0.02*rollerLen)
	band := B/2 - rollerLen/2 - axialClr
	const epsClr = 0.10
	return land >= epsClr && overlap >= epsClr && band >= epsClr
}

// revolveFlangedOuterRing revolves the outer ring as an inward-opening channel carrying two integral
// guide flanges; falls back to a plain ring when flangesFit is false.
func (b *PartBuilder) revolveFlangedOuterRing(rm ResolvedMember) error {
	if !flangesFit(rm) {
		return b.revolveRing("outer_race_dia", "outer_dia")
	}
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedFlangedRingSection("outer_dia", "outer_race_dia", "flange_bore_dia", "flange_inner_z", "width"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}
```

Add `import "math"` to `roller_bearing.go`. In `deriveRollerParams`, append `deriveFlangeParams(b)` after `deriveRacesClearing`. In `Build`, replace `return b.revolveRing("outer_race_dia", "outer_dia")` with `return b.revolveFlangedOuterRing(rm)` (thread `rm` through; `Build` already has it).

- [ ] **Step 5: Update the pre-existing test** `TestRollerBearingBuildsRollersAndRings` — its `for i, rv := range h.revolves` block still holds (both revolves are z/360/new), so no change there; it asserts `len(h.revolves) == 2` which STILL holds (inner + flanged outer). Leave it. Confirm.

- [ ] **Step 6: Run — expect PASS:** `cd designer && go test ./build/ -run 'Roller|Flange' -v` → PASS. Then `go test ./build/` (whole package) → PASS.

- [ ] **Step 7: Kernel-diagnostic the section** (host repo, throwaway) to confirm the ⊐ meridian solves `DOF=0, Redundant=0` and `Solve().Converged` for NU206 numbers, per the geometry-brief recipe. Delete the diagnostic after.

- [ ] **Step 8: Lint + commit:**
```bash
cd designer && golangci-lint run ./build/ && cd ..
git add designer/build/section_flange.go designer/build/roller_bearing.go designer/build/roller_bearing_test.go
git commit -m "feat(#53): cylindrical-roller outer-ring guide flanges

Outer ring becomes an inward-opening channel with two integral flanges
that axially locate the rollers; flange bore at pitch_dia keeps a land
above the plain inner ring and a locating overlap below the roller
crest. flangesFit guards the fallback to a plain ring."
```

---

### Task 2: Roller-end chamfers (revolve about own centerline)

**Files:**
- Create: `designer/build/section_roller_chamfer.go`
- Modify: `designer/build/roller_bearing.go` (`deriveRollerChamferParams`; rebuild `patternRollers` roller body)
- Test: `designer/build/roller_bearing_test.go` (replace the extrude assertions with revolve-about-centerline assertions; add chamfer tests)

**Interfaces:**
- Produces: `deriveRollerChamferParams(b) error`; `rollerChamferFits(rm ResolvedMember) bool`; `(*SketchContext).GroundedChamferedRollerSection(axisXExpr, diameterExpr, lengthExpr, chamferExpr string) error`; `(*PartBuilder).buildRoller(rm ResolvedMember) (feature string, err error)`.
- Consumes: `GroundedOffsetCircle` (plain-roller fallback), `RevolveAboutCenterline`, `AddCenterline`.

- [ ] **Step 1: Write failing tests.** Replace the extrude block in `TestRollerBearingBuildsRollersAndRings` (the roller is now a REVOLVE about a centerline, not an extrude) and add:

```go
// The chamfered roller is revolved about its own centerline; assert the chamfer param + that the
// roller feature is a 360deg/new revolve (about the centerline, not the Z axis).
func TestRollerChamferParams(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rollerMember("NU206", 30, 62, 16, 13)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "roller_chamfer", "roller_dia * 0.1")
}

func TestRollerChamferFits(t *testing.T) {
	if !rollerChamferFits(rollerMember("x", 15, 35, 11, 11)) { // NU202, min leg 0.56mm
		t.Error("rollerChamferFits false for NU202; every member should chamfer")
	}
	if rollerChamferFits(rollerMember("x", 100, 100.2, 11, 11)) { // ~0 gap → sub-floor leg
		t.Error("rollerChamferFits true for a sub-floor chamfer; want plain-roller fallback")
	}
}
```

In `TestRollerBearingBuildsRollersAndRings`, delete the `if len(h.extrudes) != 1 …` and `h.extrudes[0].Distance …` assertions; replace with: the roller complement is now revolves — after this task, `h.revolves` includes the roller (about centerline) + 2 rings. Assert `len(h.revolves) >= 3` and that the ring revolves (last two) are z/360/new. Keep `TestRollerBearingPatternsByCount` (still one pattern of `roller_count`).

- [ ] **Step 2: Run — expect FAIL** (`roller_chamfer` undefined; extrude assertions removed): `go test ./build/ -run 'RollerChamfer|RollerBearingBuilds' -v` → FAIL.

- [ ] **Step 3: Create `section_roller_chamfer.go`.** Model on `GroundedDomedRollerSection`'s centerline idiom (`section_roller.go`) but the axis is a vertical line at `X = axisXExpr` (`= half("pitch_dia")`), the meridian is the 6-point rod to its +X side. `disableSketchInference()` first. Add a centerline through two seed points at `(pitchR, -len/2)`,`(pitchR, +len/2)`; `closedPolyline` the 6 points (spec Feature 2 layout with `axis=pitch_dia/2`, `rd=roller_dia`, `rl=roller_length`, `c=roller_chamfer`); pin P1,P2 on the centerline via `con.PointOnLine`; dimension: `dim.Distance(P1,P2, roller_length)` (axis span), the crest offset `dim.Offset(P1edge?, …)` — use the same dimensioning shape as `dimensionChamferedRod` (radius = `half(diameter)`, chamfer feet = radius−chamfer, hypotenuses = `chamfer*sqrt(2)`) but measured from the centerline. Then `RevolveAboutCenterline`.

- [ ] **Step 4: Add to `roller_bearing.go`:**

```go
const rollerChamferFraction = "0.1" // 45deg chamfer leg as a fraction of roller_dia

// deriveRollerChamferParams adds the roller-end chamfer leg (45deg, equal axial & radial leg).
func deriveRollerChamferParams(b *PartBuilder) error {
	return b.DeriveParam("roller_chamfer", "roller_dia * "+rollerChamferFraction)
}

// rollerChamferFits reports whether a visible 45deg end chamfer fits: leg below half the roller
// radius (end disc stays real) and above the visibility floor. Else the roller is built plain.
func rollerChamferFits(rm ResolvedMember) bool {
	gap := rm.Value("D") - rm.Value("d")
	rollerDia := 0.28 * gap
	leg := 0.10 * rollerDia
	const cMin, epsClr = 0.15, 0.10
	return leg < rollerDia/2-epsClr && leg >= cMin
}

// buildRoller builds one cylindrical roller standing on the pitch circle as a body of revolution
// about its own centerline: chamfered when rollerChamferFits, else a plain cylinder. Returns the
// roller feature name so the caller can pattern it.
func (b *PartBuilder) buildRoller(rm ResolvedMember) (string, error) {
	if !rollerChamferFits(rm) {
		return b.buildPlainRoller()
	}
	sk, err := b.Sketch("XZ")
	if err != nil {
		return "", err
	}
	if err := sk.GroundedChamferedRollerSection(half("pitch_dia"), "roller_dia", "roller_length", "roller_chamfer"); err != nil {
		return "", err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return "", err
	}
	return b.RevolveAboutCenterline(sk, "360 deg", "new")
}
```

Extract the old extrude path into `buildPlainRoller() (string, error)` (the current `patternRollers` circle+extrude, but returning the feature name; `ExtrudeNamed` already returns it). Rewrite `patternRollers` to: `roller, err := b.buildRoller(rm)` … then (Task 3 inserts the bar here) … `return b.PatternCircular(roller, "roller_count")`. Append `deriveRollerChamferParams(b)` in `deriveRollerParams`. Thread `rm` into `patternRollers` (change its signature to `patternRollers(rm ResolvedMember)` and its call in `Build`).

- [ ] **Step 5: Run — expect PASS:** `go test ./build/ -run 'Roller' -v` then `go test ./build/`.

- [ ] **Step 6: Kernel-diagnostic** the chamfered-roller meridian (off-axis centerline at pitch radius) → `DOF=0, Redundant=0, Converged` for NU206. Delete after.

- [ ] **Step 7: Lint + commit:**
```bash
git add designer/build/section_roller_chamfer.go designer/build/roller_bearing.go designer/build/roller_bearing_test.go
git commit -m "feat(#53): cylindrical roller-end chamfers

Roller is now revolved about its own centerline from a chamfered
meridian (45deg end chamfers built into the profile, no edge refs),
preserving the +-roller_length/2 x pitch_dia+-roller_dia envelope.
rollerChamferFits falls back to the plain cylinder."
```

---

### Task 3: Cylindrical cage (phased bridge bars)

**Files:**
- Create: `designer/build/roller_cage.go`
- Modify: `designer/build/roller_bearing.go` (`deriveRollerCageParams` in `deriveRollerParams`; insert `buildRollerCageBar` before the pattern in `patternRollers`)
- Test: `designer/build/roller_cage_test.go` (create)

**Interfaces:**
- Produces: `deriveRollerCageParams(b) error`; `cageBarsFit(rm ResolvedMember) bool`; `(*PartBuilder).buildRollerCageBar() error`.
- Consumes: `AngledOrientedSketch`, `GroundedRingSection`, `RevolveTwoSided`. Model on `tapered_cage.go`.

- [ ] **Step 1: Write failing tests** in new `roller_cage_test.go`:

```go
func TestRollerCageParams(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rollerMember("NU206", 30, 62, 16, 13)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "roller_subtend", "asin(roller_dia / pitch_dia)")
	assertParam(t, h.added, "cage_half_pitch", "180 deg / roller_count")
	assertParam(t, h.added, "bar_half_angle", "0.4 * (cage_half_pitch - roller_subtend)")
	assertParam(t, h.added, "bar_id", "pitch_dia - bar_radial_thick")
	assertParam(t, h.added, "bar_od", "pitch_dia + bar_radial_thick")
	assertParam(t, h.added, "bar_axial_len", "roller_length * 0.7")
}

// The cage bar is a two-sided revolve about Z; with a bar there are 2 pattern-source bodies.
func TestRollerCageBarRevolvedBeforePattern(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rollerMember("NU206", 30, 62, 16, 13)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if len(h.patterns) != 1 || h.patterns[0].CountExpr != "roller_count" {
		t.Fatalf("patterns = %+v, want one roller_count pattern over roller+bar", h.patterns)
	}
	// A bar_half_angle two-sided revolve about Z exists among the revolves (roller+bar+rings).
	var found bool
	for _, rv := range h.revolves {
		if rv.Angle == "bar_half_angle" || rv.Angle == "bar_half_angle * 2" {
			found = true
		}
	}
	if !found {
		t.Errorf("no cage-bar revolve found in %+v", h.revolves)
	}
}

func TestCageBarsFitFamily(t *testing.T) {
	// NU204 is the tightest (free_half_gap 1.96deg) but still > floor → gets bars.
	if !cageBarsFit(rollerMember("x", 20, 47, 14, 12)) {
		t.Error("cageBarsFit false for NU204; it should still get bars")
	}
	// A dense synthetic complement (many fat rollers) → no room → no bars.
	if cageBarsFit(rollerMember("x", 30, 62, 16, 40)) {
		t.Error("cageBarsFit true for a dense complement; want no-bars fallback")
	}
}
```

(Confirm `RevolveTwoSided` records the angle it stores — read `builder.go:238` to see whether it stores `halfAngleExpr` verbatim or `×2`; match the test's `rv.Angle` accordingly.)

- [ ] **Step 2: Run — expect FAIL:** `go test ./build/ -run 'Cage' -v` → FAIL (undefined).

- [ ] **Step 3: Create `roller_cage.go`** modelled on `tapered_cage.go` but simpler (no cone). Constants and derivations:

```go
const (
	cageBarFill          = "0.4"  // psi as a fraction of the free half-gap
	cageBarThickFraction = "0.25" // bar radial thickness / roller_dia
	cageBarAxialFraction = "0.7"  // bar axial length / roller_length (short of the flanges)
	cageBarGapFloor      = 0.020  // radians; below this the gap is too tight to seat a bar
)

func rollerCageDerivations() []struct{ name, expr string } {
	return []struct{ name, expr string }{
		{"roller_subtend", "asin(roller_dia / pitch_dia)"},
		{"cage_half_pitch", "180 deg / roller_count"},
		{"bar_half_angle", cageBarFill + " * (cage_half_pitch - roller_subtend)"},
		{"bar_radial_thick", "roller_dia * " + cageBarThickFraction},
		{"bar_id", "pitch_dia - bar_radial_thick"},
		{"bar_od", "pitch_dia + bar_radial_thick"},
		{"bar_axial_len", "roller_length * " + cageBarAxialFraction},
	}
}

func deriveRollerCageParams(b *PartBuilder) error {
	for _, d := range rollerCageDerivations() {
		if err := b.DeriveParam(d.name, d.expr); err != nil {
			return err
		}
	}
	return nil
}

// cageBarsFit reports whether the inter-roller gap can seat a bridge bar: the free half-gap
// pi/Z - asin(roller_dia/pitch_dia) must clear the floor. Mirrors the parametric derivation.
func cageBarsFit(rm ResolvedMember) bool {
	d, D, n := rm.Value("d"), rm.Value("D"), rm.Value("Z")
	if n < 3 {
		return false
	}
	gap := D - d
	rollerDia, pitchDia := 0.28*gap, (d+D)/2
	freeHalfGap := math.Pi/n - math.Asin(rollerDia/pitchDia)
	return freeHalfGap > cageBarGapFloor
}

// buildRollerCageBar builds one bridge bar as a thin ring-section revolved a small +-angle about Z
// on a plane at the half-pitch azimuth, so it sits centred in the gap between two rollers. Built
// BEFORE the roller pattern so the single pattern arrays roller+bar together.
func (b *PartBuilder) buildRollerCageBar() error {
	sk, err := b.AngledOrientedSketch("cage_half_pitch")
	if err != nil {
		return err
	}
	if err := sk.GroundedRingSection("bar_id", "bar_od", "bar_axial_len"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	_, err = b.RevolveTwoSided(sk, "origin/axis/z", "bar_half_angle", "new")
	return err
}
```

Add `import "math"` (already added in Task 1). Append `deriveRollerCageParams(b)` in `deriveRollerParams` (after chamfer params). In `patternRollers`, between `buildRoller` and `PatternCircular`, insert:
```go
if cageBarsFit(rm) {
	if err := b.buildRollerCageBar(); err != nil {
		return err
	}
}
```

- [ ] **Step 4: Run — expect PASS:** `go test ./build/ -run 'Cage|Roller' -v`, then `go test ./build/`.

- [ ] **Step 5: Kernel-diagnostic** the angled-plane bar ring section → DOF-0/Redundant-0/Converged. Delete after.

- [ ] **Step 6: Lint + commit:**
```bash
git add designer/build/roller_cage.go designer/build/roller_cage_test.go designer/build/roller_bearing.go
git commit -m "feat(#53): cylindrical-roller phased bridge-bar cage

One bar per inter-roller gap at the pitch radius, revolved a small
+-angle on a half-pitch-azimuth plane and built before the roller
pattern so a single pattern arrays roller+bar together (bar lands in
every gap). cageBarsFit omits bars for dense complements. Bars only —
the roller-free end band is claimed by the guide flanges."
```

---

### Task 4: Ball 2Z shields (both faces)

**Files:**
- Create: `designer/build/section_shield.go`
- Modify: `designer/build/bearing.go` (`deriveShieldParams` in `deriveBearingParams`; `revolveShields` after the two grooved rings in `Build`)
- Test: `designer/build/bearing_test.go`

**Interfaces:**
- Produces: `deriveShieldParams(b) error`; `shieldsFit(rm ResolvedMember) bool`; `(*SketchContext).GroundedShieldSection(idDia, odDia, nearZ, farZ string) error`; `(*PartBuilder).revolveShields(rm ResolvedMember) error`.
- Consumes: `GroundedRingSection` idiom, `Revolve`. Ball member helper in `bearing_test.go` (reuse the existing `ballMember`/equivalent — read `bearing_test.go` for its name; if none, add one mirroring `rollerMember` with a `ball_count` column).

- [ ] **Step 1: Write failing tests** in `bearing_test.go` (use the existing ball member helper; 6206 = 30×62×16, count 9):

```go
func TestBallShieldParams(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (BallBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), ballMember("6206", 30, 62, 16, 9)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "shield_near_z", "ball_dia / 2 + 0.2")
	assertParam(t, h.added, "shield_thick", "min(width * 0.12, width / 2 - ball_dia / 2 - 0.4)")
	assertParam(t, h.added, "shield_far_z", "shield_near_z + shield_thick")
	assertParam(t, h.added, "shield_id", "inner_shoulder_dia + 0.3")
	assertParam(t, h.added, "shield_od", "outer_shoulder_dia - 0.3")
}

// Two shields (both faces) are revolved after the two grooved rings: >=4 revolves, all z/360/new.
func TestBallShieldsRevolvedBothFaces(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (BallBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), ballMember("6206", 30, 62, 16, 9)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if len(h.revolves) < 4 { // ball + 2 rings + 2 shields (ball is 1 revolve, rings 2, shields 2 => 5)
		t.Fatalf("revolves = %d, want >=4 with two shields", len(h.revolves))
	}
	shields := h.revolves[len(h.revolves)-2:]
	for i, rv := range shields {
		if rv.AxisRef != "origin/axis/z" || rv.Angle != "360 deg" || rv.Operation != "new" {
			t.Errorf("shield revolve[%d] = %+v, want z/360 deg/new", i, rv)
		}
	}
}

func TestShieldsFit(t *testing.T) {
	if !shieldsFit(ballMember("6200", 10, 30, 9, 8)) { // worst slack 1.70mm
		t.Error("shieldsFit false for 6200; every 60/62/63 member should get shields")
	}
	if shieldsFit(ballMember("x", 10, 40, 2, 8)) { // fat ball, thin ring → no room
		t.Error("shieldsFit true for a fat-ball/thin-ring member; want no-shield fallback")
	}
}
```

- [ ] **Step 2: Run — expect FAIL:** `go test ./build/ -run 'Shield' -v` → FAIL.

- [ ] **Step 3: Create `section_shield.go`.** `GroundedShieldSection` = a flat annular rectangle in XZ NOT centred on the mid-plane; model exactly on `GroundedCageRingSection` (which already positions a ring by near/far |z| offsets from the origin), parameterized by `idDia, odDia, nearZ, farZ`. `disableSketchInference()` first. Reuse `constrainCageRing`-style constraints (rename locals for a shield). Both shields are the SAME section revolved; the second is mirrored by building a second sketch with `nearZ`/`farZ` negated — implement `revolveShields` to build the +face shield then the −face shield.

- [ ] **Step 4: Add to `bearing.go`:**

```go
const shieldThickFraction = "0.12" // shield axial thickness cap as a fraction of width

// deriveShieldParams adds the 2Z shield band: near face just outboard of the ball equator, thickness
// capped by the axial slack, and the radial span a hair inside the two raceway shoulders.
func deriveShieldParams(b *PartBuilder) error {
	dims := []struct{ name, expr string }{
		{"shield_near_z", "ball_dia / 2 + 0.2"},
		{"shield_thick", "min(width * " + shieldThickFraction + ", width / 2 - ball_dia / 2 - 0.4)"},
		{"shield_far_z", "shield_near_z + shield_thick"},
		{"shield_id", "inner_shoulder_dia + 0.3"},
		{"shield_od", "outer_shoulder_dia - 0.3"},
	}
	for _, d := range dims {
		if err := b.DeriveParam(d.name, d.expr); err != nil {
			return err
		}
	}
	return nil
}

// shieldsFit reports whether both 2Z shields fit axially between the ball equator and the ring end
// faces: axial slack width/2 - ball_dia/2 must exceed clearance+inset+min-thickness. Else no shields.
func shieldsFit(rm ResolvedMember) bool {
	gap := rm.Value("D") - rm.Value("d")
	ballDia, B := 0.28*gap, rm.Value("B")
	const sMin = 0.70 // shield_clr_ax 0.2 + ring_inset 0.2 + t_min 0.3
	return B/2-ballDia/2 >= sMin
}

// revolveShields revolves the two metal shields (one per face) after the grooved rings, when they
// fit; each is a flat annulus spanning the groove mouth, axially outboard of the ball equator.
func (b *PartBuilder) revolveShields(rm ResolvedMember) error {
	if !shieldsFit(rm) {
		return nil
	}
	for _, z := range [][2]string{{"shield_near_z", "shield_far_z"}, {"-shield_near_z", "-shield_far_z"}} {
		sk, err := b.Sketch("XZ")
		if err != nil {
			return err
		}
		if err := sk.GroundedShieldSection("shield_id", "shield_od", z[0], z[1]); err != nil {
			return err
		}
		if err := sk.AssertFullyConstrained(); err != nil {
			return err
		}
		if _, err := b.Revolve(sk, "origin/axis/z", "360 deg", "new"); err != nil {
			return err
		}
	}
	return nil
}
```

(If `GroundedShieldSection`'s Offset-dim can't take a negated expression cleanly, instead pass a `bool bottomFace` and build the −face section with negated seed points + `dim.Offset` on the mirrored edges — match how `GroundedCageRingSection` handles the −Z side. Decide during Step 3.) Append `deriveShieldParams(b)` in `deriveBearingParams`; in `Build`, after the second `revolveGroovedRing`, `return b.revolveShields(rm)` (currently `Build` returns that ring call — capture its error first, then shields).

- [ ] **Step 5: Run — expect PASS:** `go test ./build/ -run 'Ball|Shield' -v`, then `go test ./build/`.

- [ ] **Step 6: Kernel-diagnostic** the shield annulus (both faces) → DOF-0/Redundant-0/Converged for 6206. Delete after.

- [ ] **Step 7: Lint + commit:**
```bash
git add designer/build/section_shield.go designer/build/bearing.go designer/build/bearing_test.go
git commit -m "feat(#53): deep-groove ball 2Z shields

Two flat annular metal shields (both faces) revolved after the grooved
rings, axially outboard of the ball equator and radially between the
raceway shoulders — provably disjoint from balls (axial gap) and rings
(radial gap). shieldsFit falls back to no shields for fat-ball members."
```

---

### Task 5: Live verification, full suite, coverage & docs

**Files:**
- Modify: `README.md` / SOURCES provenance if a standards note is warranted (bearing detail is representational; note the k-fractions' provenance = geometry brief).
- Test: whole-package + live MCP.

- [ ] **Step 1: Full suite + lint + coverage/dup:**
```bash
cd designer && go test ./... -coverprofile=/tmp/c.out && go tool cover -func=/tmp/c.out | tail -1
golangci-lint run ./... && cd ..
# coverage >80% on build/; if any new draw/guard path is uncovered, add a fakeHost test.
```
Expected: all pass, coverage >80%.

- [ ] **Step 2: Build head against current API + install add-in** (per the live-test recipe): `make install`; launch the head with `DISPLAY=:1` + `OBK_ADDINS_DIR`; MCP bridge on `127.0.0.1:7800`. Use the Popen-from-one-foreground-process pattern (never background the head).

- [ ] **Step 3: Live place + screenshot NU206** (cylindrical): drive the panel (or `PartDesigner.Place` headless) to place NU206; `capture_window`/`capture_viewport`. Confirm visually: two guide flanges on the outer ring, chamfered roller ends, bars sitting in the azimuthal gaps between rollers (use an exposed/section capture). Confirm `analysis_mass_properties` volume within ~0.3% of an analytic model (rings-as-channel + chamfered rollers + bars).

- [ ] **Step 4: Live place + screenshot 6206** (ball): confirm two shields visible on both faces, clearing the balls; volume within ~0.3%.

- [ ] **Step 5: Re-drive check:** edit a driver parameter (e.g. `bore`) via MCP; confirm the whole assembly re-drives DOF-0 (flanges/chamfers/bars/shields track).

- [ ] **Step 6: Commit any doc/provenance + push:**
```bash
git add -A && git commit -m "docs(#53): bearing-detail provenance + live-verification notes"
git push -u origin feat/53-bearing-detail
```

- [ ] **Step 7: Open the PR** (`Closes #53`), body focusing on WHAT/WHY (four representational refinements, each an independent provably-disjoint body; add-in-only). Auto-merge when green (all CI checks incl. SonarCloud Code Analysis SUCCESS). Delete the branch after merge.

---

## Self-Review

**Spec coverage:** Feature 1 flanges → Task 1; Feature 2 chamfers → Task 2; Feature 3 cage → Task 3; Feature 4 shields → Task 4; testing/live/coverage → Task 5. Global constraints (disableSketchInference, whole-body-pattern ordering, fall-back-to-plain, ε_clr) are in the Global Constraints block and each task's guard. All spec sections mapped.

**Placeholder scan:** guards, formulas, meridian layouts, and test bodies are concrete. The two "decide during Step 3" notes (flange edge dimensioning shape; shield −face negation vs bool) are genuine local implementation choices with both options stated — not deferred requirements.

**Type consistency:** `flangesFit`/`rollerChamferFits`/`cageBarsFit`/`shieldsFit` all take `ResolvedMember` and return `bool`. `buildRoller` returns `(string, error)` (feature name for pattern). `patternRollers(rm)`, `revolveFlangedOuterRing(rm)`, `revolveShields(rm)` all thread `rm`. Param names (`flange_inner_z`, `roller_chamfer`, `bar_half_angle`, `shield_near_z`, …) match between derivation, section calls, and test assertions. `math` imported once in `roller_bearing.go`.
