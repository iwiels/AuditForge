package backup

import "time"

type Manifest struct {
	ID        string          `json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	RootDir   string          `json:"root_dir"`
	Entries   []ManifestEntry `json:"entries"`
}

type ManifestEntry struct {
	OriginalPath string `json:"original_path"`
	BackupPath   string `json:"backup_path,omitempty"`
	Checksum     string `json:"checksum,omitempty"`
	Existed      bool   `json:"existed"`
}
