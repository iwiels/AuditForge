# Security Policy

## Supported Versions

This project is evolving quickly. Until formal release channels stabilize, please assume only the latest published release on GitHub is supported for security fixes.

## Reporting a Vulnerability

Please **do not open a public issue** for suspected vulnerabilities in Orquestador Auditor.

Instead, report privately with:

- affected version or commit SHA
- operating system and environment details
- reproduction steps
- impact assessment
- logs, screenshots, or proof-of-concept if available

Send reports to:

- GitHub Security Advisories / private vulnerability reporting when enabled
- or the maintainer contact listed in the repository profile

## Response Targets

We will try to:

- acknowledge receipt within **3 business days**
- validate or reject the report within **7 business days**
- provide a remediation plan or status update after triage

These targets are goals, not guarantees.

## Disclosure Expectations

Please allow reasonable time for validation and remediation before public disclosure.

We prefer coordinated disclosure so users can patch safely.

## Scope

Security reports are especially relevant for:

- release and installer integrity
- MCP server exposure and unsafe tool behavior
- privilege escalation through client sync/injection flows
- unsafe handling of secrets, tokens, prompts, or memory data
- backup / restore flows that could corrupt or expose user config

## Out of Scope

The following are generally out of scope unless they demonstrably affect this repository itself:

- vulnerabilities in third-party security tools installed separately by the user
- unsafe operator behavior outside the documented workflow
- unsupported local modifications to generated config after sync
