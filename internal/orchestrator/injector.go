package orchestrator

import (
	"path/filepath"
	"strings"

	"orquestador-auditor/internal/agents"
	"orquestador-auditor/internal/assets"
	componentcommands "orquestador-auditor/internal/components/commands"
	componentmcp "orquestador-auditor/internal/components/mcp"
	componentnativeagents "orquestador-auditor/internal/components/nativeagents"
	componentoutputstyles "orquestador-auditor/internal/components/outputstyles"
	componentprompts "orquestador-auditor/internal/components/prompts"
	componentskills "orquestador-auditor/internal/components/skills"
	componentsubagents "orquestador-auditor/internal/components/subagents"
	"orquestador-auditor/internal/components/tooldetection"
	"orquestador-auditor/internal/memory"
	"orquestador-auditor/internal/model"
)

type Injector struct {
	HomeDir string
	Profile model.AuditProfile
	// UseMarkers enables drift-free injection via ORQUESTADOR markers.
	// When true, repeated syncs replace content instead of appending.
	UseMarkers bool
	// MemoryStore enables Engram Protocol context injection.
	MemoryStore memory.Store
	// EngramTarget is the target for historical context lookup.
	EngramTarget string
	// EngramCampaign narrows the context lookup.
	EngramCampaign string
}

func (i *Injector) InjectMCP(adapter agents.Adapter) error {
	if !i.Profile.IncludesComponent(model.ComponentMCP) {
		return nil
	}
	return componentmcp.Inject(i.HomeDir, adapter)
}

func (i *Injector) InjectPrompts(adapter agents.Adapter) error {
	opts := componentprompts.InjectOptions{}
	if i.MemoryStore.Root != "" {
		opts.MemoryStore = &i.MemoryStore
		opts.EngramTarget = i.EngramTarget
		opts.EngramCampaign = i.EngramCampaign
	}
	return componentprompts.InjectWithOptions(i.HomeDir, adapter, i.Profile, i.UseMarkers, opts)
}

func (i *Injector) InjectCommands(adapter agents.Adapter) error {
	return componentcommands.Inject(i.HomeDir, adapter, i.Profile)
}

func (i *Injector) InjectSkills(adapter agents.Adapter) error {
	return componentskills.Inject(i.HomeDir, adapter, i.Profile)
}

func (i *Injector) InjectOutputStyles(adapter agents.Adapter) error {
	return componentoutputstyles.Inject(i.HomeDir, adapter, i.Profile)
}

func (i *Injector) InjectSubAgents(adapter agents.Adapter) error {
	return componentsubagents.Inject(i.HomeDir, adapter, i.Profile, i.UseMarkers)
}

func (i *Injector) InjectNativeAgents(adapter agents.Adapter) error {
	return componentnativeagents.Inject(i.HomeDir, adapter, i.Profile, i.UseMarkers)
}

func (i *Injector) InjectToolDetection(adapter agents.Adapter) error {
	return tooldetection.Inject(i.HomeDir, adapter)
}

func (i *Injector) InjectAll(adapter agents.Adapter) error {
	steps := []func(agents.Adapter) error{
		i.InjectMCP,
		i.InjectPrompts,
		i.InjectCommands,
		i.InjectSkills,
		i.InjectOutputStyles,
		i.InjectSubAgents,
		i.InjectNativeAgents,
		i.InjectToolDetection,
	}
	for _, step := range steps {
		if err := step(adapter); err != nil {
			return err
		}
	}
	return nil
}

