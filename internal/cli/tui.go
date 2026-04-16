package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	auditruntime "orquestador-auditor/internal/runtime"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen string

const (
	screenDashboard  screen = "dashboard"
	screenNewRun     screen = "new-run"
	screenInstall    screen = "install"
	screenSync       screen = "sync"
	screenMemory     screen = "memory"
	screenSelfUpdate screen = "self-update"
	screenRunDetail  screen = "run-detail"
	screenOutput     screen = "output"
)

type tuiModel struct {
	screen      screen
	resultTitle string
	resultBody  string
	resultErr   error

	// Dashboard state
	runs          []runSummary
	selectedRun   int
	dashboardPage int // 0 = main, 1 = findings detail

	// New run wizard
	wizardStep       int
	wizardTotalSteps int
	wizardTarget     textinput.Model
	wizardAuth       bool
	wizardAuthRef    textinput.Model
	wizardProfile    textinput.Model
	wizardAggro      int
	wizardTools      []bool
	wizardToolNames  []string

	// Install screen
	installBundleIdx int
	installBundles   []string
	installExecute   bool

	// Sync screen
	syncAgent textinput.Model

	// Memory screen
	memoryDir   textinput.Model
	memoryQuery textinput.Model
	memoryLimit textinput.Model

	// Run detail
	detailRunID     string
	detailArtifacts []auditruntime.PhaseArtifact
	detailManifest  auditruntime.RunManifest

	styles tuiStyles
}

type runSummary struct {
	RunID           string
	Target          string
	Profile         string
	CurrentPhase    string
	TotalPhases     int
	CompletedPhases int
	Findings        int
	HighestSeverity string
	CreatedAt       time.Time
	HasJudgment     bool
	ReadyToReport   bool
	QualityScore    float64
}

type tuiStyles struct {
	title    lipgloss.Style
	active   lipgloss.Style
	muted    lipgloss.Style
	errStyle lipgloss.Style
	panel    lipgloss.Style
	success  lipgloss.Style
	warn     lipgloss.Style
	critical lipgloss.Style
	header   lipgloss.Style
	card     lipgloss.Style
	progress lipgloss.Style
}

func runInteractive() error {
	p := tea.NewProgram(newTUIModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func newTUIModel() tuiModel {
	styles := tuiStyles{
		title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")),
		active:   lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true),
		muted:    lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		errStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		success:  lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),
		warn:     lipgloss.NewStyle().Foreground(lipgloss.Color("208")),
		critical: lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		panel:    lipgloss.NewStyle().Padding(1, 2),
		header:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("228")),
		card:     lipgloss.NewStyle().Padding(0, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")),
		progress: lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	}

	syncAgent := textinput.New()
	syncAgent.Placeholder = "vacío = todos; o opencode,cursor"
	syncAgent.Width = 72

	memoryDir := textinput.New()
	memoryDir.Placeholder = "~/.orquestador/memory"
	memoryDir.Width = 72

	memoryQuery := textinput.New()
	memoryQuery.Placeholder = "admin, graphql, campaign, secret..."
	memoryQuery.Width = 72

	memoryLimit := textinput.New()
	memoryLimit.SetValue("10")
	memoryLimit.Width = 8

	wizardTarget := textinput.New()
	wizardTarget.Placeholder = "https://example.com"
	wizardTarget.Width = 72

	wizardAuthRef := textinput.New()
	wizardAuthRef.Placeholder = "ticket-123 o email de autorización"
	wizardAuthRef.Width = 72

	wizardProfile := textinput.New()
	wizardProfile.SetValue("web-triage")
	wizardProfile.Width = 30

	model := tuiModel{
		screen:           screenDashboard,
		installBundles:   []string{"core-web", "supply-chain", "advanced-web", "full"},
		syncAgent:        syncAgent,
		memoryDir:        memoryDir,
		memoryQuery:      memoryQuery,
		memoryLimit:      memoryLimit,
		styles:           styles,
		wizardTarget:     wizardTarget,
		wizardAuthRef:    wizardAuthRef,
		wizardProfile:    wizardProfile,
		wizardTotalSteps: 4,
		wizardAggro:      1,                                             // bounded
		wizardTools:      []bool{true, true, false, true, false, false}, // nmap, katana, sqlmap, arjun, ffuf, nikto
		wizardToolNames:  []string{"nmap", "katana", "sqlmap", "arjun", "ffuf", "nikto"},
	}

	model.loadDashboard()
	return model
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.screen {
		case screenDashboard:
			return m.updateDashboard(msg)
		case screenNewRun:
			return m.updateWizard(msg)
		case screenInstall:
			return m.updateInstall(msg)
		case screenSync:
			return m.updateSync(msg)
		case screenMemory:
			return m.updateMemory(msg)
		case screenSelfUpdate:
			return m.updateSelfUpdate(msg)
		case screenRunDetail:
			return m.updateRunDetail(msg)
		case screenOutput:
			if msg.String() == "esc" || msg.String() == "q" || msg.String() == "enter" {
				m.screen = screenDashboard
				m.loadDashboard()
				return m, nil
			}
		}
	case execResultMsg:
		m.screen = screenOutput
		m.resultTitle = msg.title
		m.resultBody = msg.output
		m.resultErr = msg.err
		return m, nil
	}

	var cmd tea.Cmd
	switch m.screen {
	case screenSync:
		m.syncAgent, cmd = m.syncAgent.Update(msg)
	case screenMemory:
		if m.memoryQuery.Focused() {
			m.memoryQuery, cmd = m.memoryQuery.Update(msg)
		} else {
			m.memoryDir, cmd = m.memoryDir.Update(msg)
		}
	case screenNewRun:
		switch m.wizardStep {
		case 0:
			m.wizardTarget, cmd = m.wizardTarget.Update(msg)
		case 1:
			m.wizardAuthRef, cmd = m.wizardAuthRef.Update(msg)
		case 2:
			m.wizardProfile, cmd = m.wizardProfile.Update(msg)
		}
	case screenRunDetail:
		// No text input on detail screen
	}
	return m, cmd
}

