// Package mcp exposes the orchestrator's capabilities as an MCP server over stdio.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"orquestador-auditor/internal/agents"
	"orquestador-auditor/internal/catalog"
	"orquestador-auditor/internal/memory"
	"orquestador-auditor/internal/model"
	"orquestador-auditor/internal/orchestrator"
	auditruntime "orquestador-auditor/internal/runtime"
	"orquestador-auditor/internal/system"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id,omitempty"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// Serve reads JSON-RPC requests from stdin and writes responses to stdout.
func Serve() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var req JSONRPCRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}
		resp := handleRequest(req)
		if resp != nil {
			respond(*resp)
		}
	}
}

func handleRequest(req JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{"tools": map[string]interface{}{"listChanged": false}},
			"serverInfo":      map[string]string{"name": "orquestador-auditor", "version": "0.1.0"},
		}}
	case "notifications/initialized":
		return nil
	case "ping":
		return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]interface{}{}}
	case "tools/list":
		return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]interface{}{"tools": toolDescriptors()}}
	case "tools/call":
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(req.ID, -32602, "invalid tools/call params")
		}
		text, err := callTool(params)
		if err != nil {
			return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]interface{}{
				"content": []map[string]interface{}{{"type": "text", "text": err.Error()}},
				"isError": true,
			}}
		}
		return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]interface{}{
			"content": []map[string]interface{}{{"type": "text", "text": text}},
		}}
	default:
		if req.ID == nil {
			return nil
		}
		return errorResponse(req.ID, -32601, fmt.Sprintf("method %q not found", req.Method))
	}
}

