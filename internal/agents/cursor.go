package agents

import (
	"context"
	"os"
	"path/filepath"

	"orquestador-auditor/internal/model"
)

type CursorAdapter struct{}

func (a *CursorAdapter) ID() model.AgentID { return model.AgentCursor }

func (a *CursorAdapter) IsInstalled(ctx context.Context, homeDir string) bool {
	_ = ctx
	_, err := os.Stat(a.ConfigDir(homeDir))
	return err == nil
}

func (a *CursorAdapter) ConfigDir(homeDir string) string {
	return filepath.Join(homeDir, ".cursor")
}

func (a *CursorAdapter) PromptFile(homeDir string) string {
	return filepath.Join(homeDir, ".cursor", "rules", "gentle-ai.mdc")
}

func (a *CursorAdapter) SettingsPath(homeDir string) string {
	return filepath.Join(homeDir, ".cursor", "settings.json")
}

func (a *CursorAdapter) SupportsSystemPrompt() bool { return true }

func (a *CursorAdapter) SystemPromptStrategy() model.SystemPromptStrategy {
	return model.StrategyFileReplace
}

func (a *CursorAdapter) SupportsOutputStyles() bool { return false }

func (a *CursorAdapter) OutputStyleDir(_ string) string { return "" }

func (a *CursorAdapter) SupportsSlashCommands() bool { return false }

func (a *CursorAdapter) CommandsDir(_ string) string { return "" }

func (a *CursorAdapter) SupportsSkills() bool { return true }

func (a *CursorAdapter) SkillsDir(homeDir string) string {
	return filepath.Join(homeDir, ".cursor", "skills")
}

func (a *CursorAdapter) SupportsMCP() bool { return true }

func (a *CursorAdapter) MCPStrategy() model.MCPStrategy { return model.StrategyMCPConfigFile }

func (a *CursorAdapter) MCPConfigPath(homeDir string, _ string) string {
	return filepath.Join(homeDir, ".cursor", "mcp.json")
}

func (a *CursorAdapter) SupportsSubAgents() bool { return true }

func (a *CursorAdapter) SubAgentsDir(homeDir string) string {
	return filepath.Join(homeDir, ".cursor", "agents")
}

func (a *CursorAdapter) EmbeddedSubAgentsDir() string { return "cursor/agents" }

func (a *CursorAdapter) ToolsFilePath(homeDir string) string {
	return filepath.Join(homeDir, ".cursor", "tools-availability.md")
}
