package auditruntime

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"orquestador-auditor/internal/model"
)

type Aggressiveness string

const (
	AggressivenessPassive Aggressiveness = "passive"
	AggressivenessBounded Aggressiveness = "bounded"
	AggressivenessActive  Aggressiveness = "active"
)

type PhaseID string

const (
	PhaseScope                PhaseID = "scope"
	PhaseNetworkRecon         PhaseID = "network-recon"
	PhaseSurfaceDiscovery     PhaseID = "surface-discovery"
	PhaseJSIntel              PhaseID = "js-intel"
	PhaseAPIDiscovery         PhaseID = "api-discovery"
	PhaseVulnHypothesis       PhaseID = "vuln-hypothesis"
	PhaseAuthorizedValidation PhaseID = "authorized-validation"
	PhaseCorrelation          PhaseID = "correlation"
	PhaseJudgment             PhaseID = "judgment"
)

type PhaseStatus string

const (
	PhaseStatusPending         PhaseStatus = "pending"
	PhaseStatusObserved        PhaseStatus = "observed"
	PhaseStatusSuspected       PhaseStatus = "suspected"
	PhaseStatusValidated       PhaseStatus = "validated"
	PhaseStatusBlockedByPolicy PhaseStatus = "blocked-by-policy"
	PhaseStatusCorrelated      PhaseStatus = "correlated"
	PhaseStatusJudged          PhaseStatus = "judged"
)

type FindingState string

const (
	FindingStateObserved        FindingState = "observed"
	FindingStateSuspected       FindingState = "suspected"
	FindingStateValidated       FindingState = "validated"
	FindingStateBlockedByPolicy FindingState = "blocked-by-policy"
)

