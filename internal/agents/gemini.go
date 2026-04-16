package agents

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"orquestador-auditor/internal/model"
)

// GeminiAdapter implementa el Adapter para Gemini CLI.
// Gemini CLI usa:
//   - ~/.gemini/GEMINI.md         → system prompt (context file)
//   - ~/.gemini/settings.json     → MCP servers (clave mcpServers)
//   - ~/.gemini/commands/*.toml   → custom slash commands
//   - ~/.gemini/agents/           → subagentes (feature experimental)
//
// No soporta skills como directorio estructurado — se inyectan
// como secciones del GEMINI.md.
type GeminiAdapter struct{}

func (a *GeminiAdapter) ID() model.AgentID { return model.AgentGemini }

func (a *GeminiAdapter) IsInstalled(ctx context.Context, homeDir string) bool {
	_ = ctx
	// gemini CLI se instala via npm como @google/gemini-cli
	if _, err := exec.LookPath("gemini"); err == nil {
		return true
	}
	// También puede estar en node_modules/.bin
	if _, err := exec.LookPath("npx"); err == nil {
		// Verificamos si el directorio de config ya existe (indica uso previo)
		if _, err := os.Stat(a.ConfigDir(homeDir)); err == nil {
			return true
		}
	}
	return false
}

func (a *GeminiAdapter) ConfigDir(homeDir string) string {
	return filepath.Join(homeDir, ".gemini")
}

func (a *GeminiAdapter) PromptFile(homeDir string) string {
	// Gemini CLI carga GEMINI.md como context file (system prompt)
	return filepath.Join(homeDir, ".gemini", "GEMINI.md")
}

func (a *GeminiAdapter) SettingsPath(homeDir string) string {
	return filepath.Join(homeDir, ".gemini", "settings.json")
}

func (a *GeminiAdapter) SupportsSystemPrompt() bool { return true }

func (a *GeminiAdapter) SystemPromptStrategy() model.SystemPromptStrategy {
	// GEMINI.md soporta secciones con separadores → usamos MarkdownSections
	return model.StrategyMarkdownSections
}

func (a *GeminiAdapter) SupportsOutputStyles() bool { return false }

func (a *GeminiAdapter) OutputStyleDir(_ string) string { return "" }

func (a *GeminiAdapter) SupportsSlashCommands() bool { return true }

func (a *GeminiAdapter) CommandsDir(homeDir string) string {
	// Gemini CLI espera archivos .toml en ~/.gemini/commands/
	return filepath.Join(homeDir, ".gemini", "commands")
}

func (a *GeminiAdapter) SupportsSkills() bool {
	// Gemini CLI no tiene un directorio de skills estructurado.
	// Las skills se inyectan como secciones del GEMINI.md.
	return false
}

func (a *GeminiAdapter) SkillsDir(_ string) string { return "" }

func (a *GeminiAdapter) SupportsMCP() bool { return true }

func (a *GeminiAdapter) MCPStrategy() model.MCPStrategy {
	// MCP se configura en settings.json bajo la clave "mcpServers"
	return model.StrategyMCPConfigFile
}

func (a *GeminiAdapter) MCPConfigPath(homeDir string, _ string) string {
	return filepath.Join(homeDir, ".gemini", "settings.json")
}

func (a *GeminiAdapter) SupportsSubAgents() bool { return true }

func (a *GeminiAdapter) SubAgentsDir(homeDir string) string {
	return filepath.Join(homeDir, ".gemini", "agents")
}

func (a *GeminiAdapter) EmbeddedSubAgentsDir() string {
	return "gemini/agents"
}

func (a *GeminiAdapter) ToolsFilePath(homeDir string) string {
	return filepath.Join(homeDir, ".gemini", "tools-availability.md")
}
