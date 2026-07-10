#!/usr/bin/env python3
"""Place an ISO 104 53206 self-aligning thrust ball bearing (532xx) and confirm the sphered seat is
real (#54): the placed assembly's total volume must match the analytic model — a grooved shaft
washer, a grooved housing washer whose BACK is a concave conical seat (a shallow cone chord of the
seat sphere), a separate seat washer whose concave underside nests over it, and a full ball
complement — proving both washer sections solved to DOF 0 and revolved to correct solids (the earlier
near-flat back ARCs did not converge and collapsed the section; conical backs are exact and well-
conditioned). Then screenshot it (the domed seat washer must sit outboard of the housing). Usage:
verify_self_aligning.py <outdir>"""
import json, math, sys, time
import mcp

PANEL = "com.oblikovati.part-designer.panel"
FAMILY, SIZE = "ISO 104 Self-Aligning", "53206"
BORE, OUTER, HEIGHT, BALLS = 30.0, 47.0, 11.0, 16  # 53206 (mm)


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
    w = math.sqrt(r_g * r_g - land_off * land_off)
    cap = 0.06 * OUTER
    sphere_r = ((OUTER / 2) ** 2 + cap * cap) / (2 * cap)
    centre_z = HEIGHT / 2 + math.sqrt(sphere_r ** 2 - (OUTER / 2) ** 2)
    seat_od = 1.12 * OUTER
    seat_top = HEIGHT / 2 + 0.35 * HEIGHT
    seat_r = sphere_r - 0.02 * HEIGHT
    return dict(pitch_r=pitch_r, ball_dia=ball_dia, r_g=r_g, land=land_off, w=w,
                sphere_r=sphere_r, centre_z=centre_z, seat_od=seat_od, seat_top=seat_top, seat_r=seat_r)


def shells(x0, x1, lo, hi, n=8000):
    """V = integral 2*pi*x*(hi(x) - lo(x)) dx by the shell method."""
    acc, dx = 0.0, (x1 - x0) / n
    for i in range(n):
        x = x0 + (i + 0.5) * dx
        acc += x * (hi(x) - lo(x))
    return 2 * math.pi * acc * dx


def grooved_front(g, x, sign):
    """The ball-facing land at sign*land_off, dipping to the arc floor (sign*sqrt) in the groove band."""
    if abs(x - g["pitch_r"]) <= g["w"]:
        return sign * math.sqrt(max(g["r_g"] ** 2 - (x - g["pitch_r"]) ** 2, 0.0))
    return sign * g["land"]


def chord(x0, z0, x1, z1):
    """Straight line z(x) through the two rim stations — the shallow cone that replaces the sub-floor
    seat arc (the built back edge; see pinConicalBack)."""
    m = (z1 - z0) / (x1 - x0)
    return lambda x: z0 + m * (x - x0)


def analytic_volume():
    g = geom()
    bore_r, outer_r = BORE / 2, OUTER / 2
    seat_or = g["seat_od"] / 2
    # Shaft washer (lower): flat back at -H/2, grooved front on top (toward the balls).
    shaft = shells(bore_r, outer_r, lambda x: -HEIGHT / 2, lambda x: grooved_front(g, x, -1))
    # Housing washer (upper): grooved front below, concave conical back above (chord of the seat sphere
    # between the bore-back and OD-back rim stations — the near-flat arc is sub-floor, so it is built as
    # a shallow cone, < 0.1 mm off the sphere).
    z_od = g["centre_z"] - math.sqrt(g["sphere_r"] ** 2 - outer_r ** 2)
    z_bore = g["centre_z"] - math.sqrt(g["sphere_r"] ** 2 - bore_r ** 2)
    z_back = chord(bore_r, z_bore, outer_r, z_od)
    housing = shells(bore_r, outer_r, lambda x: grooved_front(g, x, +1), z_back)
    # Seat washer: concave conical underside (chord of its own, slightly smaller sphere) below a flat top.
    zs_od = g["centre_z"] - math.sqrt(g["seat_r"] ** 2 - seat_or ** 2)
    zs_bore = g["centre_z"] - math.sqrt(g["seat_r"] ** 2 - bore_r ** 2)
    z_seat = chord(bore_r, zs_bore, seat_or, zs_od)
    seat = shells(bore_r, seat_or, z_seat, lambda x: g["seat_top"])
    balls = BALLS * (4.0 / 3.0) * math.pi * (g["ball_dia"] / 2) ** 3
    return shaft + housing + seat + balls


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
    print(f"volume live={vol} mm^3  analytic(shaft + conical housing + seat washer + balls)={exp:.0f}  err={err:.3f}%")
    if vol is None or err > 0.5:
        raise SystemExit(f"FAIL: bearing volume {vol} off analytic {exp:.0f} by {err}% (>0.5%)")
    mcp.call("set_view_orientation", {"document": 0, "orientation": 10759, "fit": True})
    time.sleep(1.5)
    out = outdir.rstrip("/\\") + "/thrust_53206_self_aligning.png"
    mcp.call("capture_viewport", {"path": out})
    print("captured:", out, "PASS")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else ".")
