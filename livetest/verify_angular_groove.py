#!/usr/bin/env python3
"""Place an ISO 15 7206-B angular-contact ball bearing and confirm the tilted race grooves are real
(issue #54): the placed assembly's total volume must match the analytic model — two rings whose
grooves are offset axially off the mid-plane by (r_g−R)·sin α so the contact normal tilts at the
contact angle, each flanked by an asymmetric tall retaining shoulder and a relieved low shoulder,
plus a full ball complement — proving the angular ring sections solved to DOF 0 and revolved to
correct solids (not degenerate/empty). Then screenshot it. Usage: verify_angular_groove.py <outdir>"""
import json, math, sys, time
import mcp

PANEL = "com.oblikovati.part-designer.panel"
FAMILY, SIZE = "ISO 15 Angular Contact", "7206-B"
BORE, OUTER, WIDTH, ALPHA, BALLS = 30.0, 62.0, 16.0, 40.0, 13  # 7206-B (mm, deg)


def txt(r):
    try:
        return r["result"]["content"][0]["text"]
    except Exception:
        return json.dumps(r)[:400]


def ring_bounds():
    """Derive the geometry both rings share: pitch radius, ball radius, groove radius, and the axial
    /radial groove-centre offsets for the contact angle (matching the generator's derived params)."""
    gap = OUTER - BORE
    pitch_r = (BORE + OUTER) / 4.0
    ball_r = 0.28 * gap / 2.0
    r_g = 0.52 * (0.28 * gap)
    off = r_g - ball_r                       # the ball-groove clearance radius (r_g − R)
    ar = math.radians(ALPHA)
    return pitch_r, ball_r, r_g, off * math.cos(ar), off * math.sin(ar)


def ring_volume(inner):
    """V = pi * integral over z of (X_out(z)^2 − X_in(z)^2) dz (Pappus). The outer ring's groove
    centre sits at (+radial, +axial), the inner ring's at (−radial, −axial); the tall shoulder is on
    the contact face (0.55·r_g off the groove centre) and the relieved shoulder (0.85·r_g) on the
    other. The far edge is the OD for the outer ring, the bore for the inner ring."""
    pitch_r, ball_r, r_g, gc_radial, gc_axial = ring_bounds()
    w2 = WIDTH / 2.0
    zs_high, zs_relief = r_g * math.sqrt(1 - 0.55 ** 2), r_g * math.sqrt(1 - 0.85 ** 2)
    if inner:
        gc_x, gc_z = pitch_r - gc_radial, -gc_axial
        hi_sh, re_sh = gc_x - 0.55 * r_g, gc_x - 0.85 * r_g   # shoulders open inward
        hi_end, re_end = gc_z - zs_high, gc_z + zs_relief     # tall shoulder on −z
        far_r = BORE / 2.0
    else:
        gc_x, gc_z = pitch_r + gc_radial, gc_axial
        hi_sh, re_sh = gc_x + 0.55 * r_g, gc_x + 0.85 * r_g   # shoulders open outward
        hi_end, re_end = gc_z + zs_high, gc_z - zs_relief     # tall shoulder on +z
        far_r = OUTER / 2.0

    def raceway_x(z):
        """The raceway (ball-facing) boundary radius at axial z: shoulder land, or the groove arc."""
        if inner:
            if z <= hi_end:
                return hi_sh
            if z >= re_end:
                return re_sh
            return gc_x - math.sqrt(max(r_g * r_g - (z - gc_z) ** 2, 0.0))
        if z >= hi_end:
            return hi_sh
        if z <= re_end:
            return re_sh
        return gc_x + math.sqrt(max(r_g * r_g - (z - gc_z) ** 2, 0.0))

    N, acc = 6000, 0.0
    for i in range(N):
        z = -w2 + (i + 0.5) * WIDTH / N
        x = raceway_x(z)
        acc += (x * x - far_r * far_r) if inner else (far_r * far_r - x * x)
    return math.pi * acc * WIDTH / N


def analytic_volume():
    _, ball_r, _, _, _ = ring_bounds()
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
    print(f"volume live={vol} mm^3  analytic(tilted-groove rings + balls)={exp:.0f}  err={err:.3f}%")
    if vol is None or err > 2.0:
        raise SystemExit(f"FAIL: bearing volume {vol} off analytic {exp:.0f} by {err}% (>2%)")
    mcp.call("set_view_orientation", {"document": 0, "orientation": 10759, "fit": True})
    time.sleep(1.5)
    out = outdir.rstrip("/\\") + "/angular_7206b_groove.png"
    mcp.call("capture_viewport", {"path": out})
    print("captured:", out, "PASS")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else ".")
