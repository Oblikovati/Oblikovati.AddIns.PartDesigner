# Circlip lug-ears (#61) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Add two flush plier-hole lug eyes at the split-gap edges of the retaining ring.

**Architecture:** Add-in-only (no API/host change). Two derived params + a Go guard, a new
`section_ear.go` (flat annular eye placed by polar Distance+Angle on an offset plane, hole
built into the profile — no boolean), wired into `circlip.go` `Build` after the ring revolve.

**Tech Stack:** Go, `oblikovati.org/api/client`, `fakeHost` double, kernel sketch solver.

## Global Constraints

- No boolean; each ear is one **annulus** (two concentric circles). Independent body,
  `Operation:"new"`, overlapping the ring band at the gap edge.
- DOF-0 parametric; the eye sketch reaches `AssertFullyConstrained()`.
- **Unit-strict evaluator:** every additive/clearance constant carries a unit (`"0.3 mm"`);
  a bare literal + length → 0. (ASME rings are `in`-unit — the live test verifies the `mm`
  constant converts; if not, the guard uses per-family units.)
- **Fall-back-to-plain:** `circlipEarsFit(rm)` gates the ears; on failure the ring builds
  with no ears.
- SPDX GPL-2.0-only; funcs 4–20 lines; explicit types; `fakeHost` tests; coverage >80%.

**Existing to reuse:** `circlip.go` (`Circlip.Build`, `splitGapAngle="330 deg"`);
`OffsetPlaneSketch(offsetExpr)`, `ExtrudeDirected(sk, distanceExpr, op, direction)` (symmetric
span is the TOTAL), `DeriveParam`, `AssertFullyConstrained`; sketch client `AddPoint`,
`AddLine(a,b,construction)`, `AddCircleByCenterRadius(center,radius,construction)`,
`Constrain().{Ground,Coincident,Horizontal,Fix}`, `Dimension().{Distance,Angle,Diameter}`;
`ResolvedMember.Value(col)`, `rm.Family.Category` (a `catalog.CategoryPath` ending in
`External`/`Internal`). Model the guard-with-fallback on `roller_cage.go`'s `rollerCageBarsFit`.

---

### Task 1: Ear parameters + `circlipEarsFit` guard (external/internal)

**Files:**
- Modify: `designer/build/circlip.go` (add `deriveCirclipEarParams`, `circlipEarsFit`, `circlipIsExternal`)
- Test: `designer/build/circlip_test.go`

**Interfaces:**
- Produces: `circlipIsExternal(rm ResolvedMember) bool`; `deriveCirclipEarParams(b *PartBuilder, external bool) error`; `circlipEarsFit(rm ResolvedMember) bool`.

