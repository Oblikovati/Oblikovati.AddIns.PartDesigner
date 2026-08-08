// SPDX-License-Identifier: GPL-2.0-only

// Package releaseguard holds guards on the files the Release workflow executes.
// It is test-only: the guards run in the ordinary `go test ./...` sweep.
package releaseguard_test

import (
	"os/exec"
	"strings"
	"testing"
)

// wantMode is git's mode for a regular executable file; 100644 is the non-executable one.
const wantMode = "100755"

// TestReleaseScriptsAreExecutableInGit fails when a shell script under scripts/ is committed
// without its executable bit. The Release workflow invokes scripts/publish-catalogue.sh
// directly (not via `bash <script>`), so a 100644 mode makes every non-Windows leg die with
// "Permission denied" (exit 126) — which is exactly how the catalogue publish broke and then
// blocked the GitHub release for a month, unnoticed because no test covered the release path.
//
// The mode is read from the git INDEX rather than from os.Stat: this test also runs on the
// Windows CI leg, where the filesystem carries no executable bit but the index still does,
// and the index is what a Linux/macOS runner checks out.
func TestReleaseScriptsAreExecutableInGit(t *testing.T) {
	for _, entry := range strings.Split(strings.TrimSpace(gitOutput(t, "ls-files", "-s", "scripts/")), "\n") {
		mode, path, ok := splitIndexEntry(entry)
		if !ok || !strings.HasSuffix(path, ".sh") {
			continue
		}
		if mode != wantMode {
			t.Errorf("%s is committed with mode %s, want %s — the Release workflow executes it "+
				"directly, so a non-executable mode fails the build with exit 126. "+
				"Fix with: git update-index --chmod=+x %s", path, mode, wantMode, path)
		}
	}
}

// splitIndexEntry pulls the mode and path out of a `git ls-files -s` line, which reads
// "<mode> <sha> <stage>\t<path>".
func splitIndexEntry(entry string) (mode, path string, ok bool) {
	meta, path, found := strings.Cut(entry, "\t")
	if !found {
		return "", "", false
	}
	fields := strings.Fields(meta)
	if len(fields) == 0 {
		return "", "", false
	}
	return fields[0], path, true
}

// gitOutput runs git with args at the module root and returns stdout, skipping the test when
// git or the checkout is unavailable (the guard is meaningless outside a working tree).
func gitOutput(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = ".."
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("git %v unavailable: %v", args, err)
	}
	return string(out)
}
