package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"orquestador-auditor/internal/agents"
	"orquestador-auditor/internal/assets"
	"orquestador-auditor/internal/components/filemerge"
	"orquestador-auditor/internal/markers"
	"orquestador-auditor/internal/memory"
	"orquestador-auditor/internal/model"
)

// InjectOptions controls optional behaviors during prompt injection.
type InjectOptions struct {
	// MemoryStore enables Engram Protocol context injection when set.
	MemoryStore *memory.Store
	// EngramTarget is the target to search for in memory.
	EngramTarget string
	// EngramCampaign narrows the memory search.
	EngramCampaign string
}

func Inject(homeDir string, adapter agents.Adapter, profile model.AuditProfile, useMarkers bool) error {
	return InjectWithOptions(homeDir, adapter, profile, useMarkers, InjectOptions{})
}

// InjectWithOptions performs prompt injection with optional Engram context.
func InjectWithOptions(homeDir string, adapter agents.Adapter, profile model.AuditProfile, useMarkers bool, opts InjectOptions) error {
	if !adapter.SupportsSystemPrompt() || !profile.IncludesComponent(model.ComponentPrompts) {
		return nil
	}

	path := adapter.PromptFile(homeDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	content := buildPrompt(profile)

	// Inject Engram context if memory store is available
	if opts.MemoryStore != nil && strings.TrimSpace(opts.EngramTarget) != "" {
		engramPreamble := memory.EngramPreamble(opts.MemoryStore, opts.EngramTarget, opts.EngramCampaign)
		if engramPreamble != "" {
			content += engramPreamble
		}
	}

	if useMarkers {
		return injectWithMarkers(path, content)
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch adapter.SystemPromptStrategy() {
	case model.StrategyMarkdownSections:
		return filemerge.InjectMarkdownSection(path, "security-audit", content)
	case model.StrategyFileReplace:
		return filemerge.WriteIfChanged(path, []byte(content+"\n"))
	case model.StrategyAppendToFile, model.StrategyInstructionsFile:
		if ext == ".md" || ext == ".mdc" {
			return filemerge.InjectMarkdownSection(path, "security-audit", content)
		}
		return filemerge.WriteIfChanged(path, []byte(content+"\n"))
	default:
		return nil
	}
}

// injectWithMarkers uses ORQUESTADOR markers for drift-free prompt injection.
// Replaces existing block if present, appends otherwise.
func injectWithMarkers(path string, content string) error {
	marker := markers.ContentMarker{
		Component: "prompts",
		Content:   []byte("\n## Active Audit Profile (injected by orquestador-auditor)\n\n" + content),
	}
	return markers.InjectWithMarkers(path, marker)
}

func buildPrompt(profile model.AuditProfile) string {
	base := strings.TrimSpace(assets.MustRead("prompts/security-audit.md"))
	focus := bulletList(profile.FocusAreas)
	commands := make([]string, 0, len(profile.Commands))
	for _, item := range profile.Commands {
		commands = append(commands, fmt.Sprintf("`%s`", item))
	}
	allowed := bulletList(profile.Risk.AllowedActions)
	blocked := bulletList(profile.Risk.BlockedActions)
	perms := profile.Risk.Permissions

	return base + "\n\n## Active Audit Profile\n\n" +
		fmt.Sprintf("- **Profile:** %s (`%s`)\n", profile.Name, profile.ID) +
		fmt.Sprintf("- **Purpose:** %s\n", profile.Description) +
		fmt.Sprintf("- **Execution mode:** %s\n", profile.Risk.Mode) +
		fmt.Sprintf("- **Risk policy:** %s\n", profile.Risk.Summary) +
		fmt.Sprintf("- **Enabled OpenCode commands/assets:** %s\n", strings.Join(commands, ", ")) +
		fmt.Sprintf("- **OpenCode tool permissions:** read=%t, write=%t, edit=%t, bash=%t\n", perms.Read, perms.Write, perms.Edit, perms.Bash) +
		"- **Focus areas:**\n" + focus + "\n" +
		"- **Allowed actions:**\n" + allowed + "\n" +
		"- **Blocked actions:**\n" + blocked
}

func bulletList(items []string) string {
	if len(items) == 0 {
		return "- none"
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, "  - "+item)
	}
	return strings.Join(lines, "\n")
}
