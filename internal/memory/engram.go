package memory

import (
	"fmt"
	"strings"
	"time"
)

// EngramProtocol builds cross-session context for injection into AI agent prompts.
// It queries the memory store for historical observations about a target/campaign
// and formats them as a concise preamble that the agent reads before starting work.
//
// This is the "Gentle AI Engram" equivalent: persistent memory that travels with
// every new session, so agents don't start blind.

// EngramConfig controls what context gets injected.
type EngramConfig struct {
	// Target is the primary target to search for (e.g. "example.com").
	Target string
	// Campaign narrows the search to a specific campaign ID.
	Campaign string
	// MaxFindings limits how many prior findings are included (default: 10).
	MaxFindings int
	// MaxNotes limits how many operational notes are included (default: 5).
	MaxNotes int
	// IncludeRuns whether to show prior run summaries.
	IncludeRuns bool
	// MaxRuns limits how many prior runs are shown (default: 3).
	MaxRuns int
}

// EngramContext is the formatted text block to inject into agent prompts.
type EngramContext struct {
	// Target is the target this context is about.
	Target string
	// HasHistory is true if prior observations exist.
	HasHistory bool
	// FormattedBlock is the full markdown text to inject.
	FormattedBlock string
	// FindingCount is the number of prior findings included.
	FindingCount int
	// RunCount is the number of prior runs included.
	RunCount int
	// GeneratedAt is when this context was built.
	GeneratedAt time.Time
}

// BuildEngram queries the store and returns formatted context for prompt injection.
func BuildEngram(store *Store, cfg EngramConfig) EngramContext {
	ctx := EngramContext{
		Target:      cfg.Target,
		GeneratedAt: time.Now().UTC(),
	}

	if cfg.MaxFindings <= 0 {
		cfg.MaxFindings = 10
	}
	if cfg.MaxNotes <= 0 {
		cfg.MaxNotes = 5
	}
	if cfg.MaxRuns <= 0 {
		cfg.MaxRuns = 3
	}

	var sb strings.Builder
	sb.WriteString("## Historical Context (auto-injected by Engram Protocol)\n\n")
	sb.WriteString(fmt.Sprintf("**Target:** %s\n", cfg.Target))

	if cfg.Campaign != "" {
		sb.WriteString(fmt.Sprintf("**Campaign:** %s\n", cfg.Campaign))
	}

	// Search for prior observations about this target
	observations, err := store.Search(cfg.Target, cfg.MaxFindings)
	if err != nil {
		// If search fails, still return context but note the failure
		sb.WriteString("\n> ⚠ Memory search unavailable — no historical context loaded.\n")
		ctx.FormattedBlock = sb.String()
		return ctx
	}

	// Filter to relevant observations
	var findings []Observation
	var notes []Observation
	var runs []Observation
	for _, obs := range observations {
		// Skip if campaign filter is set and doesn't match
		if cfg.Campaign != "" && obs.Campaign != "" && obs.Campaign != cfg.Campaign {
			continue
		}
		switch obs.Kind {
		case "finding", "vulnerability", "observation":
			findings = append(findings, obs)
		case "note", "operator-note", "decision":
			notes = append(notes, obs)
		case "audit-run", "engagement":
			if cfg.IncludeRuns {
				runs = append(runs, obs)
			}
		default:
			// General observations about the target
			if len(findings) < cfg.MaxFindings {
				findings = append(findings, obs)
			}
		}
	}

	ctx.FindingCount = len(findings)
	ctx.RunCount = len(runs)

	if len(findings) == 0 && len(notes) == 0 && len(runs) == 0 {
		sb.WriteString("\n> No prior observations found for this target.\n")
		ctx.FormattedBlock = sb.String()
		return ctx
	}

	ctx.HasHistory = true

	// Prior runs
	if len(runs) > 0 {
		sb.WriteString("\n### Prior Audit Runs\n\n")
		limit := cfg.MaxRuns
		if len(runs) < limit {
			limit = len(runs)
		}
		for i := 0; i < limit; i++ {
			obs := runs[i]
			sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n",
				safeStr(obs.Title, "Run"),
				obs.CreatedAt.Format("2006-01-02"),
				safeStr(obs.Body, "No details")))
		}
	}

	// Prior findings
	if len(findings) > 0 {
		sb.WriteString("\n### Prior Findings\n\n")
		limit := cfg.MaxFindings
		if len(findings) < limit {
			limit = len(findings)
		}
		for i := 0; i < limit; i++ {
			obs := findings[i]
			severity := extractSeverity(obs)
			sb.WriteString(fmt.Sprintf("- [%s] **%s**", strings.ToUpper(severity), safeStr(obs.Title, obs.Kind)))
			if obs.Tags != nil && len(obs.Tags) > 0 {
				sb.WriteString(fmt.Sprintf(" `[%s]`", strings.Join(obs.Tags[:min(len(obs.Tags), 3)], ", ")))
			}
			sb.WriteString("\n")
			if obs.Body != "" {
				// Truncate long bodies
				body := obs.Body
				if len(body) > 200 {
					body = body[:200] + "..."
				}
				sb.WriteString(fmt.Sprintf("  - %s\n", body))
			}
		}
	}

	// Operator notes
	if len(notes) > 0 {
		sb.WriteString("\n### Operator Notes\n\n")
		limit := cfg.MaxNotes
		if len(notes) < limit {
			limit = len(notes)
		}
		for i := 0; i < limit; i++ {
			obs := notes[i]
			sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n",
				safeStr(obs.Title, "Note"),
				obs.CreatedAt.Format("2006-01-02"),
				safeStr(obs.Body, "")))
		}
	}

	sb.WriteString("\n> Use this context to avoid duplicating findings and to validate if previously reported issues have been remediated.\n")

	ctx.FormattedBlock = sb.String()
	return ctx
}

