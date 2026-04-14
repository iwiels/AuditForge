package agents

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCodeAdapterPrefersCanonicalConfigDir(t *testing.T) {
	home := t.TempDir()
	canonical := filepath.Join(home, ".config", "opencode")
	legacyRoot := filepath.Join(home, "AppData", "Roaming")
	legacy := filepath.Join(legacyRoot, "opencode")

	if err := os.MkdirAll(canonical, 0o755); err != nil {
		t.Fatalf("MkdirAll(canonical) error = %v", err)
	}
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatalf("MkdirAll(legacy) error = %v", err)
	}

	t.Setenv("APPDATA", legacyRoot)
	t.Setenv("OPENCODE_CONFIG_DIR", "")

	adapter := OpenCodeAdapter{}
	if got := adapter.ConfigDir(home); got != canonical {
		t.Fatalf("ConfigDir() = %q, want %q", got, canonical)
	}
}

func TestOpenCodeAdapterUsesOverrideConfigDir(t *testing.T) {
	home := t.TempDir()
	override := filepath.Join(home, "custom-opencode")

	t.Setenv("OPENCODE_CONFIG_DIR", override)

	adapter := OpenCodeAdapter{}
	if got := adapter.ConfigDir(home); got != override {
		t.Fatalf("ConfigDir() = %q, want %q", got, override)
	}
}

func TestOpenCodeAdapterDetectsDesktopBinaryWithoutGlobalCLI(t *testing.T) {
	home := t.TempDir()
	localAppData := filepath.Join(home, "LocalAppData")
	desktopDir := filepath.Join(localAppData, "OpenCode")
	if err := os.MkdirAll(desktopDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	desktopCLI := filepath.Join(desktopDir, "opencode-cli.exe")
	if err := os.WriteFile(desktopCLI, []byte("stub"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("LOCALAPPDATA", localAppData)
	t.Setenv("APPDATA", filepath.Join(home, "Roaming"))
	t.Setenv("OPENCODE_CONFIG_DIR", "")

	adapter := OpenCodeAdapter{}
	if !adapter.IsInstalled(context.Background(), home) {
		t.Fatal("IsInstalled() = false, want true when desktop binary exists")
	}
}
