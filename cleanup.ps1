# cleanup.ps1
# Elimina todos los archivos que eran del motor de auditoría.
# Ejecutar desde la raíz del repo con: .\cleanup.ps1

$toDelete = @(
    "gentle-ai",
    "internal\orchestrator\engine.go",
    "internal\orchestrator\compat.go",
    "internal\orchestrator\state.go",
    "internal\orchestrator\stages",
    "internal\orchestrator\discovery",
    "internal\orchestrator\phase",
    "internal\orchestrator\pipeline_test.go",
    "internal\parsers",
    "internal\tools",
    "internal\webintel",
    "internal\teams",
    "internal\report",
    "internal\llm",
    "internal\model\audit.go",
    "internal\model\audit_test.go",
    "internal\model\system_prompt_strategy_ext.go",
    "internal\config\profiles.go",
    "internal\cli\e2e_test.go"
)

foreach ($path in $toDelete) {
    $full = Join-Path $PSScriptRoot $path
    if (Test-Path $full) {
        Remove-Item -Recurse -Force $full
        Write-Host "Deleted: $path"
    } else {
        Write-Host "Not found (skipped): $path"
    }
}

Write-Host ""
Write-Host "Listo. Ahora corré:"
Write-Host "  go mod tidy"
Write-Host "  go build ./..."
