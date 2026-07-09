#!/usr/bin/env python3
"""Place an ISO 104 51106 thrust ball bearing and confirm the ground race grooves are real (issue
#54): the placed assembly's total volume must match the analytic model — two grooved washers (each a
revolved meridian whose ball-facing face carries a conformity groove, integrated by Pappus) plus a
full complement of balls — proving the grooved washer sections solved to DOF 0 and revolved to
correct solids. Then screenshot it. Usage: verify_thrust_groove.py <outdir>"""
import json, math, sys, time
import mcp

PANEL = "com.oblikovati.part-designer.panel"
FAMILY, SIZE = "ISO 104 Ball", "51106"
BORE, OUTER, HEIGHT, BALLS = 30.0, 47.0, 11.0, 16  # 51106 (mm)


def txt(r):
    try:
        return r["result"]["content"][0]["text"]
    except Exception:
        return json.dumps(r)[:400]


def geom():
    pitch_r = (BORE + OUTER) / 4.0
    ball_dia = 0.45 * HEIGHT
    r_g = 0.53 * ball_dia
    land_off = 0.7 * r_g
    w = math.sqrt(r_g * r_g - land_off * land_off)  # groove half-width where the arc meets the land
    return pitch_r, ball_dia, r_g, land_off, w


def washer_volume():
    """One washer, by shells: V = integral over x of 2*pi*x*(z_top(x) + H/2) dx. z_top is the flat
    land at −land_off except in the groove band, where it dips to the arc floor (the two washers are
    mirror-equal, so double this)."""
    pitch_r, _, r_g, land_off, w = geom()
    bore_r, outer_r = BORE / 2.0, OUTER / 2.0

    def z_top(x):  # top boundary of the shaft washer's meridian (measured from the mid-plane)
        if abs(x - pitch_r) <= w:
            return -math.sqrt(max(r_g * r_g - (x - pitch_r) ** 2, 0.0))  # groove floor
        return -land_off

    n, acc, dx = 6000, 0.0, (outer_r - bore_r) / 6000
    for i in range(n):
        x = bore_r + (i + 0.5) * dx
        acc += x * (z_top(x) + HEIGHT / 2.0)
    return 2 * math.pi * acc * dx


def analytic_volume():
    _, ball_dia, _, _, _ = geom()
    balls = BALLS * (4.0 / 3.0) * math.pi * (ball_dia / 2.0) ** 3
    return 2 * washer_volume() + balls


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
    for _ in range(60):
        try:
            vol = json.loads(txt(mcp.call("analysis_mass_properties", {}))).get("volumeMm3")
        except Exception:
            vol = None
        if vol is not None and last is not None and abs(vol - last) < 1e-2:
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
    exp = analytic_volume()
    err = abs(vol - exp) / exp * 100 if vol else None
    print(f"volume live={vol} mm^3  analytic(grooved washers + balls)={exp:.0f}  err={err:.3f}%")
    if vol is None or err > 2.0:
        raise SystemExit(f"FAIL: bearing volume {vol} off analytic {exp:.0f} by {err}% (>2%)")
    mcp.call("set_view_orientation", {"document": 0, "orientation": 10759, "fit": True})
    time.sleep(1.5)
    out = outdir.rstrip("/\\") + "/thrust_51106_groove.png"
    mcp.call("capture_viewport", {"path": out})
    print("captured:", out, "PASS")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else ".")
