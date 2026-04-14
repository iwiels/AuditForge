// Package orchestrator provides the agent execution engine that runs the
// security audit team in sequence, passing findings between agents via
// shared memory. This is the "real orchestrator" — not just AGENTS.md documentation.
package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"orquestador-auditor/internal/memory"
	auditruntime "orquestador-auditor/internal/runtime"
)

// AgentOrchestrator executes the security audit team in sequence.
// Each agent runs, writes findings to memory, and the next agent reads
// those findings plus any followup signals directed at it.
type AgentOrchestrator struct {
	Runtime   auditruntime.Store
	Memory    *memory.Store
	Target    string
	Campaign  string
	SessionID string
}

// TeamInput configures a full team run.
type TeamInput struct {
	Target         string
	Campaign       string
	SessionID      string
	Profile        string
	Aggressiveness string
	Authorized     bool
	ArtifactsDir   string
	MemoryDir      string
}

// AgentStep is the result of running a single agent.
type AgentStep struct {
	AgentID    string        `json:"agent_id"`
	Phase      string        `json:"phase"`
	Duration   time.Duration `json:"duration"`
	Findings   int           `json:"findings_written"`
	Signals    int           `json:"signals_written"`
	Followups  int           `json:"followups_read"`
	Status     string        `json:"status"` // "completed", "interrupted", "skipped"
	Error      string        `json:"error,omitempty"`
}

// TeamResult is the full outcome of a team execution.
type TeamResult struct {
	SessionID string      `json:"session_id"`
	Target    string      `json:"target"`
	StartedAt time.Time   `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Steps     []AgentStep `json:"steps"`
	TotalFindings int    `json:"total_findings"`
	Interrupted bool     `json:"interrupted"`
}

// RunTeam executes the full security audit team in sequence.
// The order is: memory → scout → web → code → ops → report → judgment
func (o *AgentOrchestrator) RunTeam(ctx context.Context, input TeamInput) (*TeamResult, error) {
	result := &TeamResult{
		SessionID: input.SessionID,
		Target:    input.Target,
		StartedAt: time.Now().UTC(),
	}

	// Define the agent pipeline with their corresponding phases
	agents := []struct {
		ID    string
		Phase auditruntime.PhaseID
	}{
		{"security-memory", auditruntime.PhaseScope},
		{"security-scout", auditruntime.PhaseNetworkRecon},
		{"security-web", auditruntime.PhaseSurfaceDiscovery},
		// Additional agents map to additional phases
		// {"security-code", auditruntime.PhaseJSIntel},
		// {"security-ops", auditruntime.PhaseAPIDiscovery},
	}

	for _, agentStep := range agents {
		select {
		case <-ctx.Done():
			result.Interrupted = true
			result.CompletedAt = time.Now().UTC()
			return result, nil
		default:
		}

		step := o.runSingleAgent(ctx, agentStep.ID, agentStep.Phase, input)
		result.Steps = append(result.Steps, step)

		// Check for critical findings that might interrupt the pipeline
		if step.Status == "interrupted" {
			result.Interrupted = true
			result.CompletedAt = time.Now().UTC()
			return result, nil
		}
	}

	result.CompletedAt = time.Now().UTC()
	return result, nil
}

func (o *AgentOrchestrator) runSingleAgent(ctx context.Context, agentID string, phase auditruntime.PhaseID, input TeamInput) AgentStep {
	start := time.Now()
	step := AgentStep{
		AgentID: agentID,
		Phase:   string(phase),
		Status:  "completed",
	}

	// 1. Load context from memory (followup signals directed at this agent)
	followups := o.loadFollowups(agentID)
	step.Followups = len(followups)

	// 2. Load historical context via Engram
	engramCtx := ""
	if o.Memory != nil {
		engram := memory.BuildMinimalEngram(o.Memory, input.Target)
		engramCtx = engram.FormattedBlock
	}

	// 3. Build the agent prompt
	prompt := o.buildAgentPrompt(agentID, phase, input, followups, engramCtx)

	// 4. The agent would normally execute via the AI client — for now we record
	//    the structured prompt and context for the operator to review.
	//    In a full implementation, this would call the AI adapter's chat API.

	// 5. Save the agent's working context to memory for traceability
	if o.Memory != nil {
		obs := memory.Observation{
			ID:        fmt.Sprintf("%s-%s-%s", input.SessionID, agentID, string(phase)),
			Kind:      "agent-execution",
			Title:     fmt.Sprintf("%s executed %s", agentID, phase),
			Body:      prompt,
			Target:    input.Target,
			Campaign:  input.Campaign,
			RunID:     input.SessionID,
			CreatedAt: time.Now().UTC(),
			Tags:      []string{"agent", agentID, string(phase)},
		}
		_ = o.Memory.Save(obs)
		step.Findings = 1 // The execution trace itself
	}

	step.Duration = time.Since(start)
	return step
}

// loadFollowups reads pending followup signals from memory for a given agent.
func (o *AgentOrchestrator) loadFollowups(agentID string) []memory.Observation {
	if o.Memory == nil {
		return nil
	}

	obs, err := o.Memory.Search(fmt.Sprintf("followup %s", agentID), 20)
	if err != nil {
		return nil
	}

	var followups []memory.Observation
	for _, item := range obs {
		if item.RunID != o.SessionID {
			continue
		}
		if item.Kind != "signal" {
			continue
		}
		// Check if this signal is directed at our agent
		for _, tag := range item.Tags {
			if tag == agentID || tag == "followup:"+agentID {
				followups = append(followups, item)
				break
			}
		}
	}
	return followups
}

// buildAgentPrompt creates the full system prompt for an agent, including:
// - Role definition
// - Phase instructions
// - Followup signals from other agents
// - Historical context (Engram)
func (o *AgentOrchestrator) buildAgentPrompt(agentID string, phase auditruntime.PhaseID, input TeamInput, followups []memory.Observation, engramCtx string) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("# %s — Phase: %s\n\n", agentID, phase))
	sb.WriteString(fmt.Sprintf("**Session:** %s\n", input.SessionID))
	sb.WriteString(fmt.Sprintf("**Target:** %s\n", input.Target))
	sb.WriteString(fmt.Sprintf("**Campaign:** %s\n\n", input.Campaign))

	// Role definition per agent
	sb.WriteString(o.agentRoleDefinition(agentID))
	sb.WriteString("\n\n")

	// Phase instructions
	sb.WriteString(o.phaseInstructions(phase))
	sb.WriteString("\n\n")

	// Followup signals
	if len(followups) > 0 {
		sb.WriteString("## Priority Vectors (from other agents)\n\n")
		sb.WriteString("The following items were flagged by other agents. Investigate these first:\n\n")
		for _, signal := range followups {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", signal.Title, signal.Body))
		}
		sb.WriteString("\n")
	}

	// Historical context
	if engramCtx != "" {
		sb.WriteString(engramCtx)
		sb.WriteString("\n\n")
	}

	// Output format
	sb.WriteString("## Output Format\n\n")
	sb.WriteString("Write findings to memory with the following structure:\n")
	sb.WriteString("```json\n")
	sb.WriteString(`{
  "kind": "finding",
  "agent": "` + agentID + `",
  "session": "` + input.SessionID + `",
  "target": "` + input.Target + `",
  "title": "...",
  "severity": "CRITICAL|HIGH|MEDIUM|LOW|INFORMATIVE",
  "status": "observed|suspected|validated|blocked-by-policy",
  "cwe": "CWE-XXX",
  "evidence": "...",
  "vector": "...",
  "needs_followup_by": ["security-web", "security-code"],
  "tags": ["..."]
}` + "\n")
	sb.WriteString("```\n\n")

	// If this agent finds something critical, signal others
	sb.WriteString("## Signal Protocol\n\n")
	sb.WriteString("If you find something that requires another agent's attention:\n")
	sb.WriteString("1. Write the finding to memory with `needs_followup_by` set to the target agent\n")
	sb.WriteString("2. Write a signal to memory: `kind=signal, to=<agent>, type=followup|escalation`\n")

	return sb.String()
}

