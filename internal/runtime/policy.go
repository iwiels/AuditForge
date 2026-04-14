package auditruntime

import (
	"fmt"
	"sort"
	"strings"

	"orquestador-auditor/internal/model"
)

type ToolPolicy struct {
	Name                     string
	Phases                   []PhaseID
	Profiles                 []model.AuditProfileID
	MinAggressiveness        Aggressiveness
	RequiresExplicitApproval bool
	Reason                   string
}

var aggressivenessRank = map[Aggressiveness]int{
	AggressivenessPassive: 1,
	AggressivenessBounded: 2,
	AggressivenessActive:  3,
}

var toolPolicies = map[string]ToolPolicy{
	"nmap": {
		Name:                     "nmap",
		Phases:                   []PhaseID{PhaseNetworkRecon},
		Profiles:                 []model.AuditProfileID{model.AuditProfileRecon, model.AuditProfileWebTriage},
		MinAggressiveness:        AggressivenessBounded,
		RequiresExplicitApproval: true,
		Reason:                   "Port and service enumeration is active network interaction and must be explicitly approved.",
	},
	"whatweb": {
		Name:              "whatweb",
		Phases:            []PhaseID{PhaseSurfaceDiscovery},
		Profiles:          []model.AuditProfileID{model.AuditProfileRecon, model.AuditProfileWebTriage},
		MinAggressiveness: AggressivenessBounded,
		Reason:            "Fingerprinting is allowed only in bounded or higher analysis modes.",
	},
	"katana": {
		Name:                     "katana",
		Phases:                   []PhaseID{PhaseSurfaceDiscovery},
		Profiles:                 []model.AuditProfileID{model.AuditProfileRecon, model.AuditProfileWebTriage},
		MinAggressiveness:        AggressivenessBounded,
		RequiresExplicitApproval: true,
		Reason:                   "Crawling touches the target actively and must be approved per engagement.",
	},
	"waymore": {
		Name:              "waymore",
		Phases:            []PhaseID{PhaseSurfaceDiscovery},
		Profiles:          []model.AuditProfileID{model.AuditProfileRecon, model.AuditProfileWebTriage},
		MinAggressiveness: AggressivenessPassive,
		Reason:            "Historical URL discovery is passive enough for baseline recon.",
	},
	"js-beautify": {
		Name:              "js-beautify",
		Phases:            []PhaseID{PhaseJSIntel},
		Profiles:          []model.AuditProfileID{model.AuditProfileWebTriage},
		MinAggressiveness: AggressivenessPassive,
		Reason:            "Beautifying already collected JavaScript is permitted in passive mode.",
	},
	"jsluice": {
		Name:              "jsluice",
		Phases:            []PhaseID{PhaseJSIntel},
		Profiles:          []model.AuditProfileID{model.AuditProfileWebTriage},
		MinAggressiveness: AggressivenessPassive,
		Reason:            "Static JavaScript extraction is permitted when artifacts have already been collected.",
	},
	"chromedp": {
		Name:                     "chromedp",
		Phases:                   []PhaseID{PhaseJSIntel},
		Profiles:                 []model.AuditProfileID{model.AuditProfileWebTriage},
		MinAggressiveness:        AggressivenessBounded,
		RequiresExplicitApproval: true,
		Reason:                   "Browser automation drives the target actively and must be explicitly approved.",
	},
	"mitmproxy": {
		Name:                     "mitmproxy",
		Phases:                   []PhaseID{PhaseJSIntel, PhaseAPIDiscovery},
		Profiles:                 []model.AuditProfileID{model.AuditProfileWebTriage},
		MinAggressiveness:        AggressivenessBounded,
		RequiresExplicitApproval: true,
		Reason:                   "Traffic interception changes the analysis posture and requires explicit approval.",
	},
	"openapi-harvest": {
		Name:              "openapi-harvest",
		Phases:            []PhaseID{PhaseAPIDiscovery},
		Profiles:          []model.AuditProfileID{model.AuditProfileWebTriage},
		MinAggressiveness: AggressivenessPassive,
		Reason:            "Schema harvesting from existing artifacts or public endpoints is allowed in passive mode.",
	},
	"arjun": {
		Name:                     "arjun",
		Phases:                   []PhaseID{PhaseAPIDiscovery},
		Profiles:                 []model.AuditProfileID{model.AuditProfileWebTriage},
		MinAggressiveness:        AggressivenessBounded,
		RequiresExplicitApproval: true,
		Reason:                   "Parameter discovery sends active requests and must be approved explicitly.",
	},
	"ffuf": {
		Name:                     "ffuf",
		Phases:                   []PhaseID{PhaseAPIDiscovery},
		Profiles:                 []model.AuditProfileID{model.AuditProfileWebTriage},
		MinAggressiveness:        AggressivenessActive,
		RequiresExplicitApproval: true,
		Reason:                   "Fuzzing is intrusive enough to require active aggressiveness plus explicit approval.",
	},
	"sqlmap": {
		Name:                     "sqlmap",
		Phases:                   []PhaseID{PhaseAuthorizedValidation},
		Profiles:                 []model.AuditProfileID{model.AuditProfileWebTriage},
		MinAggressiveness:        AggressivenessActive,
		RequiresExplicitApproval: true,
		Reason:                   "Automated SQLi validation is only allowed in authorized validation with explicit approval.",
	},
	"burpsuite": {
		Name:                     "burpsuite",
		Phases:                   []PhaseID{PhaseJSIntel, PhaseAPIDiscovery, PhaseVulnHypothesis},
		Profiles:                 []model.AuditProfileID{model.AuditProfileWebTriage},
		MinAggressiveness:        AggressivenessBounded,
		RequiresExplicitApproval: true,
		Reason:                   "Proxy interception and active manipulation requires bounded analysis mode plus explicit approval.",
	},
	"ysoserial": {
		Name:                     "ysoserial",
		Phases:                   []PhaseID{PhaseVulnHypothesis, PhaseAuthorizedValidation},
		Profiles:                 []model.AuditProfileID{model.AuditProfileWebTriage},
		MinAggressiveness:        AggressivenessActive,
		RequiresExplicitApproval: true,
		Reason:                   "Deserialization exploit generation requires active aggressiveness and explicit approval for validation.",
	},
	"exiftool": {
		Name:                     "exiftool",
		Phases:                   []PhaseID{PhaseVulnHypothesis},
		Profiles:                 []model.AuditProfileID{model.AuditProfileWebTriage},
		MinAggressiveness:        AggressivenessBounded,
		RequiresExplicitApproval: true,
		Reason:                   "Metadata analysis for file upload attacks requires bounded mode and explicit approval.",
	},
	"polyglot-generator": {
		Name:                     "polyglot-generator",
		Phases:                   []PhaseID{PhaseVulnHypothesis},
		Profiles:                 []model.AuditProfileID{model.AuditProfileWebTriage},
		MinAggressiveness:        AggressivenessBounded,
		RequiresExplicitApproval: true,
		Reason:                   "Polyglot file generation for upload testing requires bounded mode and explicit approval.",
	},
	"jwt-tool": {
		Name:                     "jwt-tool",
		Phases:                   []PhaseID{PhaseVulnHypothesis, PhaseAuthorizedValidation},
		Profiles:                 []model.AuditProfileID{model.AuditProfileWebTriage},
		MinAggressiveness:        AggressivenessBounded,
		RequiresExplicitApproval: true,
		Reason:                   "JWT manipulation and cracking requires bounded mode and explicit approval.",
	},
}

