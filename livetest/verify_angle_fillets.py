#!/usr/bin/env python3
"""Place an EN 10056 L 40x40x4 equal angle through the panel and confirm its root fillet + toe radii
are real (issue #51): the placed solid's volume must match the analytic filleted section area
(t(a+b-t) + root fillet - two toe radii = the tabulated 3.08 cm²), distinct from a sharp L. Then
screenshot the section. Usage: verify_angle_fillets.py <outdir>"""
import json, math, sys, time
import mcp

PANEL = "com.oblikovati.part-designer.panel"
FAMILY, SIZE = "EN 10056 L Equal", "L 40x40x4"
A, B, T, R1, R2, LEN = 40.0, 40.0, 4.0, 6.0, 3.0, 6000.0  # mm


def txt(r):
    try:
        return r["result"]["content"][0]["text"]
    except Exception:
        return json.dumps(r)[:400]


def sharp_area(a, b, t):
    return t * (a + b - t)


def filleted_area(a, b, t, r1, r2):
    k = 1 - math.pi / 4
    return sharp_area(a, b, t) + k * r1 * r1 - 2 * k * r2 * r2  # root adds, two toes remove


def place():
    mcp.call("close_all_documents", {"force": True})
    mcp.call("execute_command", {"id": "PartDesigner.Show"})
    time.sleep(0.6)
    mcp.call("set_panel_value", {"windowId": PANEL, "controlId": "family", "value": FAMILY})
    time.sleep(0.4)
    mcp.call("set_panel_value", {"windowId": PANEL, "controlId": "size", "value": SIZE})
    time.sleep(0.4)
    print("place:", txt(mcp.call("execute_command", {"id": "PartDesigner.Place"})))
    time.sleep(1.5)
    print("status:", txt(mcp.call("status_get_text", {})))


def stable_volume():
    last, stable = None, 0
    for _ in range(50):
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
    exp = filleted_area(A, B, T, R1, R2) * LEN
    sharp = sharp_area(A, B, T) * LEN
    err = abs(vol - exp) / exp * 100 if vol else None
    print(f"volume live={vol} mm^3  filleted={exp:.0f}  sharp={sharp:.0f}  err={err:.4f}%")
    if vol is None or err > 0.5:
        raise SystemExit(f"FAIL: volume {vol} off filleted analytic {exp:.0f} by {err}% (>0.5%)")
    if abs(vol - sharp) < abs(vol - exp):
        raise SystemExit("FAIL: volume matches the SHARP section — fillet/toe radii missing")
    mcp.call("ui_set_object_visibility", {"visibility": {
        "workPlanes": False, "workAxes": False, "workPoints": False, "sketches": False}})
    mcp.call("set_view_orientation", {"document": 0, "orientation": 10759, "fit": True})
    time.sleep(1.5)
    out = outdir.rstrip("/\\") + "/angle_40x40x4_fillets.png"
    mcp.call("capture_viewport", {"path": out})
    print("captured:", out, "PASS")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else ".")
