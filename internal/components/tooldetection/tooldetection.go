package tooldetection

import (
	"fmt"
	"os"
	"path/filepath"

	"orquestador-auditor/internal/agents"
	"orquestador-auditor/internal/system"
)

// Inject generates e inyecta un archivo con las ubicaciones de herramientas disponibles.
// Este archivo le dice a los agentes de IA cómo ejecutar cada herramienta (vía WSL o nativo).
func Inject(homeDir string, adapter agents.Adapter) error {
	if homeDir == "" {
		return fmt.Errorf("homeDir is required")
	}

	// Detectar dónde están las herramientas
	locations := system.DetectToolLocations()
	instructions := system.GenerateToolInstructions(locations)

	// Generar el contenido del archivo
	content := fmt.Sprintf(`# Security Tools Availability

## Información del Sistema

Este archivo fue generado automáticamente por el orquestador. Indica **dónde** están instaladas las herramientas de seguridad y **cómo ejecutarlas**.

%s

## Referencia Rápida para Agentes

### Cuando ejecutes herramientas:

**Si la herramienta está en WSL:**
Ejemplo: ` + "```bash\nwsl nmap -sV target.com\nwsl sqlmap -u \"http://target.com/page?id=1\"\nwsl katana -u http://target.com\n```" + `

**Si la herramienta es nativa de Windows:**
Ejemplo: ` + "```bash\nnmap -sV target.com\nsqlmap -u \"http://target.com/page?id=1\"\n```" + `

### Reglas Importantes

1. **SIEMPRE** usa el prefijo ` + "`wsl`" + ` para herramientas marcadas como "en WSL"
2. **NO** intentes ejecutar herramientas marcadas como "no disponibles"
3. **VERIFICA** que la herramienta existe antes de usarla: ` + "`wsl which nmap`" + ` o ` + "`where nmap`" + `
4. **REPORTA** en tus hallazgos qué herramienta usaste y su ubicación

---

*Generado automáticamente por orquestador-auditor*
`, instructions)

	// Determinar dónde guardar el archivo
	toolsFile := adapter.ToolsFilePath(homeDir)
	if toolsFile == "" {
		// Si el agente no soporta archivo de herramientas, usar ubicación por defecto
		toolsFile = filepath.Join(homeDir, ".orquestador-auditor", "tools-availability.md")
	}

	// Crear directorio si no existe
	dir := filepath.Dir(toolsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("crear directorio %s: %w", dir, err)
	}

	// Escribir el archivo
	if err := os.WriteFile(toolsFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("escribir archivo %s: %w", toolsFile, err)
	}

	fmt.Printf("✅ Tool availability generated: %s\n", toolsFile)
	return nil
}
