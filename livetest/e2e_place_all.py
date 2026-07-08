#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Cross-category live end-to-end test for the Part Designer add-in (milestone F3).

Drives a running Oblikovati head over the MCP bridge (127.0.0.1:7800) to:
  1. place one representative member from EVERY top-level category into a part,
     verifying each shows published parameters + named features and captures a
     golden screenshot;
  2. re-drive a placed part by editing a driver parameter (proving it is truly
     parametric);
  3. place a part into an active assembly, confirming the occurrence path.

Prerequisites: build + install the add-in, launch the head with the MCP bridge
add-in loaded (see livetest/README.md), then:  python3 e2e_place_all.py [outdir]
"""
import json
import sys
import time

import mcp

PANEL = "com.oblikovati.part-designer.panel"

# One representative per top-level category: (standard, family label, size, screenshot, camera eye).
CATEGORIES = [
    ("Fasteners", "ISO", "ISO 4017 Hex Head", "10x50", "e2e_fastener.png", [8, 6, 9]),
    ("Structural", "EN", "EN 10365 IPE", "IPE 300", "e2e_structural.png", [60, 45, 68]),
    ("Shaft Parts", "DIN", "DIN 6885 Parallel", "12x8x40", "e2e_shaft.png", [7, 5, 8]),
    ("Bearings", "ISO", "ISO 15 Deep Groove", "6205", "e2e_bearing.png", [9, 7, 10]),
]


def text(resp):
    r = resp.get("result", resp)
    out = [c["text"] for c in r.get("content", []) if c.get("type") == "text"]
    return "\n".join(out) if out else json.dumps(r)


def call(name, args):
    return text(mcp.call(name, args))


def settle():
    """Wait until the part's volume stops changing (recompute finished)."""
    prev = None
    for _ in range(70):
        time.sleep(0.5)
        d = json.loads(text(mcp.call("analysis_mass_properties", {})) or "{}")
        v = d.get("volumeMm3")
        if v and v == prev:
            return v
        prev = v
    return prev


def place(standard, family, size):
    call("close_all_documents", {"force": True})
    call("execute_command", {"id": "PartDesigner.Show"})
    for ctrl, val in [("standard", standard), ("family", family), ("size", size)]:
        call("set_panel_value", {"windowId": PANEL, "controlId": ctrl, "value": val})
    call("execute_command", {"id": "PartDesigner.Place"})
    return settle()


def check_placement(outdir):
    """Place one representative per category and assert it is a healthy, parametric part."""
    ok = True
    for cat, std, fam, size, png, eye in CATEGORIES:
        vol = place(std, fam, size)
        tree = json.loads(call("get_model_tree", {}))
        params = [p for p in tree.get("parameters", []) if not p.startswith("d")]
        sick = [f["name"] for f in tree.get("features", []) if f.get("health")]
        healthy = bool(vol) and bool(params) and bool(tree.get("features")) and not sick
        print(f"  [{'ok' if healthy else 'FAIL'}] {cat:12} {fam} {size}: "
              f"vol={vol} bodies={tree['bodies']} params={len(params)} sick={sick}")
        ok = ok and healthy
        call("set_camera", {"target": [0, 0, 0], "eye": eye, "up": [0, 1, 0], "fov": 0.6})
        call("capture_viewport", {"path": f"{outdir}/{png}"})
    return ok


def check_redrive():
    """Prove a placed part re-drives when a driver parameter is edited."""
    before = place("ISO", "ISO 15 Deep Groove", "6205")
    call("set_parameter", {"name": "bore", "expression": "15 mm"})
    after = settle()
    changed = abs((before or 0) - (after or 0)) > 1
    print(f"  [{'ok' if changed else 'FAIL'}] re-drive bore 25->15mm: vol {before} -> {after}")
    return changed


def check_assembly():
    """Place a part while an assembly is active and confirm an occurrence lands in it."""
    call("close_all_documents", {"force": True})
    call("create_document", {"type": "assembly", "name": "E2E Assembly"})
    call("execute_command", {"id": "PartDesigner.Show"})
    for ctrl, val in [("standard", "ISO"), ("family", "ISO 4017 Hex Head"), ("size", "8x40")]:
        call("set_panel_value", {"windowId": PANEL, "controlId": ctrl, "value": val})
    call("execute_command", {"id": "PartDesigner.Place"})
    time.sleep(3)
    call("activate_document", {"name": "E2E Assembly"})
    occ = json.loads(call("list_occurrences", {}) or "{}")
    n = len(occ.get("occurrences", [])) if isinstance(occ, dict) else 0
    print(f"  [{'ok' if n >= 1 else 'FAIL'}] assembly occurrences: {n}")
    return n >= 1


def main():
    outdir = sys.argv[1] if len(sys.argv) > 1 else "."
    mcp.initialize()
    print("=== Placement (every category) ===")
    placed = check_placement(outdir)
    print("=== Re-drive (parametric) ===")
    redrove = check_redrive()
    print("=== Assembly occurrence ===")
    assembled = check_assembly()
    ok = placed and redrove and assembled
    print(f"\n{'PASS' if ok else 'FAIL'}: placement={placed} redrive={redrove} assembly={assembled}")
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
