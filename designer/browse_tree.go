// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// familyTreeNodes converts the catalog's category tree into wire TreeNodes for a PanelTree, with
// FAMILY leaves: interior nodes are categories (ID = category path) and each family hangs as a
// childless leaf whose ID is the family ID, so a node-click identifies a family directly. The top
// two category levels are seeded open (Expanded) for a useful initial view; deeper levels start
// closed. The host owns expand/collapse after first render.
func familyTreeNodes(root *catalog.CategoryNode) []wire.TreeNode {
	return childNodes(root, 0)
}

// childNodes builds the wire nodes for one category node's children (sub-categories then families).
func childNodes(node *catalog.CategoryNode, depth int) []wire.TreeNode {
	out := make([]wire.TreeNode, 0, len(node.Children)+len(node.Families))
	for _, ch := range node.Children {
		out = append(out, wire.TreeNode{
			ID:       ch.Path.String(),
			Label:    ch.Name,
			Expanded: depth < 2,
			Children: childNodes(ch, depth+1),
		})
	}
	for _, fam := range node.Families {
		out = append(out, wire.TreeNode{ID: fam.ID, Label: familyLabel(fam)})
	}
	return out
}

// filteredTree returns a category tree containing only the families that pass the current filters,
// so the browse tree shows the same set the old Part dropdown did.
func (e *Engine) filteredTree(sel panelState) *catalog.CategoryNode {
	return catalog.TreeOf(e.filteredFamilies(sel))
}
