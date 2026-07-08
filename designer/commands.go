// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// ShowCommandID is the single ribbon command the add-in registers: it opens the Part
// Designer window. Browsing and placement happen INSIDE that window, so the ribbon stays a
// single button rather than one-button-per-part.
const ShowCommandID = "PartDesigner.Show"

// RegisterCommands registers the add-in's one ribbon button — a "Part Designer" button
// (with its own SVG glyph) that opens the standard-parts window. Headless place commands
// for scripting/MCP are added in A5, alongside the catalogue that backs them.
func (e *Engine) RegisterCommands() error {
	_, err := e.api.Commands().Create(wire.CreateCommandArgs{
		ID:          ShowCommandID,
		DisplayName: "Part Designer",
		Category:    "Part Designer",
		Tooltip:     "Open the Part Designer window to browse and place standard parts.",
		IconSVG:     partIconSVG,
		ButtonStyle: types.LargeIconButton,
	})
	return err
}

// partIconSVG is the ribbon button glyph: a hex-bolt head + threaded shank, in the host's
// icon convention (24×24, #00ff00 backplate, black primary linework, #ff0000 accent),
// recoloured per theme (Oblikovati#671).
const partIconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">` +
	`<rect x="1" y="1" width="22" height="22" rx="4" fill="#00ff00" stroke="none"/>` +
	`<polygon points="8,3 14,3 17,7.5 14,12 8,12 5,7.5"/>` +
	`<path d="M9.5 12 V20 M12.5 12 V20"/>` +
	`<path d="M9.5 14.5 H12.5 M9.5 16.5 H12.5 M9.5 18.5 H12.5"/>` +
	`<circle cx="11" cy="7.5" r="1.5" fill="#ff0000" stroke="none"/></svg>`
