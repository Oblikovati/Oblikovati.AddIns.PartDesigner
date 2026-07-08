// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"oblikovati.org/api/client"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// PanelID is the stable dockable-window id the add-in owns.
const PanelID = "com.oblikovati.part-designer.panel"

// ShowPanel creates (or replaces) the Part Designer dockable window. A1 renders a
// placeholder; A5 replaces placeholderControls with the cascading
// Category → Family → Standard → Size dropdowns + Place button that follow Inventor's
// "Place from Content Center" flow.
func (e *Engine) ShowPanel() (wire.OKResult, error) {
	return e.api.DockableWindows().Set(wire.DockableWindowSpec{
		ID:       PanelID,
		Title:    "Part Designer",
		Dock:     types.DockRight,
		Visible:  true,
		Controls: placeholderControls(),
	})
}

// placeholderControls is the A1 stand-in surface: a heading and a "coming soon" note. It is
// replaced by the real catalogue browser in A5.
func placeholderControls() []wire.PanelControlSpec {
	return []wire.PanelControlSpec{
		client.PanelLabel("hdr", "— Part Designer —"),
		client.PanelLabel("todo", "Standard-parts catalogue coming soon."),
	}
}