type Finding struct {
	ID          string            `json:"id"`
	Phase       PhaseID           `json:"phase"`
	Category    string            `json:"category"`
	Title       string            `json:"title"`
	State       FindingState      `json:"state"`
	Severity    string            `json:"severity,omitempty"`
	Confidence  float64           `json:"confidence,omitempty"`
	Evidence    []string          `json:"evidence,omitempty"`
	Sources     []string          `json:"sources,omitempty"`
	CWE         string            `json:"cwe,omitempty"`
	OWASP       string            `json:"owasp,omitempty"`
	Remediation string            `json:"remediation,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type ToolDecision struct {
	Name                     string         `json:"name"`
	Phase                    PhaseID        `json:"phase"`
	Allowed                  bool           `json:"allowed"`
	Reason                   string         `json:"reason"`
	RequiresExplicitApproval bool           `json:"requires_explicit_approval"`
	MinAggressiveness        Aggressiveness `json:"min_aggressiveness"`
}

type CorrelationSummary struct {
	TotalFindings      int            `json:"total_findings"`
	HighestSeverity    string         `json:"highest_severity,omitempty"`
	ByState            map[string]int `json:"by_state,omitempty"`
	ByCategory         map[string]int `json:"by_category,omitempty"`
	BySeverity         map[string]int `json:"by_severity,omitempty"`
	Recommendations    []string       `json:"recommendations,omitempty"`
	MissingPhases      []PhaseID      `json:"missing_phases,omitempty"`
	CorrelatedFindings int            `json:"correlated_findings"`
}

type PhaseArtifact struct {
	RunID               string               `json:"run_id"`
	Profile             model.AuditProfileID `json:"profile"`
	Target              string               `json:"target"`
	TargetKind          string               `json:"target_kind,omitempty"`
	Phase               PhaseID              `json:"phase"`
	Status              PhaseStatus          `json:"status"`
	Summary             string               `json:"summary,omitempty"`
	DependsOn           []PhaseID            `json:"depends_on,omitempty"`
	RequestedTools      []string             `json:"requested_tools,omitempty"`
	RequestedToolPolicy []ToolDecision       `json:"requested_tool_policy,omitempty"`
	CandidateToolPolicy []ToolDecision       `json:"candidate_tool_policy,omitempty"`
	Notes               []string             `json:"notes,omitempty"`
	Findings            []Finding            `json:"findings,omitempty"`
	Correlation         *CorrelationSummary  `json:"correlation,omitempty"`
	Judgment            *JudgmentResult      `json:"judgment,omitempty"`
	GeneratedAt         time.Time            `json:"generated_at"`
	ArtifactPath        string               `json:"artifact_path,omitempty"`
}

type PhaseState struct {
	Phase        PhaseID     `json:"phase"`
	Status       PhaseStatus `json:"status"`
	ArtifactPath string      `json:"artifact_path"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type RunManifest struct {
	RunID            string               `json:"run_id"`
	Target           string               `json:"target"`
	TargetKind       string               `json:"target_kind,omitempty"`
	Campaign         string               `json:"campaign,omitempty"`
	Authorized       bool                 `json:"authorized"`
	AuthorizationRef string               `json:"authorization_ref,omitempty"`
	Profile          model.AuditProfileID `json:"profile"`
	ProfileMode      string               `json:"profile_mode,omitempty"`
	PolicySummary    string               `json:"policy_summary,omitempty"`
	Aggressiveness   Aggressiveness       `json:"aggressiveness"`
	ApprovedTools    []string             `json:"approved_tools,omitempty"`
	PhasePlan        []PhaseID            `json:"phase_plan"`
	CurrentPhase     PhaseID              `json:"current_phase,omitempty"`
	Phases           []PhaseState         `json:"phases"`
	ArtifactsDir     string               `json:"artifacts_dir"`
	ManifestPath     string               `json:"manifest_path"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

type StartRunInput struct {
	Target           string
	TargetKind       string
	Campaign         string
	Authorized       bool
	AuthorizationRef string
	Profile          model.AuditProfile
	Aggressiveness   Aggressiveness
	ApprovedTools    []string
}

type RecordPhaseInput struct {
	RunID          string
	Phase          PhaseID
	Status         PhaseStatus
	Summary        string
	RequestedTools []string
	Findings       []Finding
	Notes          []string
}

func NormalizeAggressiveness(raw string) (Aggressiveness, error) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", string(AggressivenessPassive):
		return AggressivenessPassive, nil
	case string(AggressivenessBounded):
		return AggressivenessBounded, nil
	case string(AggressivenessActive):
		return AggressivenessActive, nil
	default:
		return "", fmt.Errorf("unsupported aggressiveness %q", raw)
	}
}

func NormalizePhaseID(raw string) (PhaseID, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	for _, phase := range AllPhases() {
		if string(phase) == value {
			return phase, nil
		}
	}
	return "", fmt.Errorf("unsupported phase %q", raw)
}

func NormalizePhaseStatus(raw string) (PhaseStatus, error) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", string(PhaseStatusObserved):
		return PhaseStatusObserved, nil
	case string(PhaseStatusPending):
		return PhaseStatusPending, nil
	case string(PhaseStatusSuspected):
		return PhaseStatusSuspected, nil
	case string(PhaseStatusValidated):
		return PhaseStatusValidated, nil
	case string(PhaseStatusBlockedByPolicy):
		return PhaseStatusBlockedByPolicy, nil
	case string(PhaseStatusCorrelated):
		return PhaseStatusCorrelated, nil
	case string(PhaseStatusJudged):
		return PhaseStatusJudged, nil
	default:
		return "", fmt.Errorf("unsupported phase status %q", raw)
	}
}

func AllPhases() []PhaseID {
	return []PhaseID{
		PhaseScope,
		PhaseNetworkRecon,
		PhaseSurfaceDiscovery,
		PhaseJSIntel,
		PhaseAPIDiscovery,
		PhaseVulnHypothesis,
		PhaseAuthorizedValidation,
		PhaseCorrelation,
		PhaseJudgment,
	}
}

func PhasePlanForProfile(id model.AuditProfileID) []PhaseID {
	switch id {
	case model.AuditProfileRecon:
		return []PhaseID{PhaseScope, PhaseNetworkRecon, PhaseSurfaceDiscovery, PhaseCorrelation, PhaseJudgment}
	case model.AuditProfileWebTriage, model.AuditProfileRedTeam:
		return AllPhases()
	case model.AuditProfileSupplyChain:
		return []PhaseID{PhaseScope, PhaseVulnHypothesis, PhaseCorrelation, PhaseJudgment}
	case model.AuditProfileReporting:
		return []PhaseID{PhaseScope, PhaseCorrelation, PhaseJudgment}
	case model.AuditProfileMemoryOnly:
		return []PhaseID{PhaseScope, PhaseCorrelation, PhaseJudgment}
	default:
		return AllPhases()
	}
}

func DefaultAggressiveness(profile model.AuditProfile) Aggressiveness {
	switch profile.ID {
	case model.AuditProfileRecon, model.AuditProfileReporting, model.AuditProfileMemoryOnly:
		return AggressivenessPassive
	case model.AuditProfileWebTriage, model.AuditProfileSupplyChain:
		return AggressivenessBounded
	case model.AuditProfileRedTeam:
		return AggressivenessActive
	default:
		return AggressivenessPassive
	}
}

func PhaseDependencies(plan []PhaseID, phase PhaseID) []PhaseID {
	index := -1
	for i, item := range plan {
		if item == phase {
			index = i
			break
		}
	}
	if index <= 0 {
		return nil
	}
	out := make([]PhaseID, index)
	copy(out, plan[:index])
	return out
}

func NormalizeToolNames(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(strings.ToLower(raw))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
