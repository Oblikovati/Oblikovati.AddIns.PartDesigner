#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Live area check for the taper-flange UPN channel (#69).

Places each UPN member, reads its extruded volume, and derives the cross-section
area (volume / length) to compare against the published EN 10279 sectional area.
A placement that reaches DOF 0 (AssertFullyConstrained) is a precondition — a
failed placement prints the host error.  Run against a live head with the
part-designer + mcp-bridge add-ins loaded:  python3 verify_upn.py
"""
import json
import sys
import time

import mcp

PANEL = "com.oblikovati.part-designer.panel"

# designation -> published sectional area A (cm^2), length is the catalogue 6000 mm.
PUBLISHED = {"UPN 80": 11.0, "UPN 100": 13.5, "UPN 160": 24.0, "UPN 200": 32.2}
LENGTH_MM = 6000.0


def text(resp):
    r = resp.get("result", resp)
    out = [c["text"] for c in r.get("content", []) if c.get("type") == "text"]
    return "\n".join(out) if out else json.dumps(r)


def call(name, args):
    return text(mcp.call(name, args))


def settle():
    prev = None
    for _ in range(80):
        time.sleep(0.5)
        d = json.loads(text(mcp.call("analysis_mass_properties", {})) or "{}")
        v = d.get("volumeMm3")
        if v and v == prev:
            return v
        prev = v
    return prev


def place(size):
    print(call("close_all_documents", {"force": True}))
    print("show:", call("execute_command", {"id": "PartDesigner.Show"}))
    for ctrl, val in [("standard", "EN"), ("family", "EN 10279 UPN"), ("size", size)]:
        print(f"set {ctrl}={val}:", call("set_panel_value", {"windowId": PANEL, "controlId": ctrl, "value": val}))
    print("place:", call("execute_command", {"id": "PartDesigner.Place"}))
    return settle()


def main():
    mcp.initialize()
    sizes = sys.argv[1:] or list(PUBLISHED)
    for size in sizes:
        vol = place(size)
        if not vol:
            print(f"[FAIL] {size}: no volume (placement failed — see host errors above)\n")
            continue
        area_cm2 = (vol / LENGTH_MM) / 100.0  # mm^3 / mm = mm^2 -> cm^2
        pub = PUBLISHED[size]
        err = (area_cm2 - pub) / pub * 100.0
        tag = "ok" if abs(err) < 1.0 else "CHECK"
        print(f"[{tag}] {size}: area {area_cm2:.3f} cm^2 vs published {pub} ({err:+.2f}%)  vol={vol:.0f} mm^3\n")


if __name__ == "__main__":
    main()
