package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquestador-auditor/internal/model"
)

func TestHandleRequestToolsCallAuditScout(t *testing.T) {
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	params, _ := json.Marshal(ToolCallParams{
		Name: "audit.scout",
		Arguments: map[string]interface{}{
			"target": target,
		},
	})

	response := handleRequest(JSONRPCRequest{JSONRPC: "2.0", ID: 1, Method: "tools/call", Params: params})
	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected result type: %T", response.Result)
	}
	content := result["content"].([]map[string]interface{})
	text := content[0]["text"].(string)
	if !strings.Contains(text, `"target_kind": "path"`) {
		t.Fatalf("unexpected scout output: %s", text)
	}
}

func TestAuditReportRendersMarkdown(t *testing.T) {
	state := model.NewAuditState("./repo", ".orquestador", model.TargetKindPath)
	state.AddNote("note")
	path := filepath.Join(t.TempDir(), "state.json")
	raw, _ := json.Marshal(state)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	text, err := auditReport(map[string]interface{}{"stateFile": path})
	if err != nil {
		t.Fatalf("auditReport() error = %v", err)
	}
	if !strings.Contains(text, "# Security Audit Report") {
		t.Fatalf("unexpected markdown: %s", text)
	}
}
