// package markers provides content marker boundaries for drift-free injection.
// When sync runs multiple times, markers ensure content is replaced atomically
// rather than appended, keeping AI client configs clean and deterministic.
//
// CRITICAL: Do NOT use these markers on JSON files, as they are HTML/XML style
// comments (<!-- ... -->) which are invalid JSON and will corrupt the file.
package markers

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

const (
	markerPrefix = "<!-- ORQUESTADOR:"
	markerSuffix = " -->"
	markerStart  = ":start -->"
	markerEnd    = ":end -->"
)

// ContentMarker wraps a block of injectable content with identifiable boundaries.
type ContentMarker struct {
	Component string // e.g. "commands", "skills", "mcp", "agents", "prompts"
	Content   []byte // The raw content to inject (without markers)
}

// MarkedBlock returns the full content with start/end markers wrapping it.
func (m ContentMarker) MarkedBlock() []byte {
	var buf bytes.Buffer
	buf.WriteString(markerPrefix + m.Component + markerStart)
	buf.WriteByte('\n')
	buf.Write(m.Content)
	if !bytes.HasSuffix(bytes.TrimSpace(m.Content), []byte("\n")) {
		buf.WriteByte('\n')
	}
	buf.WriteString(markerPrefix + m.Component + markerEnd)
	buf.WriteByte('\n')
	return buf.Bytes()
}

// InjectWithMarkers performs drift-free injection.
// If a previous block exists for the same component, it is replaced.
// If not, the content is appended.
func InjectWithMarkers(path string, marker ContentMarker) error {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if os.IsNotExist(err) || len(bytes.TrimSpace(existing)) == 0 {
		return os.WriteFile(path, marker.MarkedBlock(), 0o644)
	}

	startTag := markerPrefix + marker.Component + markerStart
	endTag := markerPrefix + marker.Component + markerEnd

	startIdx := bytes.Index(existing, []byte(startTag))
	endIdx := bytes.Index(existing, []byte(endTag))

	if startIdx >= 0 && endIdx > startIdx {
		newContent := replaceMarkedBlock(existing, startIdx, endIdx+len(endTag), marker)
		return os.WriteFile(path, newContent, 0o644)
	}

	var buf bytes.Buffer
	buf.Write(bytes.TrimSpace(existing))
	buf.WriteByte('\n')
	buf.WriteByte('\n')
	buf.Write(marker.MarkedBlock())
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// FormatMarkdownSection formats content as a markdown section with a header.
func FormatMarkdownSection(sectionID string, title string, content string) string {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("\n<!-- ORQUESTADOR:%s:start -->\n", sectionID))
	buf.WriteString(fmt.Sprintf("## %s\n\n", title))
	buf.WriteString(content)
	buf.WriteString(fmt.Sprintf("\n\n<!-- ORQUESTADOR:%s:end -->\n", sectionID))
	return buf.String()
}

func replaceMarkedBlock(existing []byte, startIdx, endIdx int, marker ContentMarker) []byte {
	var buf bytes.Buffer
	buf.Write(existing[:startIdx])
	buf.Write(marker.MarkedBlock())
	if endIdx < len(existing) {
		buf.Write(existing[endIdx:])
	}
	return buf.Bytes()
}

// ListMarkedComponents returns all ORQUESTADOR component names found in the file.
func ListMarkedComponents(path string) ([]string, error) {
	existing, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var components []string
	lines := strings.Split(string(existing), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, markerPrefix) && strings.Contains(line, markerStart) {
			trimmed := strings.TrimPrefix(line, markerPrefix)
			if idx := strings.Index(trimmed, markerStart); idx > 0 {
				components = append(components, trimmed[:idx])
			}
		}
	}
	return components, nil
}