func CandidateToolDecisions(manifest RunManifest, phase PhaseID) []ToolDecision {
	out := []ToolDecision{}
	for _, policy := range policiesForPhase(phase) {
		out = append(out, evaluateTool(manifest, phase, policy.Name))
	}
	return out
}

func RequestedToolDecisions(manifest RunManifest, phase PhaseID, requested []string) []ToolDecision {
	normalized := NormalizeToolNames(requested)
	out := make([]ToolDecision, 0, len(normalized))
	for _, name := range normalized {
		out = append(out, evaluateTool(manifest, phase, name))
	}
	return out
}

func evaluateTool(manifest RunManifest, phase PhaseID, raw string) ToolDecision {
	name := strings.TrimSpace(strings.ToLower(raw))
	policy, ok := toolPolicies[name]
	if !ok {
		return ToolDecision{
			Name:                     name,
			Phase:                    phase,
			Allowed:                  false,
			Reason:                   "Tool is not registered in the orchestrator policy catalog.",
			RequiresExplicitApproval: true,
			MinAggressiveness:        AggressivenessActive,
		}
	}

	decision := ToolDecision{
		Name:                     policy.Name,
		Phase:                    phase,
		Allowed:                  false,
		Reason:                   policy.Reason,
		RequiresExplicitApproval: policy.RequiresExplicitApproval,
		MinAggressiveness:        policy.MinAggressiveness,
	}

	if !manifest.Authorized && phase != PhaseScope {
		decision.Reason = "Run is not authorized yet; only scope validation is allowed until authorization is recorded."
		return decision
	}
	if !containsPhase(policy.Phases, phase) {
		decision.Reason = fmt.Sprintf("Tool %s is not allowed in phase %s.", policy.Name, phase)
		return decision
	}
	if !containsProfile(policy.Profiles, manifest.Profile) {
		decision.Reason = fmt.Sprintf("Tool %s is not available for profile %s.", policy.Name, manifest.Profile)
		return decision
	}
	if aggressivenessRank[manifest.Aggressiveness] < aggressivenessRank[policy.MinAggressiveness] {
		decision.Reason = fmt.Sprintf("Tool %s requires aggressiveness %s or higher.", policy.Name, policy.MinAggressiveness)
		return decision
	}
	if policy.RequiresExplicitApproval && !containsString(manifest.ApprovedTools, policy.Name) {
		decision.Reason = fmt.Sprintf("Tool %s requires explicit approval via policy before it can be used.", policy.Name)
		return decision
	}

	decision.Allowed = true
	decision.Reason = policy.Reason
	return decision
}

func policiesForPhase(phase PhaseID) []ToolPolicy {
	out := []ToolPolicy{}
	for _, policy := range toolPolicies {
		if containsPhase(policy.Phases, phase) {
			out = append(out, policy)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func containsProfile(items []model.AuditProfileID, target model.AuditProfileID) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func containsPhase(items []PhaseID, target PhaseID) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}
