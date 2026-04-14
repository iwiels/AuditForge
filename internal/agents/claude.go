package agents

import (
	"context"
	"os"
	"path/filepath"

	"orquestador-auditor/internal/model"
)

type ClaudeCodeAdapter struct{}

type ClaudeAdapter struct{ ClaudeCodeAdapter }

func (a *ClaudeAdapter) ID() model.AgentID { return model.AgentClaude }

func (a *ClaudeCodeAdapter) ID() model.AgentID { return model.AgentClaudeCode }

func (a *ClaudeCodeAdapter) IsInstalled(ctx context.Context, homeDir string) bool {
	_ = ctx
	_, err := os.Stat(a.ConfigDir(homeDir))
	return err == nil
}

func (a *ClaudeCodeAdapter) ConfigDir(homeDir string) string {
	return filepath.Join(homeDir, ".claude")
}

func (a *ClaudeCodeAdapter) PromptFile(homeDir string) string {
	return filepath.Join(homeDir, ".claude", "CLAUDE.md")
}

func (a *ClaudeCodeAdapter) SettingsPath(homeDir string) string {
	return filepath.Join(homeDir, ".claude", "settings.json")
}

func (a *ClaudeCodeAdapter) SupportsSystemPrompt() bool { return true }

func (a *ClaudeCodeAdapter) SystemPromptStrategy() model.SystemPromptStrategy {
	return model.StrategyMarkdownSections
}

func (a *ClaudeCodeAdapter) SupportsOutputStyles() bool { return true }

func (a *ClaudeCodeAdapter) OutputStyleDir(homeDir string) string {
	return filepath.Join(homeDir, ".claude", "output-styles")
}

func (a *ClaudeCodeAdapter) SupportsSlashCommands() bool { return true }

func (a *ClaudeCodeAdapter) CommandsDir(homeDir string) string {
	return filepath.Join(homeDir, ".claude", "commands")
}

func (a *ClaudeCodeAdapter) SupportsSkills() bool { return true }

func (a *ClaudeCodeAdapter) SkillsDir(homeDir string) string {
	return filepath.Join(homeDir, ".claude", "skills")
}

func (a *ClaudeCodeAdapter) SupportsSubAgents() bool { return false }

func (a *ClaudeCodeAdapter) SubAgentsDir(_ string) string { return "" }

func (a *ClaudeCodeAdapter) EmbeddedSubAgentsDir() string { return "" }

func (a *ClaudeCodeAdapter) ToolsFilePath(homeDir string) string {
	return filepath.Join(homeDir, ".claude", "tools-availability.md")
}

func (a *ClaudeCodeAdapter) SupportsMCP() bool { return true }

func (a *ClaudeCodeAdapter) MCPStrategy() model.MCPStrategy { return model.StrategySeparateMCPFiles }

func (a *ClaudeCodeAdapter) MCPConfigPath(homeDir string, name string) string {
	return filepath.Join(homeDir, ".claude", "mcp", name+".json")
}
