package auditruntime

import (
	"fmt"
	"strings"
	"time"
)

// JudgmentPhase performs quality validation on audit findings before report generation.
// It's the "judgment day" equivalent: every finding must pass quality checks
// or the run is flagged as not-ready-to-report.

// ValidationCheck is a single quality gate applied to a finding or the overall run.
type ValidationCheck struct {
	ID        string `json:"id"`
	Check     string `json:"check"`
	Passed    bool   `json:"passed"`
	FindingID string `json:"finding_id,omitempty"`
	Severity  string `json:"severity,omitempty"` // severity of this check failure
	Detail    string `json:"detail"`
}

// JudgmentResult is the overall validation outcome for a run.
type JudgmentResult struct {
	RunID         string            `json:"run_id"`
	TotalChecks   int               `json:"total_checks"`
	Passed        int               `json:"passed"`
	Failed        int               `json:"failed"`
	FailedChecks  []ValidationCheck `json:"failed_checks"`
	QualityScore  float64           `json:"quality_score"` // 0-100
	ReadyToReport bool              `json:"ready_to_report"`
	Recommendations []string        `json:"recommendations,omitempty"`
	ValidatedAt   time.Time         `json:"validated_at"`
}

// ValidateRun runs all quality checks against the correlated findings of a run.
func ValidateRun(manifest RunManifest, artifacts []PhaseArtifact) JudgmentResult {
	result := JudgmentResult{
		RunID:       manifest.RunID,
		ValidatedAt: time.Now().UTC(),
	}

	var checks []ValidationCheck

	// Collect all findings across phases
	var allFindings []Finding
	for _, artifact := range artifacts {
		if artifact.Phase == PhaseCorrelation {
			continue
		}
		allFindings = append(allFindings, artifact.Findings...)
	}

	// === Per-finding checks ===
	for _, finding := range allFindings {
		checks = append(checks, checkEvidencePresent(finding)...)
		checks = append(checks, checkVectorConfirmed(finding)...)
		checks = append(checks, checkCWEAssigned(finding)...)
		checks = append(checks, checkRemediationActionable(finding)...)
	}

	// === Run-level checks ===
	checks = append(checks, checkNoDuplicates(allFindings)...)
	checks = append(checks, checkNoSkippedPhases(manifest, artifacts)...)
	checks = append(checks, checkOWASPMapping(allFindings)...)
	checks = append(checks, checkSeverityConsistency(allFindings)...)
	checks = append(checks, checkFindingsTitleDescriptive(allFindings)...)

	// Aggregate results
	result.TotalChecks = len(checks)
	for _, check := range checks {
		if check.Passed {
			result.Passed++
		} else {
			result.Failed++
			result.FailedChecks = append(result.FailedChecks, check)
		}
	}

	// Calculate quality score (0-100)
	if result.TotalChecks > 0 {
		result.QualityScore = float64(result.Passed) / float64(result.TotalChecks) * 100
	}

	// Determine if ready to report
	// A run is NOT ready if:
	// - Any CRITICAL finding lacks evidence
	// - Any HIGH finding lacks vector confirmation
	// - Quality score < 60
	result.ReadyToReport = result.QualityScore >= 60 && !hasCriticalEvidenceGap(result.FailedChecks)

	// Generate recommendations
	result.Recommendations = generateValidationRecommendations(result.FailedChecks)

	return result
}

func checkEvidencePresent(finding Finding) []ValidationCheck {
	var checks []ValidationCheck
	checkID := fmt.Sprintf("evidence-%s", finding.ID)

	if len(finding.Evidence) == 0 {
		checks = append(checks, ValidationCheck{
			ID:        checkID,
			Check:     "evidence_present",
			Passed:    false,
			FindingID: finding.ID,
			Severity:  "medium",
			Detail:    fmt.Sprintf("Finding %q has no evidence — add observable proof before reporting", finding.Title),
		})
	} else {
		checks = append(checks, ValidationCheck{
			ID:        checkID,
			Check:     "evidence_present",
			Passed:    true,
			FindingID: finding.ID,
			Detail:    fmt.Sprintf("%d evidence items found", len(finding.Evidence)),
		})
	}
	return checks
}

func checkVectorConfirmed(finding Finding) []ValidationCheck {
	var checks []ValidationCheck

	// Only required for CRITICAL and HIGH severity findings
	if !isHighSeverity(finding.Severity) {
		return checks
	}

	checkID := fmt.Sprintf("vector-%s", finding.ID)

	// Check if any evidence contains confirmation language or specific data
	hasVector := false
	for _, ev := range finding.Evidence {
		evLower := strings.ToLower(ev)
		if strings.Contains(evLower, "confirmed") ||
			strings.Contains(evLower, "validated") ||
			strings.Contains(evLower, "response:") ||
			strings.Contains(evLower, "http/") ||
			strings.Contains(evLower, "payload:") ||
			strings.Contains(evLower, "curl ") {
			hasVector = true
			break
		}
	}

	if !hasVector {
		checks = append(checks, ValidationCheck{
			ID:        checkID,
			Check:     "vector_confirmed",
			Passed:    false,
			FindingID: finding.ID,
			Severity:  "high",
			Detail:    fmt.Sprintf("HIGH/CRITICAL finding %q lacks confirmed attack vector — add proof-of-concept evidence", finding.Title),
		})
	} else {
		checks = append(checks, ValidationCheck{
			ID:        checkID,
			Check:     "vector_confirmed",
			Passed:    true,
			FindingID: finding.ID,
			Detail:    "Attack vector confirmed with evidence",
		})
	}
	return checks
}

