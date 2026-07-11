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

# designation -> published EN 10279 sectional area A (cm^2), length is the catalogue 6000 mm.
PUBLISHED = {
    "UPN 50": 7.12, "UPN 65": 9.03, "UPN 80": 11.0, "UPN 100": 13.5, "UPN 120": 17.0,
    "UPN 140": 20.4, "UPN 160": 24.0, "UPN 180": 28.0, "UPN 200": 32.2, "UPN 220": 37.4,
    "UPN 240": 42.3, "UPN 260": 48.3, "UPN 280": 53.3, "UPN 300": 58.8, "UPN 320": 75.8,
    "UPN 350": 77.3, "UPN 380": 80.4, "UPN 400": 91.5,
}
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


FAMILY = "upn-en10279"  # the catalog family id


def place(size):
    call("close_all_documents", {"force": True})
    call("execute_command", {"id": "PartDesigner.Show"})
    # The panel is a catalog tree (family id) + a members table (member key).
    call("set_panel_value", {"windowId": PANEL, "controlId": "catalog", "value": FAMILY})
    call("set_panel_value", {"windowId": PANEL, "controlId": "members", "value": "designation=" + size})
    call("execute_command", {"id": "PartDesigner.Place"})
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