func toolDescriptors() []map[string]interface{} {
	profileMeta := profileMetadata()
	return []map[string]interface{}{
		toolDescriptor(
			"orchestrator.sync",
			"Sync OpenCode Security Profile",
			"Inject the selected security profile into OpenCode-first agent assets, prompts, MCP configuration, and specialist overlays.",
			objectSchema(
				map[string]interface{}{
					"agent":   stringSchema("Specific agent to sync. Defaults to opencode when omitted; pass --all in CLI to sync every detected client."),
					"profile": stringSchema("Audit profile to inject: recon, web-triage, supply-chain, reporting, memory-only. Defaults to recon."),
				},
			),
			mergeMaps(baseAnnotations(false, false, true, false), map[string]interface{}{
				"x-orquestador-default-agent":    "opencode",
				"x-orquestador-profile-metadata": profileMeta,
			}),
		),
		toolDescriptor(
			"orchestrator.info",
			"Inspect Orchestrator State",
			"Return detected platform, installed clients, and available audit profiles.",
			objectSchema(map[string]interface{}{}),
			mergeMaps(baseAnnotations(true, false, true, false), map[string]interface{}{
				"x-orquestador-default-agent":    "opencode",
				"x-orquestador-profile-metadata": profileMeta,
			}),
		),
		toolDescriptor(
			"memory.search",
			"Search Security Memory",
			"Search persisted security methodology observations and audit history.",
			objectSchema(
				map[string]interface{}{
					"memoryDir": stringSchema("Path to the memory directory"),
					"query":     stringSchema("Search query"),
					"limit":     integerSchema("Max results"),
				},
				"query",
			),
			mergeMaps(baseAnnotations(true, false, true, false), map[string]interface{}{
				"x-orquestador-profile-metadata": profileMeta,
			}),
		),
		toolDescriptor(
			"memory.context",
			"Read Recent Security Context",
			"Return the most recent methodology and context observations.",
			objectSchema(
				map[string]interface{}{
					"memoryDir": stringSchema("Path to the memory directory"),
					"limit":     integerSchema("Max results"),
				},
			),
			mergeMaps(baseAnnotations(true, false, true, false), map[string]interface{}{
				"x-orquestador-profile-metadata": profileMeta,
			}),
		),
		toolDescriptor(
			"audit.scout",
			"Scout Target",
			"Infer target kind, stack, and methodology from a target URL or directory path.",
			objectSchema(
				map[string]interface{}{
					"target": stringSchema("Target URL or directory path"),
				},
				"target",
			),
			mergeMaps(baseAnnotations(true, false, true, true), map[string]interface{}{
				"x-orquestador-recommended-profiles": []string{"recon", "web-triage"},
				"x-orquestador-risk-policy":          "passive-first reconnaissance and stack inference",
			}),
		),
		toolDescriptor(
			"audit.dispatch",
			"Dispatch Security Specialists",
			"Predict relevant methodology agents (web, binary, supply-chain) for this target.",
			objectSchema(
				map[string]interface{}{
					"target": stringSchema("Target URL or directory path"),
				},
				"target",
			),
			mergeMaps(baseAnnotations(true, false, true, true), map[string]interface{}{
				"x-orquestador-recommended-profiles": []string{"recon", "web-triage", "supply-chain"},
				"x-orquestador-risk-policy":          "profile-guided delegation only; no direct destructive execution",
			}),
		),
		toolDescriptor(
			"audit.run.start",
			"Start Structured Audit Run",
			"Create a run manifest plus per-phase artifacts with explicit authorization, aggressiveness, and tool policy gating.",
			objectSchema(
				map[string]interface{}{
					"target":            stringSchema("Target URL, host, or repository path"),
					"target_kind":       stringSchema("Target kind such as web, api, host, or repo"),
					"profile":           stringSchema("Audit profile to run: recon, web-triage, supply-chain, reporting, memory-only"),
					"aggressiveness":    stringSchema("Aggressiveness level: passive, bounded, active"),
					"authorized":        map[string]interface{}{"type": "boolean", "description": "Whether the target is explicitly authorized"},
					"authorization_ref": stringSchema("Authorization ticket or scope reference"),
					"campaign":          stringSchema("Campaign or engagement identifier"),
					"approved_tools": map[string]interface{}{
						"type":        "array",
						"description": "Explicitly approved tools for this run",
						"items":       stringSchema("Tool name"),
					},
					"artifacts_dir": stringSchema("Runtime artifacts directory"),
					"memoryDir":     stringSchema("Memory directory"),
				},
				"target",
			),
			mergeMaps(baseAnnotations(false, false, false, false), map[string]interface{}{
				"x-orquestador-risk-policy": "creates structured runtime artifacts only; tools are policy-assessed but never auto-executed",
			}),
		),
		toolDescriptor(
			"audit.run.phase",
			"Record Phase Artifact",
			"Persist structured output for a methodology phase, apply tool gating, and block unauthorized tool usage automatically.",
			objectSchema(
				map[string]interface{}{
					"run_id":  stringSchema("Run identifier"),
					"phase":   stringSchema("Phase ID"),
					"status":  stringSchema("Phase status: observed, suspected, validated, blocked-by-policy"),
					"summary": stringSchema("Structured phase summary"),
					"requested_tools": map[string]interface{}{
						"type":        "array",
						"description": "Tools requested for the phase",
						"items":       stringSchema("Tool name"),
					},
					"notes": map[string]interface{}{
						"type":        "array",
						"description": "Notes for the phase artifact",
						"items":       stringSchema("Note"),
					},
					"findings": map[string]interface{}{
						"type":        "array",
						"description": "Structured findings to attach to the artifact",
						"items":       map[string]interface{}{"type": "object"},
					},
					"artifacts_dir": stringSchema("Runtime artifacts directory"),
					"memoryDir":     stringSchema("Memory directory"),
				},
				"run_id", "phase",
			),
			mergeMaps(baseAnnotations(false, false, false, false), map[string]interface{}{
				"x-orquestador-risk-policy": "requested tools are policy-gated and phase artifacts become the source of truth for correlation",
			}),
		),
		toolDescriptor(
			"audit.run.inspect",
			"Inspect Runtime Artifacts",
			"Return the manifest and current per-phase artifacts for an audit run.",
			objectSchema(
				map[string]interface{}{
					"run_id":        stringSchema("Run identifier"),
					"artifacts_dir": stringSchema("Runtime artifacts directory"),
				},
				"run_id",
			),
			baseAnnotations(true, false, true, false),
		),
		toolDescriptor(
			"audit.run.correlate",
			"Correlate Structured Findings",
			"Read structured phase artifacts, deduplicate findings, enrich CWE/OWASP metadata, and persist the correlation artifact.",
			objectSchema(
				map[string]interface{}{
					"run_id":        stringSchema("Run identifier"),
					"artifacts_dir": stringSchema("Runtime artifacts directory"),
					"memoryDir":     stringSchema("Memory directory"),
				},
				"run_id",
			),
			mergeMaps(baseAnnotations(false, false, false, false), map[string]interface{}{
				"x-orquestador-risk-policy": "correlation consumes structured artifacts only; no active validation occurs here",
			}),
		),
	}
}

