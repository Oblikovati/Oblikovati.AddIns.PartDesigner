// SPDX-License-Identifier: GPL-2.0-only

// Package designer is the cgo-free engine of the oblikovati-part-designer add-in. It owns
// the standards catalogue and the procedural part generators, and drives the host over the
// Apache-2.0 api/client transport to realize a placed standard part as parameters +
// sketches + features. Keeping it cgo-free makes it unit-testable on every OS with a fake
// host; the cgo shell (package main) supplies the real transport at Activate.
package designer

import (
	"encoding/json"
	"fmt"
	"sync"

	"oblikovati.org/api/client"
	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/build"
	"oblikovati.org/part-designer/designer/catalog"
)

// HostCaller is the transport the engine talks to the host through — exactly the
// api/client Caller contract, supplied by the cgo shell at Activate (or a fake in tests).
// Keeping it an interface here keeps this package cgo-free and testable.
type HostCaller interface {
	Call(method string, req []byte) ([]byte, error)
}

// Engine drives the host to browse and place standard parts: it owns the standards
// catalogue and the generator registry, and turns a chosen family+member into a placed,
// stamped, parametric part (see placement.go). The panel browser (A5) reads the catalogue;
// the ribbon button + placeholder panel are the A1 scaffold.
type Engine struct {
	host    HostCaller
	api     *client.Client
	catalog *catalog.Catalog
	catErr  error // catalogue load error, surfaced by operations that need it
	gens    *build.Registry

	mu    sync.Mutex // guards sel + bound
	sel   panelState // the panel's current cascading selection
	bound bool       // the active document is a stamped Part Designer part, so a size change re-drives it in place (Change-Size)
}

// NewEngine binds the engine to the host transport, loading the embedded standards catalogue
// and the built-in generator registry. A catalogue load failure (a malformed embedded table,
// which the build/tests guard against) is stored and surfaced by the operations that need
// it, so a bad table never crashes the host at Activate. The panel opens on the first family.
func NewEngine(host HostCaller) *Engine {
	cat, err := catalog.Load()
	e := &Engine{
		host: host, api: client.New(host),
		catalog: cat, catErr: err, gens: build.DefaultRegistry(),
	}
	e.sel = e.defaultSelection()
	return e
}

// Catalog exposes the loaded standards catalogue (nil if it failed to load) for the panel
// browser.
func (e *Engine) Catalog() *catalog.Catalog { return e.catalog }

// API exposes the underlying typed client (used by the panel + placement code).
func (e *Engine) API() *client.Client { return e.api }

// Setup performs the one-time host-facing initialization: register the ribbon command and
// show the dockable panel. It MUST NOT run on the host's session goroutine (e.g. directly
// inside the C-ABI Activate) — those host calls block until the frame loop drains the
// dispatcher, so calling them on the session goroutine before the loop starts deadlocks
// the head. The cgo shell runs Setup on its own goroutine, where the live frame loop
// drains the calls. Errors are returned for logging; partial setup never crashes the host.
func (e *Engine) Setup() error {
	if err := e.RegisterCommands(); err != nil {
		return err
	}
	_, err := e.ShowPanel()
	return err
}

// Notify receives host event bytes: a command (the Show button or the Place command) or a
// panel dropdown edit.
//
// CRITICAL: Notify is invoked ON the host's session goroutine (events are emitted from
// inside the frame loop). A host call from this goroutine blocks until the frame loop
// drains the dispatcher — which cannot happen while we're still inside it — so any host
// work is dispatched to a SEPARATE goroutine, where the live frame loop drains its calls.
func (e *Engine) Notify(ev []byte) {
	var hdr struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(ev, &hdr) != nil {
		return
	}
	switch hdr.Type {
	case wire.EventCommandStarted:
		e.handleCommand(ev)
	case wire.EventPanelValueChanged:
		e.handlePanelEdit(ev)
	case wire.EventDocumentActivated:
		go e.bindActiveDocument() // reads the stamp + re-shows off the session goroutine
	}
}

// handleCommand routes the add-in's commands: the "Part Designer" button (re)opens the
// panel; the Place command places the current selection. Both make host calls (which
// deadlock inline — see Notify), so they run on their own goroutines.
func (e *Engine) handleCommand(ev []byte) {
	var c struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(ev, &c) != nil {
		return
	}
	switch c.Command {
	case ShowCommandID:
		go func() { _, _ = e.ShowPanel() }()
	case PlaceCommandID:
		go e.placeSelection()
	}
}

// handlePanelEdit applies one dropdown edit to the cascading selection and re-shows the panel
// with the updated downstream choices. The selection mutation is cheap (no host call) and
// safe on the session goroutine; the re-show makes host calls, so it runs on its own.
func (e *Engine) handlePanelEdit(ev []byte) {
	var p struct {
		WindowID  string `json:"windowId"`
		ControlID string `json:"controlId"`
		Value     string `json:"value"`
	}
	if json.Unmarshal(ev, &p) != nil || p.WindowID != PanelID {
		return
	}
	e.mu.Lock()
	e.applySelection(p.ControlID, p.Value)
	resize := e.bound && p.ControlID == sizeControlID
	memberKey := e.sel.memberKey
	e.mu.Unlock()
	// When the panel is bound to a stamped part, changing its Size re-drives that document in
	// place (Change-Size) rather than only updating the browse selection.
	go func() {
		if resize {
			_ = e.ChangeSize(memberKey)
		}
		_, _ = e.ShowPanel()
	}()
}

// placeSelection places the panel's current Part + Size selection (the Place button and the
// headless Place command both land here), reporting the outcome on the host status bar so a
// failure is visible rather than silently producing nothing.
func (e *Engine) placeSelection() {
	e.mu.Lock()
	sel := e.sel
	e.mu.Unlock()
	if sel.familyID == "" || sel.memberKey == "" {
		_, _ = e.api.Status().SetText("Part Designer: choose a part and size first")
		return
	}
	res, err := e.Place(sel.familyID, sel.memberKey)
	if err != nil {
		_, _ = e.api.Status().SetText("Part Designer: " + err.Error())
		return
	}
	msg := fmt.Sprintf("Part Designer: placed %s %s", sel.familyID, sel.memberKey)
	if res.Occurrence {
		msg += " (occurrence added to the active assembly)"
	}
	_, _ = e.api.Status().SetText(msg)
	// Bind the panel to the just-placed part so an immediate Size change re-drives it. The
	// document.activated event fired during creation, BEFORE the stamp was applied, so it left
	// the panel unbound; reconcile now that the part is stamped (unless an assembly is active).
	e.bindActiveDocument()
}
