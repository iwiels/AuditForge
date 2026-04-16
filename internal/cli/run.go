package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"orquestador-auditor/internal/catalog"
	auditruntime "orquestador-auditor/internal/runtime"
)

func runAudit(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("run requires a subcommand: start, phase, correlate, validate, inspect")
	}
	switch args[0] {
	case "start":
		return runAuditStart(args[1:])
	case "phase":
		return runAuditPhase(args[1:])
	case "correlate":
		return runAuditCorrelate(args[1:])
	case "validate":
		return runAuditValidate(args[1:])
	case "inspect":
		return runAuditInspect(args[1:])
	default:
		return fmt.Errorf("unknown run subcommand %q", args[0])
	}
}

func runAuditStart(args []string) error {
	fs := flag.NewFlagSet("run start", flag.ContinueOnError)
	var target string
	var targetKind string
	var profileName string
	var aggressiveness string
	var campaign string
	var authorizationRef string
	var approvedTools string
	var artifactsDir string
	var memoryDir string
	var authorized bool
	fs.StringVar(&target, "target", "", "Target URL, host, or path")
	fs.StringVar(&targetKind, "target-kind", "", "Target kind: web, api, host, repo, etc.")
	fs.StringVar(&profileName, "profile", "web-triage", "Audit profile")
	fs.StringVar(&aggressiveness, "aggressiveness", "", "Aggressiveness level: passive, bounded, active")
	fs.StringVar(&campaign, "campaign", "", "Campaign or engagement identifier")
	fs.StringVar(&authorizationRef, "authorization-ref", "", "Authorization ticket or evidence reference")
	fs.StringVar(&approvedTools, "approved-tools", "", "Comma-separated tools explicitly approved by policy")
	fs.StringVar(&artifactsDir, "artifacts-dir", filepath.Join(".orquestador", "runs"), "Runtime artifacts directory")
	fs.StringVar(&memoryDir, "memory-dir", filepath.Join(".orquestador", "memory"), "Memory directory")
	fs.BoolVar(&authorized, "authorized", false, "Mark the run as explicitly authorized")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("run start requires --target")
	}

	profile, err := catalog.AuditProfileByID(profileName)
	if err != nil {
		return err
	}
	level, err := auditruntime.NormalizeAggressiveness(aggressiveness)
	if err != nil {
		return err
	}
	if strings.TrimSpace(aggressiveness) == "" {
		level = auditruntime.DefaultAggressiveness(profile)
	}

	store := auditruntime.NewStore(artifactsDir, memoryDir)
	manifest, err := store.StartRun(auditruntime.StartRunInput{
		Target:           target,
		TargetKind:       targetKind,
		Campaign:         campaign,
		Authorized:       authorized,
		AuthorizationRef: authorizationRef,
		Profile:          profile,
		Aggressiveness:   level,
		ApprovedTools:    splitCSV(approvedTools),
	})
	if err != nil {
		return err
	}
	return printJSON(manifest)
}

func runAuditPhase(args []string) error {
	fs := flag.NewFlagSet("run phase", flag.ContinueOnError)
	var runID string
	var phaseName string
	var statusName string
	var summary string
	var requestedTools string
	var notes string
	var findingsFile string
	var artifactsDir string
	var memoryDir string
	fs.StringVar(&runID, "run-id", "", "Run identifier")
	fs.StringVar(&phaseName, "phase", "", "Phase ID")
	fs.StringVar(&statusName, "status", string(auditruntime.PhaseStatusObserved), "Phase status")
	fs.StringVar(&summary, "summary", "", "Structured summary for this phase")
	fs.StringVar(&requestedTools, "requested-tools", "", "Comma-separated tools requested for this phase")
	fs.StringVar(&notes, "notes", "", "Comma-separated notes to append to the artifact")
	fs.StringVar(&findingsFile, "findings-file", "", "Path to a JSON array of findings")
	fs.StringVar(&artifactsDir, "artifacts-dir", filepath.Join(".orquestador", "runs"), "Runtime artifacts directory")
	fs.StringVar(&memoryDir, "memory-dir", filepath.Join(".orquestador", "memory"), "Memory directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(runID) == "" {
		return fmt.Errorf("run phase requires --run-id")
	}
	phase, err := auditruntime.NormalizePhaseID(phaseName)
	if err != nil {
		return err
	}
	status, err := auditruntime.NormalizePhaseStatus(statusName)
	if err != nil {
		return err
	}
	findings, err := loadFindings(findingsFile)
	if err != nil {
		return err
	}

	store := auditruntime.NewStore(artifactsDir, memoryDir)
	artifact, err := store.RecordPhase(auditruntime.RecordPhaseInput{
		RunID:          runID,
		Phase:          phase,
		Status:         status,
		Summary:        summary,
		RequestedTools: splitCSV(requestedTools),
		Findings:       findings,
		Notes:          splitCSV(notes),
	})
	if err != nil {
		return err
	}
	return printJSON(artifact)
}

func runAuditCorrelate(args []string) error {
	fs := flag.NewFlagSet("run correlate", flag.ContinueOnError)
	var runID string
	var artifactsDir string
	var memoryDir string
	fs.StringVar(&runID, "run-id", "", "Run identifier")
	fs.StringVar(&artifactsDir, "artifacts-dir", filepath.Join(".orquestador", "runs"), "Runtime artifacts directory")
	fs.StringVar(&memoryDir, "memory-dir", filepath.Join(".orquestador", "memory"), "Memory directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(runID) == "" {
		return fmt.Errorf("run correlate requires --run-id")
	}
	store := auditruntime.NewStore(artifactsDir, memoryDir)
	artifact, err := store.CorrelateRun(runID)
	if err != nil {
		return err
	}
	return printJSON(artifact)
}

func runAuditValidate(args []string) error {
	fs := flag.NewFlagSet("run validate", flag.ContinueOnError)
	var runID string
	var artifactsDir string
	var memoryDir string
	fs.StringVar(&runID, "run-id", "", "Run identifier")
	fs.StringVar(&artifactsDir, "artifacts-dir", filepath.Join(".orquestador", "runs"), "Runtime artifacts directory")
	fs.StringVar(&memoryDir, "memory-dir", filepath.Join(".orquestador", "memory"), "Memory directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(runID) == "" {
		return fmt.Errorf("run validate requires --run-id")
	}
	store := auditruntime.NewStore(artifactsDir, memoryDir)
	artifact, err := store.ValidateRun(runID)
	if err != nil {
		return err
	}
	return printJSON(artifact)
}

func runAuditInspect(args []string) error {
	fs := flag.NewFlagSet("run inspect", flag.ContinueOnError)
	var runID string
	var artifactsDir string
	var memoryDir string
	fs.StringVar(&runID, "run-id", "", "Run identifier")
	fs.StringVar(&artifactsDir, "artifacts-dir", filepath.Join(".orquestador", "runs"), "Runtime artifacts directory")
	fs.StringVar(&memoryDir, "memory-dir", filepath.Join(".orquestador", "memory"), "Memory directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(runID) == "" {
		return fmt.Errorf("run inspect requires --run-id")
	}
	store := auditruntime.NewStore(artifactsDir, memoryDir)
	manifest, artifacts, err := store.InspectRun(runID)
	if err != nil {
		return err
	}
	return printJSON(map[string]interface{}{
		"manifest":  manifest,
		"artifacts": artifacts,
	})
}

func loadFindings(path string) ([]auditruntime.Finding, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var findings []auditruntime.Finding
	if err := json.Unmarshal(raw, &findings); err != nil {
		return nil, err
	}
	return findings, nil
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func printJSON(value interface{}) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(raw))
	return nil
}
