# setup-gemini-adapter.ps1
# Crea la estructura de assets para Gemini CLI y escribe todos los archivos sin BOM.
# Ejecutar desde la raiz del repo: .\setup-gemini-adapter.ps1

$base = Join-Path $PSScriptRoot "internal\assets\gemini"
$utf8NoBOM = New-Object System.Text.UTF8Encoding($false)

# Crear directorios
foreach ($dir in @("commands", "agents")) {
    $full = Join-Path $base $dir
    New-Item -ItemType Directory -Force -Path $full | Out-Null
}

# ─── GEMINI.md ──────────────────────────────────────────────────────────────
$geminiMdContent = @"
# Security Audit Orchestrator — Team Protocol v2.0

## Identidad
Sos el **Lead Security Strategist** de un equipo de auditoría de seguridad autorizada.
Tu trabajo: coordinar el equipo, no ejecutar el análisis vos solo.

## Comandos disponibles
| Comando | Acción |
|---------|--------|
| /scout [target] | Recon y superficie |
| /deep-web [target] | Análisis web profundo |
| /supply-chain [path] | Código y dependencias |
| /report | Síntesis y reporte final |
| /team [target] | Pipeline completo |
| /memory-search [query] | Contexto histórico |

---
## Skills disponibles
@~/.gemini/skills/surface-discovery.md
@~/.gemini/skills/web-triage.md
@~/.gemini/skills/threat-modeling.md
@~/.gemini/skills/evidence-reporting.md
"@

[System.IO.File]::WriteAllText((Join-Path $base "GEMINI.md"), $geminiMdContent, $utf8NoBOM)

# ─── COMMANDS (.toml) ─────────────────────────────────────────────────────────
$commands = @{
"scout.toml" = @"
description = "Recon y surface discovery — primer agente del Security Audit Team"
prompt = """
Ejecutá recon completo del target autorizado. Sos el primer agente del pipeline.
Target: {{args}}
"""
"@

"memory-search.toml" = @"
description = "Buscar en el contexto histórico de auditorías anteriores"
prompt = """
Buscá en la memoria persistente usando la herramienta MCP security-audit.
Query a buscar: {{args}}
"""
"@
}

$cmdDir = Join-Path $base "commands"
foreach ($name in $commands.Keys) {
    [System.IO.File]::WriteAllText((Join-Path $cmdDir $name), $commands[$name], $utf8NoBOM)
}

# ─── AGENTS (.md) ─────────────────────────────────────────────────────────────
$agents = @{
"security-scout.md" = @"
---
name: security-scout
description: Recon & Surface Discovery
model: gemini-2.0-flash
---
Sos el Security Scout del Security Audit Team.
"@

"security-report.md" = @"
---
name: security-report
description: Security Report Specialist
model: gemini-2.0-flash
---
Sos el Security Report Specialist del Security Audit Team.
"@
}

$agentDir = Join-Path $base "agents"
foreach ($name in $agents.Keys) {
    [System.IO.File]::WriteAllText((Join-Path $agentDir $name), $agents[$name], $utf8NoBOM)
}

Write-Host "[+] Script corregido. Archivos generados sin BOM."
Write-Host "[!] IMPORTANTE: Borra los archivos viejos en C:\Users\victo\.gemini\ para evitar conflictos."
