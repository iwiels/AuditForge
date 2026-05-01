# Changelog

## [2.0.0] - 2024-01-15 - Proxy & Smart Replay Release

### 🚀 New Features

#### AuditForge Proxy Server
- **Native HTTP/HTTPS Proxy** (`cmd/proxy-server/`)
  - Intercepts traffic from any application (browsers, mobile apps, CLI tools)
  - MITM HTTPS with dynamic certificate generation
  - SQLite persistence for request/response history
  - MCP interface for OpenCode integration

#### Smart Replay Engine
- **Differential Analysis Engine** (`smart_replay.go`)
  - Automated mutation generation for security testing
  - Detects: IDOR, auth bypass, privilege escalation, info disclosure, timing attacks
  - Semantic comparison of baseline vs variations
  - Automatic finding generation with CWE mapping

#### New Skills
- **`auditforge-proxy`** - Proxy integration and usage documentation
- **`smart-replay-engine`** - Differential analysis workflows and examples

### 🔧 MCP Tools Added

```javascript
// Interception
proxy.intercept.enable({filters})
proxy.intercept.disable()

// History
proxy.history.search({host, path, method, limit})
proxy.request.get({request_id})

// Modification
proxy.request.modify({request_id, headers, body})
proxy.request.forward({request_id})
proxy.request.drop({request_id, status_code})

// Replay & Analysis
proxy.replay.execute({request_id, smart_mode, mutations})
proxy.findings.list({severity, type})
proxy.stats.get()
proxy.export.har({output_path, filters})
```

### 📊 Detection Capabilities

| Vulnerability | Detection Method | Severity |
|---------------|------------------|----------|
| Authentication Bypass | Status code change (401→200) | CRITICAL |
| IDOR | 404→200 with ID mutation | HIGH |
| Privilege Escalation | Schema field differences | MEDIUM |
| Information Disclosure | Error message analysis | MEDIUM |
| Timing Attacks | Response time differential | LOW |

### 🔐 Security

- CA certificate management for HTTPS interception
- Configurable interception filters
- Automatic scope enforcement
- Secure storage of sensitive data

### 📚 Documentation

- New proxy setup guide (`cmd/proxy-server/README.md`)
- Practical example walkthrough (`cmd/proxy-server/EXAMPLE.md`)
- Skill documentation for both new capabilities
- Automated setup script (`setup.sh`)

---

## [1.0.0] - Initial Release

### Core Features
- Security agent team (6 specialized agents)
- 27 security skills
- Chrome DevTools MCP integration
- Methodology-first approach
- Authorization and evidence tracking
- OpenCode-first architecture

### Agents
- security-scout (Reconnaissance)
- security-web (Web/API analysis)
- security-code (Source code review)
- security-ops (Compliance & architecture)
- security-report (Synthesis & reporting)
- security-memory (Context continuity)
