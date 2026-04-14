package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"orquestador-auditor/internal/agents"
	"orquestador-auditor/internal/backup"
	"orquestador-auditor/internal/installcmd"
	"orquestador-auditor/internal/memory"
	"orquestador-auditor/internal/model"
	"orquestador-auditor/internal/orchestrator"
	"orquestador-auditor/internal/system"
	"orquestador-auditor/internal/verify"
)

// runSetup instala las dependencias npm (chrome-devtools-mcp, @opencode-ai/memory)
// y luego ejecuta sync contra opencode, todo en un solo comando.
//
// Uso:
//
//	orquestador-auditor setup                   # instala deps + sync recon
//	orquestador-auditor setup --profile web-triage
//	orquestador-auditor setup --dry-run
//	orquestador-auditor setup --skip-deps       # solo sync
//	orquestador-auditor setup --skip-sync       # solo instalar deps
func runSetup(args []string) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	var profileName string
	var dryRun bool
	var skipDeps bool
	var skipSync bool
	fs.StringVar(&profileName, "profile", string(model.AuditProfileRecon),
		"Audit profile: recon, web-triage, supply-chain, reporting, memory-only")
	fs.BoolVar(&dryRun, "dry-run", false, "Muestra los comandos sin ejecutarlos")
	fs.BoolVar(&skipDeps, "skip-deps", false, "Omitir instalación de dependencias npm")
	fs.BoolVar(&skipSync, "skip-sync", false, "Omitir el sync de opencode")
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
	if !detection.Profile.Supported {
		return fmt.Errorf("unsupported platform %s/%s", detection.Profile.OS, detection.Profile.Arch)
	}

	// ── Paso 1: instalar dependencias npm ────────────────────────────────────
	if !skipDeps {
		printBanner("PASO 1 — Instalar dependencias npm de opencode")
		fmt.Println("  Paquetes requeridos:")
		fmt.Println("  • chrome-devtools-mcp   → MCP server para captura de tráfico del browser")
		fmt.Println("  (La memoria Engram cross-session ya está integrada en el binario)")
		fmt.Println()

		resolver := installcmd.NewResolver(detection.Profile)
		depCmds, err := resolver.ResolveOpenCodeDeps()
		if err != nil {
			return fmt.Errorf("resolver deps: %w", err)
		}

		for _, cmd := range depCmds {
			if len(cmd) == 0 {
				continue
			}
			fmt.Printf("  → %s\n", strings.Join(cmd, " "))
			if !dryRun {
				c := exec.Command(cmd[0], cmd[1:]...)
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				if err := c.Run(); err != nil {
					return fmt.Errorf("instalar %q: %w", cmd[len(cmd)-1], err)
				}
				fmt.Println()
			}
		}

		if dryRun {
			fmt.Println("  [dry-run] no se ejecutó nada.")
		} else {
			fmt.Println("  ✓ Dependencias instaladas.")
		}
		fmt.Println()
	}

	// ── Paso 2: sync opencode ─────────────────────────────────────────────────
	if !skipSync {
		printBanner("PASO 2 — Sync de agentes y config en opencode")

		profile, err := resolveAuditProfile(profileName)
		if err != nil {
			return err
		}

		homeDir := detection.Profile.HomeDir

		adapter, err := agents.NewAdapter(model.AgentOpenCode)
		if err != nil {
			return fmt.Errorf("crear adapter opencode: %w", err)
		}
		if !adapter.IsInstalled(context.Background(), homeDir) {
			fmt.Println("  ⚠  opencode no detectado. Instalalo primero con:")
			fmt.Println("       npm install -g opencode-ai")
			fmt.Println("  Luego volvé a correr: orquestador-auditor setup")
			return nil
		}

		if dryRun {
			fmt.Printf("  [dry-run] sync --agent opencode --profile %s\n\n", profileName)
		} else {
			injector := orchestrator.Injector{
				HomeDir:     homeDir,
				Profile:     profile,
				UseMarkers:  true,
				MemoryStore: memory.New(filepath.Join(".orquestador", "memory")),
			}

			paths := injector.ManagedPaths(adapter)
			backupPaths := injector.BackupPaths(adapter)
			backupDir := filepath.Join(backup.DefaultBackupDir(homeDir), string(adapter.ID()), string(profile.ID))

			manifest, err := backup.NewSnapshotter().Create(backupDir, backupPaths)
			if err != nil {
				return fmt.Errorf("backup: %w", err)
			}
			if err := injector.InjectAll(adapter); err != nil {
				_ = backup.Restore(manifest)
				return fmt.Errorf("sync opencode: %w", err)
			}

			results := verify.RunChecks(context.Background(), buildSyncChecks(adapter, paths))
			for _, r := range results {
				if r.Status == verify.CheckStatusFailed {
					_ = backup.Restore(manifest)
					return fmt.Errorf("verify failed:\n%s", verify.RenderReport(results))
				}
			}

			fmt.Printf("  ✓ Synced opencode — perfil %s (%s)\n", profile.ID, profile.Risk.Mode)
			fmt.Printf("  ✓ Backup en: %s\n", manifest.RootDir)
		}
		fmt.Println()
	}

	// ── Resumen final ─────────────────────────────────────────────────────────
	if dryRun {
		printBanner("Setup completo (dry-run — no se ejecutó nada)")
	} else {
		printBanner("Setup completo ✓")
		fmt.Println("  Próximos pasos:")
		fmt.Println("  1. Reiniciá opencode desktop para que tome los cambios")
		fmt.Println("  2. El agente security-orchestrator estará disponible")
		fmt.Println("  3. Los MCP chrome-devtools y security-audit estarán activos")
		fmt.Println("  4. Memoria Engram persistente en .orquestador/memory/")
	}
	fmt.Println()

	return nil
}

func printBanner(title string) {
	border := strings.Repeat("═", 58)
	fmt.Printf("╔%s╗\n", border)
	padded := fmt.Sprintf("  %s", title)
	if len(padded) < 58 {
		padded += strings.Repeat(" ", 58-len(padded))
	}
	fmt.Printf("║%s║\n", padded)
	fmt.Printf("╚%s╝\n\n", border)
}
