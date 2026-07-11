#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Live geometry gate for the shaft-parts size-coverage expansion.

Same discipline as verify_size_coverage.py (fasteners): the kernel's parameter engine is
unit-strict, so a wrong or blank member cell collapses the part into a degenerate solid that the
fakeHost unit tests cannot see. Every newly-added size must be proven on real geometry.

This samples ONE family per shaft generator (circlip, key, gib_head_key, pin, clevis_pin,
split_pin), places its NEW smallest and NEW largest size, and asserts each is non-degenerate
(volume above a small floor) and size-monotonic (largest > smallest). A screenshot of the largest
placement per family is captured for a visual shape check.

Each family runs in its own child process with its own fresh head (`--one <label>`): placing many
parts into one long-lived head session eventually wedges its recompute path (a known head-stability
limitation — the fastener gate happens to survive 7 sequential placements, the shaft generators do
not), and restarting the head inside a single process proved flaky (the second head's MCP bridge
races its own startup). A fresh process + fresh head per family sidesteps both.

Usage: verify_shaft_coverage.py <outdir>            # orchestrator: runs every family
       verify_shaft_coverage.py --one <label> <outdir>   # one family (spawned by the orchestrator)
"""
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import mcp  # noqa: E402

HEAD = os.environ.get("OBK_HEAD", "/tmp/claude-1000/-home-vmiguel-git-oblikovati-workspace/"
                      "884d1d20-f65b-417f-a4a4-34a3b439e7fb/scratchpad/oblikovati-head")
ADDINS = "/home/vmiguel/git/oblikovati-workspace/Oblikovati/head/addins"
PANEL = "com.oblikovati.part-designer.panel"

# label -> (catalog family id, new-smallest member key, new-largest member key), one per generator.
# Keys are the canonical keyColumns join (see Family.memberKey): only key columns appear, so the
# gib-head key omits h2 (its key columns are b,h,l).
CASES = {
    "din471_circlip": ("din471-external-circlip", "d=3", "d=50"),
    "din472_circlip": ("din472-internal-circlip", "d=8", "d=62"),
    "din6885_key": ("din6885-parallel-key", "b=2,h=2,l=6", "b=50,h=28,l=160"),
    "din6887_gib_key": ("din6887-gib-head-key", "b=8,h=7,l=32", "b=50,h=28,l=220"),
    "iso2338_dowel": ("iso2338-dowel-pin", "d=1,l=5", "d=50,l=200"),
    "iso2341_clevis": ("iso2341-clevis-pin", "d=3,l=12", "d=36,l=140"),
    "iso1234_split_pin": ("iso1234-split-pin", "d=0.6,l=6", "d=20,l=200"),
}
ORDER = ["din471_circlip", "din472_circlip", "din6885_key", "din6887_gib_key",
         "iso2338_dowel", "iso2341_clevis", "iso1234_split_pin"]
VOL_FLOOR = 0.1  # mm^3 — anything at/below this is a degenerate/collapsed solid


def bridge_up():
    try:
        urllib.request.urlopen(urllib.request.Request("http://127.0.0.1:7800/mcp", method="GET"), timeout=2)
    except urllib.error.HTTPError as e:
        return e.code == 400
    except Exception:
        return False
    return False


def txt(r):
    try:
        return r["result"]["content"][0]["text"]
    except Exception:
        return str(r)[:300]


def place_and_volume(head, catalog_id, member_key):
    mcp.call("close_all_documents", {"force": True})
    mcp.call("execute_command", {"id": "PartDesigner.Show"})
    time.sleep(0.6)
    mcp.call("set_panel_value", {"windowId": PANEL, "controlId": "catalog", "value": catalog_id})
    time.sleep(0.5)
    mcp.call("set_panel_value", {"windowId": PANEL, "controlId": "members", "value": member_key})
    time.sleep(0.5)
    mcp.call("execute_command", {"id": "PartDesigner.Place"})
    last, stable = None, 0
    for _ in range(40):
        if head.poll() is not None:
            return None
        try:
            vol = json.loads(txt(mcp.call("analysis_mass_properties", {}))).get("volumeMm3")
        except Exception:
            vol = None
        if vol is not None and last is not None and abs(vol - last) < 1e-4:
            stable += 1
            if stable >= 2:
                return vol
        else:
            stable = 0
        last = vol
        time.sleep(0.4)
    return last


def run_one(label, outdir):
    """One family in this process's own fresh head; prints a RESULT line the parent parses."""
    cat, small, large = CASES[label]
    subprocess.run(["pkill", "-9", "oblikovati-head"], capture_output=True)
    time.sleep(2)
    env = dict(os.environ, DISPLAY=":1", OBK_ADDINS_DIR=ADDINS)
    head = subprocess.Popen([HEAD], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    try:
        ready = False
        for _ in range(120):
            if bridge_up():
                ready = True
                break
            if head.poll() is not None:
                break
            time.sleep(0.5)
        if not ready:
            print(f"RESULT {label} vs=None vl=None err=no_bridge")
            return
        for _ in range(30):
            try:
                mcp.initialize()
                break
            except Exception:
                time.sleep(1)
        else:
            print(f"RESULT {label} vs=None vl=None err=no_init")
            return
        vs = place_and_volume(head, cat, small)
        vl = place_and_volume(head, cat, large)
        mcp.call("execute_command", {"id": "view.fit"})
        time.sleep(0.6)
        mcp.call("capture_viewport", {"path": f"{outdir.rstrip('/')}/shaft_{label}.png"})
        print(f"RESULT {label} vs={vs} vl={vl} err=")
    finally:
        head.terminate()
        try:
            head.wait(timeout=8)
        except Exception:
            head.kill()


def orchestrate(outdir):
    failures = []
    for label in ORDER:
        cat, small, large = CASES[label]
        proc = subprocess.run([sys.executable, os.path.abspath(__file__), "--one", label, outdir],
                              capture_output=True, text=True, timeout=240)
        line = next((ln for ln in proc.stdout.splitlines() if ln.startswith("RESULT ")), "")
        vs = vl = None
        for tok in line.split():
            if tok.startswith("vs="):
                vs = None if tok[3:] == "None" else float(tok[3:])
            elif tok.startswith("vl="):
                vl = None if tok[3:] == "None" else float(tok[3:])
        ok = (vs is not None and vl is not None and vs > VOL_FLOOR and vl > VOL_FLOOR and vl > vs)
        print(f"{label:18s} small({small})={vs} mm^3  large({large})={vl} mm^3  [{'PASS' if ok else 'FAIL'}]")
        if not ok:
            failures.append(f"{label}: small={vs} large={vl} (need large>small>{VOL_FLOOR}); child said: {line or proc.stderr[-200:]}")
    if failures:
        print("\nFAIL:\n  " + "\n  ".join(failures))
        return 1
    print("\nPASS: every sampled new shaft size placed as a non-degenerate, size-monotonic solid.")
    return 0


def main(argv):
    if len(argv) >= 3 and argv[0] == "--one":
        run_one(argv[1], argv[2])
        return 0
    return orchestrate(argv[0] if argv else ".")


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
