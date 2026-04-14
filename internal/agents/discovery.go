package agents

import (
	"context"

	"orquestador-auditor/internal/model"
)

type InstalledAgent struct {
	ID        model.AgentID
	ConfigDir string
}

func DiscoverInstalled(ctx context.Context, reg *Registry, homeDir string) []InstalledAgent {
	var out []InstalledAgent
	seenConfigRoots := map[string]struct{}{}
	for _, id := range reg.SupportedAgents() {
		adapter, ok := reg.Get(id)
		if !ok {
			continue
		}
		if !adapter.IsInstalled(ctx, homeDir) {
			continue
		}
		configDir := adapter.ConfigDir(homeDir)
		if _, exists := seenConfigRoots[configDir]; exists {
			continue
		}
		seenConfigRoots[configDir] = struct{}{}
		out = append(out, InstalledAgent{ID: id, ConfigDir: configDir})
	}
	return out
}

func ConfigRootsForBackup(ctx context.Context, reg *Registry, homeDir string) []string {
	installed := DiscoverInstalled(ctx, reg, homeDir)
	seen := map[string]struct{}{}
	dirs := make([]string, 0, len(installed))
	for _, item := range installed {
		if _, ok := seen[item.ConfigDir]; ok || item.ConfigDir == "" {
			continue
		}
		seen[item.ConfigDir] = struct{}{}
		dirs = append(dirs, item.ConfigDir)
	}
	return dirs
}