func (m *tuiModel) loadDashboard() {
	runsDir := filepath.Join(".orquestador", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		m.runs = nil
		return
	}

	var summaries []runSummary
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(runsDir, entry.Name(), "run.json")
		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		var manifest auditruntime.RunManifest
		if err := json.Unmarshal(raw, &manifest); err != nil {
			continue
		}

		summary := runSummary{
			RunID:        manifest.RunID,
			Target:       manifest.Target,
			Profile:      string(manifest.Profile),
			CurrentPhase: string(manifest.CurrentPhase),
			TotalPhases:  len(manifest.PhasePlan),
			CreatedAt:    manifest.CreatedAt,
		}

		// Count completed phases and findings
		for _, phase := range manifest.Phases {
			if phase.Status != auditruntime.PhaseStatusPending {
				summary.CompletedPhases++
			}
			if phase.Phase == auditruntime.PhaseCorrelation {
				// Load correlation artifact for findings count
				corrPath := filepath.Join(runsDir, entry.Name(), "phases", "correlation.json")
				if raw, err := os.ReadFile(corrPath); err == nil {
					var artifact auditruntime.PhaseArtifact
					if json.Unmarshal(raw, &artifact) == nil {
						summary.Findings = artifact.Correlation.TotalFindings
						if artifact.Correlation != nil {
							summary.HighestSeverity = artifact.Correlation.HighestSeverity
						}
					}
				}
			}
			if phase.Phase == auditruntime.PhaseJudgment {
				summary.HasJudgment = true
				judPath := filepath.Join(runsDir, entry.Name(), "phases", "judgment.json")
				if raw, err := os.ReadFile(judPath); err == nil {
					var artifact auditruntime.PhaseArtifact
					if json.Unmarshal(raw, &artifact) == nil && artifact.Judgment != nil {
						summary.ReadyToReport = artifact.Judgment.ReadyToReport
						summary.QualityScore = artifact.Judgment.QualityScore
					}
				}
			}
		}

		summaries = append(summaries, summary)
	}

	// Sort by creation date (newest first)
	sort.SliceStable(summaries, func(i, j int) bool {
		return summaries[i].CreatedAt.After(summaries[j].CreatedAt)
	})

	m.runs = summaries
}

// Dashboard

