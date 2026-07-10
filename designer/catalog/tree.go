// SPDX-License-Identifier: GPL-2.0-only

package catalog

import "sort"

// CategoryNode is one node of the browse tree. Interior nodes carry Children; a node whose
// Path equals a family's Category carries that family in Families. The root is a synthetic
// node with an empty Path.
type CategoryNode struct {
	Name     string
	Path     CategoryPath
	Children []*CategoryNode
	Families []*Family
}

// Tree assembles the category tree from every family, mirroring Inventor's data-driven
// category browser (the tree is derived from the families, not a hard-coded enum).
func (c *Catalog) Tree() *CategoryNode {
	return TreeOf(c.Families())
}

// TreeOf assembles a category tree from an explicit set of families (used to show a filtered
// subtree, e.g. the browse tree under the Category/Standard/Search filters). Children are sorted
// by name and families by id, so the tree is deterministic.
func TreeOf(families []*Family) *CategoryNode {
	root := &CategoryNode{}
	for _, fam := range families {
		node := root.descend(fam.Category)
		node.Families = append(node.Families, fam)
	}
	root.sortRecursive()
	return root
}

// descend walks from the receiver down to the node for path, creating missing nodes.
func (n *CategoryNode) descend(path CategoryPath) *CategoryNode {
	node := n
	for i, seg := range path {
		node = node.childNamed(seg, path[:i+1])
	}
	return node
}

// childNamed returns the child with the given name, creating it (with the full path so far)
// when absent.
func (n *CategoryNode) childNamed(name string, path CategoryPath) *CategoryNode {
	for _, ch := range n.Children {
		if ch.Name == name {
			return ch
		}
	}
	child := &CategoryNode{Name: name, Path: append(CategoryPath{}, path...)}
	n.Children = append(n.Children, child)
	return child
}

// sortRecursive orders children by name and families by id, depth-first.
func (n *CategoryNode) sortRecursive() {
	sort.Slice(n.Children, func(i, j int) bool { return n.Children[i].Name < n.Children[j].Name })
	sort.Slice(n.Families, func(i, j int) bool { return n.Families[i].ID < n.Families[j].ID })
	for _, ch := range n.Children {
		ch.sortRecursive()
	}
}