- [ ] **Step 1: Write failing tests** (the circlip test helper is `circlipMember` — read `circlip_test.go` for its exact signature; add a `category` argument or a second helper if it doesn't set `Family.Category`):

```go
func TestCirclipEarParamsExternal(t *testing.T) {
	h := &fakeHost{dof: 0}
	// DIN 471 d30: di=28.6, do=38.6, s=1.5, external
	if err := (Circlip{}).Build(newBuilder(h, catalog.UnitsMillimetre), circlipExtMember("d30", 30, 28.6, 38.6, 1.5)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "ear_band_width", "(outer_dia - inner_dia) / 2")
	assertParam(t, h.added, "eye_outer_dia", "ear_band_width * 1.0")
	assertParam(t, h.added, "plier_hole_dia", "eye_outer_dia * 0.45")
	assertParam(t, h.added, "eye_radius_pos", "outer_dia / 2 + eye_outer_dia * 0.3")
}

func TestCirclipEarParamsInternal(t *testing.T) {
	h := &fakeHost{dof: 0}
	// DIN 472 d20: di=15 (bore side)… internal kEye=0.9, eye projects inward
	if err := (Circlip{}).Build(newBuilder(h, catalog.UnitsMillimetre), circlipIntMember("d20", 20, 21.0, 15.0, 1.0)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "eye_outer_dia", "ear_band_width * 0.9")
	assertParam(t, h.added, "eye_radius_pos", "inner_dia / 2 - eye_outer_dia * 0.3")
}

func TestCirclipEarsFit(t *testing.T) {
	if !circlipEarsFit(circlipExtMember("d30", 30, 28.6, 38.6, 1.5)) {
		t.Error("ears should fit DIN471 d30 (external)")
	}
	if !circlipEarsFit(circlipIntMember("d20", 20, 21.0, 15.0, 1.0)) {
		t.Error("ears should fit DIN472 d20 (internal, the binding worst case, 15.4% margin)")
	}
	// Synthetic tiny internal: small bore + wide band → ears collide → skip.
	if circlipEarsFit(circlipIntMember("x", 6, 9.0, 4.0, 1.0)) {
		t.Error("ears must NOT fit a tiny internal ring; want skip-ears fallback")
	}
}
```

- [ ] **Step 2: Run — FAIL** (`deriveCirclipEarParams`/`circlipEarsFit` undefined): `cd designer && go test ./build/ -run Circlip -v`.

- [ ] **Step 3: Implement in `circlip.go`:**

```go
// Lug-ear proportions (representational — DIN 471/472 catalogues carry no lug/hole columns;
// see the geometry-math-advisor derivation, #61). Every additive constant carries a unit
// because the parameter evaluator is unit-strict (a bare literal + a length evaluates to 0).
const (
	earEyeFracExternal = "1.0"    // eye Ø as a fraction of the ring's radial band width
	earEyeFracInternal = "0.9"    // internal ears sit at a smaller radius → trimmed to avoid collision
	earHoleFrac        = "0.45"   // plier-hole Ø as a fraction of the eye Ø (leaves a real rim)
	earOutwardFrac     = "0.3"    // eye centre this fraction of its own Ø past the band edge (60% overlap)
	earMinClr          = 0.3      // mm; rim floor + two-ear non-collision clearance (float, for the guard)
)

// circlipIsExternal reports whether the ring is an external (shaft) ring — ears project radially
// OUTWARD — vs an internal (bore) ring — ears project INWARD. Keyed off the family category
// ("Shaft Parts/Retaining Rings/External" | ".../Internal"), the same signal the data encodes.
func circlipIsExternal(rm ResolvedMember) bool {
	c := rm.Family.Category
	return len(c) == 0 || c[len(c)-1] != "Internal"
}

// deriveCirclipEarParams publishes the lug-eye geometry: the ring's radial band width, the eye and
// plier-hole diameters, and the eye-centre radius (beyond OD for external, inside ID for internal).
func deriveCirclipEarParams(b *PartBuilder, external bool) error {
	eyeFrac := earEyeFracInternal
	radius := "inner_dia / 2 - eye_outer_dia * " + earOutwardFrac
	if external {
		eyeFrac, radius = earEyeFracExternal, "outer_dia / 2 + eye_outer_dia * "+earOutwardFrac
	}
	steps := []struct{ name, expr string }{
		{"ear_band_width", "(outer_dia - inner_dia) / 2"},
		{"eye_outer_dia", "ear_band_width * " + eyeFrac},
		{"plier_hole_dia", "eye_outer_dia * " + earHoleFrac},
		{"eye_radius_pos", radius},
	}
	for _, s := range steps {
		if err := b.DeriveParam(s.name, s.expr); err != nil {
			return err
		}
	}
	return nil
}

// circlipEarsFit reports whether both lug ears fit: a positive eye rim, and the two ears (30° apart
// on the eye-centre circle) not colliding — 2·R·sin15° ≥ eye_dia + clearance. Internal rings are
// the binding case (smaller R). Mirrors the parametric formulas so the Go build decision matches.
func circlipEarsFit(rm ResolvedMember) bool {
	di, do := rm.Value("di"), rm.Value("do")
	band := (do - di) / 2
	external := circlipIsExternal(rm)
	eye := band
	if !external {
		eye = band * 0.9
	}
	hole := eye * 0.45
	var r float64
	if external {
		r = do/2 + eye*0.3
	} else {
		r = di/2 - eye*0.3
	}
	const sin15 = 0.2588190451
	rimOK := hole+2*earMinClr <= eye
	noCollide := 2*r*sin15 >= eye+earMinClr
	posRadius := r-eye/2 > 0
	return rimOK && noCollide && posRadius
}
```

(NOTE: read `circlip_test.go`'s member helper and the JSON to confirm the column names — the
data uses `di`/`do`/`s` mapped to params `inner_dia`/`outer_dia`/`thickness`; `rm.Value` takes
the COLUMN name (`"di"`,`"do"`), the param expression uses the PARAM name (`inner_dia`). Verify
which the internal `di`/`do` are for DIN 472 — for a bore ring `do` may be the larger free
diameter; use whichever column the guard needs so external/internal both compute a positive band.)

- [ ] **Step 4: Run — PASS:** `go test ./build/ -run Circlip -v`, then `go test ./build/`.

- [ ] **Step 5: Commit:**
```bash
git add designer/build/circlip.go designer/build/circlip_test.go
git commit -m "feat(#61): circlip lug-ear parameters + fit guard (external/internal)"
```

---

### Task 2: `GroundedEyeSection` + circle-at-point helper + wire two ears into `Build`

**Files:**
- Create: `designer/build/section_ear.go`
- Modify: `designer/build/circlip.go` (`buildCirclipEar`; call it twice in `Build` after the revolve, guarded)
- Test: `designer/build/circlip_test.go`

**Interfaces:**
- Produces: `(*SketchContext).GroundedEyeSection(centreRadius, azimuth, eyeDia, holeDia string) error`; `(*SketchContext).circleAtPoint(centre uint64, diameterExpr string) error`; `(*PartBuilder).buildCirclipEar(azimuthExpr string) error`.
- Consumes: `OffsetPlaneSketch`, `ExtrudeDirected`, the Task-1 params.

- [ ] **Step 1: Write failing tests:**

```go
// Two ears are extruded (thickness/new/symmetric) AFTER the ring revolve; the ring is 1 revolve.
func TestCirclipBuildsTwoEars(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (Circlip{}).Build(newBuilder(h, catalog.UnitsMillimetre), circlipExtMember("d30", 30, 28.6, 38.6, 1.5)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if len(h.revolves) != 1 {
		t.Fatalf("revolves = %d, want 1 (the ring)", len(h.revolves))
	}
	if len(h.extrudes) != 2 {
		t.Fatalf("extrudes = %d, want 2 (the two lug ears)", len(h.extrudes))
	}
	for i, e := range h.extrudes {
		if e.Distance != "thickness" || e.Operation != "new" || e.Direction != "symmetric" {
			t.Errorf("ear extrude[%d] = %+v, want thickness/new/symmetric", i, e)
		}
	}
}

// A member that fails the fit guard builds the ring only — no ear extrudes.
func TestCirclipSkipsEarsWhenUnfit(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (Circlip{}).Build(newBuilder(h, catalog.UnitsMillimetre), circlipIntMember("x", 6, 9.0, 4.0, 1.0)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if len(h.extrudes) != 0 {
		t.Errorf("extrudes = %d, want 0 (ears skipped for an unfit ring)", len(h.extrudes))
	}
}
```

- [ ] **Step 2: Run — FAIL:** `go test ./build/ -run Circlip -v`.

- [ ] **Step 3: Create `section_ear.go`.** `GroundedEyeSection` on the current (offset-plane) sketch: ground an origin point at (0,0); a **fixed** construction reference line origin→(1,0) (the +X reference the `Angle` measures from — `Fix` its far endpoint so the reference is rigid); a construction radial line origin→centre-point (centre seeded at a non-degenerate polar guess); `Dimension().Distance(origin, centre, centreRadius)` + `Dimension().Angle(refLineEntity, radialLineEntity, azimuth)` to pin the centre DOF-0; then `circleAtPoint(centre, eyeDia)` and `circleAtPoint(centre, holeDia)` — each adds a circle (seeded near the centre) and `Coincident`s its centre onto the eye-centre point, then `Diameter`-dimensions it. DOF-0 count: origin grounded, ref line fixed, centre pinned by Distance+Angle (2), each circle centre Coincident (2) + Diameter (1). Reuse `AddLine`/`AddPoint`/`AddCircleByCenterRadius`/`Constrain`/`Dimension` as in `GroundedRadialSection`/`GroundedCircle`. Two concentric circles = the annulus (no boolean).

Add to `circlip.go`:

```go
// buildCirclipEar builds one plier-lug eye at the given gap-edge azimuth: a flat annulus on a
// work plane at the ring's axial mid-level, extruded through the ring thickness as a separate body
// overlapping the band end. Called once per gap edge (0° and splitGapAngle).
func (b *PartBuilder) buildCirclipEar(azimuthExpr string) error {
	sk, err := b.OffsetPlaneSketch("thickness / 2")
	if err != nil {
		return err
	}
	if err := sk.GroundedEyeSection("eye_radius_pos", azimuthExpr, "eye_outer_dia", "plier_hole_dia"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	return b.ExtrudeDirected(sk, "thickness", "new", "symmetric")
}
```

Wire into `Circlip.Build` after the ring `Revolve`:

```go
	if _, err = b.Revolve(sk, "origin/axis/z", splitGapAngle, "new"); err != nil {
		return err
	}
	if !circlipEarsFit(rm) {
		return nil
	}
	if err := deriveCirclipEarParams(b, circlipIsExternal(rm)); err != nil {
		return err
	}
	if err := b.buildCirclipEar("0 deg"); err != nil {
		return err
	}
	return b.buildCirclipEar(splitGapAngle)
```

(`ExtrudeDirected` must return the same way the test observes `h.extrudes` — confirm its
signature/recording in `builder.go`/`host_fake_test.go`; if it doesn't record Direction, use
`ExtrudeNamed(sk,"thickness","new","symmetric")` and adjust the assertion accordingly.)

- [ ] **Step 4: Run — PASS:** `go test ./build/ -run Circlip -v`, then `go test ./build/`.

- [ ] **Step 5: Kernel diagnostic** (host `model/sketch`): build the eye sketch (offset plane,
  grounded origin, fixed ref line, centre via Distance+Angle, two Coincident circles) for DIN 471
  d30 and confirm `DOF=0, Redundant=0, Converged`, and that the solved centre sits at the intended
  polar `(eye_radius_pos, azimuth)`. Delete the diagnostic after.

- [ ] **Step 6: Lint + commit:**
```bash
cd designer && golangci-lint run ./build/ && cd ..
git add designer/build/section_ear.go designer/build/circlip.go designer/build/circlip_test.go
git commit -m "feat(#61): plier-hole lug eyes at the split-gap edges

Flat annular eye (outer disc minus concentric plier hole — no boolean)
placed by polar Distance+Angle on an offset plane at the ring's axial
mid-level, extruded through the thickness. One eye at each gap edge
(0deg and splitGapAngle); circlipEarsFit skips them on an unfit ring."
```

---

### Task 3: Live verification + PR

- [ ] **Step 1: Full suite + lint + coverage:** `cd designer && go test ./... -coverprofile=/tmp/c.out && go tool cover -func=/tmp/c.out | tail -1 && golangci-lint run ./... && gofmt -l .` — all green, `build/` coverage >80% (add fakeHost tests for any uncovered new branch, esp. the internal path + skip-ears fallback).

- [ ] **Step 2: `make install`; launch head + MCP bridge** (Popen-from-one-foreground pattern; head is current at API 0.134). Placement recipe: `set_panel_value catalog=<familyID>`, `members=<memberKey>`, `execute_command PartDesigner.Place`. Family IDs: `din471-external-circlip`, `din472-internal-circlip`, `<asme id>`.

- [ ] **Step 3: WINDING-SIGN CHECK (critical).** Place **DIN 471 d30** (external); capture a top/iso screenshot. Confirm the two eyes bracket the **actual band-terminus edges** (one at the +X/0° edge, one at the 330° edge on the band side) — NOT mirrored to +30° on the empty gap side. If ear B is on the wrong side, negate/adjust `ear_b_azimuth` (e.g. `-30 deg` vs `330 deg`) and re-verify. Confirm each eye shows its plier hole and touches the band.

- [ ] **Step 4: Internal + ASME.** Place a **DIN 472** internal ring → eyes project inward, clear each other and the axis. Place an **ASME** ring → verifies the `mm` clearance constant converts against the `in` family (if the ears wrongly skip or the guard misbehaves, switch `earMinClr` to a per-family unit). Per-body volumes: 2 ear bodies present, each ≈ `π·((eye/2)² − (hole/2)²)·thickness`. Re-drive a driver param (edit `nominal_dia`/size) → recomputes DOF-0.

- [ ] **Step 5: Commit any provenance note + push; open PR** (`Closes #61`, WHAT/WHY body). Auto-merge when green (all CI checks SUCCESS); delete branch.

---

## Self-Review

**Spec coverage:** ear params + external/internal + guard → Task 1; eye section + circle-at-point
+ two-ear wiring + fallback → Task 2; live winding-sign/ASME/internal verification + PR → Task 3.
Global constraints (no boolean annulus, unit-strict constants, fall-back-to-plain, DOF-0) are in
the Global Constraints block and each task.

**Placeholder scan:** guard/param formulas and test bodies are concrete. The "read the member
helper / confirm di-vs-do / confirm ExtrudeDirected recording" notes are genuine
verify-against-existing-code steps, not deferred requirements.

**Type consistency:** `circlipIsExternal`/`circlipEarsFit` take `ResolvedMember`→`bool`;
`deriveCirclipEarParams(b, external bool)`; `buildCirclipEar(azimuthExpr string)`;
`GroundedEyeSection(centreRadius, azimuth, eyeDia, holeDia string)`; `circleAtPoint(centre uint64,
diameterExpr string)`. Param names (`ear_band_width`, `eye_outer_dia`, `plier_hole_dia`,
`eye_radius_pos`) consistent across derivation, section calls, and tests.
