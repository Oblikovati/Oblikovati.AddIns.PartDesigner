//go:build js && wasm

// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"sync"

	"oblikovati.org/api/client"
	"oblikovati.org/api/wasmaddin"
	"oblikovati.org/part-designer/designer"
)

const addInID = "com.oblikovati.part-designer"

type wasmPartDesigner struct {
	mu     sync.Mutex
	engine *designer.Engine
}

func (a *wasmPartDesigner) ID() string       { return addInID }
func (a *wasmPartDesigner) Manifest() string { return manifestJSON }

func (a *wasmPartDesigner) Activate(caller client.Caller) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	eng := designer.NewEngine(caller)
	a.engine = eng
	go func() { _ = eng.Setup() }()
	return nil
}

func (a *wasmPartDesigner) Deactivate() error {
	a.mu.Lock()
	a.engine = nil
	a.mu.Unlock()
	return nil
}

func (a *wasmPartDesigner) Notify(event []byte) {
	a.mu.Lock()
	eng := a.engine
	a.mu.Unlock()
	if eng != nil {
		eng.Notify(event)
	}
}

func main() {
	wasmaddin.Serve(&wasmPartDesigner{})
}