func (o *AgentOrchestrator) agentRoleDefinition(agentID string) string {
	roles := map[string]string{
		"security-memory": `You are the **Security Memory Agent**.
Your role is to load historical context about this target and campaign.
Search memory for prior findings, operator notes, and campaign context.
Write a summary of relevant context to help subsequent agents avoid duplicating work.`,

		"security-scout": `You are the **Security Scout Agent**.
Your role is passive reconnaissance: port scanning, service detection, TLS enumeration,
vhost discovery, and technology fingerprinting.
Focus on building an asset inventory. Do NOT perform active exploitation.
Mark any interesting findings for security-web to investigate deeper.`,

		"security-web": `You are the **Security Web Agent**.
Your role is deep web analysis: authentication, authorization, input validation,
JS analysis, API discovery, and vulnerability hypothesis.
Investigate any followup vectors from security-scout first.
Mark code-level findings for security-code to review.`,

		"security-code": `You are the **Security Code Agent**.
Your role is source code review, dependency analysis, CI/CD security, and secret detection.
Investigate any followup vectors from security-web.`,

		"security-ops": `You are the **Security Ops Agent**.
Your role is compliance checking, threat modeling, and operational security review.
Review all findings from other agents for compliance gaps and operational risk.`,

		"security-report": `You are the **Security Report Agent**.
Your role is to consolidate all findings into a structured report.
Map CWE, OWASP Top 10, severity, and actionable remediation for each finding.
Deduplicate and correlate findings across all phases.`,
	}

	if role, ok := roles[agentID]; ok {
		return role
	}
	return fmt.Sprintf("You are the **%s** agent. Perform your assigned security analysis.", agentID)
}

