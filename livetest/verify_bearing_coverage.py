#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Live geometry gate for the bearings size-coverage top-up.

Same discipline as the other category gates: the kernel's parameter engine is unit-strict, so a
wrong or blank member cell collapses the bearing into a degenerate solid the fakeHost unit tests
cannot see. Every newly-added designation must be proven on real geometry.

This samples ONE family per bearing generator (ball_bearing, angular_contact, roller_bearing,
tapered_roller, thrust_bearing, thrust_self_aligning, plain_bush), places its smallest and largest
sampled designation, and asserts each is non-degenerate and size-monotonic (the summed ring +
rolling-element volume grows with bore). The self-aligning case is the load-bearing one: its seed
dimensions were corrected to the real 532xx (the sphered-seat OD changed), so this confirms the
concave spherical seat still builds a valid solid at both ends of the series.

Fresh head + fresh process per family (`--one <label>`), and a non-zero settle, for the reasons
documented in verify_shaft_coverage.py / verify_structural_coverage.py.

Usage: verify_bearing_coverage.py <outdir>
       verify_bearing_coverage.py --one <label> <outdir>   # spawned per family
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

CASES = {
    "ball": ("iso15-deep-groove-ball-bearing", "designation=6009", "designation=6310"),
    "angular": ("iso15-angular-contact-ball-bearing", "designation=7200-B", "designation=7210-B"),
    "roller": ("iso15-cylindrical-roller-bearing", "designation=NU202", "designation=NU310"),
    "tapered": ("iso355-tapered-roller-bearing", "designation=30203", "designation=30310"),
    "thrust": ("iso104-thrust-ball-bearing", "designation=51100", "designation=51109"),
    "self_aligning": ("iso104-self-aligning-thrust-ball-bearing", "designation=53200", "designation=53210"),
    "plain_bush": ("iso4379-sleeve-bush", "d=6,L=10", "d=50,L=50"),
}
ORDER = ["ball", "angular", "roller", "tapered", "thrust", "self_aligning", "plain_bush"]
VOL_FLOOR = 0.1  # mm^3


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
    # settle on a NON-ZERO stable volume (see verify_structural_coverage.py).
    last, stable = None, 0
    for _ in range(50):
        if head.poll() is not None:
            return None
        try:
            vol = json.loads(txt(mcp.call("analysis_mass_properties", {}))).get("volumeMm3")
        except Exception:
            vol = None
        if vol is not None and vol > VOL_FLOOR and last is not None and abs(vol - last) < 1e-3:
            stable += 1
            if stable >= 2:
                return vol
        else:
            stable = 0
        last = vol
        time.sleep(0.4)
    return last


def run_one(label, outdir):
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
        mcp.call("capture_viewport", {"path": f"{outdir.rstrip('/')}/bearing_{label}.png"})
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
        print(f"{label:14s} small({small})={vs} mm^3  large({large})={vl} mm^3  [{'PASS' if ok else 'FAIL'}]")
        if not ok:
            failures.append(f"{label}: small={vs} large={vl} (need large>small>{VOL_FLOOR}); child: {line or proc.stderr[-200:]}")
    if failures:
        print("\nFAIL:\n  " + "\n  ".join(failures))
        return 1
    print("\nPASS: every sampled bearing designation placed as a non-degenerate, size-monotonic solid.")
    return 0


def main(argv):
    if len(argv) >= 3 and argv[0] == "--one":
        run_one(argv[1], argv[2])
        return 0
    return orchestrate(argv[0] if argv else ".")


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
