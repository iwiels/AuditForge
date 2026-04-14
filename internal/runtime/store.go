package auditruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"orquestador-auditor/internal/memory"
)

type Store struct {
	ArtifactsRoot string
	MemoryRoot    string
}

func NewStore(artifactsRoot, memoryRoot string) Store {
	return Store{
		ArtifactsRoot: artifactsRoot,
		MemoryRoot:    memoryRoot,
	}
}

func (s Store) StartRun(input StartRunInput) (RunManifest, error) {
	target := strings.TrimSpace(input.Target)
	if target == "" {
		return RunManifest{}, fmt.Errorf("target is required")
	}
	if input.Profile.ID == "" {
		return RunManifest{}, fmt.Errorf("profile is required")
	}

	aggressiveness := input.Aggressiveness
	if aggressiveness == "" {
		aggressiveness = DefaultAggressiveness(input.Profile)
	}
	runID := buildRunID(target)
	runDir := s.runDir(runID)
	phaseDir := filepath.Join(runDir, "phases")
	if err := os.MkdirAll(phaseDir, 0o755); err != nil {
		return RunManifest{}, err
	}

	now := time.Now().UTC()
	manifest := RunManifest{
		RunID:            runID,
		Target:           target,
		TargetKind:       strings.TrimSpace(input.TargetKind),
		Campaign:         strings.TrimSpace(input.Campaign),
		Authorized:       input.Authorized,
		AuthorizationRef: strings.TrimSpace(input.AuthorizationRef),
		Profile:          input.Profile.ID,
		ProfileMode:      input.Profile.Risk.Mode,
		PolicySummary:    input.Profile.Risk.Summary,
		Aggressiveness:   aggressiveness,
		ApprovedTools:    NormalizeToolNames(input.ApprovedTools),
		PhasePlan:        PhasePlanForProfile(input.Profile.ID),
		ArtifactsDir:     runDir,
		ManifestPath:     s.manifestPath(runID),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	manifest.Phases = make([]PhaseState, 0, len(manifest.PhasePlan))
	for _, phase := range manifest.PhasePlan {
		artifact := PhaseArtifact{
			RunID:               runID,
			Profile:             manifest.Profile,
			Target:              manifest.Target,
			TargetKind:          manifest.TargetKind,
			Phase:               phase,
			Status:              PhaseStatusPending,
			Summary:             "Pending phase artifact. Candidate tools are policy-evaluated and never auto-executed by the orchestrator.",
			DependsOn:           PhaseDependencies(manifest.PhasePlan, phase),
			CandidateToolPolicy: CandidateToolDecisions(manifest, phase),
			GeneratedAt:         now,
			ArtifactPath:        s.phasePath(runID, phase),
		}
		if phase == PhaseScope {
			artifact.Status = PhaseStatusObserved
			artifact.Summary = fmt.Sprintf("Authorization=%t. Target kind=%s. Aggressiveness=%s. Approved tools=%s.", manifest.Authorized, safeValue(manifest.TargetKind, "unknown"), manifest.Aggressiveness, strings.Join(manifest.ApprovedTools, ", "))
			artifact.Notes = []string{
				"Phase 0 defines scope, authorization, target kind, and aggressiveness before any active activity.",
				"Tool policy is explicit: candidate tools may be approved, but the orchestrator does not auto-execute them.",
			}
			if !manifest.Authorized {
				artifact.Status = PhaseStatusBlockedByPolicy
				artifact.Notes = append(artifact.Notes, "Run started without authorization; every non-scope phase remains blocked until authorization is recorded.")
			}
			manifest.CurrentPhase = PhaseScope
		}
		if err := writeJSON(artifact.ArtifactPath, artifact); err != nil {
			return RunManifest{}, err
		}
		manifest.Phases = append(manifest.Phases, PhaseState{
			Phase:        phase,
			Status:       artifact.Status,
			ArtifactPath: artifact.ArtifactPath,
			UpdatedAt:    now,
		})
	}
	if err := writeJSON(manifest.ManifestPath, manifest); err != nil {
		return RunManifest{}, err
	}
	_ = s.saveObservation(memory.Observation{
		ID:        runID + "-start",
		Kind:      "audit-run",
		Title:     fmt.Sprintf("Started %s run for %s", input.Profile.ID, target),
		Body:      fmt.Sprintf("Authorized=%t, target_kind=%s, aggressiveness=%s, approved_tools=%s", manifest.Authorized, safeValue(manifest.TargetKind, "unknown"), manifest.Aggressiveness, strings.Join(manifest.ApprovedTools, ", ")),
		Target:    target,
		Campaign:  manifest.Campaign,
		RunID:     runID,
		CreatedAt: now,
		Tags:      []string{"runtime", "audit-run", string(input.Profile.ID), string(manifest.Aggressiveness)},
		Metadata:  map[string]string{"phase": string(PhaseScope), "manifest_path": manifest.ManifestPath},
	})
	return manifest, nil
}

func (s Store) RecordPhase(input RecordPhaseInput) (PhaseArtifact, error) {
	manifest, err := s.LoadRun(input.RunID)
	if err != nil {
		return PhaseArtifact{}, err
	}
	if !containsPhase(manifest.PhasePlan, input.Phase) {
		return PhaseArtifact{}, fmt.Errorf("phase %s is not enabled for profile %s", input.Phase, manifest.Profile)
	}

	artifact, err := s.loadPhase(input.RunID, input.Phase)
	if err != nil {
		return PhaseArtifact{}, err
	}

	candidate := CandidateToolDecisions(manifest, input.Phase)
	requested := RequestedToolDecisions(manifest, input.Phase, input.RequestedTools)
	status := input.Status
	if status == "" {
		status = PhaseStatusObserved
	}

	notes := append([]string(nil), input.Notes...)
	if !manifest.Authorized && input.Phase != PhaseScope {
		status = PhaseStatusBlockedByPolicy
		notes = append(notes, "Blocked: the run has no recorded authorization yet.")
	}
	if missing := s.pendingDependencies(manifest, input.Phase); len(missing) > 0 {
		notes = append(notes, fmt.Sprintf("Methodology warning: missing prior phase artifacts: %s", strings.Join(phasesToStrings(missing), ", ")))
	}
	if input.Phase == PhaseAuthorizedValidation && !s.hasPriorHypothesis(manifest) {
		status = PhaseStatusBlockedByPolicy
		notes = append(notes, "Blocked: authorized validation requires a prior vuln-hypothesis artifact with findings.")
	}
	for _, decision := range requested {
		if !decision.Allowed {
			status = PhaseStatusBlockedByPolicy
			notes = append(notes, decision.Reason)
		}
	}

	artifact.Status = status
	if strings.TrimSpace(input.Summary) != "" {
		artifact.Summary = strings.TrimSpace(input.Summary)
	}
	artifact.RequestedTools = NormalizeToolNames(input.RequestedTools)
	artifact.RequestedToolPolicy = requested
	artifact.CandidateToolPolicy = candidate
	artifact.Notes = dedupeStrings(notes)
	artifact.Findings = normalizeFindings(input.Phase, status, input.Findings)
	artifact.GeneratedAt = time.Now().UTC()
	if artifact.ArtifactPath == "" {
		artifact.ArtifactPath = s.phasePath(input.RunID, input.Phase)
	}

	if err := writeJSON(artifact.ArtifactPath, artifact); err != nil {
		return PhaseArtifact{}, err
	}
	manifest.UpdatedAt = artifact.GeneratedAt
	manifest.CurrentPhase = input.Phase
	manifest = updateManifestPhaseState(manifest, artifact)
	if err := writeJSON(manifest.ManifestPath, manifest); err != nil {
		return PhaseArtifact{}, err
	}
	_ = s.saveObservation(memory.Observation{
		ID:        fmt.Sprintf("%s-%s", input.RunID, input.Phase),
		Kind:      "phase-artifact",
		Title:     fmt.Sprintf("Recorded %s for %s", input.Phase, manifest.Target),
		Body:      safeValue(artifact.Summary, "No summary provided"),
		Target:    manifest.Target,
		Campaign:  manifest.Campaign,
		RunID:     manifest.RunID,
		CreatedAt: artifact.GeneratedAt,
		Tags:      []string{"runtime", "phase-artifact", string(input.Phase), string(artifact.Status), string(manifest.Profile)},
		Metadata:  map[string]string{"artifact_path": artifact.ArtifactPath},
	})
	return artifact, nil
}

func (s Store) CorrelateRun(runID string) (PhaseArtifact, error) {
	manifest, artifacts, err := s.InspectRun(runID)
	if err != nil {
		return PhaseArtifact{}, err
	}
	if !containsPhase(manifest.PhasePlan, PhaseCorrelation) {
		return PhaseArtifact{}, fmt.Errorf("correlation phase is not enabled for profile %s", manifest.Profile)
	}

	allFindings := []Finding{}
	missingPhases := []PhaseID{}
	for _, phase := range manifest.PhasePlan {
		if phase == PhaseCorrelation {
			continue
		}
		artifact := findArtifact(artifacts, phase)
		if artifact == nil || artifact.Status == PhaseStatusPending {
			missingPhases = append(missingPhases, phase)
			continue
		}
		allFindings = append(allFindings, artifact.Findings...)
	}

	correlated := dedupeAndEnrichFindings(allFindings)
	summary := buildCorrelationSummary(correlated, missingPhases)
	artifact := PhaseArtifact{
		RunID:               manifest.RunID,
		Profile:             manifest.Profile,
		Target:              manifest.Target,
		TargetKind:          manifest.TargetKind,
		Phase:               PhaseCorrelation,
		Status:              PhaseStatusCorrelated,
		Summary:             fmt.Sprintf("Correlated %d findings across %d phases.", summary.CorrelatedFindings, len(manifest.PhasePlan)-1),
		DependsOn:           PhaseDependencies(manifest.PhasePlan, PhaseCorrelation),
		CandidateToolPolicy: CandidateToolDecisions(manifest, PhaseCorrelation),
		Findings:            correlated,
		Correlation:         &summary,
		Notes:               summary.Recommendations,
		GeneratedAt:         time.Now().UTC(),
		ArtifactPath:        s.phasePath(runID, PhaseCorrelation),
	}

	if err := writeJSON(artifact.ArtifactPath, artifact); err != nil {
		return PhaseArtifact{}, err
	}
	manifest.UpdatedAt = artifact.GeneratedAt
	manifest.CurrentPhase = PhaseCorrelation
	manifest = updateManifestPhaseState(manifest, artifact)
	if err := writeJSON(manifest.ManifestPath, manifest); err != nil {
		return PhaseArtifact{}, err
	}
	_ = s.saveObservation(memory.Observation{
		ID:        runID + "-correlation",
		Kind:      "correlation-artifact",
		Title:     fmt.Sprintf("Correlated findings for %s", manifest.Target),
		Body:      artifact.Summary,
		Target:    manifest.Target,
		Campaign:  manifest.Campaign,
		RunID:     manifest.RunID,
		CreatedAt: artifact.GeneratedAt,
		Tags:      []string{"runtime", "correlation", string(manifest.Profile)},
		Metadata:  map[string]string{"artifact_path": artifact.ArtifactPath},
	})
	return artifact, nil
}

func (s Store) ValidateRun(runID string) (PhaseArtifact, error) {
	manifest, artifacts, err := s.InspectRun(runID)
	if err != nil {
		return PhaseArtifact{}, err
	}
	if !containsPhase(manifest.PhasePlan, PhaseJudgment) {
		return PhaseArtifact{}, fmt.Errorf("judgment phase is not enabled for profile %s", manifest.Profile)
	}

	// Check that correlation has been completed
	correlationArtifact := findArtifact(artifacts, PhaseCorrelation)
	if correlationArtifact == nil || correlationArtifact.Status == PhaseStatusPending {
		return PhaseArtifact{}, fmt.Errorf("judgment requires correlation to be completed first")
	}

	// Run validation
	judgment := ValidateRun(manifest, artifacts)

	artifact := PhaseArtifact{
		RunID:        manifest.RunID,
		Profile:      manifest.Profile,
		Target:       manifest.Target,
		TargetKind:   manifest.TargetKind,
		Phase:        PhaseJudgment,
		Status:       PhaseStatusJudged,
		Summary:      fmt.Sprintf("Validated %d checks: %d passed, %d failed. Quality score: %.0f%%. Ready to report: %t.", judgment.TotalChecks, judgment.Passed, judgment.Failed, judgment.QualityScore, judgment.ReadyToReport),
		DependsOn:    PhaseDependencies(manifest.PhasePlan, PhaseJudgment),
		Judgment:     &judgment,
		Notes:        judgment.Recommendations,
		GeneratedAt:  time.Now().UTC(),
		ArtifactPath: s.phasePath(runID, PhaseJudgment),
	}

	if err := writeJSON(artifact.ArtifactPath, artifact); err != nil {
		return PhaseArtifact{}, err
	}
	manifest.UpdatedAt = artifact.GeneratedAt
	manifest.CurrentPhase = PhaseJudgment
	manifest = updateManifestPhaseState(manifest, artifact)
	if err := writeJSON(manifest.ManifestPath, manifest); err != nil {
		return PhaseArtifact{}, err
	}
	_ = s.saveObservation(memory.Observation{
		ID:        runID + "-judgment",
		Kind:      "judgment-artifact",
		Title:     fmt.Sprintf("Judgment for %s: score=%.0f%%", manifest.Target, judgment.QualityScore),
		Body:      artifact.Summary,
		Target:    manifest.Target,
		Campaign:  manifest.Campaign,
		RunID:     manifest.RunID,
		CreatedAt: artifact.GeneratedAt,
		Tags:      []string{"runtime", "judgment", string(manifest.Profile)},
		Metadata:  map[string]string{"artifact_path": artifact.ArtifactPath, "quality_score": fmt.Sprintf("%.0f", judgment.QualityScore), "ready_to_report": fmt.Sprintf("%t", judgment.ReadyToReport)},
	})
	return artifact, nil
}

func (s Store) LoadRun(runID string) (RunManifest, error) {
	raw, err := os.ReadFile(s.manifestPath(runID))
	if err != nil {
		return RunManifest{}, err
	}
	var manifest RunManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return RunManifest{}, err
	}
	return manifest, nil
}

