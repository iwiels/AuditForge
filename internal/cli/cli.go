package cli

import (
	"fmt"
	"os"
	"strings"

	"orquestador-auditor/internal/mcp"
)

// Run dispatches the CLI command based on args.
func Run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "ui":
		return runInteractive()
	case "install":
		return runInstall(args[1:])
	case "setup":
		return runSetup(args[1:])
	case "sync":
		return runSync(args[1:])
	case "run":
		return runAudit(args[1:])
	case "memory":
		return runMemory(args[1:])
	case "self-update":
		return runSelfUpdate(args[1:])
	case "--mcp", "mcp":
		mcp.Serve()
		return nil
	case "--help", "-h", "help":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command %q - run without arguments to see usage", args[0])
	}
}

// RunFromOSArgs calls Run with the real os.Args.
func RunFromOSArgs() error {
	return Run(os.Args[1:])
}

func printHelp() {
	lines := []string{
		"orquestador-auditor - orchestrator de ciberseguridad OpenCode-first",
		"",
		"Uso:",
		"  orquestador-auditor setup                              # instala deps npm + sync opencode (recon)",
		"  orquestador-auditor setup --profile web-triage         # mismo con perfil web-triage",
		"  orquestador-auditor setup --dry-run                    # muestra qué haría sin ejecutar",
		"  orquestador-auditor setup --skip-deps                  # solo sync (sin instalar npm)",
		"  orquestador-auditor install --bundle full --execute",
		"  orquestador-auditor sync --profile recon",
		"  orquestador-auditor sync --profile web-triage",
		"  orquestador-auditor run start --target https://example.com --profile web-triage --authorized --aggressiveness bounded",
		"  orquestador-auditor run phase --run-id 20260409-120000-example-com --phase network-recon --status observed --requested-tools nmap",
		"  orquestador-auditor run correlate --run-id 20260409-120000-example-com",
		"  orquestador-auditor run validate --run-id 20260409-120000-example-com",
		"  orquestador-auditor sync --profile supply-chain",
		"  orquestador-auditor memory search --query admin",
		"  orquestador-auditor memory context --limit 5",
		"  orquestador-auditor self-update",
		"  orquestador-auditor ui",
		"  orquestador-auditor --mcp",
		"",
		"Perfiles de sync:",
		"  recon | web-triage | supply-chain | reporting | memory-only",
		"",
		"Comandos:",
		"  setup         Instala deps npm (chrome-devtools-mcp, @opencode-ai/memory) + sync opencode",
		"  install       Resuelve e instala bundles de herramientas de seguridad",
		"  sync          Inyecta en OpenCode el perfil activo con policy-driven prompts, MCP y agentes",
		"  run           Crea artefactos metodologicos por fase, aplica policy gating y correlaciona hallazgos",
		"  memory        Busca o lista observaciones persistidas",
		"  self-update   Actualiza el binario desde GitHub releases",
		"  ui            Abre la TUI interactiva",
		"  --mcp         Corre como servidor MCP (stdio)",
		"",
		"Notas:",
		"  - 'setup' es el punto de entrada recomendado: instala todo lo necesario en un solo paso",
		"  - sync apunta a opencode por defecto",
		"  - run no autoejecuta herramientas de riesgo: registra policy, artefactos y decisiones por fase",
		"  - usá --all o --agent solo si necesitás compatibilidad avanzada con otros clientes",
	}
	fmt.Println(strings.Join(lines, "\n"))
}
