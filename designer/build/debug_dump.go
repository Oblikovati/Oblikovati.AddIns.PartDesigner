// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"log/slog"
	"os"
)

// debugEnabled reports whether the OBK_PD_DEBUG env var is set — the live-diagnostic switch for
// DumpGeometry, so generator geometry/DOF can be traced through the host's logs without shipping
// the noise (dev-only).
func debugEnabled() bool { return os.Getenv("OBK_PD_DEBUG") != "" }

// DumpGeometry logs, at debug level, the sketch's degrees of freedom and every entity with its
// defining-point coordinates (cm) — the tool for diagnosing a section's live solve (why a face
// collapsed, whether it reached DOF 0). It is a no-op unless OBK_PD_DEBUG is set. label names the
// dump so several can be told apart within one placement.
func (s *SketchContext) DumpGeometry(label string) {
	if !debugEnabled() {
		return
	}
	sk := s.b.api.Sketch()
	st, err := sk.ConstraintStatus(s.index)
	if err != nil {
		slog.Warn("pd.dumpGeometry: constraint status", "label", label, "err", err)
		return
	}
	slog.Info("pd.dumpGeometry", "label", label, "dof", st.DOF, "status", st.Status)
	ents, err := sk.Entities(s.index)
	if err != nil {
		slog.Warn("pd.dumpGeometry: entities", "label", label, "err", err)
		return
	}
	for _, e := range ents.Entities {
		slog.Info("pd.dumpGeometry.entity", "label", label, "id", e.ID, "kind", e.Kind, "radius", e.Radius, "points", e.Points)
	}
}