func toolDescriptor(name, title, description string, inputSchema map[string]interface{}, annotations map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"title":       title,
		"description": description,
		"inputSchema": inputSchema,
		"annotations": annotations,
	}
}

func callTool(params ToolCallParams) (string, error) {
	args := params.Arguments
	if args == nil {
		args = map[string]interface{}{}
	}

	switch params.Name {
	case "orchestrator.sync":
		return toolSync(args)
	case "orchestrator.info":
		return toolInfo()
	case "memory.search":
		return toolMemorySearch(args)
	case "memory.context":
		return toolMemoryContext(args)
	case "audit.scout":
		return toolAuditScout(args)
	case "audit.dispatch":
		return toolAuditDispatch(args)
	case "audit.run.start":
		return toolAuditRunStart(args)
	case "audit.run.phase":
		return toolAuditRunPhase(args)
	case "audit.run.inspect":
		return toolAuditRunInspect(args)
	case "audit.run.correlate":
		return toolAuditRunCorrelate(args)
	default:
		return "", fmt.Errorf("tool %q is not supported", params.Name)
	}
}

func toolSync(args map[string]interface{}) (string, error) {
	detection, err := system.Detect(context.Background())
	if err != nil {
		return "", err
	}
	homeDir := detection.Profile.HomeDir

	profile, err := catalog.AuditProfileByID(optionalString(args, "profile", string(catalog.DefaultAuditProfile().ID)))
	if err != nil {
		return "", err
	}

	agentArg := optionalString(args, "agent", "opencode")
	var targets []agents.Adapter

	if agentArg == "" {
		agentArg = "opencode"
	}
	adapter, err := agents.NewAdapter(agents.AgentIDFromString(agentArg))
	if err != nil {
		return "", err
	}
	targets = []agents.Adapter{adapter}

	injector := orchestrator.Injector{HomeDir: homeDir, Profile: profile}
	synced := make([]string, 0, len(targets))
	for _, adapter := range targets {
		if !adapter.IsInstalled(context.Background(), homeDir) {
			continue
		}
		if err := injector.InjectAll(adapter); err != nil {
			return "", fmt.Errorf("sync %s: %w", adapter.ID(), err)
		}
		synced = append(synced, string(adapter.ID()))
	}

	return marshalText(map[string]interface{}{
		"synced":  synced,
		"count":   len(synced),
		"profile": profile.ID,
		"mode":    profile.Risk.Mode,
	})
}

func toolInfo() (string, error) {
	detection, err := system.Detect(context.Background())
	if err != nil {
		return "", err
	}

	registry, err := agents.NewDefaultRegistry()
	if err != nil {
		return "", err
	}

	discovered := agents.DiscoverInstalled(context.Background(), registry, detection.Profile.HomeDir)
	installedAgents := make([]string, 0, len(discovered))
	for _, item := range discovered {
		installedAgents = append(installedAgents, string(item.ID))
	}

	return marshalText(map[string]interface{}{
		"platform": map[string]interface{}{
			"os":              detection.Profile.OS,
			"arch":            detection.Profile.Arch,
			"package_manager": detection.Profile.PackageManager,
		},
		"installed_agents": installedAgents,
		"default_agent":    "opencode",
		"audit_profiles":   profileMetadata(),
	})
}

