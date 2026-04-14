package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquestador-auditor/internal/agents"
	"orquestador-auditor/internal/catalog"
)

func TestInjectOpenCodeWritesSecurityCommands(t *testing.T) {
	home := t.TempDir()
	profile := catalog.DefaultAuditProfile()
	if err := Inject(home, &agents.OpenCodeAdapter{}, profile); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	dir := filepath.Join(home, ".config", "opencode", "commands")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected commands directory to contain files")
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", entry.Name(), err)
		}
		if !strings.Contains(string(content), "security") {
			t.Fatalf("unexpected command content in %s: %s", entry.Name(), string(content))
		}
	}
}
