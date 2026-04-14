package agents

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverInstalledReturnsKnownConfigRoots(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	reg, err := NewDefaultRegistry()
	if err != nil {
		t.Fatalf("NewDefaultRegistry() error = %v", err)
	}
	installed := DiscoverInstalled(context.Background(), reg, home)
	if len(installed) == 0 {
		t.Fatalf("expected installed agents, got none")
	}
}
