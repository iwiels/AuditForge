package nativeagents

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"orquestador-auditor/internal/model"
)

type fakeAdapter struct {
	root string
}

func (f fakeAdapter) ID() model.AgentID { return model.AgentOpenCode }

func (f fakeAdapter) IsInstalled(context.Context, string) bool { return true }

func (f fakeAdapter) ConfigDir(string) string { return f.root }

func (f fakeAdapter) PromptFile(string) string { return filepath.Join(f.root, "AGENTS.md") }

func (f fakeAdapter) SettingsPath(string) string { return filepath.Join(f.root, "opencode.json") }

func (f fakeAdapter) SupportsSystemPrompt() bool { return true }

func (f fakeAdapter) SystemPromptStrategy() model.SystemPromptStrategy {
	return model.StrategyFileReplace
}

func (f fakeAdapter) SupportsOutputStyles() bool { return false }

func (f fakeAdapter) OutputStyleDir(string) string { return "" }

func (f fakeAdapter) SupportsSlashCommands() bool { return true }

func (f fakeAdapter) CommandsDir(string) string { return filepath.Join(f.root, "commands") }

func (f fakeAdapter) SupportsSkills() bool { return true }

func (f fakeAdapter) SkillsDir(string) string { return filepath.Join(f.root, "skills") }

func (f fakeAdapter) SupportsMCP() bool { return true }

func (f fakeAdapter) MCPStrategy() model.MCPStrategy { return model.StrategyMergeIntoSettings }

func (f fakeAdapter) MCPConfigPath(string, string) string {
	return filepath.Join(f.root, "opencode.json")
}

func (f fakeAdapter) SupportsSubAgents() bool { return false }

func (f fakeAdapter) SubAgentsDir(string) string { return "" }

func (f fakeAdapter) EmbeddedSubAgentsDir() string { return "" }

func (f fakeAdapter) ToolsFilePath(string) string {
	return filepath.Join(f.root, "tools-availability.md")
}

func (f fakeAdapter) SupportsAgentOverlay() bool { return false }

func (f fakeAdapter) AgentOverlayPath(string) string { return "" }

func (f fakeAdapter) EmbeddedAgentOverlayAsset() string { return "" }

func (f fakeAdapter) SupportsAgentPlugins() bool { return true }

func (f fakeAdapter) PluginsDir(string) string { return filepath.Join(f.root, "plugins") }

func (f fakeAdapter) EmbeddedPluginsDir() string { return "opencode/plugins" }

func (f fakeAdapter) SupportsNativeAgentFiles() bool { return true }

func (f fakeAdapter) NativeAgentsDir(string) string { return filepath.Join(f.root, "agents") }

func (f fakeAdapter) EmbeddedNativeAgentsDir() string { return "opencode/agents" }

func TestInjectFileAgentsWritesFrontmatterAtStart(t *testing.T) {
	homeDir := t.TempDir()
	adapter := fakeAdapter{root: homeDir}
	profile := model.AuditProfile{
		Components:   []model.ComponentID{model.ComponentSubAgents},
		NativeAgents: []string{"security-orchestrator"},
	}

	if err := injectFileAgents(homeDir, adapter, profile, true); err != nil {
		t.Fatalf("injectFileAgents() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(homeDir, "agents", "security-orchestrator.md"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if bytes.HasPrefix(content, utf8BOM) {
		t.Fatalf("expected BOM to be stripped, got %q", content[:3])
	}
	if !bytes.HasPrefix(content, []byte("---\n")) {
		t.Fatalf("expected agent file to start with frontmatter, got %q", content[:min(32, len(content))])
	}
	if bytes.Contains(content, []byte("<!-- ORQUESTADOR:")) {
		t.Fatalf("expected no HTML markers in agent file, got %q", content)
	}
}

func TestInjectPluginsWritesValidTypeScript(t *testing.T) {
	homeDir := t.TempDir()
	adapter := fakeAdapter{root: homeDir}
	profile := model.AuditProfile{
		Components: []model.ComponentID{model.ComponentSubAgents},
	}

	if err := injectPlugins(homeDir, adapter, profile, true); err != nil {
		t.Fatalf("injectPlugins() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(homeDir, "plugins", "background-agents.ts"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if bytes.Contains(content, []byte("<!-- ORQUESTADOR:")) {
		t.Fatalf("expected no HTML markers in plugin file, got %q", content)
	}
	if !bytes.HasPrefix(content, []byte("import ")) {
		t.Fatalf("expected plugin file to start with TypeScript import, got %q", content[:min(32, len(content))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