func (s Store) InspectRun(runID string) (RunManifest, []PhaseArtifact, error) {
	manifest, err := s.LoadRun(runID)
	if err != nil {
		return RunManifest{}, nil, err
	}
	artifacts := make([]PhaseArtifact, 0, len(manifest.PhasePlan))
	for _, phase := range manifest.PhasePlan {
		artifact, err := s.loadPhase(runID, phase)
		if err != nil {
			return RunManifest{}, nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	sort.SliceStable(artifacts, func(i, j int) bool {
		return phaseIndex(artifacts[i].Phase, manifest.PhasePlan) < phaseIndex(artifacts[j].Phase, manifest.PhasePlan)
	})
	return manifest, artifacts, nil
}

func (s Store) pendingDependencies(manifest RunManifest, phase PhaseID) []PhaseID {
	required := PhaseDependencies(manifest.PhasePlan, phase)
	if len(required) == 0 {
		return nil
	}
	missing := []PhaseID{}
	for _, dependency := range required {
		artifact, err := s.loadPhase(manifest.RunID, dependency)
		if err != nil || artifact.Status == PhaseStatusPending {
			missing = append(missing, dependency)
		}
	}
	return missing
}

func (s Store) hasPriorHypothesis(manifest RunManifest) bool {
	if !containsPhase(manifest.PhasePlan, PhaseVulnHypothesis) {
		return false
	}
	artifact, err := s.loadPhase(manifest.RunID, PhaseVulnHypothesis)
	if err != nil {
		return false
	}
	return artifact.Status != PhaseStatusPending && len(artifact.Findings) > 0
}

func (s Store) loadPhase(runID string, phase PhaseID) (PhaseArtifact, error) {
	raw, err := os.ReadFile(s.phasePath(runID, phase))
	if err != nil {
		return PhaseArtifact{}, err
	}
	var artifact PhaseArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return PhaseArtifact{}, err
	}
	return artifact, nil
}

func (s Store) saveObservation(obs memory.Observation) error {
	if strings.TrimSpace(s.MemoryRoot) == "" {
		return nil
	}
	return memory.New(s.MemoryRoot).Save(obs)
}

func (s Store) runDir(runID string) string {
	return filepath.Join(s.ArtifactsRoot, runID)
}

func (s Store) manifestPath(runID string) string {
	return filepath.Join(s.runDir(runID), "run.json")
}

func (s Store) phasePath(runID string, phase PhaseID) string {
	return filepath.Join(s.runDir(runID), "phases", string(phase)+".json")
}

func writeJSON(path string, value interface{}) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func buildRunID(target string) string {
	return time.Now().UTC().Format("20060102-150405") + "-" + slug(target)
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		":", "-",
		"/", "-",
		"\\", "-",
		".", "-",
		"@", "-",
		" ", "-",
		"_", "-",
	)
	value = replacer.Replace(value)
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	if len(parts) == 0 {
		return "target"
	}
	return strings.Join(parts, "-")
}

