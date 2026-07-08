// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// errFake is returned by the fake host for a method configured to fail.
var errFake = errors.New("fake host: forced failure")

// fakeHost is the HostCaller double for the engine tests. It models just enough host state
// for the placement flow — a document table (with the active flag), an attribute store, and
// the set of parameters on the document — and records the calls the tests assert on
// (registered command, shown panel, published parameters, placed occurrences). It is
// mutex-guarded because the engine dispatches some work to background goroutines.
type fakeHost struct {
	mu sync.Mutex

	nextDocID  uint64
	docs       []wire.DocumentInfo // document table; Active marks the current document
	attrs      map[string]string   // "doc/set/name" -> string attribute value
	params     map[string]bool     // parameter names currently on the document
	dof        int                 // DOF returned by sketch.constraintStatus (0 = constrained)
	failMethod string              // when non-empty, this wire method returns an error

	methods  []string
	commands []wire.CreateCommandArgs
	windows  []wire.DockableWindowSpec
	added    []wire.ParameterSetArgs
	set      []wire.ParameterSetArgs
	placed   []wire.PlaceOccurrenceArgs
	updates  int
}

func newFakeHost() *fakeHost {
	return &fakeHost{attrs: map[string]string{}, params: map[string]bool{}}
}

// seedActiveAssembly adds an already-open, active assembly document, so Place exercises the
// "drop an occurrence into the active assembly" path. Returns its id.
func (h *fakeHost) seedActiveAssembly(name string) uint64 {
	return h.seedActive(name, "assembly")
}

// seedActivePart adds an already-open, active part document (no Part Designer stamp), for
// exercising Change-Size guards.
func (h *fakeHost) seedActivePart(name string) uint64 {
	return h.seedActive(name, "part")
}

// seedActive appends an active document of the given type and returns its id.
func (h *fakeHost) seedActive(name, docType string) uint64 {
	h.nextDocID++
	id := h.nextDocID
	for i := range h.docs {
		h.docs[i].Active = false
	}
	h.docs = append(h.docs, wire.DocumentInfo{ID: id, Name: name, Type: docType, Active: true})
	return id
}

// Call records the method and services it against the modelled state.
func (h *fakeHost) Call(method string, req []byte) ([]byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.methods = append(h.methods, method)
	if method == h.failMethod {
		return nil, errFake
	}
	switch method {
	case wire.MethodCommandsCreate:
		h.commands = append(h.commands, decodeReq[wire.CreateCommandArgs](req))
	case wire.MethodDockableWindowsSet:
		h.windows = append(h.windows, decodeReq[wire.SetDockableWindowArgs](req).Window)
	case wire.MethodDocumentsCreate:
		return h.createDoc(req)
	case wire.MethodDocumentsActivate:
		h.activate(decodeReq[struct {
			ID uint64 `json:"id"`
		}](req).ID)
	case wire.MethodDocumentsList:
		return json.Marshal(wire.ListDocumentsResult{Documents: h.docs})
	case wire.MethodDocumentsUpdate:
		h.updates++
	case wire.MethodAttributesSet:
		h.setAttr(decodeReq[wire.SetAttributeArgs](req))
	case wire.MethodAttributesGet:
		return h.getAttr(decodeReq[wire.GetAttributeArgs](req))
	case wire.MethodAssemblyPlace:
		h.placed = append(h.placed, decodeReq[wire.PlaceOccurrenceArgs](req))
	case wire.MethodParametersList:
		return h.listParams()
	case wire.MethodParametersAdd:
		h.addParam(decodeReq[wire.ParameterSetArgs](req))
	case wire.MethodParametersSet:
		h.set = append(h.set, decodeReq[wire.ParameterSetArgs](req))
	case wire.MethodSketchCreate:
		return []byte(`{"sketchIndex":1}`), nil
	case wire.MethodSketchAddEntity:
		return []byte(`{"entityId":10,"pointIds":[11,12]}`), nil
	case wire.MethodSketchConstraintStatus:
		return json.Marshal(wire.ConstraintStatusResult{DOF: h.dof})
	}
	return []byte("{}"), nil
}

// createDoc adds a new document (initially inactive; the engine activates it next).
func (h *fakeHost) createDoc(req []byte) ([]byte, error) {
	a := decodeReq[wire.CreateDocumentArgs](req)
	h.nextDocID++
	info := wire.DocumentInfo{ID: h.nextDocID, Name: a.Name, Type: a.Type}
	h.docs = append(h.docs, info)
	return json.Marshal(info)
}

// activate marks id the active document and clears the flag on the rest.
func (h *fakeHost) activate(id uint64) {
	for i := range h.docs {
		h.docs[i].Active = h.docs[i].ID == id
	}
}

// setAttr stores one string attribute.
func (h *fakeHost) setAttr(a wire.SetAttributeArgs) {
	s, _ := a.Value.Str()
	h.attrs[attrKey(a.Document, a.Set, a.Name)] = s
}

// getAttr returns a stored string attribute (or Found=false).
func (h *fakeHost) getAttr(a wire.GetAttributeArgs) ([]byte, error) {
	v, ok := h.attrs[attrKey(a.Document, a.Set, a.Name)]
	res := wire.AttributeResult{Found: ok}
	if ok {
		res.Attribute = wire.AttributeInfo{Name: a.Name, Value: types.StringVariant(v)}
	}
	return json.Marshal(res)
}

// listParams reports the parameters currently on the document.
func (h *fakeHost) listParams() ([]byte, error) {
	var res wire.ListParametersResult
	for name := range h.params {
		res.Parameters = append(res.Parameters, wire.ParameterInfo{Name: name})
	}
	return json.Marshal(res)
}

// addParam records an added parameter and marks it present.
func (h *fakeHost) addParam(a wire.ParameterSetArgs) {
	h.added = append(h.added, a)
	h.params[a.Name] = true
}

func attrKey(doc uint64, set, name string) string {
	return strconv.FormatUint(doc, 10) + "/" + set + "/" + name
}

// decodeReq unmarshals a recorded request into T (zero value on malformed input).
func decodeReq[T any](req []byte) T {
	var v T
	_ = json.Unmarshal(req, &v)
	return v
}

// called reports whether a method was invoked at least once.
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

// waitFor polls cond until it holds or a short deadline elapses, for effects the engine
// produces from a goroutine.
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
