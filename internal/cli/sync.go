package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"orquestador-auditor/internal/agents"
	"orquestador-auditor/internal/backup"
	"orquestador-auditor/internal/catalog"
	"orquestador-auditor/internal/memory"
	"orquestador-auditor/internal/model"
	"orquestador-auditor/internal/orchestrator"
	"orquestador-auditor/internal/system"
	"orquestador-auditor/internal/verify"
)

// resolveAuditProfile es una helper compartida por sync y setup.
func resolveAuditProfile(profileName string) (model.AuditProfile, error) {
	return catalog.AuditProfileByID(profileName)
}

func runSync(args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	var agentList string
	var profileName string
	var syncAll bool
	fs.StringVar(&agentList, "agent", "", "Comma-separated agents to sync (defaults to opencode)")
	fs.StringVar(&profileName, "profile", string(model.AuditProfileRecon), "Audit profile to inject: recon, web-triage, supply-chain, reporting, memory-only")
	fs.BoolVar(&syncAll, "all", false, "Sync all detected clients (legacy / advanced mode)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	profile, err := catalog.AuditProfileByID(profileName)
	if err != nil {
		return err
	}

	if err := system.EnsureCurrentOSSupported(); err != nil {
		return err
	}

	detection, err := system.Detect(context.Background())
	if err != nil {
		return err
	}
	if !detection.Profile.Supported {
		return fmt.Errorf("unsupported platform %s/%s", detection.Profile.OS, detection.Profile.Arch)
	}

	homeDir := detection.Profile.HomeDir
	registry, err := agents.NewDefaultRegistry()
	if err != nil {
		return err
	}

	targets, err := resolveSyncAgents(context.Background(), registry, homeDir, agentList, syncAll)
	if err != nil {
		return err
	}

	injector := orchestrator.Injector{
		HomeDir:     homeDir,
		Profile:     profile,
		UseMarkers:  true,
		MemoryStore: memory.New(filepath.Join(".orquestador", "memory")),
	}
	for _, adapter := range targets {
		paths := injector.ManagedPaths(adapter)
		backupPaths := injector.BackupPaths(adapter)
		backupDir := filepath.Join(backup.DefaultBackupDir(homeDir), string(adapter.ID()), string(profile.ID))
		manifest, err := backup.NewSnapshotter().Create(backupDir, backupPaths)
		if err != nil {
			return fmt.Errorf("backup %s: %w", adapter.ID(), err)
		}
		if err := injector.InjectAll(adapter); err != nil {
			_ = backup.Restore(manifest)
			return fmt.Errorf("sync %s: %w", adapter.ID(), err)
		}

		results := verify.RunChecks(context.Background(), buildSyncChecks(adapter, paths))
		failed := false
		for _, result := range results {
			if result.Status == verify.CheckStatusFailed {
				failed = true
				break
			}
		}
		if failed {
			_ = backup.Restore(manifest)
			return fmt.Errorf("verify %s failed:\n%s", adapter.ID(), verify.RenderReport(results))
		}

		fmt.Printf("Synced %s with profile %s (%s)\n", adapter.ID(), profile.ID, profile.Risk.Mode)
		fmt.Printf("Backup: %s\n", manifest.RootDir)
	}
	return nil
}

func resolveSyncAgents(ctx context.Context, reg *agents.Registry, homeDir, agentList string, syncAll bool) ([]agents.Adapter, error) {
	selected := map[model.AgentID]struct{}{}
	if syncAll {
		for _, item := range agents.DiscoverInstalled(ctx, reg, homeDir) {
			selected[item.ID] = struct{}{}
		}
	} else if strings.TrimSpace(agentList) == "" {
		selected[model.AgentOpenCode] = struct{}{}
	} else {
		for _, part := range strings.Split(agentList, ",") {
			value := model.AgentID(strings.TrimSpace(part))
			if value != "" {
				selected[value] = struct{}{}
			}
		}
	}

	out := make([]agents.Adapter, 0, len(selected))
	seenConfigRoots := map[string]struct{}{}
	for id := range selected {
		adapter, err := agents.NewAdapter(id)
		if err != nil {
			return nil, err
		}
		if !adapter.IsInstalled(ctx, homeDir) {
			fmt.Printf("Skipping %s because it does not look installed yet\n", id)
			continue
		}
		configRoot := adapter.ConfigDir(homeDir)
		if _, ok := seenConfigRoots[configRoot]; ok {
			continue
		}
		seenConfigRoots[configRoot] = struct{}{}
		out = append(out, adapter)
	}
	return out, nil
}

func buildSyncChecks(adapter agents.Adapter, paths []string) []verify.Check {
	checks := make([]verify.Check, 0, len(paths)+1)
	for _, path := range paths {
		path := path
		checks = append(checks, verify.Check{
			ID:          path,
			Description: "managed integration file exists and contains expected markers",
			Run: func(context.Context) error {
				raw, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				text := string(raw)
				lower := strings.ToLower(text)
				if strings.TrimSpace(text) == "" {
					return fmt.Errorf("file is empty")
				}
				name := filepath.Base(path)
				parent := filepath.Base(filepath.Dir(path))
				switch {
				case strings.EqualFold(parent, "agents") && strings.HasSuffix(strings.ToLower(name), ".md"):
					if strings.Contains(text, "<!-- ORQUESTADOR:") {
						return fmt.Errorf("native agent file contains HTML markers")
					}
					if !strings.HasPrefix(strings.TrimPrefix(text, "\ufeff"), "---") {
						return fmt.Errorf("native agent file must start with markdown frontmatter")
					}
				case strings.EqualFold(name, "CLAUDE.md") || strings.EqualFold(name, "AGENTS.md") || strings.HasSuffix(name, ".md"):
					if !strings.Contains(text, "Security Audit") && !strings.Contains(text, "security-") && !strings.Contains(text, "Skill:") && !strings.Contains(lower, "security") {
						return fmt.Errorf("missing expected security content")
					}
				case strings.EqualFold(parent, "plugins") && strings.HasSuffix(strings.ToLower(name), ".ts"):
					if strings.Contains(text, "<!-- ORQUESTADOR:") {
						return fmt.Errorf("plugin file contains HTML markers")
					}
					trimmed := strings.TrimSpace(text)
					if !strings.HasPrefix(trimmed, "import ") && !strings.HasPrefix(trimmed, "export ") {
						return fmt.Errorf("plugin file does not start with valid TypeScript")
					}
				case strings.HasSuffix(name, ".json"):
					if strings.EqualFold(name, "security-audit.json") {
						if !strings.Contains(text, "orquestador-auditor") {
							return fmt.Errorf("missing orchestrator command reference")
						}
						return nil
					}
					if !strings.Contains(text, "security-audit") && !strings.Contains(lower, "security-") {
						return fmt.Errorf("missing security markers")
					}
				}
				return nil
			},
		})
	}
	checks = append(checks, verify.Check{
		ID:          "agent-capabilities",
		Description: "declared capabilities map to at least one managed file",
		Run: func(context.Context) error {
			if adapter.SupportsMCP() && adapter.MCPConfigPath("", "security-audit") == "" {
				return fmt.Errorf("mcp path is empty")
			}
			if adapter.SupportsSkills() && adapter.SkillsDir("x") == "" {
				return fmt.Errorf("skills dir is empty")
			}
			return nil
		},
	})
	return checks
}
