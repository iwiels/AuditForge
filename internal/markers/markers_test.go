package markers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectWithMarkersCreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	marker := ContentMarker{
		Component: "test",
		Content:   []byte("hello world"),
	}
	if err := InjectWithMarkers(path, marker); err != nil {
		t.Fatalf("InjectWithMarkers() first error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), "<!-- ORQUESTADOR:test:start -->") {
		t.Fatalf("expected start marker, got: %s", string(content))
	}
	if !strings.Contains(string(content), "hello world") {
		t.Fatalf("expected content, got: %s", string(content))
	}
}

func TestInjectWithMarkersReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	marker1 := ContentMarker{
		Component: "test",
		Content:   []byte("old content"),
	}
	if err := InjectWithMarkers(path, marker1); err != nil {
		t.Fatalf("InjectWithMarkers() first error = %v", err)
	}

	marker2 := ContentMarker{
		Component: "test",
		Content:   []byte("new content"),
	}
	if err := InjectWithMarkers(path, marker2); err != nil {
		t.Fatalf("InjectWithMarkers() second error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	// Should have exactly ONE start marker (not two)
	count := strings.Count(string(content), "<!-- ORQUESTADOR:test:start -->")
	if count != 1 {
		t.Fatalf("expected 1 start marker after replace, got %d in: %s", count, string(content))
	}
	if !strings.Contains(string(content), "new content") {
		t.Fatalf("expected new content, got: %s", string(content))
	}
	if strings.Contains(string(content), "old content") {
		t.Fatalf("should not contain old content, got: %s", string(content))
	}
}

func TestInjectWithMarkersAppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	// Write pre-existing content
	if err := os.WriteFile(path, []byte("# Existing file\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	marker := ContentMarker{
		Component: "test",
		Content:   []byte("injected"),
	}
	if err := InjectWithMarkers(path, marker); err != nil {
		t.Fatalf("InjectWithMarkers() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), "# Existing file") {
		t.Fatalf("expected original content preserved, got: %s", string(content))
	}
	if !strings.Contains(string(content), "injected") {
		t.Fatalf("expected injected content, got: %s", string(content))
	}
}

func TestListMarkedComponents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	// Write content with multiple markers
	content := `# File
<!-- ORQUESTADOR:prompts:start -->
prompt content
<!-- ORQUESTADOR:prompts:end -->

<!-- ORQUESTADOR:commands:start -->
command content
<!-- ORQUESTADOR:commands:end -->
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	components, err := ListMarkedComponents(path)
	if err != nil {
		t.Fatalf("ListMarkedComponents() error = %v", err)
	}
	if len(components) != 2 {
		t.Fatalf("expected 2 components, got %d: %v", len(components), components)
	}
	if components[0] != "prompts" || components[1] != "commands" {
		t.Fatalf("unexpected components: %v", components)
	}
}

func TestFormatMarkdownSection(t *testing.T) {
	result := FormatMarkdownSection("test", "Test Section", "some content")
	if !strings.Contains(result, "<!-- ORQUESTADOR:test:start -->") {
		t.Fatalf("expected start marker, got: %s", result)
	}
	if !strings.Contains(result, "## Test Section") {
		t.Fatalf("expected title, got: %s", result)
	}
	if !strings.Contains(result, "some content") {
		t.Fatalf("expected content, got: %s", result)
	}
	if !strings.Contains(result, "<!-- ORQUESTADOR:test:end -->") {
		t.Fatalf("expected end marker, got: %s", result)
	}
}
