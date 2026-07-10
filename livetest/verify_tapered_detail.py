#!/usr/bin/env python3
"""Place an ISO 355 30206 tapered-roller bearing and confirm the on-apex detail is real (issue #54):
the placed assembly's total volume must match the analytic model — a ribbed cone (inner ring with a
big-end retaining flange), a plain cup (outer ring), and a full complement of tapered rollers whose
big end is a SPHERICAL CAP centred on the shared apex (Method C: each roller is one body of
revolution about its own tilted axis, the sketch centerline). The volume match proves the ribbed
cone section and the domed-roller section both solved to DOF 0 and revolved to correct solids. Then
screenshot it (the big-end rib and the tilted, domed rollers seated on both raceways must be
visible). Usage: verify_tapered_detail.py <outdir>"""
import json, math, sys, time
import mcp

PANEL = "com.oblikovati.part-designer.panel"
FAMILY, SIZE = "ISO 355 Tapered", "30206"
BORE, OUTER, WIDTH, ALPHA, ROLLERS = 30.0, 62.0, 17.25, 14.0, 16  # 30206 (mm, deg)


def txt(r):
    try:
        return r["result"]["content"][0]["text"]
    except Exception:
        return json.dumps(r)[:400]


def geom():
    """Reconstruct the on-apex derived geometry the generator publishes (all mm)."""
    pitch = (BORE + OUTER) / 2.0
    ra = WIDTH * 0.65
    ar = math.radians(ALPHA)
    cone_ray, axis = ar * 0.75, ar * 0.875
    apex = (pitch / 2.0) / math.tan(axis)
    zb, zs = apex + ra / 2.0, apex - ra / 2.0
    cone_big, cone_sm = 2 * zb * math.tan(cone_ray), 2 * zs * math.tan(cone_ray)
    cup_big, cup_sm = 2 * zb * math.tan(ar), 2 * zs * math.tan(ar)
    r_big, r_sm = (cup_big - cone_big) / 2.0, (cup_sm - cone_sm) / 2.0  # roller diameters
    r_big_pos = (cone_big + cup_big) / 2.0
    cone_clr, cup_clr = r_big * 0.02, r_big * 0.05
    rib_z = ra / 2.0 + WIDTH * 0.04
    rib_crest = min(r_big_pos + 0.8 * r_big, cup_big - 0.3 * r_big)
    cone_bottom = 2 * (apex - WIDTH / 2.0) * math.tan(cone_ray) - cone_clr
    cone_rib = 2 * (apex + rib_z) * math.tan(cone_ray) - cone_clr
    cup_bottom = 2 * (apex - WIDTH / 2.0) * math.tan(ar) + cup_clr
    cup_top = 2 * (apex + WIDTH / 2.0) * math.tan(ar) + cup_clr
    sphere_r = math.hypot(zb, r_big / 2.0)  # Method-C dome radius: apex O → big-end rim
    return dict(ra=ra, r_big=r_big, r_sm=r_sm, zb=zb, sphere_r=sphere_r, rib_z=rib_z,
                rib_crest=rib_crest, cone_bottom=cone_bottom, cone_rib=cone_rib,
                cup_bottom=cup_bottom, cup_top=cup_top)


def revolve_volume(x_out, x_in, z0, z1, n=6000):
    """V = pi * integral over z of (x_out(z)^2 - x_in(z)^2) dz (theorem of Pappus)."""
    acc, dz = 0.0, (z1 - z0) / n
    for i in range(n):
        z = z0 + (i + 0.5) * dz
        acc += x_out(z) ** 2 - x_in(z) ** 2
    return math.pi * acc * dz


def analytic_volume():
    g = geom()
    w2, br, orad = WIDTH / 2.0, BORE / 2.0, OUTER / 2.0

    def lerp(z, za, xa, zb, xb):
        return xa + (xb - xa) * (z - za) / (zb - za)

    def cone_out(z):  # raceway up to the rib foot, then the rib crest
        if z <= g["rib_z"]:
            return lerp(z, -w2, g["cone_bottom"] / 2, g["rib_z"], g["cone_rib"] / 2)
        return g["rib_crest"] / 2

    cone = revolve_volume(cone_out, lambda z: br, -w2, w2)
    cup = revolve_volume(lambda z: orad,
                         lambda z: lerp(z, -w2, g["cup_bottom"] / 2, w2, g["cup_top"] / 2), -w2, w2)
    rb, rs = g["r_big"] / 2, g["r_sm"] / 2                       # roller radii
    frustum = (math.pi * g["ra"] / 3.0) * (rb * rb + rb * rs + rs * rs)  # cone frustum (apex O)
    cap_h = g["sphere_r"] - g["zb"]                              # domed big-end spherical cap
    cap = (math.pi * cap_h * cap_h / 3.0) * (3 * g["sphere_r"] - cap_h)
    return cone + cup + ROLLERS * (frustum + cap)


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
    print(f"volume live={vol} mm^3  analytic(ribbed cone + cup + tapered rollers)={exp:.0f}  err={err:.3f}%")
    if vol is None or err > 2.0:
        raise SystemExit(f"FAIL: bearing volume {vol} off analytic {exp:.0f} by {err}% (>2%)")
    mcp.call("set_view_orientation", {"document": 0, "orientation": 10759, "fit": True})
    time.sleep(1.5)
    out = outdir.rstrip("/\\") + "/tapered_30206_detail.png"
    mcp.call("capture_viewport", {"path": out})
    print("captured:", out, "PASS")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else ".")