func updateManifestPhaseState(manifest RunManifest, artifact PhaseArtifact) RunManifest {
	updated := false
	for i, item := range manifest.Phases {
		if item.Phase == artifact.Phase {
			manifest.Phases[i].Status = artifact.Status
			manifest.Phases[i].ArtifactPath = artifact.ArtifactPath
			manifest.Phases[i].UpdatedAt = artifact.GeneratedAt
			updated = true
			break
		}
	}
	if !updated {
		manifest.Phases = append(manifest.Phases, PhaseState{
			Phase:        artifact.Phase,
			Status:       artifact.Status,
			ArtifactPath: artifact.ArtifactPath,
			UpdatedAt:    artifact.GeneratedAt,
		})
	}
	return manifest
}

func normalizeFindings(phase PhaseID, status PhaseStatus, findings []Finding) []Finding {
	if len(findings) == 0 {
		return nil
	}
	out := make([]Finding, 0, len(findings))
	for i, finding := range findings {
		item := finding
		item.Phase = phase
		if strings.TrimSpace(item.ID) == "" {
			item.ID = fmt.Sprintf("%s-%d", phase, i+1)
		}
		item.Category = normalizeCategory(item.Category)
		if strings.TrimSpace(item.Title) == "" {
			item.Title = strings.ReplaceAll(item.Category, "-", " ")
		}
		if item.State == "" {
			item.State = defaultFindingState(status)
		}
		item.Evidence = dedupeStrings(item.Evidence)
		item.Sources = dedupeStrings(item.Sources)
		out = append(out, item)
	}
	return out
}

