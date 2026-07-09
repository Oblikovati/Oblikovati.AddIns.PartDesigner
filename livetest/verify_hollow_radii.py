#!/usr/bin/env python3
"""Place an EN 10219 SHS through the Part Designer panel and confirm its cold-formed corner
radii are real (issue #52): the placed solid's volume must match the analytic rounded tube
(outer radius 2·t, inner radius t), which an under-constrained profile could never hit. Then
screenshot the end so the rounded corner is visible. Usage: verify_hollow_radii.py <outdir>"""
import json, math, sys, time
import mcp

PANEL = "com.oblikovati.part-designer.panel"
FAMILY, SIZE = "EN 10219 SHS", "SHS 40x40x3"
B, H, T, LEN = 40.0, 40.0, 3.0, 6000.0  # SHS 40x40x3, 6 m stock (mm)


def txt(r):
    try:
        return r["result"]["content"][0]["text"]
    except Exception:
        return json.dumps(r)[:400]


def analytic_volume(b, h, t, length):
    """Rounded hollow-tube volume: outer rounded rect (r=2t) minus inner rounded rect (r=t)."""
    ro, ri = 2 * t, t
    outer = b * h - (4 - math.pi) * ro * ro
    inner = (b - 2 * t) * (h - 2 * t) - (4 - math.pi) * ri * ri
    return (outer - inner) * length  # mm^2 * mm


def place():
    mcp.call("close_all_documents", {"force": True})
    mcp.call("execute_command", {"id": "PartDesigner.Show"})
    time.sleep(0.6)
    mcp.call("set_panel_value", {"windowId": PANEL, "controlId": "family", "value": FAMILY})
    time.sleep(0.4)
    mcp.call("set_panel_value", {"windowId": PANEL, "controlId": "size", "value": SIZE})
    time.sleep(0.4)
    print("place:", txt(mcp.call("execute_command", {"id": "PartDesigner.Place"})))


def stable_volume():
    last, stable = None, 0
    for _ in range(40):
        vol = None
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


def main(outdir):
    mcp.initialize()
    place()
    vol = stable_volume()
    exp = analytic_volume(B, H, T, LEN)
    err = abs(vol - exp) / exp * 100 if vol else None
    print(f"volume live={vol} mm^3  analytic={exp:.0f} mm^3  err={err:.4f}%")
    if vol is None or err > 0.5:
        raise SystemExit(f"FAIL: volume {vol} off analytic {exp:.0f} by {err}% (>0.5%)")
    mcp.call("ui_set_object_visibility", {"visibility": {
        "workPlanes": False, "workAxes": False, "workPoints": False, "sketches": False}})
    mcp.call("set_view_orientation", {"document": 0, "orientation": 10763, "fit": True})
    time.sleep(1.5)
    out = outdir.rstrip("/\\") + "/hollow_shs_40x40x3_corner.png"
    mcp.call("capture_viewport", {"path": out})
    print("captured:", out, "PASS")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else ".")
