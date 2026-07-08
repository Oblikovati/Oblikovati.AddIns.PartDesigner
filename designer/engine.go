// SPDX-License-Identifier: GPL-2.0-only

// Package designer is the cgo-free engine of the oblikovati-part-designer add-in. It owns
// the standards catalogue and the procedural part generators, and drives the host over the
// Apache-2.0 api/client transport to realize a placed standard part as parameters +
// sketches + features. Keeping it cgo-free makes it unit-testable on every OS with a fake
// host; the cgo shell (package main) supplies the real transport at Activate.
package designer

import (
	"encoding/json"

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
}

// NewEngine binds the engine to the host transport, loading the embedded standards catalogue
// and the built-in generator registry. A catalogue load failure (a malformed embedded table,
// which the build/tests guard against) is stored and surfaced by the operations that need
// it, so a bad table never crashes the host at Activate.
func NewEngine(host HostCaller) *Engine {
	cat, err := catalog.Load()
	return &Engine{
		host: host, api: client.New(host),
		catalog: cat, catErr: err, gens: build.DefaultRegistry(),
	}
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

// Notify receives host event bytes. The "Part Designer" command re-opens the panel;
// everything else is ignored for now.
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
	if hdr.Type == wire.EventCommandStarted {
		e.handleCommand(ev)
	}
}

// handleCommand routes the add-in's commands. The "Part Designer" button (re)opens the
// dockable panel; ShowPanel makes host calls (which deadlock inline — see Notify), so it
// runs on its own goroutine.
func (e *Engine) handleCommand(ev []byte) {
	var c struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(ev, &c) != nil {
		return
	}
	if c.Command == ShowCommandID {
		go func() { _, _ = e.ShowPanel() }()
	}
}