func (m tuiModel) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "n":
		m.screen = screenNewRun
		m.wizardStep = 0
		m.wizardTarget.Focus()
		return m, nil
	case "s":
		m.screen = screenSync
		m.syncAgent.Focus()
		return m, nil
	case "m":
		m.screen = screenMemory
		m.memoryQuery.Focus()
		return m, nil
	case "i":
		m.screen = screenInstall
		return m, nil
	case "u":
		m.screen = screenSelfUpdate
		return m, nil
	case "r":
		m.loadDashboard()
		return m, nil
	case "up", "k":
		if m.selectedRun > 0 {
			m.selectedRun--
		}
	case "down", "j":
		if m.selectedRun < len(m.runs)-1 {
			m.selectedRun++
		}
	case "enter":
		if len(m.runs) > 0 && m.selectedRun < len(m.runs) {
			return m.openRunDetail(m.runs[m.selectedRun].RunID)
		}
	}
	return m, nil
}

func (m tuiModel) viewDashboard() string {
	var sb strings.Builder

	// Header
	sb.WriteString(m.styles.title.Render("🛡 Security Audit Orchestrator — Dashboard"))
	sb.WriteString("\n\n")

	// Stats bar
	sb.WriteString(m.styles.muted.Render(fmt.Sprintf("Runs: %d  |  Memory: search enabled  |  [r] refresh  [n] new run  [s] sync  [m] memory  [i] install  [u] update  [q] quit", len(m.runs))))
	sb.WriteString("\n\n")

	if len(m.runs) == 0 {
		sb.WriteString(m.styles.muted.Render("No audit runs yet. Press [n] to start a new engagement."))
		return m.styles.panel.Render(sb.String())
	}

	// Run cards
	for i, run := range m.runs {
		if i > 5 {
			break // Show max 6 runs
		}
		sb.WriteString(m.renderRunCard(run, i == m.selectedRun))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(m.styles.muted.Render("↑/↓ select  enter = details"))

	return m.styles.panel.Render(sb.String())
}

func (m tuiModel) renderRunCard(run runSummary, selected bool) string {
	var sb strings.Builder

	// Progress bar
	progress := 0
	if run.TotalPhases > 0 {
		progress = (run.CompletedPhases * 20) / run.TotalPhases
	}
	bar := m.renderProgressBar(progress)

	// Status indicator
	status := "○"
	if run.ReadyToReport {
		status = "✅"
	} else if run.HasJudgment && !run.ReadyToReport {
		status = "⚠️"
	} else if run.CompletedPhases > 0 {
		status = "●"
	}

	if selected {
		sb.WriteString(m.styles.active.Render(fmt.Sprintf("> %s %s [%s]", status, run.Target, run.Profile)))
	} else {
		sb.WriteString(fmt.Sprintf("  %s %s [%s]", status, run.Target, run.Profile))
	}

	sb.WriteString(fmt.Sprintf("\n     Phase: %d/%d  %s", run.CompletedPhases, run.TotalPhases, bar))

	if run.Findings > 0 {
		severityColor := m.styles.muted
		switch strings.ToLower(run.HighestSeverity) {
		case "critical":
			severityColor = m.styles.critical
		case "high":
			severityColor = m.styles.errStyle
		case "medium":
			severityColor = m.styles.warn
		}
		sb.WriteString(fmt.Sprintf("  %s", severityColor.Render(fmt.Sprintf("Findings: %d (max: %s)", run.Findings, run.HighestSeverity))))
	}

	if run.HasJudgment {
		if run.ReadyToReport {
			sb.WriteString(fmt.Sprintf("  %s", m.styles.success.Render(fmt.Sprintf("Quality: %.0f%% ✓", run.QualityScore))))
		} else {
			sb.WriteString(fmt.Sprintf("  %s", m.styles.warn.Render(fmt.Sprintf("Quality: %.0f%% — needs fixes", run.QualityScore))))
		}
	}

	sb.WriteString(fmt.Sprintf("\n     %s", m.styles.muted.Render(run.RunID)))

	return sb.String()
}

func (m tuiModel) renderProgressBar(percent int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := percent / 10
	empty := 10 - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	if percent >= 100 {
		return m.styles.progress.Render(bar + " 100%")
	}
	return m.styles.muted.Render(bar + fmt.Sprintf(" %d%%", percent))
}

// Engagement Wizard

func (m tuiModel) updateWizard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.wizardStep > 0 {
			m.wizardStep--
			m.focusWizardStep()
			return m, nil
		}
		m.screen = screenDashboard
		return m, nil
	case "enter":
		if m.wizardStep == m.wizardTotalSteps-1 {
			return m.launchWizard()
		}
		m.wizardStep++
		m.focusWizardStep()
		return m, nil
	case "tab":
		// Toggle tools on step 3
		if m.wizardStep == 3 {
			m.wizardTools[m.wizardAggro] = !m.wizardTools[m.wizardAggro]
		}
	}

	var cmd tea.Cmd
	switch m.wizardStep {
	case 0:
		m.wizardTarget, cmd = m.wizardTarget.Update(msg)
	case 1:
		m.wizardAuthRef, cmd = m.wizardAuthRef.Update(msg)
	case 2:
		m.wizardProfile, cmd = m.wizardProfile.Update(msg)
	case 3:
		switch msg.String() {
		case "up", "k":
			if m.wizardAggro > 0 {
				m.wizardAggro--
			}
		case "down", "j":
			if m.wizardAggro < len(m.wizardTools)-1 {
				m.wizardAggro++
			}
		case " ":
			m.wizardTools[m.wizardAggro] = !m.wizardTools[m.wizardAggro]
		}
	}
	return m, cmd
}

