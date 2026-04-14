package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"orquestador-auditor/internal/agents"
	"orquestador-auditor/internal/assets"
	"orquestador-auditor/internal/components/filemerge"
	"orquestador-auditor/internal/model"
)

func Inject(homeDir string, adapter agents.Adapter, profile model.AuditProfile) error {
	if !adapter.SupportsSlashCommands() {
		return nil
	}

	dir := adapter.CommandsDir(homeDir)
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	pattern, err := commandPattern(adapter)
	if err != nil {
		return err
	}
	entries, err := assets.List(pattern)
	if err != nil {
		return err
	}

	keep := map[string]struct{}{}
	for _, entry := range entries {
		base := filepath.Base(entry)
		ext := filepath.Ext(base)
		name := strings.TrimSuffix(base, ext)
		targetPath := filepath.Join(dir, base)

		if profile.IncludesComponent(model.ComponentCommands) && profile.AllowsCommand(name) {
			content, err := assets.Read(entry)
			if err != nil {
				return err
			}
			if err := filemerge.WriteIfChanged(targetPath, []byte(content)); err != nil {
				return err
			}
			keep[targetPath] = struct{}{}
		}
	}

	// Limpiar comandos que ya no están en el perfil activo
	for _, entry := range entries {
		targetPath := filepath.Join(dir, filepath.Base(entry))
		if _, ok := keep[targetPath]; ok {
			continue
		}
		if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func commandPattern(adapter agents.Adapter) (string, error) {
	switch adapter.ID() {
	case model.AgentOpenCode:
		return "opencode/commands/*.md", nil
	case model.AgentClaudeCode, model.AgentClaude:
		return "claude/commands/*.md", nil
	case model.AgentGemini:
		// Gemini CLI usa archivos .toml para los comandos
		return "gemini/commands/*.toml", nil
	default:
		return "", fmt.Errorf("commands are not defined for agent %q", adapter.ID())
	}
}