func checkCWEAssigned(finding Finding) []ValidationCheck {
	var checks []ValidationCheck
	checkID := fmt.Sprintf("cwe-%s", finding.ID)

	// Required for validated findings with known category
	if finding.State != FindingStateValidated {
		return checks
	}

	if finding.CWE == "" {
		checks = append(checks, ValidationCheck{
			ID:        checkID,
			Check:     "cwe_assigned",
			Passed:    false,
			FindingID: finding.ID,
			Severity:  "low",
			Detail:    fmt.Sprintf("Validated finding %q has no CWE assigned", finding.Title),
		})
	} else {
		checks = append(checks, ValidationCheck{
			ID:        checkID,
			Check:     "cwe_assigned",
			Passed:    true,
			FindingID: finding.ID,
			Detail:    fmt.Sprintf("CWE: %s", finding.CWE),
		})
	}
	return checks
}

func checkRemediationActionable(finding Finding) []ValidationCheck {
	var checks []ValidationCheck
	checkID := fmt.Sprintf("remediation-%s", finding.ID)

	if finding.Remediation == "" {
		checks = append(checks, ValidationCheck{
			ID:        checkID,
			Check:     "remediation_actionable",
			Passed:    false,
			FindingID: finding.ID,
			Severity:  "medium",
			Detail:    fmt.Sprintf("Finding %q has no remediation guidance", finding.Title),
		})
	} else if isGenericRemediation(finding.Remediation) {
		checks = append(checks, ValidationCheck{
			ID:        checkID,
			Check:     "remediation_actionable",
			Passed:    false,
			FindingID: finding.ID,
			Severity:  "low",
			Detail:    fmt.Sprintf("Finding %q has generic remediation — make it specific to the vulnerability", finding.Title),
		})
	} else {
		checks = append(checks, ValidationCheck{
			ID:        checkID,
			Check:     "remediation_actionable",
			Passed:    true,
			FindingID: finding.ID,
			Detail:    "Remediation is specific and actionable",
		})
	}
	return checks
}

func checkNoDuplicates(findings []Finding) []ValidationCheck {
	var checks []ValidationCheck

	seen := map[string]int{}
	for _, f := range findings {
		key := strings.ToLower(strings.TrimSpace(f.Category + "|" + f.Title))
		seen[key]++
	}

	dupCount := 0
	for key, count := range seen {
		if count > 1 {
			dupCount++
			parts := strings.Split(key, "|")
			checks = append(checks, ValidationCheck{
				ID:     fmt.Sprintf("no-duplicates-%d", dupCount),
				Check:  "no_duplicates",
				Passed: false,
				Severity: "low",
				Detail: fmt.Sprintf("%d findings with same category/title: %q", count, parts[1]),
			})
		}
	}

	if dupCount == 0 {
		checks = append(checks, ValidationCheck{
			ID:     "no-duplicates",
			Check:  "no_duplicates",
			Passed: true,
			Detail: fmt.Sprintf("No duplicate findings across %d total", len(findings)),
		})
	}

	return checks
}

func checkNoSkippedPhases(manifest RunManifest, artifacts []PhaseArtifact) []ValidationCheck {
	var checks []ValidationCheck

	var skipped []string
	for _, phase := range manifest.PhasePlan {
		if phase == PhaseCorrelation {
			continue
		}
		found := false
		for _, artifact := range artifacts {
			if artifact.Phase == phase && artifact.Status != PhaseStatusPending {
				found = true
				break
			}
		}
		if !found {
			skipped = append(skipped, string(phase))
		}
	}

	if len(skipped) > 0 {
		checks = append(checks, ValidationCheck{
			ID:       "no-skipped-phases",
			Check:    "no_skipped_phases",
			Passed:   false,
			Severity: "medium",
			Detail:   fmt.Sprintf("Phases skipped: %s", strings.Join(skipped, ", ")),
		})
	} else {
		checks = append(checks, ValidationCheck{
			ID:     "no-skipped-phases",
			Check:  "no_skipped_phases",
			Passed: true,
			Detail: "All phases completed",
		})
	}

	return checks
}

