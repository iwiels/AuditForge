package outputstyles

import (
	"os"
	"path/filepath"

	"orquestador-auditor/internal/agents"
	"orquestador-auditor/internal/assets"
	"orquestador-auditor/internal/components/filemerge"
	"orquestador-auditor/internal/model"
)

func Inject(homeDir string, adapter agents.Adapter, profile model.AuditProfile) error {
	if !adapter.SupportsOutputStyles() {
		return nil
	}
	dir := adapter.OutputStyleDir(homeDir)
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	entries, err := assets.List("claude/*.md")
	if err != nil {
		return err
	}
	keep := map[string]struct{}{}
	for _, entry := range entries {
		if filepath.Base(entry) != "output-style-security.md" {
			continue
		}
		targetPath := filepath.Join(dir, filepath.Base(entry))
		if profile.IncludesComponent(model.ComponentOutputStyles) {
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
	for _, entry := range entries {
		if filepath.Base(entry) != "output-style-security.md" {
			continue
		}
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