func defaultFindingState(status PhaseStatus) FindingState {
	switch status {
	case PhaseStatusValidated, PhaseStatusCorrelated:
		return FindingStateValidated
	case PhaseStatusSuspected:
		return FindingStateSuspected
	case PhaseStatusBlockedByPolicy:
		return FindingStateBlockedByPolicy
	default:
		return FindingStateObserved
	}
}

func dedupeAndEnrichFindings(findings []Finding) []Finding {
	type key string
	merged := map[key]Finding{}
	order := []key{}
	for _, raw := range findings {
		item := enrichFinding(raw)
		k := key(strings.ToLower(strings.TrimSpace(item.Category + "|" + item.Title + "|" + string(item.State))))
		if existing, ok := merged[k]; ok {
			existing.Evidence = dedupeStrings(append(existing.Evidence, item.Evidence...))
			existing.Sources = dedupeStrings(append(existing.Sources, item.Sources...))
			existing.Severity = maxSeverity(existing.Severity, item.Severity)
			if item.Confidence > existing.Confidence {
				existing.Confidence = item.Confidence
			}
			if existing.CWE == "" {
				existing.CWE = item.CWE
			}
			if existing.OWASP == "" {
				existing.OWASP = item.OWASP
			}
			if existing.Remediation == "" {
				existing.Remediation = item.Remediation
			}
			merged[k] = existing
			continue
		}
		order = append(order, k)
		merged[k] = item
	}
	out := make([]Finding, 0, len(order))
	for _, k := range order {
		out = append(out, merged[k])
	}
	return out
}

