# Security Ops

This project is intended to become a **defensive security orchestration layer for AI clients**, not a one-click offensive toolkit.

## Operating posture

- **authorization-first**: no engagement starts without explicit scope and authorization
- **methodology-first**: prioritize structured review over random tool execution
- **evidence-first**: every finding must preserve the observation that justified it
- **defensive-by-default**: prefer discovery, analysis, hardening, and remediation guidance
- **human-in-the-loop**: a human operator decides when to escalate beyond low-risk actions

## Default workflow

1. Confirm target kind, scope, and authorization.
2. Start with read-only discovery and passive review.
3. Escalate only when the methodology calls for it and the engagement allows it.
4. Keep evidence attached to each finding.
5. Report remediation, not just exploitation paths.

## Non-goals

- blind offensive automation
- high-risk scanner orchestration without operator review
- unsupported claims without evidence
- silent privilege escalation in AI clients
