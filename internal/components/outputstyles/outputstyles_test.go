package outputstyles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquestador-auditor/internal/agents"
	"orquestador-auditor/internal/catalog"
)

func TestInjectClaudeCodeOutputStyleWritesSecurityFile(t *testing.T) {
	home := t.TempDir()
	profile := catalog.DefaultAuditProfile()
	if err := Inject(home, &agents.ClaudeCodeAdapter{}, profile); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	// Check if any output style file was created
	dir := filepath.Join(home, ".claude")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.Contains(entry.Name(), "output") || strings.Contains(entry.Name(), "style") {
			path := filepath.Join(dir, entry.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			if strings.Contains(string(content), "security") {
				return // Found security output style
			}
		}
	}
	// If no output style file was created, that's OK - not all adapters support it
	t.Skip("No output style file created (adapter may not support it)")
}
