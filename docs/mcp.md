# MCP

The orchestrator exposes a lightweight stdio MCP surface for integration with AI clients.

Current exposed tools focus on audit operations and scanner capabilities.

Use MCP when you want an external agent to request:

- reconnaissance
- web fingerprinting
- secret scanning
- dependency scanning
- report rendering

## Built-in Audit Tools

- `audit.scout` infer target kind, stack, package manager, and project assets
- `audit.dispatch` predict which internal audit agents should run for a target
- `audit.teams` run the internal security subagent runtime against a persisted state file
- `audit.report` render Markdown from a persisted `*.state.json` file

## Scanner Wrappers

- `tool.nmap`
- `tool.whatweb`
- `tool.katana`
- `tool.jsluice`
- `tool.browser.capture`
- `tool.har.inspect`
- `tool.openapi.inspect`
- `tool.mitmproxy.capture`
- `tool.mitmproxy2swagger`
- `tool.nuclei`
- `tool.nikto`
- `tool.sqlmap`
- `tool.searchsploit`
- `tool.arjun`
- `tool.ffuf`
- `tool.waymore`
- `tool.semgrep`
- `tool.trivy`
- `tool.grype`
- `tool.gitleaks`

## Notes

- `tools/call` is wired for the listed tools.
- `tools/list` now includes `inputSchema` metadata for typed invocation.
- Scanner wrappers return raw tool output.
- Tools that generate artifacts like `Arjun`, `ffuf`, `waymore`, HAR capture, and `mitmproxy2swagger` return parsed summaries when possible.
- `audit.report` expects a `stateFile` argument pointing to a persisted audit state JSON file.
- Client-side `sync` preserves unrelated OpenCode settings through JSON merge instead of full overwrite.
- Browser and JS intel can enrich the attack surface even when active scanners are unavailable.
