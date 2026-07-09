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

## Note on reading results mid-recompute

A part recomputes asynchronously off the session goroutine after Place, so its features and
bodies appear incrementally. Always poll `analysis_mass_properties` until the volume is
**stable** (the `settle()` helper) before reading `get_model_tree` / counting bodies —
reading too early can catch a partial state (e.g. a bearing showing only its balls before the
ring revolves have recomputed). Each rolling bearing's ball/roller count comes straight from
the family's `ball_count`/`roller_count` column (e.g. deep-groove 6204 = 8 balls, 6205 = 9).