// BuildMinimalEngram creates a compact context for agents with limited prompt space.
// It only includes the count of prior findings and the highest severity.
func BuildMinimalEngram(store *Store, target string) EngramContext {
	ctx := EngramContext{
		Target:      target,
		GeneratedAt: time.Now().UTC(),
	}

	observations, err := store.Search(target, 50)
	if err != nil || len(observations) == 0 {
		sb := strings.Builder{}
		sb.WriteString("## Historical Context\n\n")
		sb.WriteString("> No prior observations.\n")
		ctx.FormattedBlock = sb.String()
		return ctx
	}

	var findingCount int
	var severities []string
	for _, obs := range observations {
		if obs.Kind == "finding" || obs.Kind == "vulnerability" {
			findingCount++
			sev := extractSeverity(obs)
			if sev != "" {
				severities = append(severities, sev)
			}
		}
	}

	ctx.FindingCount = findingCount

	sb := strings.Builder{}
	sb.WriteString("## Historical Context\n\n")
	if findingCount > 0 {
		highest := highestSeverity(severities)
		sb.WriteString(fmt.Sprintf("- **Prior findings:** %d (highest severity: %s)\n", findingCount, highest))
		sb.WriteString("> Review full memory for details before duplicating analysis.\n")
	} else {
		sb.WriteString(fmt.Sprintf("- **Prior observations:** %d\n", len(observations)))
		sb.WriteString("> No security findings yet. Begin with reconnaissance.\n")
	}
	ctx.HasHistory = findingCount > 0
	ctx.FormattedBlock = sb.String()
	return ctx
}

// EngramPreamble returns the block to append to the system prompt.
// If there's no history, returns empty string (nothing to inject).
func EngramPreamble(store *Store, target string, campaign string) string {
	if store == nil || strings.TrimSpace(target) == "" {
		return ""
	}

	cfg := EngramConfig{
		Target:       target,
		Campaign:     campaign,
		MaxFindings:  8,
		MaxNotes:     3,
		IncludeRuns:  true,
		MaxRuns:      2,
	}

	ctx := BuildEngram(store, cfg)
	if !ctx.HasHistory {
		return ""
	}
	return "\n\n" + ctx.FormattedBlock
}

func extractSeverity(obs Observation) string {
	// Try to extract severity from tags
	for _, tag := range obs.Tags {
		switch strings.ToLower(tag) {
		case "critical":
			return "critical"
		case "high":
			return "high"
		case "medium":
			return "medium"
		case "low":
			return "low"
		case "info", "informational":
			return "info"
		}
	}
	// Try to extract from title/body
	text := strings.ToLower(obs.Title + " " + obs.Body)
	if strings.Contains(text, "critical") {
		return "critical"
	}
	if strings.Contains(text, "high") {
		return "high"
	}
	if strings.Contains(text, "medium") {
		return "medium"
	}
	if strings.Contains(text, "low") {
		return "low"
	}
	return ""
}

func highestSeverity(severities []string) string {
	order := map[string]int{"": 0, "info": 1, "low": 2, "medium": 3, "high": 4, "critical": 5}
	highest := ""
	for _, sev := range severities {
		if order[sev] > order[highest] {
			highest = sev
		}
	}
	if highest == "" {
		return "unknown"
	}
	return highest
}

func safeStr(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
