#!/usr/bin/env python3
"""Place an AISC W8x31 wide-flange beam through the Part Designer panel and confirm two things at
once (issue #51): (1) the shape PLACES at all — it shares the i_beam generator, which gained a
required root_radius param, and the AISC W family had been missing that column (a regression this
proves fixed); and (2) its four web-flange root fillets are real — the placed solid's volume matches
the analytic FILLETED section area (which equals the published W8x31 area, 9.13 in²), distinct from
the sharp-corner area. Then screenshot it. Usage: verify_w_shape_fillets.py <outdir>"""
import json, math, sys, time
import mcp

PANEL = "com.oblikovati.part-designer.panel"
FAMILY, SIZE = "AISC W Shapes", "W8x31"
IN = 25.4  # mm per inch
D, BF, TW, TF, R, LEN = 8.0 * IN, 8.0 * IN, 0.285 * IN, 0.435 * IN, 0.394 * IN, 240.0 * IN


def txt(r):
    try:
        return r["result"]["content"][0]["text"]
    except Exception:
        return json.dumps(r)[:400]


def sharp_area(d, bf, tw, tf):
    return bf * d - (bf - tw) * (d - 2 * tf)


def filleted_area(d, bf, tw, tf, r):
    return sharp_area(d, bf, tw, tf) + 4 * r * r * (1 - math.pi / 4)  # four concave root fillets add


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
    exp = filleted_area(D, BF, TW, TF, R) * LEN
    sharp = sharp_area(D, BF, TW, TF) * LEN
    err = abs(vol - exp) / exp * 100 if vol else None
    print(f"volume live={vol} mm^3  filleted={exp:.0f}  sharp={sharp:.0f}  err={err:.4f}%")
    if vol is None or err > 0.5:
        raise SystemExit(f"FAIL: W8x31 volume {vol} off filleted analytic {exp:.0f} by {err}% (>0.5%)")
    if abs(vol - sharp) < abs(vol - exp):
        raise SystemExit("FAIL: volume matches the SHARP section — root fillets missing")
    mcp.call("ui_set_object_visibility", {"visibility": {
        "workPlanes": False, "workAxes": False, "workPoints": False, "sketches": False}})
    mcp.call("set_view_orientation", {"document": 0, "orientation": 10759, "fit": True})
    time.sleep(1.5)
    out = outdir.rstrip("/\\") + "/w8x31_fillets.png"
    mcp.call("capture_viewport", {"path": out})
    print("captured:", out, "PASS")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else ".")
