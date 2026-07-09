#!/usr/bin/env python3
"""Build the split-pin sweep geometry directly via the bridge and screenshot it, to eyeball the
shape (issue #58). Mirrors what the split_pin generator does: a wire circle (d/2) swept along a
hairpin path (leg up, semicircular eye, leg down). Usage: verify_split_pin.py <outdir>"""
import math, sys, time
import mcp

D, A, C, L = 8.0, 4.0, 15.0, 80.0  # ISO 1234 d=8: nominal 8, offset a=4, eye width c=15, length l=80 (mm)
MM = 0.1                            # mm -> cm (model unit)


def split_path(d, a, c, l):
    a, c, l = a * MM, c * MM, l * MM
    r = c / 2
    pts = [[0, 0, 0], [0, 0, l]]
    for i in range(1, 16):
        th = math.pi * (1 - i / 16)
        pts.append([r + r * math.cos(th), 0, l + r * math.sin(th)])
    pts += [[c, 0, l], [c, 0, -a]]
    return pts


def main(outdir):
    mcp.initialize()
    mcp.call("close_all_documents", {"force": True})
    mcp.call("create_document", {"type": "part", "name": "splitpin_iso1234_8"})
    mcp.call("create_sketch", {"plane": "XY"})
    wire_radius_mm = D / 2 / 2  # wire dia = d/2, so radius = d/4
    mcp.call("add_sketch_entity", {"sketchIndex": 0, "kind": "circle",
                                   "points": [[0, 0]], "radius": f"{wire_radius_mm} mm"})
    res = mcp.call("add_feature", {"kind": "sweep", "args": {
        "sketchIndex": 0, "profileIndex": 0, "pathPoints": split_path(D, A, C, L), "operation": "new"}})
    print("sweep:", res.get("result", res))
    props = mcp.call("get_physical_properties", {})
    print("props:", props.get("result", props))
    mcp.call("ui_set_object_visibility", {"visibility": {
        "workPlanes": False, "workAxes": False, "workPoints": False, "sketches": False}})
    mcp.call("set_view_orientation", {"orientation": 10759, "fit": True})
    time.sleep(2.0)  # let the async render settle + the fit frame the part before capturing
    mcp.call("set_view_orientation", {"orientation": 10759, "fit": True})
    out = outdir.rstrip("/\\") + "/split_pin_iso1234_8.png"
    mcp.call("capture_viewport", {"path": out})
    time.sleep(1.0)
    print("captured:", out)


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else ".")