func buildCorrelationSummary(findings []Finding, missing []PhaseID) CorrelationSummary {
	summary := CorrelationSummary{
		TotalFindings:      len(findings),
		CorrelatedFindings: len(findings),
		ByState:            map[string]int{},
		ByCategory:         map[string]int{},
		BySeverity:         map[string]int{},
		MissingPhases:      missing,
	}
	recommendations := []string{}
	highest := ""
	for _, finding := range findings {
		summary.ByState[string(finding.State)]++
		summary.ByCategory[finding.Category]++
		if finding.Severity != "" {
			summary.BySeverity[strings.ToLower(finding.Severity)]++
			highest = maxSeverity(highest, finding.Severity)
		}
		if finding.Remediation != "" {
			recommendations = append(recommendations, finding.Remediation)
		}
	}
	summary.HighestSeverity = strings.ToLower(highest)
	summary.Recommendations = dedupeStrings(recommendations)
	if len(summary.Recommendations) == 0 && len(missing) > 0 {
		summary.Recommendations = []string{"Completar las fases metodologicas pendientes antes de cerrar el reporte final."}
	}
	return summary
}

type taxonomy struct {
	CWE         string
	OWASP       string
	Severity    string
	Remediation string
}

var categoryTaxonomy = map[string]taxonomy{
	"authz":             {CWE: "CWE-285", OWASP: "A01:2021-Broken Access Control", Severity: "high", Remediation: "Revisar controles de autorizacion por objeto y por accion; negar por defecto y registrar decisiones de acceso."},
	"idor":              {CWE: "CWE-639", OWASP: "A01:2021-Broken Access Control", Severity: "high", Remediation: "Aplicar verificaciones de ownership en cada objeto accesible y testear referencias directas a recursos."},
	"sqli":              {CWE: "CWE-89", OWASP: "A03:2021-Injection", Severity: "critical", Remediation: "Usar queries parametrizadas, normalizar tipos de entrada y agregar validaciones server-side."},
	"xss":               {CWE: "CWE-79", OWASP: "A03:2021-Injection", Severity: "medium", Remediation: "Escapar output por contexto, endurecer CSP y validar contenido reflejado o almacenado."},
	"ssrf":              {CWE: "CWE-918", OWASP: "A10:2021-Server-Side Request Forgery", Severity: "high", Remediation: "Restringir destinos salientes, validar URLs contra allowlists y aislar servicios internos."},
	"command-injection": {CWE: "CWE-78", OWASP: "A03:2021-Injection", Severity: "critical", Remediation: "Eliminar shell interpolation, usar APIs nativas y sanitizar argumentos de manera estructurada."},
	"secrets":           {CWE: "CWE-798", OWASP: "A05:2021-Security Misconfiguration", Severity: "high", Remediation: "Rotar secretos expuestos, moverlos a un vault y eliminar valores hardcodeados del codigo y CI/CD."},
	"config-leak":       {CWE: "CWE-200", OWASP: "A05:2021-Security Misconfiguration", Severity: "medium", Remediation: "Reducir informacion expuesta en respuestas, bundles y archivos publicos; segmentar entornos."},
	"surface-exposure":  {CWE: "CWE-200", OWASP: "A05:2021-Security Misconfiguration", Severity: "low", Remediation: "Cerrar servicios innecesarios, revisar headers y reducir superficie accesible externamente."},
	"input-validation":  {CWE: "CWE-20", OWASP: "A03:2021-Injection", Severity: "medium", Remediation: "Validar formato, longitud y tipo de input del lado servidor antes de usarlo."},
}

