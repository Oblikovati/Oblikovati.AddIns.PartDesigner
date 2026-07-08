#!/usr/bin/env python3
"""Minimal MCP streamable-HTTP client for driving the Oblikovati bridge live-test."""
import json, sys, urllib.request

URL = "http://127.0.0.1:7800/mcp"
SESSION = {"id": None}


def _post(payload, want_body=True):
    data = json.dumps(payload).encode()
    headers = {"Content-Type": "application/json",
               "Accept": "application/json, text/event-stream"}
    if SESSION["id"]:
        headers["Mcp-Session-Id"] = SESSION["id"]
    req = urllib.request.Request(URL, data=data, headers=headers)
    with urllib.request.urlopen(req, timeout=120) as resp:
        sid = resp.headers.get("Mcp-Session-Id")
        if sid:
            SESSION["id"] = sid
        raw = resp.read().decode()
    if not want_body:
        return None
    # SSE frames: lines starting with "data: "
    for line in raw.splitlines():
        line = line.strip()
        if line.startswith("data:"):
            return json.loads(line[5:].strip())
    if raw.strip():
        return json.loads(raw)
    return None


def initialize():
    r = _post({"jsonrpc": "2.0", "id": 0, "method": "initialize",
               "params": {"protocolVersion": "2024-11-05",
                          "capabilities": {},
                          "clientInfo": {"name": "b2-live", "version": "0"}}})
    _post({"jsonrpc": "2.0", "method": "notifications/initialized"}, want_body=False)
    return r


def call(name, args, rid=1):
    r = _post({"jsonrpc": "2.0", "id": rid, "method": "tools/call",
               "params": {"name": name, "arguments": args}})
    return r


def list_tools():
    return _post({"jsonrpc": "2.0", "id": 2, "method": "tools/list"})


if __name__ == "__main__":
    initialize()
    cmd = sys.argv[1]
    if cmd == "tools":
        res = list_tools()
        names = [t["name"] for t in res["result"]["tools"]]
        print("\n".join(names))
    elif cmd == "call":
        name = sys.argv[2]
        args = json.loads(sys.argv[3]) if len(sys.argv) > 3 else {}
        res = call(name, args)
        print(json.dumps(res, indent=2)[:2000])
