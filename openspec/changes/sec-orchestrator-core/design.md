# Design: Go Security Orchestrator

## Technical Approach

We will implement a **command-and-pipeline** orchestration in Go. The system will use a specialized **Normalizer Layer** to ensure that data from tools as different as `nmap` (XML) and config scans (filesystem) can be integrated into a coherent state.

## Architecture Decisions

### Decision: Universal Finding Schema (UFS)

**Choice**: Go structs for all tool outputs.
**Rationale**: By forcing all tools to map their output to a `Finding` object, the orchestrator can perform cross-tool analysis (e.g., finding the same vulnerability via two different tools).

### Decision: Dynamic Dispatch Mapping

**Choice**: Conditional dispatch in the Go pipeline.
**Rationale**: Instead of a static sequence, the pipeline uses reconnaissance data to decide which auditors to activate.
- `if ports.80 == open -> activate web_audit`
- `if target is a project directory -> activate dependency audit`

### Decision: AI-Driven Vulnerability Review

**Choice**: Pluggable review client.
**Rationale**: Security tools (Nmap, Searchsploit) produce noise. A review client can correlate assets, service versions, and raw evidence to confirm if a vulnerability is likely exploitable in the specific target context, reducing manual triage time.

### Decision: Weaponization Artifacts

**Choice**: Metasploit Resource Scripts (`.rc`).
**Rationale**: Instead of executing exploits (risky), we generate the exact sequence of commands needed in `msfconsole`. This follows the "Human-in-the-Loop" safety principle.

## Data Flow

    [Recon Step] ──> [Audit Dispatch] ──> [Audit State]
                                            │
                                            ▼
                                   [Critical Review]
                                   (review client)
                                            │
               ┌────────────────────────────┴────────────────────────────┐
               ▼                                                         ▼
    [Confirmed Findings]                                        [False Positives]
               │                                                 (Marked/Logged)
               ▼
    [Metasploit Agent]
    (Generate .rc scripts)
               │
               ▼
    [Executive Reporter]

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/llm/client.go` | Create | Review client abstraction. |
| `internal/orchestrator/pipeline.go` | Modify | Expand pipeline with Critical and Weaponization phases. |
| `internal/report/report.go` | Create | Markdown report generator. |

## Interfaces / Contracts

### Universal Finding Schema (UFS)
```go
type Finding struct {
    ID          string
    Title       string
    Severity    Severity
    Description string
    Evidence    string
    Location    string
    Remediation string
    ToolSource  string
}
```

### Tool Execution Contract
```go
func RunKaliTool(ctx context.Context, cmd string, args []string) (string, error)
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Parsers | Feed tools' raw output files (XML/JSON/Text) and verify UFS object. |
| Integration | Dispatch routing | Verify that opening a mock port triggers the correct auditor. |
| E2E | Report accuracy | Verify vulnerability correlation (Recon find + exploit map). |

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Stack detection logic | Mock directory structures with key files. |
| Integration | Pipeline traversal | Test start-to-finish audit with mock reviewers. |
| E2E | Report generation | Verify file content against a list of findings. |

## Migration / Rollout

No migration required. This is a new project.

## Open Questions

- [ ] Should we use Pydantic for the finding models instead of raw Dicts? (Yes, recommended for validation).
- [ ] How to handle tool dependencies (e.g., semgrep not installed) gracefully? (Scout should detect tool availability).
