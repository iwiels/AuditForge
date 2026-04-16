# Agents

This project uses the `gentle-ai` idea of an orchestrator plus specialist phases.

It now also includes an internal native security-team runtime inside the binary, so multi-agent analysis is not limited to external client capabilities.

## Internal Phases

- `Scout` - detects stack, manifests, and target kind
- `Dispatch` - activates config, dependency, web, and correlator work
- `Review` - normalizes and confirms findings
- `Report` - writes report and weaponization artifacts

## Supported Adapter Model

Adapters are responsible for injecting MCP config and prompt guidance into external AI clients.

| Agent | Skills | MCP | Output Styles | Slash Commands | Native Agent Selector | Subagents | Config Path |
|-------|--------|-----|---------------|----------------|-----------------------|-----------|-------------|
| Claude Code | Yes | Yes | Yes | Yes | No | No | `~/.claude` |
| Claude | Yes | Yes | Yes | Yes | No | No | `~/.claude` |
| Cursor | Yes | Yes | No | No | No | Yes | `~/.cursor` |
| OpenCode | Yes | Yes | No | Yes | Yes | No | `~/.config/opencode` |

Security-focused assets now injected by client capability:

- skills for defensive analysis and evidence reporting
- advanced skills for JS intel, browser API mapping, proxy capture, schema harvest, parameter discovery, and archive intelligence
- Claude output styles for operator-grade remediation output
- Cursor subagents for scout, web, supply-chain, and reporting roles
- OpenCode commands for reconnaissance and reporting flows

OpenCode receives native named agents in `opencode.json`:

- `security-orchestrator`
- `security-scout`
- `security-web`
- `security-report`
- `security-memory`

Claude Code does not expose the same native visual agent selector. Its correct integration model remains commands, skills, MCP, output styles, and the native Task tool.
