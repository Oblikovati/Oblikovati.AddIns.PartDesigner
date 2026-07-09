// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/part-designer/designer/catalog"
)

// structuralSectionParams is the set of published parameters each structural-section generator
// drives its sketch dimensions against. A family bound to one of these generators MUST publish
// every listed param as a column, or the generator references an undefined parameter at solve time
// and placement fails. This regressed once: the AISC W family shared the i_beam generator but was
// not given root_radius after the I-beam gained its root fillet, so W-shapes could not be placed.
var structuralSectionParams = map[string][]string{
	"i_beam":  {"height", "flange_width", "web_thickness", "flange_thickness", "root_radius", "length"},
	"angle":   {"leg_a", "leg_b", "thickness", "root_radius", "toe_radius", "length"},
	"tee":     {"height", "flange_width", "web_thickness", "flange_thickness", "root_radius", "length"},
	"channel": {"height", "flange_width", "web_thickness", "flange_thickness", "length"},
}

// TestEmbeddedFamiliesPublishGeneratorParams guards the generator↔family parameter contract: every
// embedded family bound to a structural-section generator publishes all the params that generator
// drives geometry against. It is the static check the fakeHost cannot make (the fake records a
// dimension's expression string without resolving it), and it catches a family that adopts a
// generator without carrying the columns the generator now needs.
func TestEmbeddedFamiliesPublishGeneratorParams(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("catalog.Load() error = %v", err)
	}
	for _, fam := range cat.Families() {
		required, tracked := structuralSectionParams[fam.Generator]
		if !tracked {
			continue // non-section generators are not part of this contract
		}
		published := map[string]bool{}
		for _, col := range fam.Columns {
			published[col.Param] = true
		}
		for _, param := range required {
			if !published[param] {
				t.Errorf("family %q (generator %q) does not publish %q; the generator drives a "+
					"dimension against it, so placement would reference an undefined parameter",
					fam.ID, fam.Generator, param)
			}
		}
	}
}
