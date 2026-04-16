package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func LoadManifest(snapshotDir string) (Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(snapshotDir, ManifestFilename))
	if err != nil {
		return Manifest{}, err
	}
	manifest := Manifest{}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func Restore(manifest Manifest) error {
	for _, entry := range manifest.Entries {
		if !entry.Existed || entry.BackupPath == "" {
			continue
		}
		raw, err := os.ReadFile(entry.BackupPath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(entry.OriginalPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(entry.OriginalPath, raw, 0o644); err != nil {
			return err
		}
	}
	return nil
}
