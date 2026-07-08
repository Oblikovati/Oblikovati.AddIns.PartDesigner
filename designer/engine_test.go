// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"encoding/json"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestSetupRegistersButtonAndShowsPanel is the A1 acceptance check: activating the add-in
// must register exactly the "Part Designer" ribbon button (with its glyph + large style)
// and show the dockable panel — the two host-facing effects the cgo shell triggers on a
// background goroutine at Activate.
func TestSetupRegistersButtonAndShowsPanel(t *testing.T) {
	host := &fakeHost{}
	if err := NewEngine(host).Setup(); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	if len(host.commands) != 1 {
		t.Fatalf("commands registered = %d, want 1 (%v)", len(host.commands), host.methods)
	}
	cmd := host.commands[0]
	if cmd.ID != ShowCommandID {
		t.Errorf("command ID = %q, want %q", cmd.ID, ShowCommandID)
	}
	if cmd.DisplayName != "Part Designer" {
		t.Errorf("command DisplayName = %q, want %q", cmd.DisplayName, "Part Designer")
	}
	if cmd.ButtonStyle != types.LargeIconButton {
		t.Errorf("command ButtonStyle = %v, want LargeIconButton", cmd.ButtonStyle)
	}
	if cmd.IconSVG == "" {
		t.Error("command IconSVG is empty; the ribbon button must ship its glyph")
	}

	if len(host.windows) != 1 {
		t.Fatalf("dockable windows shown = %d, want 1", len(host.windows))
	}
	win := host.windows[0]
	if win.ID != PanelID {
		t.Errorf("panel ID = %q, want %q", win.ID, PanelID)
	}
	if !win.Visible {
		t.Error("panel Visible = false, want true")
	}
	if len(win.Controls) == 0 {
		t.Error("panel has no controls; want at least the placeholder heading")
	}
}

// TestNotifyShowCommandReopensPanel checks the event path: a command.started for the Part
// Designer button re-shows the panel. The work is dispatched to a goroutine (host calls on
// the session goroutine deadlock the head), so the test synchronizes on the panel appearing.
func TestNotifyShowCommandReopensPanel(t *testing.T) {
	host := &fakeHost{}
	e := NewEngine(host)

	ev, err := json.Marshal(map[string]string{"type": wire.EventCommandStarted, "command": ShowCommandID})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	e.Notify(ev)

	waitFor(t, func() bool { return host.called(wire.MethodDockableWindowsSet) },
		"Notify(command.started) never re-showed the panel")
}

// TestNotifyIgnoresUnknownEvents ensures non-command events and unknown commands make no
// host calls — the engine must not react to traffic it does not own.
func TestNotifyIgnoresUnknownEvents(t *testing.T) {
	host := &fakeHost{}
	e := NewEngine(host)

	e.Notify([]byte(`{"type":"document.activated","id":7}`))
	e.Notify([]byte(`not json`))
	e.Notify([]byte(`{"type":"command.started","command":"SomeOther.Command"}`))

	if len(host.methods) != 0 {
		t.Errorf("engine made host calls for unowned events: %v", host.methods)
	}
}
