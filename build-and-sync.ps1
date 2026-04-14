# Build y setup completo del orquestador-auditor
# Ejecutar desde el directorio del proyecto: .\build-and-sync.ps1
#
# Flags opcionales:
#   -DryRun      Ver qué haría sin ejecutar
#   -SkipDeps    Solo sync (no instalar npm)
#   -Profile     Perfil de audit (default: recon)

param(
    [switch]$DryRun,
    [switch]$SkipDeps,
    [string]$Profile = "recon"
)

Set-Location $PSScriptRoot

Write-Host ""
Write-Host ">>> Compilando orquestador-auditor..." -ForegroundColor Cyan
go build -o orquestador-auditor.exe ./cmd/orquestador-auditor/
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Falló el build" -ForegroundColor Red
    exit 1
}
Write-Host "Build exitoso." -ForegroundColor Green
Write-Host ""

# Construir flags para setup
$setupArgs = @("setup", "--profile", $Profile)
if ($DryRun)   { $setupArgs += "--dry-run" }
if ($SkipDeps) { $setupArgs += "--skip-deps" }

Write-Host ">>> Ejecutando: orquestador-auditor $($setupArgs -join ' ')" -ForegroundColor Cyan
Write-Host ""

& .\orquestador-auditor.exe @setupArgs
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Falló el setup" -ForegroundColor Red
    exit 1
}
