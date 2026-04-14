package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const ManifestFilename = "manifest.json"

type Snapshotter struct {
	now func() time.Time
}

func NewSnapshotter() Snapshotter {
	return Snapshotter{now: time.Now}
}

func (s Snapshotter) Create(snapshotDir string, paths []string) (Manifest, error) {
	if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
		return Manifest{}, err
	}
	manifest := Manifest{
		ID:        filepath.Base(snapshotDir),
		CreatedAt: s.now().UTC(),
		RootDir:   snapshotDir,
		Entries:   make([]ManifestEntry, 0, len(paths)),
	}
	for _, source := range paths {
		entry, err := snapshotFile(snapshotDir, source)
		if err != nil {
			return Manifest{}, err
		}
		manifest.Entries = append(manifest.Entries, entry)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Manifest{}, err
	}
	if err := os.WriteFile(filepath.Join(snapshotDir, ManifestFilename), append(data, '\n'), 0o644); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func snapshotFile(snapshotDir, source string) (ManifestEntry, error) {
	cleanSource := filepath.Clean(source)
	entry := ManifestEntry{OriginalPath: cleanSource}
	raw, err := os.ReadFile(cleanSource)
	if err != nil {
		if os.IsNotExist(err) {
			return entry, nil
		}
		return ManifestEntry{}, err
	}
	rel := strings.TrimPrefix(cleanSource, filepath.VolumeName(cleanSource))
	rel = strings.TrimPrefix(rel, string(filepath.Separator))
	backupPath := filepath.Join(snapshotDir, "files", rel)
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return ManifestEntry{}, err
	}
	if err := os.WriteFile(backupPath, raw, 0o644); err != nil {
		return ManifestEntry{}, err
	}
	sum := sha256.Sum256(raw)
	entry.BackupPath = backupPath
	entry.Checksum = hex.EncodeToString(sum[:])
	entry.Existed = true
	return entry, nil
}

func DefaultBackupDir(homeDir string) string {
	ts := time.Now().UTC().Format("20060102-150405")
	return filepath.Join(homeDir, ".orquestador", "backups", fmt.Sprintf("sync-%s", ts))
}
