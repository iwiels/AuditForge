# Proposal: Security Audit Orchestrator (Go Edition)

## Intent

Build a professional-grade multi-agent security orchestration system ("SecOrchestrator") in Go, inspired by the **Gentle AI** orchestration structure. The system coordinates Kali Linux tools, uses a lightweight review client for intelligent reasoning and false-positive filtering, and prepares Metasploit RC files for authorized security operations.

## Scope

### In Scope
- **AI-Powered "Critical Agent"**: Integration with a pluggable review client to analyze tool outputs, filter noise, and rank vulnerabilities.
- **Metasploit Weaponization**: Automated generation of `msfconsole` resource scripts (.rc) based on verified exploits.
- **Gentle AI Workflow Alignment**: Implementation of structured phases (Recon, Analysis, Critical Review, Weaponization, Reporting) with persistent state and package boundaries modeled after gentle-ai.
- **Category-based Agent Registry**: Specialized agents for Web, Network, and Exploitation.
- **Local/Remote Support**: Optimized for Kali Linux but usable against project directories and URLs.

### Out of Scope
- Automatic execution of Metasploit exploits (must remain HITL).
- Bypassing of legal/authorized boundaries (target must be authorized).

## Capabilities

### New Capabilities
- `critical-review`: review-based analysis of findings to eliminate false positives.
- `metasploit-weaponization`: Generating ready-to-use `.rc` scripts for verified vulnerabilities.
- `review-client`: Unified interface for review providers.

### Modified Capabilities
- `audit-orchestration`: Scaling to a multi-phase professional pipeline.

## Approach

We will implement a **multi-phase pipeline** in Go.
1.  **Phase I: Recon**: Passive/Active discovery (Nmap/Scout).
2.  **Phase II: Audit**: Tool dispatch (Nikto/Sqlmap/Searchsploit).
3.  **Phase III: Critical Filtering**: The review client inspects state, compares evidence, and marks findings as confirmed or false positive.
4.  **Phase IV: Weaponization**: For confirmed high-severity findings, the Metasploit agent generates the session scripts.
5.  **Phase V: Professional Reporting**: Executive Markdown summary with linked exploits and RC scripts.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/orchestrator/` | Modified | Scaling the pipeline to include Review and Weaponization. |
| `internal/llm/` | New | Review client abstractions. |
| `internal/report/` | New | Markdown report generator. |
| `internal/model/` | New | Audit state, assets, findings, and severity types. |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| LLM Hallucinations | Med | Use local evidence (tool raw output) as grounding for the AI reviews. |
| API Costs/Rate Limits | Med | Implement caching for LLM calls and optimized prompt sizing. |
| Payload Safety | Low | The orchestrator only generates commands, never executes 'exploit' automatically. |

## Dependencies

- Python 3.12+
- Kali Linux tools
- LangGraph / LangChain
- API Keys for Claude (Anthropic) or Gemini (Google Cloud/AI Studio).
- Metasploit Framework.
