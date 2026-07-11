#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Live geometry gate for the fastener size-coverage expansion.

The kernel's parameter engine is unit-strict: a wrong or blank member cell collapses the part
into a degenerate (near-zero-volume) solid, and the fakeHost unit tests cannot see it (they never
evaluate units or build real geometry). So every newly-added size must be proven on real geometry.

This drives a running head over the MCP bridge and, for a representative sample of families,
places the NEW smallest and NEW largest size, then asserts each is:
  1. non-degenerate — stable body volume above a small floor, and
  2. monotonic — the largest size's volume exceeds the smallest's.
A collapsed row (the unit-strict failure mode) fails check 1; a mis-scaled row usually fails 2.
A screenshot of the largest placement per family is captured for a visual shape check.

Usage: verify_size_coverage.py <outdir>
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

# (label, catalog family id, new-smallest member key, new-largest member key)
CASES = [
    ("din933_hex_bolt", "din933-hex-bolt", "d=1.6,l=8", "d=48,l=240"),
    ("iso4762_socket", "iso4762-socket-screw", "d=1.6,l=8", "d=48,l=240"),
    ("iso4032_hex_nut", "iso4032-hex-nut", "d=1.6", "d=48"),
    ("iso7089_washer", "iso7089-washer", "size=M1.6", "size=M36"),
    ("din976_rod", "din976-threaded-rod", "d=1.6,l=1000", "d=48,l=1000"),
    ("din939_stud", "din939-stud", "d=6,l=30", "d=48,l=200"),
    ("ansi_hex_bolt", "ansi-b18-2-1-hex-bolt", "thread=1/4-20,l=1", "thread=2-4 1/2,l=8"),
]
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


def place_and_volume(catalog_id, member_key):
    mcp.call("close_all_documents", {"force": True})
    mcp.call("execute_command", {"id": "PartDesigner.Show"})
    time.sleep(0.5)
    mcp.call("set_panel_value", {"windowId": PANEL, "controlId": "catalog", "value": catalog_id})
    time.sleep(0.4)
    mcp.call("set_panel_value", {"windowId": PANEL, "controlId": "members", "value": member_key})
    time.sleep(0.4)
    mcp.call("execute_command", {"id": "PartDesigner.Place"})
    # settle: poll until the volume stops changing (async recompute off the session goroutine)
    last, stable = None, 0
    for _ in range(80):
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


def capture(outdir, label):
    mcp.call("execute_command", {"id": "view.fit"})
    time.sleep(0.6)
    p = f"{outdir.rstrip('/')}/size_{label}.png"
    mcp.call("capture_viewport", {"path": p})
    return p


def main(outdir):
    env = dict(os.environ, DISPLAY=":1", OBK_ADDINS_DIR=ADDINS)
    head = subprocess.Popen([HEAD], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    failures = []
    try:
        for _ in range(120):
            if bridge_up():
                break
            if head.poll() is not None:
                print(f"FAIL: head exited early ({head.returncode})")
                return 1
            time.sleep(0.5)
        else:
            print("FAIL: bridge never came up")
            return 1
        mcp.initialize()
        for label, cat, small, large in CASES:
            vs = place_and_volume(cat, small)
            vl = place_and_volume(cat, large)
            path = capture(outdir, label)
            ok = (vs is not None and vl is not None and vs > VOL_FLOOR and vl > VOL_FLOOR and vl > vs)
            print(f"{label:18s} small({small})={vs} mm^3  large({large})={vl} mm^3  "
                  f"[{'PASS' if ok else 'FAIL'}]  {path}")
            if not ok:
                failures.append(f"{label}: small={vs} large={vl} (floor={VOL_FLOOR}, need large>small>floor)")
        if failures:
            print("\nFAIL:\n  " + "\n  ".join(failures))
            return 1
        print("\nPASS: every sampled new size placed as a non-degenerate, size-monotonic solid.")
        return 0
    finally:
        head.terminate()
        try:
            head.wait(timeout=10)
        except Exception:
            head.kill()


if __name__ == "__main__":
    sys.exit(main(sys.argv[1] if len(sys.argv) > 1 else "."))