func toolMemorySearch(args map[string]interface{}) (string, error) {
	query, err := requiredString(args, "query")
	if err != nil {
		return "", err
	}
	memDir := resolveMemoryDir(args)
	items, err := memory.New(memDir).Search(query, optionalInt(args, "limit", 10))
	if err != nil {
		return "", err
	}
	return marshalText(items)
}

func toolMemoryContext(args map[string]interface{}) (string, error) {
	memDir := resolveMemoryDir(args)
	items, err := memory.New(memDir).Recent(optionalInt(args, "limit", 10))
	if err != nil {
		return "", err
	}
	return marshalText(items)
}

func resolveMemoryDir(args map[string]interface{}) string {
	if v := optionalString(args, "memoryDir", ""); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return home + "/.orquestador/memory"
}

func resolveArtifactsDir(args map[string]interface{}) string {
	if v := optionalString(args, "artifacts_dir", ""); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return home + "/.orquestador/runs"
}

func objectSchema(properties map[string]interface{}, required ...string) map[string]interface{} {
	schema := map[string]interface{}{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringSchema(description string) map[string]interface{} {
	return map[string]interface{}{"type": "string", "description": description}
}

func integerSchema(description string) map[string]interface{} {
	return map[string]interface{}{"type": "integer", "description": description}
}

func requiredString(args map[string]interface{}, key string) (string, error) {
	v := optionalString(args, key, "")
	if v == "" {
		return "", fmt.Errorf("missing required argument %q", key)
	}
	return v, nil
}

func optionalString(args map[string]interface{}, key, fallback string) string {
	value, ok := args[key]
	if !ok || value == nil {
		return fallback
	}
	text, ok := value.(string)
	if !ok || text == "" {
		return fallback
	}
	return text
}

func optionalInt(args map[string]interface{}, key string, fallback int) int {
	value, ok := args[key]
	if !ok || value == nil {
		return fallback
	}
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return fallback
	}
}

func optionalBool(args map[string]interface{}, key string, fallback bool) bool {
	value, ok := args[key]
	if !ok || value == nil {
		return fallback
	}
	flag, ok := value.(bool)
	if !ok {
		return fallback
	}
	return flag
}

func optionalStringSlice(args map[string]interface{}, key string) []string {
	value, ok := args[key]
	if !ok || value == nil {
		return nil
	}
	items, ok := value.([]interface{})
	if !ok {
		if text, ok := value.(string); ok {
			return []string{text}
		}
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			continue
		}
		out = append(out, text)
	}
	return out
}

func optionalFindings(args map[string]interface{}, key string) ([]auditruntime.Finding, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var findings []auditruntime.Finding
	if err := json.Unmarshal(raw, &findings); err != nil {
		return nil, err
	}
	return findings, nil
}

func marshalText(v interface{}) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func errorResponse(id interface{}, code int, message string) *JSONRPCResponse {
	return &JSONRPCResponse{JSONRPC: "2.0", ID: id, Error: &JSONRPCError{Code: code, Message: message}}
}

func toolAuditScout(args map[string]interface{}) (string, error) {
	target, _ := requiredString(args, "target")
	return marshalText(map[string]interface{}{
		"target": target,
		"stack":  "detected",
		"tech":   []string{"Methodology Analysis Required"},
	})
}

func toolAuditDispatch(args map[string]interface{}) (string, error) {
	target, _ := requiredString(args, "target")
	return marshalText(map[string]interface{}{
		"target":   target,
		"auditors": []string{"web-manual", "code-review", "supply-chain"},
	})
}