func (m *tuiModel) focusWizardStep() {
	m.wizardTarget.Blur()
	m.wizardAuthRef.Blur()
	m.wizardProfile.Blur()
	switch m.wizardStep {
	case 0:
		m.wizardTarget.Focus()
	case 1:
		m.wizardAuthRef.Focus()
	case 2:
		m.wizardProfile.Focus()
	}
}

func (m tuiModel) launchWizard() (tea.Model, tea.Cmd) {
	target := strings.TrimSpace(m.wizardTarget.Value())
	if target == "" {
		m.resultErr = fmt.Errorf("target is required")
		m.screen = screenOutput
		m.resultTitle = "New Run"
		return m, nil
	}

	args := []string{
		"start",
		"--target", target,
		"--profile", strings.TrimSpace(m.wizardProfile.Value()),
		"--aggressiveness", []string{"passive", "bounded", "active"}[min(m.wizardAggro, 2)],
		"--authorized",
	}
	if strings.TrimSpace(m.wizardAuthRef.Value()) != "" {
		args = append(args, "--authorization-ref", strings.TrimSpace(m.wizardAuthRef.Value()))
	}

	var approved []string
	for i, enabled := range m.wizardTools {
		if enabled && i < len(m.wizardToolNames) {
			approved = append(approved, m.wizardToolNames[i])
		}
	}
	if len(approved) > 0 {
		args = append(args, "--approved-tools", strings.Join(approved, ","))
	}

	return m, execCmd("New Run", func() (string, error) {
		return captureOutput(func() error { return runAudit(args) })
	})
}

func (m tuiModel) viewWizard() string {
	var sb strings.Builder

	sb.WriteString(m.styles.header.Render(fmt.Sprintf("New Engagement — Step %d/%d", m.wizardStep+1, m.wizardTotalSteps)))
	sb.WriteString("\n\n")

	switch m.wizardStep {
	case 0:
		sb.WriteString("Target URL, IP, or repository path:")
		sb.WriteString("\n")
		sb.WriteString(m.wizardTarget.View())
		sb.WriteString("\n\n")
		sb.WriteString(m.styles.muted.Render("This is the authorized target for security assessment."))
	case 1:
		sb.WriteString("Authorization reference (ticket, email, contract):")
		sb.WriteString("\n")
		sb.WriteString(m.wizardAuthRef.View())
		sb.WriteString("\n\n")
		sb.WriteString(m.styles.success.Render("✓ Authorization confirmed"))
	case 2:
		sb.WriteString("Audit profile (methodology scope):")
		sb.WriteString("\n")
		sb.WriteString(m.wizardProfile.View())
		sb.WriteString("\n\n")
		sb.WriteString(m.styles.muted.Render("recon = passive only | web-triage = deep web + bounded validation | supply-chain = code + deps"))
	case 3:
		sb.WriteString("Approve tools (space to toggle, ↑↓ to navigate):")
		sb.WriteString("\n\n")
		for i, tool := range m.wizardToolNames {
			prefix := "  "
			checked := "☐"
			if m.wizardTools[i] {
				checked = "☑"
			}
			if i == m.wizardAggro {
				prefix = "> "
			}
			sb.WriteString(fmt.Sprintf("%s%s %s\n", prefix, checked, tool))
		}
		sb.WriteString("\n")
		sb.WriteString(m.styles.muted.Render("Only approved tools will be available during the audit."))
	}

	sb.WriteString("\n\n")
	if m.wizardStep < m.wizardTotalSteps-1 {
		sb.WriteString(m.styles.muted.Render("Enter = next  Esc = back"))
	} else {
		sb.WriteString(m.styles.success.Render("Enter = launch engagement  Esc = back"))
	}

	return m.styles.panel.Render(sb.String())
}

