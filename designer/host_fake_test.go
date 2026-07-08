// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"oblikovati.org/api/wire"
)

// fakeHost is a HostCaller test double: it records each wire method it was asked to run and
// decodes the request bodies the tests assert on (the registered command, the shown
// dockable window), then returns a minimal OK reply. It replaces the cgo transport so the
// engine is exercised end-to-end without a live host. It is mutex-guarded because the
// engine dispatches some host work to background goroutines (so `go test -race` is clean).
type fakeHost struct {
	mu       sync.Mutex
	methods  []string                  // every wire method, in call order
	commands []wire.CreateCommandArgs  // decoded commands.create requests
	windows  []wire.DockableWindowSpec // decoded dockableWindows.set requests
}

// Call records the method + request and returns a canned OK reply. Unknown methods return
// an empty JSON object, which the typed client decodes to a zero-value result without error.
func (h *fakeHost) Call(method string, req []byte) ([]byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.methods = append(h.methods, method)
	switch method {
	case wire.MethodCommandsCreate:
		var a wire.CreateCommandArgs
		_ = json.Unmarshal(req, &a)
		h.commands = append(h.commands, a)
	case wire.MethodDockableWindowsSet:
		var a wire.SetDockableWindowArgs
		_ = json.Unmarshal(req, &a)
		h.windows = append(h.windows, a.Window)
	}
	return []byte("{}"), nil
}

// called reports whether the given wire method was invoked at least once.
func (h *fakeHost) called(method string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, m := range h.methods {
		if m == method {
			return true
		}
	}
	return false
}

// waitFor polls cond until it holds or a short deadline elapses, failing the test on
// timeout. Used to synchronize on host effects the engine produces from a goroutine.
func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(msg)
}
