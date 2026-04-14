package filemerge

import (
	"strings"
	"testing"
)

func TestMergeJSONObjectsPreservesUserKeysAndReplacesTargetBranch(t *testing.T) {
	base := []byte("{\n  \"theme\": \"night\",\n  \"mcp\": {\n    \"other\": {\"type\": \"remote\"},\n    \"security-audit\": {\"type\": \"remote\", \"serverUrl\": \"https://old\"}\n  }\n}\n")
	overlay := []byte("{\n  \"mcp\": {\n    \"security-audit\": {\n      \"__replace__\": {\n        \"type\": \"local\",\n        \"command\": [\"orquestador-auditor\", \"--mcp\"]\n      }\n    }\n  }\n}\n")

	merged, err := MergeJSONObjects(base, overlay)
	if err != nil {
		t.Fatalf("MergeJSONObjects() error = %v", err)
	}
	text := string(merged)
	if !strings.Contains(text, `"theme": "night"`) || !strings.Contains(text, `"other"`) || !strings.Contains(text, `"type": "local"`) {
		t.Fatalf("unexpected merged JSON: %s", text)
	}
	if strings.Contains(text, `"serverUrl": "https://old"`) {
		t.Fatalf("old branch was not replaced: %s", text)
	}
}

func TestMergeJSONObjectsStripsHTMLComments(t *testing.T) {
	base := []byte("<!-- ORQUESTADOR:start -->\n{\n  \"mcp\": {}\n}\n<!-- ORQUESTADOR:end -->\n")
	overlay := []byte("{\n  \"mcp\": {\"security-audit\": {\"enabled\": true}}\n}\n")

	merged, err := MergeJSONObjects(base, overlay)
	if err != nil {
		t.Fatalf("MergeJSONObjects() error = %v", err)
	}
	text := string(merged)
	if strings.Contains(text, "<!--") {
		t.Fatalf("merged JSON still contains HTML comments: %s", text)
	}
	if !strings.Contains(text, `"enabled": true`) {
		t.Fatalf("overlay not applied correctly: %s", text)
	}
}