func (i *Injector) ManagedPaths(adapter agents.Adapter) []string {
	paths := make([]string, 0, 16)
	if adapter.SupportsMCP() && i.Profile.IncludesComponent(model.ComponentMCP) {
		paths = append(paths, adapter.MCPConfigPath(i.HomeDir, "security-audit"))
	}
	if adapter.SupportsSystemPrompt() && i.Profile.IncludesComponent(model.ComponentPrompts) {
		paths = append(paths, adapter.PromptFile(i.HomeDir))
	}
	if adapter.SupportsSlashCommands() {
		pattern := ""
		switch adapter.ID() {
		case model.AgentOpenCode:
			pattern = "opencode/commands/*.md"
		case model.AgentClaudeCode, model.AgentClaude:
			pattern = "claude/commands/*.md"
		}
		entries, _ := assets.List(pattern)
		for _, entry := range entries {
			name := strings.TrimSuffix(filepath.Base(entry), filepath.Ext(entry))
			if !i.Profile.IncludesComponent(model.ComponentCommands) || !i.Profile.AllowsCommand(name) {
				continue
			}
			paths = append(paths, filepath.Join(adapter.CommandsDir(i.HomeDir), filepath.Base(entry)))
		}
	}
	if adapter.SupportsSkills() {
		entries, _ := assets.List("skills/*/*.md")
		for _, entry := range entries {
			skillID := model.SkillID(filepath.Base(filepath.Dir(entry)))
			if !i.Profile.IncludesComponent(model.ComponentSkills) || !i.Profile.AllowsSkill(skillID) {
				continue
			}
			paths = append(paths, filepath.Join(adapter.SkillsDir(i.HomeDir), filepath.Base(filepath.Dir(entry)), filepath.Base(entry)))
		}
	}
	if adapter.SupportsOutputStyles() && i.Profile.IncludesComponent(model.ComponentOutputStyles) {
		entries, _ := assets.List("claude/*.md")
		for _, entry := range entries {
			if filepath.Base(entry) != "output-style-security.md" {
				continue
			}
			paths = append(paths, filepath.Join(adapter.OutputStyleDir(i.HomeDir), filepath.Base(entry)))
		}
	}
	if adapter.SupportsSubAgents() && i.Profile.IncludesComponent(model.ComponentSubAgents) {
		entries, _ := assets.List(adapter.EmbeddedSubAgentsDir() + "/*.md")
		for _, entry := range entries {
			name := strings.TrimSuffix(filepath.Base(entry), filepath.Ext(entry))
			if !i.Profile.AllowsSubAgent(name) {
				continue
			}
			paths = append(paths, filepath.Join(adapter.SubAgentsDir(i.HomeDir), filepath.Base(entry)))
		}
	}
	paths = append(paths, componentnativeagents.ManagedPaths(i.HomeDir, adapter, i.Profile)...)
	return uniquePaths(paths)
}

func (i *Injector) BackupPaths(adapter agents.Adapter) []string {
	paths := make([]string, 0, 24)
	if adapter.SupportsMCP() {
		paths = append(paths, adapter.MCPConfigPath(i.HomeDir, "security-audit"))
	}
	if adapter.SupportsSystemPrompt() {
		paths = append(paths, adapter.PromptFile(i.HomeDir))
	}
	if adapter.SupportsSlashCommands() {
		pattern := ""
		switch adapter.ID() {
		case model.AgentOpenCode:
			pattern = "opencode/commands/*.md"
		case model.AgentClaudeCode, model.AgentClaude:
			pattern = "claude/commands/*.md"
		}
		entries, _ := assets.List(pattern)
		for _, entry := range entries {
			paths = append(paths, filepath.Join(adapter.CommandsDir(i.HomeDir), filepath.Base(entry)))
		}
	}
	if adapter.SupportsSkills() {
		entries, _ := assets.List("skills/*/*.md")
		for _, entry := range entries {
			paths = append(paths, filepath.Join(adapter.SkillsDir(i.HomeDir), filepath.Base(filepath.Dir(entry)), filepath.Base(entry)))
		}
	}
	if adapter.SupportsOutputStyles() {
		entries, _ := assets.List("claude/*.md")
		for _, entry := range entries {
			if filepath.Base(entry) != "output-style-security.md" {
				continue
			}
			paths = append(paths, filepath.Join(adapter.OutputStyleDir(i.HomeDir), filepath.Base(entry)))
		}
	}
	if adapter.SupportsSubAgents() {
		entries, _ := assets.List(adapter.EmbeddedSubAgentsDir() + "/*.md")
		for _, entry := range entries {
			paths = append(paths, filepath.Join(adapter.SubAgentsDir(i.HomeDir), filepath.Base(entry)))
		}
	}
	paths = append(paths, componentnativeagents.AllManagedPaths(i.HomeDir, adapter)...)
	return uniquePaths(paths)
}

// DriftReport summarizes the marker state for an adapter's managed files.
type DriftReport struct {
	Adapter     string   `json:"adapter"`
	ManagedFile string   `json:"managed_file"`
	Components  []string `json:"orquestador_components"`
	HasMarkers  bool     `json:"has_markers"`
}

// InspectDrift returns a drift report for the adapter's prompt file.
func (i *Injector) InspectDrift(adapter agents.Adapter) (*DriftReport, error) {
	if !adapter.SupportsSystemPrompt() {
		return nil, nil
	}
	path := adapter.PromptFile(i.HomeDir)
	components, err := ListMarkedComponents(path)
	if err != nil {
		return nil, err
	}
	return &DriftReport{
		Adapter:     string(adapter.ID()),
		ManagedFile: path,
		Components:  components,
		HasMarkers:  len(components) > 0,
	}, nil
}

func uniquePaths(paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}
