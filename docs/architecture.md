# Architecture

Orquestador Auditor is evolving toward an **OpenCode-first, security-first orchestration platform**.

## Architectural intent

The product is organized around five concerns:

1. **OpenCode integration** — inject prompts, skills, commands, MCP config, and specialist agent overlays into OpenCode.
2. **Methodology delivery** — package security workflows, personas, specialist agents, and audit profiles in reusable assets.
3. **Memory and context** — persist observations so engagements can accumulate context over time.
4. **Safety and control** — backup user config before sync, verify post-sync state, and keep dangerous behavior explicit.
5. **MCP interoperability** — expose a local MCP server that OpenCode can call for orchestration and context.

## Current flow

1. Detect platform and resolve OpenCode as the default runtime.
2. Resolve the audit profile (`recon`, `web-triage`, `supply-chain`, `reporting`, `memory-only`).
3. Map the profile to:
   - assets to inject
   - agent overlay membership
   - OpenCode tool permissions (`read/write/edit/bash`)
   - prompt guardrails and MCP metadata
4. Backup managed files before modifying them.
5. Inject prompts, commands, skills, MCP config, and agent overlays filtered by profile.
6. Prune managed assets that no longer belong to the active profile.
7. Verify the resulting state and preserve rollback options.

## Design notes

- OpenCode is the primary product runtime.
- Profiles are not cosmetic: they govern both asset distribution and operational risk.
- Prefer evidence-backed output over model guesswork.
- Optimize for safe orchestration, not raw offensive automation.
- Default to least privilege (`recon`) instead of injecting every asset and capability by default.