func enrichFinding(finding Finding) Finding {
	item := finding
	item.Category = normalizeCategory(item.Category)
	tax, ok := categoryTaxonomy[item.Category]
	if ok {
		if item.CWE == "" {
			item.CWE = tax.CWE
		}
		if item.OWASP == "" {
			item.OWASP = tax.OWASP
		}
		if item.Severity == "" {
			item.Severity = tax.Severity
		}
		if item.Remediation == "" {
			item.Remediation = tax.Remediation
		}
	}
	item.Severity = strings.ToLower(strings.TrimSpace(item.Severity))
	return item
}

func maxSeverity(a, b string) string {
	order := map[string]int{"": 0, "info": 1, "low": 2, "medium": 3, "high": 4, "critical": 5}
	if order[strings.ToLower(b)] > order[strings.ToLower(a)] {
		return strings.ToLower(b)
	}
	return strings.ToLower(a)
}

func normalizeCategory(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "observation"
	}
	replacer := strings.NewReplacer("_", "-", " ", "-", "/", "-")
	value = replacer.Replace(value)
	switch value {
	case "sqli", "sql-injection":
		return "sqli"
	case "auth", "authz", "authorization":
		return "authz"
	case "idor":
		return "idor"
	case "xss":
		return "xss"
	case "ssrf":
		return "ssrf"
	case "command-injection", "cmdi", "commandinjection":
		return "command-injection"
	case "secret", "secrets", "secret-leak":
		return "secrets"
	case "config", "config-leak", "configleak":
		return "config-leak"
	default:
		return value
	}
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func safeValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func phasesToStrings(phases []PhaseID) []string {
	out := make([]string, 0, len(phases))
	for _, phase := range phases {
		out = append(out, string(phase))
	}
	return out
}

func phaseIndex(phase PhaseID, plan []PhaseID) int {
	for i, item := range plan {
		if item == phase {
			return i
		}
	}
	return len(plan) + 1
}

func findArtifact(artifacts []PhaseArtifact, phase PhaseID) *PhaseArtifact {
	for i := range artifacts {
		if artifacts[i].Phase == phase {
			return &artifacts[i]
		}
	}
	return nil
}