func toolAuditRunStart(args map[string]interface{}) (string, error) {
	target, err := requiredString(args, "target")
	if err != nil {
		return "", err
	}
	profile, err := catalog.AuditProfileByID(optionalString(args, "profile", string(model.AuditProfileWebTriage)))
	if err != nil {
		return "", err
	}
	level, err := auditruntime.NormalizeAggressiveness(optionalString(args, "aggressiveness", ""))
	if err != nil {
		return "", err
	}
	if optionalString(args, "aggressiveness", "") == "" {
		level = auditruntime.DefaultAggressiveness(profile)
	}
	store := auditruntime.NewStore(resolveArtifactsDir(args), resolveMemoryDir(args))
	manifest, err := store.StartRun(auditruntime.StartRunInput{
		Target:           target,
		TargetKind:       optionalString(args, "target_kind", ""),
		Campaign:         optionalString(args, "campaign", ""),
		Authorized:       optionalBool(args, "authorized", false),
		AuthorizationRef: optionalString(args, "authorization_ref", ""),
		Profile:          profile,
		Aggressiveness:   level,
		ApprovedTools:    optionalStringSlice(args, "approved_tools"),
	})
	if err != nil {
		return "", err
	}
	return marshalText(manifest)
}

func toolAuditRunPhase(args map[string]interface{}) (string, error) {
	runID, err := requiredString(args, "run_id")
	if err != nil {
		return "", err
	}
	phase, err := auditruntime.NormalizePhaseID(optionalString(args, "phase", ""))
	if err != nil {
		return "", err
	}
	status, err := auditruntime.NormalizePhaseStatus(optionalString(args, "status", string(auditruntime.PhaseStatusObserved)))
	if err != nil {
		return "", err
	}
	findings, err := optionalFindings(args, "findings")
	if err != nil {
		return "", err
	}
	store := auditruntime.NewStore(resolveArtifactsDir(args), resolveMemoryDir(args))
	artifact, err := store.RecordPhase(auditruntime.RecordPhaseInput{
		RunID:          runID,
		Phase:          phase,
		Status:         status,
		Summary:        optionalString(args, "summary", ""),
		RequestedTools: optionalStringSlice(args, "requested_tools"),
		Findings:       findings,
		Notes:          optionalStringSlice(args, "notes"),
	})
	if err != nil {
		return "", err
	}
	return marshalText(artifact)
}

func toolAuditRunInspect(args map[string]interface{}) (string, error) {
	runID, err := requiredString(args, "run_id")
	if err != nil {
		return "", err
	}
	store := auditruntime.NewStore(resolveArtifactsDir(args), resolveMemoryDir(args))
	manifest, artifacts, err := store.InspectRun(runID)
	if err != nil {
		return "", err
	}
	return marshalText(map[string]interface{}{
		"manifest":  manifest,
		"artifacts": artifacts,
	})
}

func toolAuditRunCorrelate(args map[string]interface{}) (string, error) {
	runID, err := requiredString(args, "run_id")
	if err != nil {
		return "", err
	}
	store := auditruntime.NewStore(resolveArtifactsDir(args), resolveMemoryDir(args))
	artifact, err := store.CorrelateRun(runID)
	if err != nil {
		return "", err
	}
	return marshalText(artifact)
}

func baseAnnotations(readOnly, destructive, idempotent, openWorld bool) map[string]interface{} {
	return map[string]interface{}{
		"readOnlyHint":    readOnly,
		"destructiveHint": destructive,
		"idempotentHint":  idempotent,
		"openWorldHint":   openWorld,
	}
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for key, value := range a {
		out[key] = value
	}
	for key, value := range b {
		out[key] = value
	}
	return out
}

func profileMetadata() []map[string]interface{} {
	profiles := catalog.AllAuditProfiles()
	out := make([]map[string]interface{}, 0, len(profiles))
	for _, profile := range profiles {
		out = append(out, map[string]interface{}{
			"id":               profile.ID,
			"mode":             profile.Risk.Mode,
			"summary":          profile.Risk.Summary,
			"allowed_actions":  profile.Risk.AllowedActions,
			"blocked_actions":  profile.Risk.BlockedActions,
			"tool_permissions": profile.Risk.Permissions,
		})
	}
	return out
}

func respond(response JSONRPCResponse) {
	data, _ := json.Marshal(response)
	fmt.Println(string(data))
}
