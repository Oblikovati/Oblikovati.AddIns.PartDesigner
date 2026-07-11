// SPDX-License-Identifier: GPL-2.0-only

package catalog

import "testing"

// TestFastenerSizeCoverage guards the 2026-07 size-coverage expansion against a silent shrink:
// the fastener families were grown from a 4–6-size seed to each standard's full preferred series
// (see data/SOURCES.md). If a family's members are accidentally truncated back toward the seed,
// this fails. It checks the aggregate floor plus representative per-family endpoints (the smallest
// and largest sizes), so a regression at either end of a series is caught.
func TestFastenerSizeCoverage(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	total := 0
	for _, f := range c.Families() {
		if len(f.Category) > 0 && f.Category[0] == "Fasteners" {
			total += len(f.Members)
		}
	}
	// The expanded catalogue carries 261 fastener members; a drop well below that means a family
	// was truncated. Floor set with headroom so adding sizes never trips it.
	if total < 240 {
		t.Errorf("total fastener members = %d, want >= 240 (the expanded preferred-series coverage)", total)
	}

	// Representative endpoints: metric spans M1.6..M48, inch spans up to 2 in.
	endpoints := []struct{ family, key string }{
		{"iso4017-hex-bolt", "d=1.6,l=8"},  // smallest metric bolt
		{"iso4017-hex-bolt", "d=48,l=240"}, // largest metric bolt
		{"din933-hex-bolt", "d=48,l=240"},  // DIN twin matches
		{"iso4032-hex-nut", "d=48"},        // largest metric nut
		{"iso4762-socket-screw", "d=48,l=240"},
		{"iso7089-washer", "size=M36"},                  // washer series tops out at M36
		{"ansi-b18-2-1-hex-bolt", "thread=2-4 1/2,l=8"}, // largest inch bolt
	}
	assertEndpoints(t, c, endpoints)
}

// TestShaftSizeCoverage is the shaft-parts twin of TestFastenerSizeCoverage: the retaining rings,
// pins, and keys were grown from a 5–8-size seed to each standard's tabulated series (see
// data/SOURCES.md). It guards the aggregate floor plus per-family smallest/largest endpoints so a
// silent truncation at either end of a series is caught.
func TestShaftSizeCoverage(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	total := 0
	for _, f := range c.Families() {
		if len(f.Category) > 0 && f.Category[0] == "Shaft Parts" {
			total += len(f.Members)
		}
	}
	// The expanded shaft catalogue carries 207 members; floor set with headroom.
	if total < 190 {
		t.Errorf("total shaft members = %d, want >= 190 (the expanded standard-series coverage)", total)
	}

	endpoints := []struct{ family, key string }{
		{"din471-external-circlip", "d=3"},          // smallest DIN 471 shaft ring
		{"din471-external-circlip", "d=50"},         // largest
		{"din472-internal-circlip", "d=8"},          // smallest DIN 472 bore ring
		{"din472-internal-circlip", "d=62"},         // largest
		{"iso2338-dowel-pin", "d=1,l=5"},            // smallest ISO 2338 dowel
		{"iso2338-dowel-pin", "d=50,l=200"},         // largest
		{"iso1234-split-pin", "d=0.6,l=6"},          // smallest ISO 1234 split pin
		{"iso1234-split-pin", "d=20,l=200"},         // largest
		{"iso2341-clevis-pin", "d=3,l=12"},          // smallest ISO 2341 clevis pin
		{"din6885-parallel-key", "b=2,h=2,l=6"},     // smallest DIN 6885 key
		{"din6885-parallel-key", "b=50,h=28,l=160"}, // largest
		{"din6887-gib-head-key", "b=50,h=28,l=220"}, // largest DIN 6887 gib-head key
	}
	assertEndpoints(t, c, endpoints)
}

// assertEndpoints checks each (family, memberKey) pair resolves in the loaded catalogue — a
// regression guard shared by the per-category coverage tests.
func assertEndpoints(t *testing.T, c *Catalog, endpoints []struct{ family, key string }) {
	t.Helper()
	for _, e := range endpoints {
		fam, ok := c.Family(e.family)
		if !ok {
			t.Errorf("family %q not loaded", e.family)
			continue
		}
		if _, ok := fam.Member(e.key); !ok {
			t.Errorf("%s: expected size %q missing (coverage shrank?)", e.family, e.key)
		}
	}
}
