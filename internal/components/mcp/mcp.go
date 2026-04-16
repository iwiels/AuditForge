package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"orquestador-auditor/internal/agents"
	"orquestador-auditor/internal/components/filemerge"
	"orquestador-auditor/internal/model"
	"orquestador-auditor/internal/system"
)

func Inject(homeDir string, adapter agents.Adapter) error {
	if !adapter.SupportsMCP() {
		return nil
	}

	path := adapter.MCPConfigPath(homeDir, "security-audit")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	content, err := configFor(adapter)
	if err != nil {
		return err
	}

	switch adapter.MCPStrategy() {
	case model.StrategySeparateMCPFiles:
		return filemerge.WriteIfChanged(path, content)
	case model.StrategyMergeIntoSettings:
		return filemerge.WriteJSONMerged(path, content)
	case model.StrategyMCPConfigFile:
		return filemerge.WriteJSONMerged(path, content)
	default:
		return fmt.Errorf("unsupported mcp strategy %d", adapter.MCPStrategy())
	}
}

func configFor(adapter agents.Adapter) ([]byte, error) {
	cmdPath := resolveOrchestratorCommand()

	switch adapter.MCPStrategy() {

	case model.StrategySeparateMCPFiles:
		// Claude Code: archivo JSON separado por servidor
		return json.MarshalIndent(map[string]any{
			"command": cmdPath,
			"args":    []string{"--mcp"},
			"enabled": true,
		}, "", "  ")

	case model.StrategyMergeIntoSettings:
		// OpenCode: merge en opencode.json bajo clave "mcp"
		return json.MarshalIndent(map[string]any{
			"mcp": mapWithReplaceSentinel(opencodeMCPServers(cmdPath)),
		}, "", "  ")

	case model.StrategyMCPConfigFile:
		// Gemini CLI: settings.json con clave "mcpServers"
		// Nota: Gemini CLI no permite guiones bajos en el nombre del servidor
		// (el parser FQN los usa como separador). Usamos "security-audit" con guión.
		return json.MarshalIndent(map[string]any{
			"mcpServers": map[string]any{
				"__replace__": map[string]any{
					"security-audit": map[string]any{
						"command": cmdPath,
						"args":    []string{"--mcp"},
						"timeout": 30000,
						"trust":   false,
					},
				},
			},
		}, "", "  ")

	default:
		return nil, fmt.Errorf("unsupported mcp strategy %d", adapter.MCPStrategy())
	}
}

func resolveOrchestratorCommand() string {
	cmdPath := "orquestador-auditor"
	if exePath, err := os.Executable(); err == nil {
		return exePath
	}

	binName := "orquestador-auditor"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	localBin := filepath.Join(system.GetBinTargetDir(), binName)
	if _, err := os.Stat(localBin); err == nil {
		return localBin
	}
	return cmdPath
}

func opencodeMCPServers(orchestratorCmd string) map[string]any {
	servers := map[string]any{
		"security-audit": map[string]any{
			"type":    "local",
			"command": []string{orchestratorCmd, "--mcp"},
			"enabled": true,
		},
	}
	if chrome := chromeDevToolsServerConfig(); chrome != nil {
		servers["chrome-devtools"] = chrome
	}
	return servers
}

func chromeDevToolsServerConfig() map[string]any {
	command := chromeDevToolsCommand()
	if len(command) == 0 {
		return nil
	}

	server := map[string]any{
		"type":    "local",
		"command": command,
		"enabled": true,
		"timeout": 30000,
	}
	if runtime.GOOS == "windows" {
		server["environment"] = map[string]any{
			"SystemRoot":   os.Getenv("SystemRoot"),
			"PROGRAMFILES": os.Getenv("PROGRAMFILES"),
		}
	}
	return server
}

func chromeDevToolsCommand() []string {
	if runtime.GOOS == "windows" {
		if hasCommand("chrome-devtools-mcp") {
			return []string{"cmd", "/c", "chrome-devtools-mcp"}
		}
		if hasCommand("npx") {
			return []string{"cmd", "/c", "npx", "-y", "chrome-devtools-mcp@latest"}
		}
		return nil
	}
	if hasCommand("chrome-devtools-mcp") {
		return []string{"chrome-devtools-mcp"}
	}
	if hasCommand("npx") {
		return []string{"npx", "-y", "chrome-devtools-mcp@latest"}
	}
	return nil
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func mapWithReplaceSentinel(entries map[string]any) map[string]any {
	out := make(map[string]any, len(entries))
	for key, value := range entries {
		out[key] = map[string]any{
			"__replace__": value,
		}
	}
	return out
}
