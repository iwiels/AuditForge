# AuditForge — AI-Powered Security Audit Orchestrator

[![Go Version](https://img.shields.io/badge/go-1.25.0-blue.svg)](https://golang.org)
[![Platform](https://img.shields.io/badge/Platform-OpenCode--first-green.svg)](https://opencode.com)
[![License](https://img.shields.io/badge/License-MIT-brightgreen.svg)](LICENSE)

> **Forge your security audits with precision.** AuditForge is the AI-powered orchestrator that transforms OpenCode into a methodological security audit platform — not another tool launcher, but a **Methodological Brain** for authorized defensive audits.

---

## Philosophy: Methodology > Automation

AuditForge elevates AI-assisted cybersecurity practice. We don't automate chaos; we orchestrate technical discipline under four unwavering pillars:

1. **Methodology-First**: Aligned with OWASP, PTES and OSSTMM. Nothing executes without a methodological "why."
2. **Authorization-First**: Engagement requires explicit scope and authorization before allowing active actions.
3. **Evidence-First**: Every finding is backed by observable, verifiable evidence — eliminating model hallucinations.
4. **Defensive-by-Default**: Prioritizes reconnaissance, design analysis and remediation over destructive exploitation.

---

## Security Agent Team

AuditForge injects into OpenCode a team of specialized agents operating via **shared memory** (MCP) and a structured findings protocol:

| Agent | Domain | Operational Focus |
|:---|:---|:---|
| **`security-scout`** | Recon & Reconnaissance | Passive OSINT, surface, TLS/vhosts and fingerprinting |
| **`security-web`** | Web & API Intel | Deep JS analysis, endpoints, authn/authz and web logic |
| **`security-code`** | Source & Supply Chain | SAST, secrets, dependencies and pipeline review |
| **`security-ops`** | Architecture & Compliance | Risk assessment, threat modeling and remediation |
| **`security-report`** | Synthesis & Reporting | Severity correlation, CWE/OWASP mapping and final report |
| **`security-memory`** | Contextual Continuity | Findings deduplication and inter-session handoff |

---

## Capabilities: Skills & Tools

### Security Skills (29 total)

**Recon & Surface:**
- `surface-discovery` —robots.txt, security.txt, DNS, certificate transparency
- `network-recon` — Port discovery, service enumeration, TLS inspection
- `tls-vhost-enum` — TLS/SAN vhost enumeration
- `osint-passive` — Passive OSINT, Shodan/Censys integration
- `archive-intel` — Historical URLs and archived responses

**Web & Dynamic Analysis:**
- `web-triage` — OWASP-based authentication, access control, input validation
- `web-js-intel` — JavaScript endpoint extraction, deobfuscation
- `js-deobfuscation-intel` — Advanced JS deobfuscation techniques
- `browser-api-mapping` — Chrome DevTools API surface inventory
- `request-interception-manipulation` — **NEW** Fetch/XHR interception, replay, token manipulation
- `websocket-security` — **NEW** CSWSH, injection, authorization analysis
- `proxy-capture-replay` — mitmproxy capture, HAR export, replay verification

**Authentication & Tokens:**
- `jwt-jwks-analysis` — **NEW** JWT algorithm confusion, key injection, kid path traversal
- `advanced-auth-bypass` — OAuth/SAML, CSRF, session fixation

**API Security:**
- `api-schema-harvest` — OpenAPI normalization, endpoint discovery
- `api-parameter-mapping` — Parameter inference and mapping
- `param-discovery-fuzz` — Hidden parameter discovery

**Input Validation:**
- `file-upload-attacks` — Extension bypass, MIME spoofing, polyglot files
- `deserialization-attacks` — Java/PHP/Python/.NET gadget chains

**Proxy & Interception:**
- `auditforge-proxy` — **NEW** Native HTTP/HTTPS proxy with MCP interface for traffic interception
- `smart-replay-engine` — **NEW** Differential analysis engine for automated vulnerability detection (IDOR, auth bypass, privilege escalation)
- `proxy-capture-replay` — mitmproxy capture, HAR export, replay verification

**Code & Supply Chain:**
- `code-review` — SAST patterns, secure coding anti-patterns
- `supply-chain-triage` — Dependency analysis, CVE correlation

**Analysis & Reporting:**
- `vulnerability-correlation` — CWE/OWASP mapping, severity scoring
- `evidence-reporting` — Structured finding format, reproducibility
- `threat-modeling` — Attack surface, trust boundaries
- `compliance-check` — Regulatory compliance verification
- `incident-response` — Breach analysis and containment

**Safety:**
- `authorization-guard` — Scope validation, policy enforcement
- `sqli-hypothesis-validation` — Bounded SQL injection confirmation

### AuditForge Proxy MCP (Native HTTP/HTTPS Interception)

AuditForge includes a **native proxy server** for comprehensive traffic interception. It's automatically installed and configured:

```javascript
// The proxy starts automatically with OpenCode (or manually)
// Configure your browser to use localhost:8080

// Start intercepting
proxy.intercept.enable({
  filters: {
    host_pattern: "api.target.com",
    path_pattern: "/api/v1"
  }
})

// Review captured requests
proxy.history.search({ limit: 20 })

// Modify and replay requests
proxy.request.modify({
  request_id: "abc-123",
  headers: { "Authorization": "Bearer ADMIN_TOKEN" },
  body: JSON.stringify({ role: "admin" })
})

// Smart Replay with differential analysis
proxy.replay.execute({
  request_id: "abc-123",
  smart_mode: true  // Auto-generates mutations for IDOR/auth testing
})

// Review automated findings
proxy.findings.list({ severity: "CRITICAL" })
```

**Key Features:**
- ✅ **Auto-installed** with `./install-assets.sh` or `.\install-assets.ps1`
- ✅ Intercepts traffic from **any application** (browsers, mobile apps, CLI tools)
- ✅ **MITM HTTPS** with dynamic certificate generation
- ✅ **Burp-style interception**: pause, modify, forward, drop
- ✅ **Smart Replay Engine**: automated mutation testing with differential analysis
- ✅ **Auto-detection**: IDOR, auth bypass, privilege escalation, info disclosure
- ✅ **Persistent storage** in SQLite with HAR export

**Manual Start (if needed):**
```bash
~/.auditforge/proxy/start-proxy.sh        # Linux/macOS
%USERPROFILE%\.auditforge\proxy\start-proxy.bat  # Windows
```

### Chrome DevTools MCP Integration

AuditForge uses the `chrome-devtools` MCP for dynamic web analysis:

```javascript
// Capture all XHR/fetch requests
chrome-devtools_list_network_requests({ resourceTypes: ["xhr", "fetch"] })

// Get full request/response details
chrome-devtools_get_network_request({ reqid: N })

// Intercept and modify requests before they leave the browser
chrome-devtools_evaluate_script({
  function: `() => {
    // Install fetch interceptor
    window.__origFetch = window.fetch;
    window.fetch = async (url, opts) => {
      console.log('Request:', url, opts);
      return window.__origFetch(url, opts);
    };
  }`
})

// DOM snapshot for evidence
chrome-devtools_take_snapshot({ verbose: true })
chrome-devtools_take_screenshot({ fullPage: true })
```

---

## Quick Start

### 1. Install the Binary

```bash
# Linux/macOS
curl -sL https://raw.githubusercontent.com/victo/auditforge/main/install.sh | bash

# Or download from releases
curl -fsSL https://github.com/victo/auditforge/releases/latest/download/auditforge -o ~/bin/auditforge
chmod +x ~/bin/auditforge
```

### 2. Install Assets (Skills, Agents, MCP Config, Proxy Server)

**Option A: PowerShell (Windows - Recommended)**
```powershell
# One-line install (includes proxy server if Go is installed)
.\install-assets.ps1

# Or specific components only
.\install-assets.ps1 -Proxy    # Build proxy server only
.\install-assets.ps1 -Skills   # Skills only
.\install-assets.ps1 -All      # Everything (default)
```

**Option B: Bash (Linux/macOS/WSL)**
```bash
# One-line install (includes proxy server if Go is installed)
chmod +x install-assets.sh
./install-assets.sh --all

# Or specific components
./install-assets.sh --proxy    # Build proxy server only
./install-assets.sh --skills   # Skills only
```

**Option C: Batch (Windows Legacy)**
```cmd
install-assets.bat
```

This installs:
- **29 security skills** to `~/.auditforge/skills/`
- **6 agent definitions** to `~/.config/opencode/agents/`
- **10 command definitions** to `~/.config/opencode/commands/`
- **MCP configurations** (Chrome DevTools + AuditForge Proxy)
- **AuditForge Proxy Server** (if Go is installed) to `~/.auditforge/proxy/`

### 3. Restart OpenCode

```bash
# Verify installation
/opencode
/memory-search test
```

### 4. Start Your First Audit

```bash
# Full team audit
/team https://target.com

# Or phase by phase
/scout https://target.com
/deep-web https://target.com
/report
```

### 5. Advanced: Proxy-based Interception (Auto-Installed)

The AuditForge Proxy is automatically installed and configured. Just start it:

```bash
# The proxy was installed to ~/.auditforge/proxy/
# Start it with:
~/.auditforge/proxy/start-proxy.sh        # Linux/macOS
%USERPROFILE%\.auditforge\proxy\start-proxy.bat  # Windows

# Or manually:
cd ~/.auditforge/proxy
./auditforge-proxy
```

Then configure your browser to use `localhost:8080` as HTTP/HTTPS proxy.

In OpenCode, use MCP tools:
```javascript
proxy.intercept.enable({})
proxy.history.search({limit: 20})
proxy.replay.execute({request_id: "abc-123", smart_mode: true})
proxy.findings.list({severity: "CRITICAL"})
```

---

## Project Structure

```
auditforge/
├── cmd/                    # CLI and MCP server entrypoints
│   └── proxy-server/       # NEW: Native HTTP/HTTPS proxy with Smart Replay
│       ├── main.go
│       ├── proxy_core.go
│       ├── smart_replay.go
│       ├── storage.go
│       ├── mcp_tools.go
│       ├── setup.sh
│       └── README.md
├── internal/
│   ├── assets/
│   │   ├── skills/         # 29 security skills (27 + 2 new)
│   │   │   ├── auditforge-proxy/      # NEW: Proxy integration skill
│   │   │   └── smart-replay-engine/   # NEW: Differential analysis skill
│   │   ├── opencode/       # OpenCode agents & commands
│   │   └── prompts/        # System prompts
│   ├── orchestrator/       # Injection engine
│   ├── mcp/                # MCP protocol server
│   ├── memory/             # Shared memory store (Engram-style)
│   ├── model/              # Domain types
│   └── catalog/            # Skills registry
├── docs/                   # Architecture & operational docs
├── install.sh              # Binary installer
├── install-assets.sh       # Assets installer (Linux/macOS)
└── install-assets.bat      # Assets installer (Windows)
```

---

## Documentation

- [**Architecture Deep Dive**](docs/architecture.md)
- [**Tool Matrix**](docs/tool-matrix.md)
- [**Security Operations Policy**](docs/security-ops.md)
- [**Agent Team Protocol**](AGENTS.md)
- [**Chrome DevTools MCP Guide**](#chrome-devtools-mcp-integration)
- [**AuditForge Proxy Setup**](cmd/proxy-server/README.md) - HTTP/HTTPS proxy with Smart Replay Engine
- [**Smart Replay Guide**](cmd/proxy-server/EXAMPLE.md) - Practical audit example
- [**Proxy Integration Skill**](internal/assets/skills/auditforge-proxy/SKILL.md)
- [**Smart Replay Skill**](internal/assets/skills/smart-replay-engine/SKILL.md)

---

## Contributing

Want to improve AuditForge? Read our [**CONTRIBUTING.md**](CONTRIBUTING.md).

**Golden rule**: Methodology first, code second. We don't integrate tools that don't provide reproducible evidence.

---

*AuditForge — Forging the standard of AI-assisted security auditing.*
