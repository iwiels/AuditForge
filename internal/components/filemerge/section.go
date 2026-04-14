package filemerge

import (
	"bytes"
	"os"
	"strings"
)

func InjectMarkdownSection(path, sectionID, content string) error {
	current, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	updated := injectSection(string(current), sectionID, content)
	if bytes.Equal(current, []byte(updated)) {
		return nil
	}
	return os.WriteFile(path, []byte(updated), 0o644)
}

func WriteIfChanged(path string, content []byte) error {
	current, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if bytes.Equal(current, content) {
		return nil
	}
	return os.WriteFile(path, content, 0o644)
}

func injectSection(existing, sectionID, content string) string {
	open := "<!-- orquestador-auditor:" + sectionID + " -->"
	close := "<!-- /orquestador-auditor:" + sectionID + " -->"
	openIdx := strings.Index(existing, open)
	closeIdx := strings.Index(existing, close)
	if openIdx >= 0 && closeIdx > openIdx {
		before := existing[:openIdx]
		after := existing[closeIdx+len(close):]
		var b strings.Builder
		b.WriteString(before)
		b.WriteString(open)
		b.WriteString("\n")
		b.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			b.WriteString("\n")
		}
		b.WriteString(close)
		b.WriteString(after)
		return b.String()
	}
	trimmed := strings.TrimRight(existing, "\r\n")
	if trimmed == "" {
		return open + "\n" + content + "\n" + close + "\n"
	}
	return trimmed + "\n\n" + open + "\n" + content + "\n" + close + "\n"
}
