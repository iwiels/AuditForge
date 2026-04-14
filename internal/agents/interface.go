package agents

import (
	"context"

	"orquestador-auditor/internal/model"
)

// Adapter defines the contract for any AI agent integration.
type Adapter interface {
	ID() model.AgentID
	IsInstalled(ctx context.Context, homeDir string) bool
	ConfigDir(homeDir string) string
	PromptFile(homeDir string) string
	SettingsPath(homeDir string) string
	SupportsSystemPrompt() bool
	SystemPromptStrategy() model.SystemPromptStrategy
	SupportsOutputStyles() bool
	OutputStyleDir(homeDir string) string
	SupportsSlashCommands() bool
	CommandsDir(homeDir string) string
	SupportsSkills() bool
	SkillsDir(homeDir string) string
	SupportsMCP() bool
	MCPStrategy() model.MCPStrategy
	MCPConfigPath(homeDir string, name string) string
	SupportsSubAgents() bool
	SubAgentsDir(homeDir string) string
	EmbeddedSubAgentsDir() string
	ToolsFilePath(homeDir string) string
}
