# Components

## Security Components

- `assets` - embedded prompts and command templates
- `backup` - snapshot and restore of managed client files before sync
- `verify` - post-sync integrity checks and reporting
- `system` - platform and dependency detection for secure runtime decisions
- `components/mcp` - MCP config injection
- `components/prompts` - system prompt injection
- `components/commands` - OpenCode slash command injection
- `components/skills` - defensive security skill pack injection
- `components/outputstyles` - Claude output-style injection for security reporting
- `components/subagents` - Cursor security subagent injection
- `components/filemerge` - safe JSON and markdown merge helpers
- `model` - audit state, assets, findings, severity
- `webintel` - JavaScript AST intel, dynamic browser capture, and OpenAPI harvesting
- `teams` - internal security subagent runtime for native multi-agent analysis
- `parsers` - `nmap`, `nikto`, `sqlmap`, `searchsploit`
- `tools` - bounded wrappers for scanner execution
- `report` - Markdown report and Metasploit RC generation
- `orchestrator` - engine, discovery, and phase orchestration

## What The System Looks For

- exposed HTTP/HTTPS services
- runtime XHR, fetch, websocket, and cookie-driven API flows
- HAR artifacts and replay-oriented traffic summaries
- embedded and external JavaScript endpoints, params, and secret-like artifacts
- public or harvestable OpenAPI schemas
- versioned network services
- secrets in config files
- container hardening gaps
- vulnerable dependencies
