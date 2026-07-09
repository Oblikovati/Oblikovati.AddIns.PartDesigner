#!/usr/bin/env python3
"""Place an EN 10055 T 50 equal-flange tee through the Part Designer panel and confirm its two
flange-stem root fillets are real (issue #51): the placed solid's volume must match the analytic
FILLETED section area (sharp + two concave root fillets = the Montanstahl EN 10055 tabulated
5.66 cm²), distinct from the sharp-corner area — proving the two root fillets solved to DOF 0.
Then screenshot the tee. Usage: verify_tee_fillets.py <outdir>"""
import json, math, sys, time
import mcp

PANEL = "com.oblikovati.part-designer.panel"
FAMILY, SIZE = "EN 10055 T", "T 50"
H, B, TW, TF, R, LEN = 50.0, 50.0, 6.0, 6.0, 2.0, 6000.0  # T 50 (mm)


def txt(r):
    try:
        return r["result"]["content"][0]["text"]
    except Exception:
        return json.dumps(r)[:400]


def sharp_area(h, b, tw, tf):
    return tf * b + tw * (h - tf)  # flange bar + stem below it


def filleted_area(h, b, tw, tf, r):
    return sharp_area(h, b, tw, tf) + 2 * r * r * (1 - math.pi / 4)  # two concave root fillets add


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
    exp = filleted_area(H, B, TW, TF, R) * LEN
    sharp = sharp_area(H, B, TW, TF) * LEN
    err = abs(vol - exp) / exp * 100 if vol else None
    print(f"volume live={vol} mm^3  filleted={exp:.0f}  sharp={sharp:.0f}  err={err:.4f}%")
    if vol is None or err > 0.5:
        raise SystemExit(f"FAIL: volume {vol} off filleted analytic {exp:.0f} by {err}% (>0.5%)")
    if abs(vol - sharp) < abs(vol - exp):
        raise SystemExit("FAIL: volume matches the SHARP section — root fillets missing")
    mcp.call("ui_set_object_visibility", {"visibility": {
        "workPlanes": False, "workAxes": False, "workPoints": False, "sketches": False}})
    mcp.call("set_view_orientation", {"document": 0, "orientation": 10759, "fit": True})
    time.sleep(1.5)
    out = outdir.rstrip("/\\") + "/tee_50_fillets.png"
    mcp.call("capture_viewport", {"path": out})
    print("captured:", out, "PASS")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else ".")