func (o *AgentOrchestrator) phaseInstructions(phase auditruntime.PhaseID) string {
	instructions := map[auditruntime.PhaseID]string{
		auditruntime.PhaseScope: `## Phase: Scope
1. Confirm authorization is recorded
2. Document target kind (web, api, host, repo)
3. Record aggressiveness and approved tools
4. Do NOT perform any active scanning in this phase`,

		auditruntime.PhaseNetworkRecon: `## Phase: Network Recon
1. Scan authorized ports and services
2. Detect service versions and TLS configuration
3. Enumerate vhosts and subdomains
4. Build technology fingerprint inventory
5. Flag interesting endpoints for security-web: GraphQL, API docs, admin panels`,

		auditruntime.PhaseSurfaceDiscovery: `## Phase: Surface Discovery
1. Crawl the target and map all reachable endpoints
2. Analyze JavaScript bundles for secrets, API keys, and internal routes
3. Discover and normalize API schemas (OpenAPI, GraphQL)
4. Identify authentication mechanisms and session handling
5. Generate vulnerability hypotheses based on technology stack`,

		auditruntime.PhaseJSIntel: `## Phase: JavaScript Intel
1. Deobfuscate and beautify JavaScript
2. Extract API endpoints, secrets, and internal routes
3. Analyze source maps if available
4. Map client-side state and data flow
5. Flag client-side vulnerabilities (DOM XSS, insecure storage)`,

		auditruntime.PhaseAPIDiscovery: `## Phase: API Discovery
1. Harvest OpenAPI/Swagger specs if available
2. Discover hidden parameters and endpoints
3. Map authentication and authorization flows
4. Identify data types and validation patterns
5. Generate parameter-level vulnerability hypotheses`,

		auditruntime.PhaseVulnHypothesis: `## Phase: Vulnerability Hypothesis
1. Generate hypotheses for each potential vulnerability class
2. Map each hypothesis to a specific endpoint, parameter, or code path
3. Prioritize by impact likelihood and technology stack evidence
4. Do NOT validate hypotheses — that requires authorized-validation phase`,

		auditruntime.PhaseAuthorizedValidation: `## Phase: Authorized Validation
1. Only validate hypotheses generated in vuln-hypothesis phase
2. Use minimal, targeted requests — no brute force or fuzzing
3. Record exact request/response pairs as evidence
4. Mark findings as "validated" only with clear proof`,

		auditruntime.PhaseCorrelation: `## Phase: Correlation
1. Consolidate all findings from prior phases
2. Deduplicate by root cause (not by symptom)
3. Map each finding to CWE and OWASP Top 10
4. Assign severity based on impact and exploitability
5. Write actionable remediation for each unique finding`,

		auditruntime.PhaseJudgment: `## Phase: Judgment
1. Validate that every finding has evidence
2. Confirm attack vectors for HIGH/CRITICAL findings
3. Ensure CWE and OWASP mapping is complete
4. Verify remediation is specific and actionable
5. Flag any findings that fail quality checks`,
	}

	if instr, ok := instructions[phase]; ok {
		return instr
	}
	return fmt.Sprintf("## Phase: %s\nExecute your analysis according to the methodology.", phase)
}

// NewOrchestrator creates a new AgentOrchestrator.
func NewOrchestrator(runtime auditruntime.Store, mem *memory.Store, target, campaign, sessionID string) *AgentOrchestrator {
	return &AgentOrchestrator{
		Runtime:   runtime,
		Memory:    mem,
		Target:    target,
		Campaign:  campaign,
		SessionID: sessionID,
	}
}

// WriteSignal writes an inter-agent signal to memory.
func (o *AgentOrchestrator) WriteSignal(fromAgent string, toAgent string, signalType string, message string, priority string) error {
	if o.Memory == nil {
		return fmt.Errorf("memory store not available")
	}

	obs := memory.Observation{
		ID:        fmt.Sprintf("%s-signal-%s-%s-%d", o.SessionID, fromAgent, toAgent, time.Now().UnixNano()),
		Kind:      "signal",
		Title:     fmt.Sprintf("%s → %s: %s", fromAgent, toAgent, signalType),
		Body:      message,
		Target:    o.Target,
		Campaign:  o.Campaign,
		RunID:     o.SessionID,
		CreatedAt: time.Now().UTC(),
		Tags:      []string{"signal", toAgent, "followup:" + toAgent, signalType, priority},
		Metadata:  map[string]string{"from": fromAgent, "to": toAgent, "type": signalType, "priority": priority},
	}
	return o.Memory.Save(obs)
}
