# Live end-to-end test

`e2e_place_all.py` drives a running Oblikovati head over the MCP bridge to exercise the
Part Designer add-in end to end: it places one representative member from **every** top-level
category, verifies each is a healthy parametric part (published parameters + named features,
no sick features), captures a golden screenshot, re-drives a part by editing a driver
parameter, and confirms the assembly-occurrence path.

It is a **manual/developer** test (it needs a live head with a GPU/llvmpipe context and the
MCP bridge), not a CI test — CI runs the cgo-free unit suite (`go test ./designer/...`).

## Running it

1. Build and install both add-ins into the head's add-in directory:

   ```sh
   make build   # -> build/oblikovati-part-designer.so
   ```

   Copy `oblikovati-part-designer.so` and `oblikovati-mcp-bridge.so` into a directory, e.g.
   `$ADDINS`.

2. Launch the head with the add-ins and a display:

   ```sh
   DISPLAY=:1 OBK_ADDINS_DIR="$ADDINS" ./oblikovati-head &
   # wait until the bridge answers: curl -s -o /dev/null -w '%{http_code}' 127.0.0.1:7800/mcp  (== 400)
   ```

3. Run the test (screenshots land in `outdir`, default `.`):

   ```sh
   python3 e2e_place_all.py /path/to/outdir
   ```

   It prints one `[ok]/[FAIL]` line per check and exits non-zero if any fails.

`mcp.py` is a minimal self-contained MCP streamable-HTTP client (no third-party deps).

## Known limitation

On small deep-groove ball bearings (e.g. 6204) the kernel's circular pattern can drop a
single ball copy at tight ball spacing, so 8 balls render where `ball_count` is 9. The
add-in requests the correct count via the `ball_count` parameter; the drop is a host-side
pattern-robustness edge, not add-in logic. Larger sizes (e.g. 6205) render the full count.
