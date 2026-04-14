package prompts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquestador-auditor/internal/agents"
	"orquestador-auditor/internal/catalog"
)

func TestInjectClaudeCodeAppendsMarkedSectionOnce(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("# User content\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	profile := catalog.DefaultAuditProfile()
	adapter := &agents.ClaudeCodeAdapter{}
	if err := Inject(home, adapter, profile, true); err != nil {
		t.Fatalf("Inject() first error = %v", err)
	}
	if err := Inject(home, adapter, profile, true); err != nil {
		t.Fatalf("Inject() second error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)
	if strings.Count(text, "<!-- ORQUESTADOR:prompts:start -->") != 1 {
		t.Fatalf("expected one injected marker, got: %s", text)
	}
}

func TestInjectOpenCodeWritesPromptFile(t *testing.T) {
	home := t.TempDir()
	profile := catalog.DefaultAuditProfile()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(home, ".config", "opencode"))
	t.Setenv("APPDATA", "")
	if err := Inject(home, &agents.OpenCodeAdapter{}, profile, false); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	path := filepath.Join(home, ".config", "opencode", "AGENTS.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "Security Audit") {
		t.Fatalf("unexpected prompt content (missing 'Security Audit'): %.200s", string(content))
	}
	if !strings.Contains(text, "chrome-devtools") {
		t.Fatalf("unexpected prompt content (missing chrome-devtools guidance): %.200s", text)
	}
}
