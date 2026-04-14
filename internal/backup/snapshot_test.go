package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAndRestoreSnapshot(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "config.json")
	if err := os.WriteFile(target, []byte("before"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manifest, err := NewSnapshotter().Create(filepath.Join(root, "backup"), []string{target})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := os.WriteFile(target, []byte("after"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := Restore(manifest); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(raw) != "before" {
		t.Fatalf("unexpected restored content: %s", string(raw))
	}
}
