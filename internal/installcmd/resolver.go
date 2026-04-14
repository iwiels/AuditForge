package installcmd

import (
	"fmt"
	"os/exec"

	"orquestador-auditor/internal/model"
	"orquestador-auditor/internal/system"
)

type CommandSequence = [][]string

type Resolver interface {
	ResolveAgentInstall(agent model.AgentID) (CommandSequence, error)
	ResolveBundleInstall(bundle string) (CommandSequence, error)
	ResolveComponentInstall(component model.ComponentID) (CommandSequence, error)
	ResolveDependencyInstall(dependency string) (CommandSequence, error)
	ResolveOpenCodeDeps() (CommandSequence, error)
}

type runtimeResolver struct {
	lookPath func(string) (string, error)
	profile  system.PlatformProfile
}

func NewResolver(profile system.PlatformProfile) Resolver {
	return runtimeResolver{lookPath: exec.LookPath, profile: profile}
}

func (r runtimeResolver) ResolveAgentInstall(agent model.AgentID) (CommandSequence, error) {
	switch agent {
	case model.AgentClaudeCode, model.AgentClaude:
		return npmInstall(r, "@anthropic-ai/claude-code"), nil
	case model.AgentOpenCode:
		if r.profile.PackageManager == "brew" {
			return CommandSequence{{"brew", "install", "anomalyco/tap/opencode"}}, nil
		}
		return npmInstall(r, "opencode-ai"), nil
	case model.AgentCursor:
		return nil, fmt.Errorf("agent %q is a desktop app; install it manually and then run sync", agent)
	default:
		return nil, fmt.Errorf("install command is not supported for agent %q", agent)
	}
}

// ResolveOpenCodeDeps returns the npm packages que opencode necesita
// para que el orquestador funcione correctamente:
//
//   - chrome-devtools-mcp : MCP server para captura de trafico del browser
//
// Nota: el sistema Engram de memoria persistente ya está integrado en el
// binario (internal/memory) y NO requiere ningún paquete npm adicional.
func (r runtimeResolver) ResolveOpenCodeDeps() (CommandSequence, error) {
	pkgs := []string{
		"chrome-devtools-mcp",
	}
	var cmds CommandSequence
	for _, pkg := range pkgs {
		cmds = append(cmds, npmInstall(r, pkg)...)
	}
	return cmds, nil
}

func (r runtimeResolver) ResolveComponentInstall(component model.ComponentID) (CommandSequence, error) {
	switch component {
	case model.ComponentAssets, model.ComponentCommands, model.ComponentMCP, model.ComponentPrompts,
		model.ComponentSkills, model.ComponentOutputStyles, model.ComponentSubAgents,
		model.ComponentBackup, model.ComponentVerify, model.ComponentSystem:
		return CommandSequence{}, nil
	default:
		return nil, fmt.Errorf("install command is not supported for component %q", component)
	}
}

func (r runtimeResolver) ResolveBundleInstall(bundle string) (CommandSequence, error) {
	deps := bundleDependencies(bundle)
	if len(deps) == 0 {
		return nil, fmt.Errorf("install bundle %q is not supported", bundle)
	}
	commands := make(CommandSequence, 0, len(deps))
	for _, dep := range deps {
		resolved, err := r.ResolveDependencyInstall(dep)
		if err != nil {
			return nil, err
		}
		commands = append(commands, resolved...)
	}
	return commands, nil
}

func (r runtimeResolver) ResolveDependencyInstall(dependency string) (CommandSequence, error) {
	if dependency == "" {
		return nil, fmt.Errorf("dependency name is required")
	}
	switch r.profile.PackageManager {
	case "brew":
		return CommandSequence{{"brew", "install", dependency}}, nil
	case "apt":
		return CommandSequence{{"sudo", "apt-get", "install", "-y", dependency}}, nil
	case "pacman":
		return CommandSequence{{"sudo", "pacman", "-S", "--noconfirm", dependency}}, nil
	case "dnf":
		return CommandSequence{{"sudo", "dnf", "install", "-y", dependency}}, nil
	case "winget":
		return CommandSequence{{"winget", "install", "--id", dependency, "-e", "--accept-source-agreements", "--accept-package-agreements"}}, nil
	default:
		return nil, fmt.Errorf("could not detect a supported package manager for dependency %q", dependency)
	}
}

func npmInstall(r runtimeResolver, pkg string) CommandSequence {
	if r.profile.OS == "linux" && hasBinary(r.lookPath, "sudo") {
		return CommandSequence{{"sudo", "npm", "install", "-g", pkg}}
	}
	return CommandSequence{{"npm", "install", "-g", pkg}}
}

func hasBinary(lookPath func(string) (string, error), name string) bool {
	_, err := lookPath(name)
	return err == nil
}

func bundleDependencies(bundle string) []string {
	switch bundle {
	case "core-web":
		return []string{"nmap", "whatweb", "katana", "nuclei", "nikto", "sqlmap", "searchsploit"}
	case "supply-chain":
		return []string{"semgrep", "trivy", "grype", "gitleaks"}
	case "advanced-web":
		return []string{"jsluice", "mitmproxy", "mitmproxy2swagger", "arjun", "ffuf", "waymore", "exiftool", "jwt-tool"}
	case "offensive-web":
		return []string{"ysoserial", "jwt-tool", "exiftool", "burpsuite", "polyglot-generator"}
	case "full":
		return []string{"nmap", "whatweb", "katana", "nuclei", "nikto", "sqlmap", "searchsploit", "semgrep", "trivy", "grype", "gitleaks", "jsluice", "mitmproxy", "mitmproxy2swagger", "arjun", "ffuf", "waymore", "exiftool", "jwt-tool", "ysoserial", "burpsuite"}
	default:
		return nil
	}
}
