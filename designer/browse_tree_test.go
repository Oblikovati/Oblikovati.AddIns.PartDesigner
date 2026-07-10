// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// familyTreeNodes must turn the catalog category tree into wire TreeNodes whose LEAVES are
// families (ID = family ID) nested under their category path, so a tree-node click identifies a
// family directly (no label lookup).
func TestFamilyTreeNodesHasFamilyLeaves(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("catalog.Load: %v", err)
	}
	nodes := familyTreeNodes(cat.Tree())
	fam := findFamilyLeaf(nodes)
	if fam == nil {
		t.Fatal("no family leaf found in tree")
	}
	if _, ok := cat.Family(fam.ID); !ok {
		t.Fatalf("family leaf ID %q is not a real family", fam.ID)
	}
	if len(fam.Children) != 0 {
		t.Fatalf("family leaf %q has children, want leaf", fam.ID)
	}
}

// findFamilyLeaf returns the first depth-first node with no children (a family leaf).
func findFamilyLeaf(nodes []wire.TreeNode) *wire.TreeNode {
	for i := range nodes {
		if len(nodes[i].Children) == 0 {
			return &nodes[i]
		}
		if leaf := findFamilyLeaf(nodes[i].Children); leaf != nil {
			return leaf
		}
	}
	return nil
}
