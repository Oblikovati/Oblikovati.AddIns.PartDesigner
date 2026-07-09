#!/usr/bin/env python3
"""Place an ISO 15 6205 deep-groove ball bearing and confirm the ground race grooves are real
(issue #53): the placed assembly's total volume must match the analytic model — two grooved rings
(each a revolved meridian whose raceway carries a conformity groove, integrated by Pappus) plus a
full complement of balls — proving the grooved ring sections solved to DOF 0 and revolved to correct
solids (not degenerate/empty). Then screenshot it. Usage: verify_ball_groove.py <outdir>"""
import json, math, sys, time
import mcp

PANEL = "com.oblikovati.part-designer.panel"
FAMILY, SIZE = "ISO 15 Deep Groove", "6205"
BORE, OUTER, WIDTH, BALLS = 25.0, 52.0, 15.0, 9  # 6205 (mm)


def txt(r):
    try:
        return r["result"]["content"][0]["text"]
    except Exception:
        return json.dumps(r)[:400]


def analytic_volume():
    """Total = inner grooved ring + outer grooved ring + N balls, in mm^3."""
    gap = OUTER - BORE
    pitch_r = (BORE + OUTER) / 4.0
    ball_r = 0.28 * gap / 2.0
    r_g = 0.52 * (0.28 * gap)            # groove arc radius
    z_s = r_g * math.sqrt(1 - 0.55 ** 2)  # groove half-axial-span
    inner_sh = pitch_r - 0.55 * r_g       # inner shoulder radius
    outer_sh = pitch_r + 0.55 * r_g
    bore_r, outer_r, w2 = BORE / 2.0, OUTER / 2.0, WIDTH / 2.0

    def ring_volume(inner):
        # V = pi * integral over z of (X_out(z)^2 - X_in(z)^2) dz  (theorem of Pappus)
        N, acc = 4000, 0.0
        for i in range(N):
            z = -w2 + (i + 0.5) * WIDTH / N
            if inner:
                x_out = pitch_r - math.sqrt(r_g * r_g - z * z) if abs(z) <= z_s else inner_sh
                acc += x_out * x_out - bore_r * bore_r
            else:
                x_in = pitch_r + math.sqrt(r_g * r_g - z * z) if abs(z) <= z_s else outer_sh
                acc += outer_r * outer_r - x_in * x_in
        return math.pi * acc * WIDTH / N

    balls = BALLS * (4.0 / 3.0) * math.pi * ball_r ** 3
    return ring_volume(True) + ring_volume(False) + balls


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
    print(f"volume live={vol} mm^3  analytic(grooved rings + balls)={exp:.0f}  err={err:.3f}%")
    if vol is None or err > 2.0:
        raise SystemExit(f"FAIL: bearing volume {vol} off analytic {exp:.0f} by {err}% (>2%)")
    mcp.call("set_view_orientation", {"document": 0, "orientation": 10759, "fit": True})
    time.sleep(1.5)
    out = outdir.rstrip("/\\") + "/ball_bearing_6205_groove.png"
    mcp.call("capture_viewport", {"path": out})
    print("captured:", out, "PASS")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else ".")
