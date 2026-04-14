package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"orquestador-auditor/internal/installcmd"
	"orquestador-auditor/internal/model"
	"orquestador-auditor/internal/system"
)

func runInstall(args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	var agent string
	var bundle string
	var component string
	var dependency string
	var execute bool
	fs.StringVar(&agent, "agent", "", "Agent to install: claude-code, claude, cursor, opencode")
	fs.StringVar(&bundle, "bundle", "", "Bundle to install: core-web, supply-chain, advanced-web, full")
	fs.StringVar(&component, "component", "", "Component to install")
	fs.StringVar(&dependency, "dependency", "", "Dependency to install")
	fs.BoolVar(&execute, "execute", false, "Execute resolved commands instead of only printing them")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := system.EnsureCurrentOSSupported(); err != nil {
		return err
	}
	detection, err := system.Detect(context.Background())
	if err != nil {
		return err
	}
	resolver := installcmd.NewResolver(detection.Profile)
	var (
		commands   installcmd.CommandSequence
		resolveErr error
	)

	switch {
	case agent != "":
		commands, resolveErr = resolver.ResolveAgentInstall(model.AgentID(strings.TrimSpace(agent)))
	case bundle != "":
		commands, resolveErr = resolver.ResolveBundleInstall(strings.TrimSpace(bundle))
	case component != "":
		commands, resolveErr = resolver.ResolveComponentInstall(model.ComponentID(strings.TrimSpace(component)))
	case dependency != "":
		commands, resolveErr = resolver.ResolveDependencyInstall(strings.TrimSpace(dependency))
	default:
		return fmt.Errorf("install requires --agent, --component, or --dependency")
	}
	if resolveErr != nil {
		return resolveErr
	}

	if len(commands) == 0 {
		fmt.Println("No external install command is required. Run sync to write local integration files.")
		return nil
	}

	if !execute {
		fmt.Println(renderCommands(commands))
		return nil
	}

	for _, cmd := range commands {
		if len(cmd) == 0 {
			continue
		}
		execCmd := exec.Command(cmd[0], cmd[1:]...)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		if err := execCmd.Run(); err != nil {
			return fmt.Errorf("run %q: %w", strings.Join(cmd, " "), err)
		}
	}

	return nil
}

func renderCommands(commands installcmd.CommandSequence) string {
	lines := make([]string, 0, len(commands))
	for _, cmd := range commands {
		lines = append(lines, strings.Join(cmd, " "))
	}
	return strings.Join(lines, "\n")
}
