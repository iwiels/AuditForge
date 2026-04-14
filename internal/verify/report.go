package verify

import "strings"

func RenderReport(results []CheckResult) string {
	var b strings.Builder
	for _, result := range results {
		b.WriteString("- [")
		b.WriteString(string(result.Status))
		b.WriteString("] ")
		b.WriteString(result.ID)
		b.WriteString(": ")
		b.WriteString(result.Description)
		if result.Error != "" {
			b.WriteString(" (")
			b.WriteString(result.Error)
			b.WriteString(")")
		}
		b.WriteString("\n")
	}
	return b.String()
}
