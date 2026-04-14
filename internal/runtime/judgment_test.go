package auditruntime

import (
	"testing"
	"time"

	"orquestador-auditor/internal/catalog"
	"orquestador-auditor/internal/memory"
)

func TestValidateRunWithEmptyFindings(t *testing.T) {
	manifest := RunManifest{
		RunID:      "test-run",
		Target:     "example.com",
		Profile:    "web-triage",
		PhasePlan:  AllPhases(),
		Authorized: true,
	}
	result := ValidateRun(manifest, nil)
	if !result.ReadyToReport {
		t.Fatalf("expected ready with no findings, score=%.0f", result.QualityScore)
	}
}

func TestValidateRunWithFindingMissingEvidence(t *testing.T) {
	manifest := RunManifest{
		RunID:     "test-run",
		Target:    "example.com",
		Profile:   "web-triage",
		PhasePlan: []PhaseID{PhaseVulnHypothesis, PhaseCorrelation},
	}
	artifacts := []PhaseArtifact{
		{
			Phase:  PhaseVulnHypothesis,
			Status: PhaseStatusObserved,
			Findings: []Finding{
				{ID: "f1", Category: "idor", Title: "IDOR in /api/users/{id}", State: FindingStateObserved, Severity: "high", Evidence: nil},
			},
		},
		{Phase: PhaseCorrelation, Status: PhaseStatusCorrelated},
	}
	result := ValidateRun(manifest, artifacts)
	if result.Failed == 0 {
		t.Fatalf("expected at least 1 failed check for missing evidence")
	}
}

func TestValidateRunWithValidFinding(t *testing.T) {
	manifest := RunManifest{
		RunID:     "test-run",
		Target:    "example.com",
		Profile:   "web-triage",
		PhasePlan: []PhaseID{PhaseVulnHypothesis, PhaseCorrelation},
	}
	artifacts := []PhaseArtifact{
		{
			Phase:  PhaseVulnHypothesis,
			Status: PhaseStatusValidated,
			Findings: []Finding{
				{
					ID: "f1", Category: "idor",
					Title: "IDOR in /api/users/{id} allows accessing other users data",
					State: FindingStateValidated, Severity: "high",
					Evidence: []string{"GET /api/users/100 returned data for user 101", "Confirmed via curl"},
					CWE:      "CWE-639", OWASP: "A01:2021-Broken Access Control",
					Remediation: "Implement ownership check in /api/users/:id handler, verify request.user.id matches resource owner",
				},
			},
		},
		{Phase: PhaseCorrelation, Status: PhaseStatusCorrelated},
	}
	result := ValidateRun(manifest, artifacts)
	if result.Failed > 0 {
		t.Fatalf("expected 0 failed checks, got %d: %v", result.Failed, result.FailedChecks)
	}
}

func TestValidateRunDetectsDuplicateFindings(t *testing.T) {
	manifest := RunManifest{
		RunID:     "test-run",
		Target:    "example.com",
		Profile:   "web-triage",
		PhasePlan: []PhaseID{PhaseVulnHypothesis, PhaseCorrelation},
	}
	artifacts := []PhaseArtifact{
		{
			Phase:  PhaseVulnHypothesis,
			Status: PhaseStatusObserved,
			Findings: []Finding{
				{ID: "f1", Category: "xss", Title: "Reflected XSS on /search", State: FindingStateObserved},
				{ID: "f2", Category: "xss", Title: "Reflected XSS on /search", State: FindingStateObserved},
			},
		},
		{Phase: PhaseCorrelation, Status: PhaseStatusCorrelated},
	}
	result := ValidateRun(manifest, artifacts)
	hasDupFail := false
	for _, c := range result.FailedChecks {
		if c.Check == "no_duplicates" {
			hasDupFail = true
			break
		}
	}
	if !hasDupFail {
		t.Fatalf("expected duplicate detection to fail, got: %v", result.FailedChecks)
	}
}

func TestJudgmentPhaseInProfilePlan(t *testing.T) {
	profile, err := catalog.AuditProfileByID("web-triage")
	if err != nil {
		t.Fatalf("AuditProfileByID error = %v", err)
	}
	plan := PhasePlanForProfile(profile.ID)
	hasJudgment := false
	for _, p := range plan {
		if p == PhaseJudgment {
			hasJudgment = true
			break
		}
	}
	if !hasJudgment {
		t.Fatalf("web-triage profile should include judgment phase")
	}
}

func TestEngramProtocolBuildsContext(t *testing.T) {
	dir := t.TempDir()
	mem := memory.New(dir)
	now := time.Now().UTC()
	_ = mem.Save(memory.Observation{
		ID: "obs-1", Kind: "finding", Title: "IDOR in /api/users",
		Body:   "GET /api/users/100 returns data for user 101",
		Target: "example.com", CreatedAt: now, Tags: []string{"high", "idor"},
	})
	cfg := memory.EngramConfig{Target: "example.com", MaxFindings: 5}
	ctx := memory.BuildEngram(&mem, cfg)
	if !ctx.HasHistory {
		t.Fatalf("expected history for example.com")
	}
	if ctx.FindingCount < 1 {
		t.Fatalf("expected at least 1 finding, got %d", ctx.FindingCount)
	}
}

func TestEngramPreambleReturnsEmptyForNoHistory(t *testing.T) {
	dir := t.TempDir()
	mem := memory.New(dir)
	result := memory.EngramPreamble(&mem, "new-target.com", "")
	if result != "" {
		t.Fatalf("expected empty preamble for new target, got content")
	}
}
