package installcmd

import (
	"fmt"
	"testing"

	"orquestador-auditor/internal/model"
	"orquestador-auditor/internal/system"
)

func TestResolveAgentInstallClaudeLinuxUsesSudo(t *testing.T) {
	r := runtimeResolver{
		profile: system.PlatformProfile{OS: "linux", PackageManager: "apt"},
		lookPath: func(name string) (string, error) {
			if name == "sudo" {
				return "/usr/bin/sudo", nil
			}
			return "", fmt.Errorf("missing %s", name)
		},
	}

	commands, err := r.ResolveAgentInstall(model.AgentClaudeCode)
	if err != nil {
		t.Fatalf("ResolveAgentInstall() error = %v", err)
	}
	if len(commands) != 1 || commands[0][0] != "sudo" {
		t.Fatalf("unexpected commands: %#v", commands)
	}
}

func TestResolveAgentInstallOpenCodeDarwinUsesBrew(t *testing.T) {
	r := runtimeResolver{
		profile: system.PlatformProfile{OS: "darwin", PackageManager: "brew"},
		lookPath: func(name string) (string, error) {
			if name == "brew" {
				return "/opt/homebrew/bin/brew", nil
			}
			return "", fmt.Errorf("missing %s", name)
		},
	}

	commands, err := r.ResolveAgentInstall(model.AgentOpenCode)
	if err != nil {
		t.Fatalf("ResolveAgentInstall() error = %v", err)
	}
	if len(commands) != 1 || commands[0][0] != "brew" {
		t.Fatalf("unexpected commands: %#v", commands)
	}
}

func TestResolveComponentInstallReturnsNoopForInternalComponent(t *testing.T) {
	r := runtimeResolver{}
	commands, err := r.ResolveComponentInstall(model.ComponentMCP)
	if err != nil {
		t.Fatalf("ResolveComponentInstall() error = %v", err)
	}
	if len(commands) != 0 {
		t.Fatalf("expected no-op commands, got %#v", commands)
	}
}