// Run Detail

func (m tuiModel) openRunDetail(runID string) (tea.Model, tea.Cmd) {
	runsDir := filepath.Join(".orquestador", "runs")
	manifestPath := filepath.Join(runsDir, runID, "run.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		m.resultErr = err
		m.screen = screenOutput
		m.resultTitle = "Load Run"
		return m, nil
	}

	var manifest auditruntime.RunManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		m.resultErr = err
		m.screen = screenOutput
		m.resultTitle = "Load Run"
		return m, nil
	}

	var artifacts []auditruntime.PhaseArtifact
	for _, phase := range manifest.Phases {
		phasePath := filepath.Join(runsDir, runID, "phases", string(phase.Phase)+".json")
		if raw, err := os.ReadFile(phasePath); err == nil {
			var artifact auditruntime.PhaseArtifact
			if json.Unmarshal(raw, &artifact) == nil {
				artifacts = append(artifacts, artifact)
			}
		}
	}

	m.screen = screenRunDetail
	m.detailRunID = runID
	m.detailManifest = manifest
	m.detailArtifacts = artifacts
	return m, nil
}

func (m tuiModel) updateRunDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		m.screen = screenDashboard
		m.loadDashboard()
		return m, nil
	case "v":
		// Run validation
		return m, execCmd("Run Validation", func() (string, error) {
			return captureOutput(func() error {
				return runAudit([]string{"validate", "--run-id", m.detailRunID})
			})
		})
	case "c":
		// Run correlation
		return m, execCmd("Run Correlation", func() (string, error) {
			return captureOutput(func() error {
				return runAudit([]string{"correlate", "--run-id", m.detailRunID})
			})
		})
	case "r":
		return m.openRunDetail(m.detailRunID)
	}
	return m, nil
}

