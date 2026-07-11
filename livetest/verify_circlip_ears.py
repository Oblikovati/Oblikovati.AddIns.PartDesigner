#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Live end-to-end verification of the circlip plier-lug ears (#61).

Drives a running Oblikovati head over the MCP bridge (127.0.0.1:7800) to place the
three circlip families and confirm each grew TWO non-degenerate lug eyes bracketing
the split gap:

  1. DIN 471 d30 (EXTERNAL, mm) — ears project radially OUTWARD, both on the
     band-terminus side of the gap (the winding-sign trap: ear B at the actual 330°
     edge, not mirrored to +30° on the empty gap side).
  2. DIN 472 d20 (INTERNAL, mm) — ears project INWARD (the kEye=0.9 trim; inner_dia
     radius formula).
  3. ASME 1/4" (EXTERNAL, in) — proves the mm clearance floor in circlipEarsFit
     converts against an inch-unit family (else the guard would silently skip ears).

Each placement must be 3 bodies (ring + 2 ears) with a total volume matching the
analytic ring + two annular ears (a collapsed/degenerate ear fails both checks — the
#53 lesson: only the live per-body-aware volume gates against a zeroed param). A
top-down screenshot (looking along the ring axis) is captured for the winding-sign
visual check, plus an iso view.

Usage: verify_circlip_ears.py <outdir>
"""
import json
import math
import sys
import time

import mcp

PANEL = "com.oblikovati.part-designer.panel"

# (label, catalog family ID, member key, external?, unit, di, do, s) — di/do/s in the family unit.
CASES = [
    ("din471_d30_external", "din471-external-circlip", "d=30", True, "mm", 28.6, 38.6, 1.5),
    ("din472_d20_internal", "din472-internal-circlip", "d=20", False, "mm", 15.0, 21.0, 1.0),
    ("ansi_quarter_external", "ansi-external-retaining-ring", "size=1/4", True, "in", 0.24, 0.415, 0.025),
]


def txt(r):
    try:
        return r["result"]["content"][0]["text"]
    except Exception:
        return json.dumps(r)[:400]


def analytic_mm3(external, unit, di, do, s):
    """Ring (330deg annular band) + two flat annular ears, summed as independent bodies
    (no boolean union — mass properties sums each body, so the representational overlap is
    counted in both, exactly as the tapered/self-aligning verifiers assume)."""
    band = (do - di) / 2.0
    eye = band * (1.0 if external else 0.9)
    hole = eye * 0.45
    v_ring = (330.0 / 360.0) * math.pi * ((do / 2) ** 2 - (di / 2) ** 2) * s
    v_ear = math.pi * ((eye / 2) ** 2 - (hole / 2) ** 2) * s
    total = v_ring + 2 * v_ear
    conv = 25.4 if unit == "in" else 1.0  # family unit -> mm; volume scales by conv^3
    return total * conv ** 3


def place(catalog_id, member_key):
    mcp.call("close_all_documents", {"force": True})
    mcp.call("execute_command", {"id": "PartDesigner.Show"})
    time.sleep(0.6)
    mcp.call("set_panel_value", {"windowId": PANEL, "controlId": "catalog", "value": catalog_id})
    time.sleep(0.4)
    mcp.call("set_panel_value", {"windowId": PANEL, "controlId": "members", "value": member_key})
    time.sleep(0.4)
    print("  place:", txt(mcp.call("execute_command", {"id": "PartDesigner.Place"})))
    time.sleep(1.5)


def stable_volume():
    last, stable = None, 0
    for _ in range(60):
        try:
            vol = json.loads(txt(mcp.call("analysis_mass_properties", {}))).get("volumeMm3")
        except Exception:
            vol = None
        if vol is not None and last is not None and abs(vol - last) < 1e-3:
            stable += 1
            if stable >= 2:
                return vol
        else:
            stable = 0
        last = vol
        time.sleep(0.5)
    return last


def body_count():
    try:
        return json.loads(txt(mcp.call("get_model_tree", {}))).get("bodies")
    except Exception:
        return None


def capture(outdir, label):
    # Top-down: look along +Z at the ring's flat face — the gap and both ears are unambiguous.
    mcp.call("set_camera", {"target": [0, 0, 0], "eye": [0, 0, 12], "up": [0, 1, 0], "fov": 0.6})
    mcp.call("execute_command", {"id": "view.fit"})
    time.sleep(1.0)
    top = f"{outdir.rstrip('/')}/circlip_{label}_top.png"
    mcp.call("capture_viewport", {"path": top})
    mcp.call("set_camera", {"target": [0, 0, 0], "eye": [7, 5, 8], "up": [0, 1, 0], "fov": 0.6})
    time.sleep(1.0)
    iso = f"{outdir.rstrip('/')}/circlip_{label}_iso.png"
    mcp.call("capture_viewport", {"path": iso})
    return top


def main(outdir):
    mcp.initialize()
    failures = []
    for label, cat, key, external, unit, di, do, s in CASES:
        print(f"=== {label} ({cat} {key}) ===")
        place(cat, key)
        vol = stable_volume()
        bodies = body_count()
        exp = analytic_mm3(external, unit, di, do, s)
        err = abs(vol - exp) / exp * 100 if vol else None
        ok = vol is not None and bodies == 3 and err is not None and err <= 2.0
        print(f"  bodies={bodies} (want 3)  vol={vol} mm^3  analytic(ring+2 ears)={exp:.2f}  "
              f"err={err if err is None else round(err, 3)}%  [{'PASS' if ok else 'FAIL'}]")
        top = capture(outdir, label)
        print(f"  captured {top}")
        if not ok:
            failures.append(f"{label}: bodies={bodies} vol={vol} exp={exp:.2f} err={err}")
    if failures:
        print("\nFAIL:\n  " + "\n  ".join(failures))
        raise SystemExit(1)
    print("\nPASS: all three circlip families grew 2 non-degenerate lug ears (3 bodies, volume within 2%).")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else ".")