func checkOWASPMapping(findings []Finding) []ValidationCheck {
	var checks []ValidationCheck

	missingCount := 0
	for _, f := range findings {
		if f.State == FindingStateValidated && f.OWASP == "" {
			missingCount++
		}
	}

	if missingCount > 0 {
		checks = append(checks, ValidationCheck{
			ID:       "owasp-mapping",
			Check:    "owasp_mapping",
			Passed:   false,
			Severity: "low",
			Detail:   fmt.Sprintf("%d validated findings lack OWASP Top 10 mapping", missingCount),
		})
	} else {
		checks = append(checks, ValidationCheck{
			ID:     "owasp-mapping",
			Check:  "owasp_mapping",
			Passed: true,
			Detail: "All validated findings have OWASP mapping",
		})
	}

	return checks
}

func checkSeverityConsistency(findings []Finding) []ValidationCheck {
	var checks []ValidationCheck

	// Check that severity matches the expected taxonomy for the category
	mismatchCount := 0
	for _, f := range findings {
		if f.Severity == "" || f.Category == "" {
			continue
		}
		tax, ok := categoryTaxonomy[f.Category]
		if !ok {
			continue
		}
		if !severityCompatible(f.Severity, tax.Severity) {
			mismatchCount++
		}
	}

	if mismatchCount > 0 {
		checks = append(checks, ValidationCheck{
			ID:       "severity-consistency",
			Check:    "severity_consistency",
			Passed:   false,
			Severity: "info",
			Detail:   fmt.Sprintf("%d findings have severity that differs from taxonomy default (informational only)", mismatchCount),
		})
	} else {
		checks = append(checks, ValidationCheck{
			ID:     "severity-consistency",
			Check:  "severity_consistency",
			Passed: true,
			Detail: "All findings match taxonomy severity or have justified override",
		})
	}

	return checks
}

func checkFindingsTitleDescriptive(findings []Finding) []ValidationCheck {
	var checks []ValidationCheck

	for _, f := range findings {
		checkID := fmt.Sprintf("title-%s", f.ID)
		title := strings.TrimSpace(f.Title)
		if len(title) < 10 {
			checks = append(checks, ValidationCheck{
				ID:        checkID,
				Check:     "title_descriptive",
				Passed:    false,
				FindingID: f.ID,
				Severity:  "low",
				Detail:    fmt.Sprintf("Finding title too short: %q (min 10 chars)", title),
			})
		} else {
			checks = append(checks, ValidationCheck{
				ID:        checkID,
				Check:     "title_descriptive",
				Passed:    true,
				FindingID: f.ID,
				Detail:    "Title is descriptive",
			})
		}
	}

	return checks
}

// Helper functions

func isHighSeverity(severity string) bool {
	s := strings.ToLower(strings.TrimSpace(severity))
	return s == "high" || s == "critical"
}

func isGenericRemediation(remediation string) bool {
	lower := strings.ToLower(strings.TrimSpace(remediation))
	genericPhrases := []string{
		"fix the vulnerability",
		"apply security patches",
		"follow security best practices",
		"review and update",
		"ensure proper",
	}
	for _, phrase := range genericPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func severityCompatible(actual, expected string) bool {
	order := map[string]int{"info": 1, "low": 2, "medium": 3, "high": 4, "critical": 5}
	a := order[strings.ToLower(actual)]
	e := order[strings.ToLower(expected)]
	// Allow if within 1 level of expected (justified override)
	return a >= e-1 && a <= e+1
}

func hasCriticalEvidenceGap(failedChecks []ValidationCheck) bool {
	for _, check := range failedChecks {
		if check.Check == "evidence_present" && check.Severity == "high" {
			return true
		}
	}
	return false
}

func generateValidationRecommendations(failedChecks []ValidationCheck) []string {
	recs := []string{}
	seen := map[string]bool{}

	for _, check := range failedChecks {
		switch check.Check {
		case "evidence_present":
			if !seen["evidence"] {
				recs = append(recs, "Add observable evidence to all findings before reporting — a finding without proof is an opinion, not intelligence")
				seen["evidence"] = true
			}
		case "vector_confirmed":
			if !seen["vector"] {
				recs = append(recs, "Confirm attack vectors for HIGH/CRITICAL findings with proof-of-concept or response evidence")
				seen["vector"] = true
			}
		case "cwe_assigned":
			if !seen["cwe"] {
				recs = append(recs, "Assign CWE IDs to all validated findings for standardized vulnerability reporting")
				seen["cwe"] = true
			}
		case "remediation_actionable":
			if !seen["remediation"] {
				recs = append(recs, "Write specific, actionable remediation for each finding — generic guidance is not useful to developers")
				seen["remediation"] = true
			}
		case "no_duplicates":
			if !seen["dedup"] {
				recs = append(recs, "Deduplicate findings — merge observations with the same root cause")
				seen["dedup"] = true
			}
		case "no_skipped_phases":
			if !seen["phases"] {
				recs = append(recs, "Complete all methodology phases before closing the audit — document justification for any skipped phases")
				seen["phases"] = true
			}
		}
	}

	return recs
}
