// SPDX-License-Identifier: GPL-2.0-only

package catalog

import "testing"

// TestTreeGroupsFamiliesByCategory checks the browse tree derived from the seed families:
// Fasteners → {Bolts → Hex Head → iso4017-hex-bolt, Nuts → Hex → din934-hex-nut}.
func TestTreeGroupsFamiliesByCategory(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	root := c.Tree()

	fasteners := child(t, root, "Fasteners")
	if len(fasteners.Children) < 2 {
		t.Fatalf("Fasteners children = %d, want at least Bolts + Nuts", len(fasteners.Children))
	}
	// Children are name-sorted: Bolts before Nuts.
	if fasteners.Children[0].Name != "Bolts" || fasteners.Children[1].Name != "Nuts" {
		t.Errorf("Fasteners children order = %q,%q, want Bolts,Nuts",
			fasteners.Children[0].Name, fasteners.Children[1].Name)
	}

	hexHead := child(t, child(t, fasteners, "Bolts"), "Hex Head")
	if !containsID(hexHead.Families, "iso4017-hex-bolt") || !containsID(hexHead.Families, "din933-hex-bolt") {
		t.Errorf("Hex Head families = %v, want both hex bolts (ISO 4017 + DIN 933)", ids(hexHead.Families))
	}
	if got := hexHead.Path.String(); got != "Fasteners/Bolts/Hex Head" {
		t.Errorf("Hex Head node path = %q, want Fasteners/Bolts/Hex Head", got)
	}

	hexNut := child(t, child(t, fasteners, "Nuts"), "Hex")
	if len(hexNut.Families) != 3 || !containsID(hexNut.Families, "din934-hex-nut") ||
		!containsID(hexNut.Families, "iso4032-hex-nut") || !containsID(hexNut.Families, "iso4035-hex-nut") {
		t.Errorf("Hex nut families = %v, want din934/iso4032/iso4035 hex-nut", ids(hexNut.Families))
	}
}

// child finds the named child of n, failing the test when absent.
func child(t *testing.T, n *CategoryNode, name string) *CategoryNode {
	t.Helper()
	for _, ch := range n.Children {
		if ch.Name == name {
			return ch
		}
	}
	t.Fatalf("node %q has no child %q", n.Name, name)
	return nil
}

// ids extracts family ids for readable assertions.
func ids(fs []*Family) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.ID
	}
	return out
}
