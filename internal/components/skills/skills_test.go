package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquestador-auditor/internal/agents"
	"orquestador-auditor/internal/catalog"
)

func TestInjectClaudeCodeSkillsWritesSecuritySkills(t *testing.T) {
	home := t.TempDir()
	profile := catalog.DefaultAuditProfile()
	if err := Inject(home, &agents.ClaudeCodeAdapter{}, profile); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	dir := filepath.Join(home, ".claude", "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected skills directory to contain files")
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name(), "SKILL.md")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s/SKILL.md) error = %v", entry.Name(), err)
		}
		if !strings.Contains(string(content), "Skill:") {
			t.Fatalf("unexpected skill content: %s", string(content))
		}
	}
}