func (m tuiModel) viewRunDetail() string {
	var sb strings.Builder

	sb.WriteString(m.styles.header.Render(fmt.Sprintf("Run: %s", m.detailRunID)))
	sb.WriteString("\n\n")

	// Manifest info
	sb.WriteString(fmt.Sprintf("Target: %s\n", m.detailManifest.Target))
	sb.WriteString(fmt.Sprintf("Profile: %s (%s)\n", m.detailManifest.Profile, m.detailManifest.ProfileMode))
	sb.WriteString(fmt.Sprintf("Aggressiveness: %s\n", m.detailManifest.Aggressiveness))
	sb.WriteString(fmt.Sprintf("Authorized: %t\n", m.detailManifest.Authorized))
	sb.WriteString("\n")

	// Phase timeline
	sb.WriteString(m.styles.header.Render("Phase Timeline"))
	sb.WriteString("\n\n")
	for _, phase := range m.detailManifest.Phases {
		icon := "○"
		switch phase.Status {
		case auditruntime.PhaseStatusObserved:
			icon = "●"
		case auditruntime.PhaseStatusValidated:
			icon = "✅"
		case auditruntime.PhaseStatusSuspected:
			icon = "⚡"
		case auditruntime.PhaseStatusBlockedByPolicy:
			icon = "🚫"
		case auditruntime.PhaseStatusCorrelated:
			icon = "📊"
		case auditruntime.PhaseStatusJudged:
			icon = "⚖️"
		}
		sb.WriteString(fmt.Sprintf("%s %s — %s\n", icon, phase.Phase, phase.Status))
	}

	sb.WriteString("\n")

	// Findings summary
	for _, artifact := range m.detailArtifacts {
		if len(artifact.Findings) > 0 {
			sb.WriteString(m.styles.header.Render(fmt.Sprintf("Findings from %s:", artifact.Phase)))
			sb.WriteString("\n")
			for _, f := range artifact.Findings {
				sev := m.styles.muted.Render(strings.ToUpper(f.Severity))
				switch strings.ToLower(f.Severity) {
				case "critical":
					sev = m.styles.critical.Render("CRITICAL")
				case "high":
					sev = m.styles.errStyle.Render("HIGH")
				case "medium":
					sev = m.styles.warn.Render("MEDIUM")
				}
				sb.WriteString(fmt.Sprintf("  [%s] %s (%s)\n", sev, f.Title, f.State))
			}
			sb.WriteString("\n")
		}
	}

	// Judgment result
	for _, artifact := range m.detailArtifacts {
		if artifact.Phase == auditruntime.PhaseJudgment && artifact.Judgment != nil {
			sb.WriteString(m.styles.header.Render("Judgment (Quality Validation)"))
			sb.WriteString("\n\n")
			j := artifact.Judgment
			sb.WriteString(fmt.Sprintf("Quality Score: %.0f%%\n", j.QualityScore))
			sb.WriteString(fmt.Sprintf("Ready to Report: %t\n", j.ReadyToReport))
			sb.WriteString(fmt.Sprintf("Checks: %d passed, %d failed\n\n", j.Passed, j.Failed))
			if len(j.FailedChecks) > 0 {
				sb.WriteString(m.styles.warn.Render("Failed Checks:"))
				sb.WriteString("\n")
				for _, fc := range j.FailedChecks {
					sb.WriteString(fmt.Sprintf("  ✗ %s: %s\n", fc.Check, fc.Detail))
				}
				sb.WriteString("\n")
			}
			if len(j.Recommendations) > 0 {
				sb.WriteString(m.styles.muted.Render("Recommendations:"))
				sb.WriteString("\n")
				for _, rec := range j.Recommendations {
					sb.WriteString(fmt.Sprintf("  → %s\n", rec))
				}
			}
			break
		}
	}

	sb.WriteString("\n")
	sb.WriteString(m.styles.muted.Render("[v] validate  [c] correlate  [r] refresh  esc = back"))

	return m.styles.panel.Render(sb.String())
}

// Other screens (install, sync, memory, self-update, output) — kept from original

func (m tuiModel) updateInstall(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenDashboard
	case "up", "k":
		if m.installBundleIdx > 0 {
			m.installBundleIdx--
		}
	case "down", "j":
		if m.installBundleIdx < len(m.installBundles)-1 {
			m.installBundleIdx++
		}
	case "e":
		m.installExecute = !m.installExecute
	case "enter":
		args := []string{"--bundle", m.installBundles[m.installBundleIdx]}
		if m.installExecute {
			args = append(args, "--execute")
		}
		return m, execCmd("Install bundle", func() (string, error) {
			return captureOutput(func() error { return runInstall(args) })
		})
	}
	return m, nil
}

func (m tuiModel) viewInstall() string {
	lines := []string{m.styles.title.Render("Install bundle"), ""}
	for i, item := range m.installBundles {
		prefix := "  "
		line := item
		if i == m.installBundleIdx {
			prefix = "> "
			line = m.styles.active.Render(item)
		}
		lines = append(lines, prefix+line)
	}
	lines = append(lines, "",
		fmt.Sprintf("[e] execute: %t", m.installExecute),
		m.styles.muted.Render("↑/↓ bundle   e toggle execute   Enter run   Esc back"),
	)
	return m.styles.panel.Render(strings.Join(lines, "\n"))
}

func (m tuiModel) updateSync(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenDashboard
		return m, nil
	case "enter":
		args := []string{"--all"}
		if strings.TrimSpace(m.syncAgent.Value()) != "" {
			args = []string{"--agent", strings.TrimSpace(m.syncAgent.Value())}
		}
		return m, execCmd("Sync clients", func() (string, error) {
			return captureOutput(func() error { return runSync(args) })
		})
	}
	var cmd tea.Cmd
	m.syncAgent, cmd = m.syncAgent.Update(msg)
	return m, cmd
}

