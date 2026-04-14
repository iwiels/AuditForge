package skills

import (
	"os"
	"path/filepath"

	"orquestador-auditor/internal/agents"
	"orquestador-auditor/internal/assets"
	"orquestador-auditor/internal/components/filemerge"
	"orquestador-auditor/internal/model"
)

func Inject(homeDir string, adapter agents.Adapter, profile model.AuditProfile) error {
	if !adapter.SupportsSkills() {
		return nil
	}
	dir := adapter.SkillsDir(homeDir)
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	entries, err := assets.List("skills/*/*.md")
	if err != nil {
		return err
	}
	keepDirs := map[string]struct{}{}
	for _, entry := range entries {
		skillDirName := filepath.Base(filepath.Dir(entry))
		skillID := model.SkillID(skillDirName)
		targetDir := filepath.Join(dir, skillDirName)
		targetPath := filepath.Join(targetDir, filepath.Base(entry))
		if profile.IncludesComponent(model.ComponentSkills) && profile.AllowsSkill(skillID) {
			content, err := assets.Read(entry)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return err
			}
			if err := filemerge.WriteIfChanged(targetPath, []byte(content)); err != nil {
				return err
			}
			keepDirs[targetDir] = struct{}{}
		}
	}

	seenDirs := map[string]struct{}{}
	for _, entry := range entries {
		targetDir := filepath.Join(dir, filepath.Base(filepath.Dir(entry)))
		if _, seen := seenDirs[targetDir]; seen {
			continue
		}
		seenDirs[targetDir] = struct{}{}
		if _, ok := keepDirs[targetDir]; ok {
			continue
		}
		if err := os.RemoveAll(targetDir); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
