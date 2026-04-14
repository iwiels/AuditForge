package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquestador-auditor/internal/agents"
)

func TestInjectClaudeCodeWritesSeparateFile(t *testing.T) {
	home := t.TempDir()
	if err := Inject(home, &agents.ClaudeCodeAdapter{}); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	path := filepath.Join(home, ".claude", "mcp", "security-audit.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)
	if !strings.Contains(text, `"command": "`) || !strings.Contains(text, `"args": [`) || !strings.Contains(text, `"--mcp"`) {
		t.Fatalf("missing command in %s", text)
	}
}

func TestInjectOpenCodeWritesSettingsJSON(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("{\n  \"theme\": \"night\"\n}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("PATH", os.Getenv("PATH"))
	if err := Inject(home, &agents.OpenCodeAdapter{}); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)
	if !strings.Contains(text, `"mcp"`) || !strings.Contains(text, `"type": "local"`) || !strings.Contains(text, `"theme": "night"`) {
		t.Fatalf("unexpected OpenCode MCP config: %s", text)
	}
	if !strings.Contains(text, `"security-audit"`) {
		t.Fatalf("missing security-audit MCP in %s", text)
	}
	if !strings.Contains(text, `"chrome-devtools"`) {
		t.Fatalf("missing chrome-devtools MCP in %s", text)
	}
}

func TestInjectCursorWritesMCPConfigFile(t *testing.T) {
	home := t.TempDir()
	if err := Inject(home, &agents.CursorAdapter{}); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	path := filepath.Join(home, ".cursor", "mcp.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)
	if !strings.Contains(text, `"mcpServers"`) || !strings.Contains(text, `"security-audit"`) {
		t.Fatalf("unexpected Cursor MCP config: %s", text)
	}
}
