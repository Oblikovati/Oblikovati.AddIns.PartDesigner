#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Live geometry gate for the structural-steel size-coverage expansion.

Same discipline as the fastener/shaft gates: the kernel's parameter engine is unit-strict, so a
wrong or blank member cell collapses the profile-extrude into a degenerate solid that the fakeHost
unit tests cannot see. Every newly-added section must be proven on real geometry.

This samples ONE family per structural generator (i_beam, channel, angle, tee, hollow_rect,
hollow_round, round_bar, flat_bar), places its NEW smallest and NEW largest section, and asserts
each is non-degenerate (volume above a floor) and size-monotonic (largest > smallest). Because these
are constant-length extrusions (l = 6000 mm / 240 in / 1000 mm), the volume ratio also tracks the
cross-section area growth. A screenshot of the largest placement per family is captured.

Each family runs in its own child process with its own fresh head (`--one <label>`), for the same
head-stability reason documented in verify_shaft_coverage.py.

Usage: verify_structural_coverage.py <outdir>
       verify_structural_coverage.py --one <label> <outdir>   # spawned per family
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
CASES = {
    "ipe_beam": ("ipe-en10365", "designation=IPE 80", "designation=IPE 600"),
    "upn_channel": ("upn-en10279", "designation=UPN 50", "designation=UPN 400"),
    "equal_angle": ("angle-equal-en10056", "designation=L 20x20x3", "designation=L 200x200x20"),
    "tee": ("tee-en10055", "designation=T 20", "designation=T 120"),
    "shs_hollow": ("shs-en10219", "designation=SHS 20x20x2", "designation=SHS 200x200x8"),
    "chs_hollow": ("chs-en10219", "designation=CHS 21.3x2.5", "designation=CHS 168.3x6.3"),
    "round_bar": ("iso1035-round-bar", "d=8,l=1000", "d=100,l=1000"),
    "flat_bar": ("en10058-flat-bar", "b=15,a=5", "b=100,a=12"),
}
ORDER = ["ipe_beam", "upn_channel", "equal_angle", "tee", "shs_hollow", "chs_hollow",
         "round_bar", "flat_bar"]
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
    # Settle on a NON-ZERO stable volume. Placement recompute is async: right after Place the
    # document exists but the body may not be built yet, so analysis reads volumeMm3 = 0. The
    # complex profile generators (i_beam, angle, tee) build slowly enough that this 0-window spans
    # several polls — accepting a "stable" 0 would falsely read them as degenerate. So only count a
    # reading above the floor toward stability; a genuinely collapsed solid never stabilizes and
    # falls through to `last` (0/None), which the caller flags as a failure.
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
        mcp.call("capture_viewport", {"path": f"{outdir.rstrip('/')}/struct_{label}.png"})
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
    print("\nPASS: every sampled new structural section placed as a non-degenerate, size-monotonic solid.")
    return 0


def main(argv):
    if len(argv) >= 3 and argv[0] == "--one":
        run_one(argv[1], argv[2])
        return 0
    return orchestrate(argv[0] if argv else ".")


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
