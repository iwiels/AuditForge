package system

import (
	"os/exec"
	"strings"
)

var securityToolSet = []string{
	"nmap",
	"whatweb",
	"katana",
	"nuclei",
	"nikto",
	"sqlmap",
	"searchsploit",
	"semgrep",
	"trivy",
	"grype",
	"gitleaks",
}

// ToolLocation describes where a security tool is available.
type ToolLocation struct {
	Name        string // tool name
	Available   bool   // is the tool available?
	ExecutionCmd string // how to execute it (e.g., "wsl nmap" or just "nmap")
	Location    string // "wsl", "native", or "not-found"
}

func RequiredSecurityTools() []string {
	out := make([]string, len(securityToolSet))
	copy(out, securityToolSet)
	return out
}

// DetectToolLocations checks where security tools are available (native Windows, WSL, or not found).
func DetectToolLocations() map[string]ToolLocation {
	locations := make(map[string]ToolLocation)

	for _, tool := range securityToolSet {
		loc := ToolLocation{
			Name:     tool,
			Location: "not-found",
		}

		// Check native Windows first
		if _, err := exec.LookPath(tool); err == nil {
			loc.Available = true
			loc.ExecutionCmd = tool
			loc.Location = "native"
		} else if isWSLAvailable() {
			// Check if tool is available in WSL
			if isToolAvailableInWSL(tool) {
				loc.Available = true
				loc.ExecutionCmd = "wsl " + tool
				loc.Location = "wsl"
			}
		}

		locations[tool] = loc
	}

	return locations
}

// isWSLAvailable checks if WSL is available on the system.
func isWSLAvailable() bool {
	cmd := exec.Command("wsl", "--status")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// isToolAvailableInWSL checks if a specific tool is available in WSL.
func isToolAvailableInWSL(tool string) bool {
	cmd := exec.Command("wsl", "which", tool)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return !strings.Contains(string(output), "not found") && strings.TrimSpace(string(output)) != ""
}

// GenerateToolInstructions generates instructions for agents on how to use tools.
func GenerateToolInstructions(locations map[string]ToolLocation) string {
	var instructions strings.Builder

	wslCount := 0
	nativeCount := 0
	notFoundCount := 0

	for _, loc := range locations {
		switch loc.Location {
		case "wsl":
			wslCount++
		case "native":
			nativeCount++
		case "not-found":
			notFoundCount++
		}
	}

	if wslCount > 0 {
		instructions.WriteString("\n## Herramientas disponibles en WSL\n")
		instructions.WriteString("Las siguientes herramientas están instaladas en WSL. Debes ejecutarlas con el prefijo `wsl`:\n\n")
		for _, loc := range locations {
			if loc.Location == "wsl" {
				instructions.WriteString("- **" + loc.Name + "**: usar como `" + loc.ExecutionCmd + "`\n")
			}
		}
	}

	if nativeCount > 0 {
		instructions.WriteString("\n## Herramientas nativas de Windows\n")
		instructions.WriteString("Estas herramientas están instaladas nativamente en Windows:\n\n")
		for _, loc := range locations {
			if loc.Location == "native" {
				instructions.WriteString("- **" + loc.Name + "**: usar directamente como `" + loc.Name + "`\n")
			}
		}
	}

	if notFoundCount > 0 {
		instructions.WriteString("\n## Herramientas no disponibles\n")
		instructions.WriteString("Estas herramientas NO están disponibles y no deben usarse:\n\n")
		for _, loc := range locations {
			if loc.Location == "not-found" {
				instructions.WriteString("- ❌ **" + loc.Name + "**\n")
			}
		}
	}

	return instructions.String()
}
