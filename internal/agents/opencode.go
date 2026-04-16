package agents

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"orquestador-auditor/internal/model"
)

type OpenCodeAdapter struct{}

func (a *OpenCodeAdapter) ID() model.AgentID { return model.AgentOpenCode }

func (a *OpenCodeAdapter) IsInstalled(ctx context.Context, homeDir string) bool {
	_ = ctx
	if _, err := exec.LookPath("opencode"); err == nil {
		return true
	}
	for _, path := range a.configDirCandidates(homeDir) {
		if dirExists(path) {
			return true
		}
	}
	for _, path := range a.desktopBinaryCandidates() {
		if fileExists(path) {
			return true
		}
	}
	return false
}

func (a *OpenCodeAdapter) ConfigDir(homeDir string) string {
	for _, path := range a.configDirCandidates(homeDir) {
		if dirExists(path) {
			return path
		}
	}
	candidates := a.configDirCandidates(homeDir)
	if len(candidates) == 0 {
		return filepath.Join(homeDir, ".config", "opencode")
	}
	return candidates[0]
}

func (a *OpenCodeAdapter) PromptFile(homeDir string) string {
	return filepath.Join(a.ConfigDir(homeDir), "AGENTS.md")
}

func (a *OpenCodeAdapter) SettingsPath(homeDir string) string {
	return filepath.Join(a.ConfigDir(homeDir), "opencode.json")
}

func (a *OpenCodeAdapter) SupportsSystemPrompt() bool { return true }

func (a *OpenCodeAdapter) SystemPromptStrategy() model.SystemPromptStrategy {
	return model.StrategyFileReplace
}

func (a *OpenCodeAdapter) SupportsOutputStyles() bool { return false }

func (a *OpenCodeAdapter) OutputStyleDir(_ string) string { return "" }

func (a *OpenCodeAdapter) SupportsSlashCommands() bool { return true }

func (a *OpenCodeAdapter) CommandsDir(homeDir string) string {
	return filepath.Join(a.ConfigDir(homeDir), "commands")
}

func (a *OpenCodeAdapter) SupportsSkills() bool { return true }

func (a *OpenCodeAdapter) SkillsDir(homeDir string) string {
	return filepath.Join(a.ConfigDir(homeDir), "skills")
}

func (a *OpenCodeAdapter) SupportsMCP() bool { return true }

func (a *OpenCodeAdapter) MCPStrategy() model.MCPStrategy { return model.StrategyMergeIntoSettings }

func (a *OpenCodeAdapter) MCPConfigPath(homeDir string, _ string) string {
	return a.SettingsPath(homeDir)
}

func (a *OpenCodeAdapter) SupportsSubAgents() bool { return false }

func (a *OpenCodeAdapter) SubAgentsDir(homeDir string) string {
	return ""
}

func (a *OpenCodeAdapter) EmbeddedSubAgentsDir() string { return "" }

func (a *OpenCodeAdapter) ToolsFilePath(homeDir string) string {
	return filepath.Join(a.ConfigDir(homeDir), "tools-availability.md")
}

func (a *OpenCodeAdapter) SupportsAgentOverlay() bool { return false }

func (a *OpenCodeAdapter) AgentOverlayPath(homeDir string) string {
	return a.SettingsPath(homeDir)
}

func (a *OpenCodeAdapter) EmbeddedAgentOverlayAsset() string {
	return "opencode/security-overlay.json"
}

func (a *OpenCodeAdapter) SupportsAgentPlugins() bool { return true }

func (a *OpenCodeAdapter) PluginsDir(homeDir string) string {
	return filepath.Join(a.ConfigDir(homeDir), "plugins")
}

func (a *OpenCodeAdapter) EmbeddedPluginsDir() string {
	return "opencode/plugins"
}

func (a *OpenCodeAdapter) SupportsNativeAgentFiles() bool { return true }

func (a *OpenCodeAdapter) NativeAgentsDir(homeDir string) string {
	return filepath.Join(a.ConfigDir(homeDir), "agents")
}

func (a *OpenCodeAdapter) EmbeddedNativeAgentsDir() string {
	return "opencode/agents"
}

func (a *OpenCodeAdapter) configDirCandidates(homeDir string) []string {
	if override := strings.TrimSpace(os.Getenv("OPENCODE_CONFIG_DIR")); override != "" {
		return []string{override}
	}

	candidates := []string{
		filepath.Join(homeDir, ".config", "opencode"),
	}
	if appData := strings.TrimSpace(os.Getenv("APPDATA")); appData != "" {
		// Legacy Windows location from older syncs.
		candidates = append(candidates, filepath.Join(appData, "opencode"))
	}
	return uniqueConfigPaths(candidates)
}

func (a *OpenCodeAdapter) desktopBinaryCandidates() []string {
	localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
	if localAppData == "" {
		return nil
	}
	return []string{
		filepath.Join(localAppData, "OpenCode", "opencode-cli.exe"),
		filepath.Join(localAppData, "OpenCode", "OpenCode.exe"),
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func uniqueConfigPaths(paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
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