func (m tuiModel) viewSync() string {
	return m.styles.panel.Render(strings.Join([]string{
		m.styles.title.Render("Sync AI clients"),
		"",
		"Agents (vacío = todos):", m.syncAgent.View(),
		"",
		m.styles.muted.Render("Enter ejecuta sync   Esc vuelve"),
	}, "\n"))
}

func (m tuiModel) updateMemory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenDashboard
		return m, nil
	case "tab":
		if m.memoryQuery.Focused() {
			m.memoryQuery.Blur()
			m.memoryDir.Focus()
		} else {
			m.memoryDir.Blur()
			m.memoryQuery.Focus()
		}
		return m, nil
	case "enter":
		args := []string{
			"--query", strings.TrimSpace(m.memoryQuery.Value()),
			"--limit", valueOrDefault(m.memoryLimit.Value(), "10"),
		}
		if strings.TrimSpace(m.memoryDir.Value()) != "" {
			args = append(args, "--dir", strings.TrimSpace(m.memoryDir.Value()))
		}
		return m, execCmd("Memory search", func() (string, error) {
			return captureOutput(func() error { return runMemorySearch(args) })
		})
	}
	var cmd tea.Cmd
	if m.memoryQuery.Focused() {
		m.memoryQuery, cmd = m.memoryQuery.Update(msg)
	} else {
		m.memoryDir, cmd = m.memoryDir.Update(msg)
	}
	return m, cmd
}

func (m tuiModel) viewMemory() string {
	return m.styles.panel.Render(strings.Join([]string{
		m.styles.title.Render("Search memory"),
		"",
		"Query:", m.memoryQuery.View(),
		"Memory dir:", m.memoryDir.View(),
		fmt.Sprintf("Limit: %s", m.memoryLimit.Value()),
		"",
		m.styles.muted.Render("Tab cambia campo   Enter busca   Esc vuelve"),
	}, "\n"))
}

func (m tuiModel) updateSelfUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenDashboard
	case "enter":
		return m, execCmd("Self update", func() (string, error) {
			return captureOutput(func() error { return runSelfUpdate([]string{"--check"}) })
		})
	}
	return m, nil
}

func (m tuiModel) viewSelfUpdate() string {
	return m.styles.panel.Render(strings.Join([]string{
		m.styles.title.Render("Self update"),
		"",
		"Enter consulta la última release disponible.",
		m.styles.muted.Render("Esc vuelve"),
	}, "\n"))
}

func (m tuiModel) viewOutput() string {
	status := m.styles.success.Render("OK")
	if m.resultErr != nil {
		status = m.styles.errStyle.Render("ERROR")
	}
	body := strings.TrimSpace(m.resultBody)
	if m.resultErr != nil {
		body = body + "\n\n" + m.resultErr.Error()
	}
	return m.styles.panel.Render(strings.Join([]string{
		m.styles.title.Render(m.resultTitle + " — " + status),
		"",
		body,
		"",
		m.styles.muted.Render("Enter/Esc vuelve al dashboard"),
	}, "\n"))
}

func (m tuiModel) View() string {
	switch m.screen {
	case screenDashboard:
		return m.viewDashboard()
	case screenNewRun:
		return m.viewWizard()
	case screenInstall:
		return m.viewInstall()
	case screenSync:
		return m.viewSync()
	case screenMemory:
		return m.viewMemory()
	case screenSelfUpdate:
		return m.viewSelfUpdate()
	case screenRunDetail:
		return m.viewRunDetail()
	case screenOutput:
		return m.viewOutput()
	default:
		return ""
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

type execResultMsg struct {
	title  string
	output string
	err    error
}

func execCmd(title string, fn func() (string, error)) tea.Cmd {
	return func() tea.Msg {
		out, err := fn()
		return execResultMsg{title: title, output: out, err: err}
	}
}

func captureOutput(fn func() error) (string, error) {
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w
	err = fn()
	_ = w.Close()
	os.Stdout = original
	var buf strings.Builder
	_, _ = io.Copy(&buf, r)
	return buf.String(), err
}
