#Requires -Version 5.1
<#
.SYNOPSIS
    AuditForge Assets Installer - PowerShell Edition
    Installs skills, agents, and MCP config
#>

[CmdletBinding()]
param(
    [switch]$Help
)

if ($Help) {
    Get-Help $MyInvocation.MyCommand.Path -Detailed
    exit 0
}

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = $ScriptDir
$OpenCodeDir = Join-Path $env:USERPROFILE ".config\opencode"
$SkillsDir = Join-Path $env:USERPROFILE ".auditforge\skills"
$AuditForgeDir = Join-Path $env:USERPROFILE ".auditforge"
$ProxyInstallDir = Join-Path $AuditForgeDir "proxy"
$AgentsDir = Join-Path $OpenCodeDir "agents"
$CommandsDir = Join-Path $OpenCodeDir "commands"
$McpDir = Join-Path $OpenCodeDir "mcp"

function Join-Path {
    param($Path, $ChildPath)
    return [System.IO.Path]::Combine($Path, $ChildPath)
}

function Write-Info($Message) {
    Write-Host "[INFO] $Message" -ForegroundColor Cyan
}
function Write-Success($Message) {
    Write-Host "[OK] $Message" -ForegroundColor Green
}
function Write-Warning($Message) {
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}
function Write-Error($Message) {
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}
function Write-Step($Message) {
    Write-Host ""
    Write-Host "[STEP] $Message" -ForegroundColor Magenta
}

function Ensure-Dir($Path) {
    if (!(Test-Path $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

# Install Skills
Write-Step "Installing skills..."
Ensure-Dir $SkillsDir
$skillsSrc = Join-Path $ProjectRoot "internal\assets\skills"
if (Test-Path $skillsSrc) {
    $count = 0
    Get-ChildItem -Path $skillsSrc -Directory | ForEach-Object {
        $skillName = $_.Name
        $skillFile = Join-Path $_.FullName "SKILL.md"
        if (Test-Path $skillFile) {
            $destDir = Join-Path $SkillsDir $skillName
            Ensure-Dir $destDir
            Copy-Item -Path $skillFile -Destination $destDir -Force
            Write-Success "  $skillName"
            $count++
        }
    }
    Write-Success "Installed $count skills to $SkillsDir"
} else {
    Write-Error "Skills source not found: $skillsSrc"
}

# Install Agents and Commands
Write-Step "Installing agents and commands..."
Ensure-Dir $AgentsDir
Ensure-Dir $CommandsDir
$count = 0

# Agents
$agentsSrc = Join-Path $ProjectRoot "internal\assets\opencode\agents"
if (Test-Path $agentsSrc) {
    Get-ChildItem -Path $agentsSrc -Filter "*.md" | ForEach-Object {
        $agentName = $_.BaseName
        Copy-Item -Path $_.FullName -Destination (Join-Path $AgentsDir "$agentName.md") -Force
        Write-Success "  agent:$agentName"
        $count++
    }
}

# Commands
$cmdsSrc = Join-Path $ProjectRoot "internal\assets\opencode\commands"
if (Test-Path $cmdsSrc) {
    Get-ChildItem -Path $cmdsSrc -Filter "*.md" | ForEach-Object {
        $cmdName = $_.BaseName
        Copy-Item -Path $_.FullName -Destination (Join-Path $CommandsDir "$cmdName.md") -Force
        Write-Success "  cmd:$cmdName"
        $count++
    }
}
Write-Success "Installed $count agents/commands to $OpenCodeDir"

# Install MCP Configuration
Write-Step "Installing MCP configuration..."
Ensure-Dir $OpenCodeDir
Ensure-Dir $McpDir

$settingsPath = Join-Path $OpenCodeDir "opencode.json"
$settings = @{}
if (Test-Path $settingsPath) {
    try {
        $raw = Get-Content -Path $settingsPath -Raw -ErrorAction Stop
        if ($raw -and $raw.Trim() -ne "") {
            $parsed = $raw | ConvertFrom-Json -AsHashtable -ErrorAction Stop
            if ($parsed) {
                $settings = $parsed
            }
        }
    } catch {
        Write-Warning "Invalid JSON at $settingsPath, preserving backup and reinitializing."
        Copy-Item -Path $settingsPath -Destination ($settingsPath + ".backup") -Force
        $settings = @{}
    }
}

if (-not ($settings.ContainsKey("mcp"))) {
    $settings["mcp"] = @{}
}

$settings["mcp"]["security-audit"] = @{
    type    = "local"
    command = @((Join-Path (Join-Path $env:USERPROFILE ".orquestador-auditor\\bin") "orquestador-auditor.exe"), "--mcp")
    enabled = $true
    disabled = $false
}

$settings["mcp"]["chrome-devtools"] = @{
    type    = "local"
    command = @("cmd", "/c", "npx", "-y", "chrome-devtools-mcp@latest")
    enabled = $true
    timeout = 30000
}

$proxyExe = Join-Path $ProxyInstallDir "auditforge-proxy.exe"
if (Test-Path $proxyExe) {
    $proxyCommand = @($proxyExe)
    $proxyDbPath = (Join-Path $ProxyInstallDir "auditforge-proxy.db")
} else {
    $goExe = "C:\\Program Files\\Go\\bin\\go.exe"
    if (Test-Path $goExe) {
        $proxyCommand = @("cmd", "/c", $goExe, "run", (Join-Path $ProjectRoot "cmd\\proxy-server"))
    } else {
        $proxyCommand = @("cmd", "/c", "go", "run", (Join-Path $ProjectRoot "cmd\\proxy-server"))
    }
    $proxyDbPath = (Join-Path $ProjectRoot ".auditforge-proxy.db")
}

$settings["mcp"]["auditforge-proxy"] = @{
    type    = "local"
    command = $proxyCommand
    enabled = $true
    disabled = $false
    environment = @{
        PROXY_PORT = "8080"
        DB_PATH    = $proxyDbPath
    }
}

$settings | ConvertTo-Json -Depth 16 | Set-Content -Path $settingsPath -Encoding UTF8
Write-Success "MCP configuration merged into $settingsPath"

# Completion
Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Success "Installation complete!"
Write-Host ""
Write-Host "  Next steps:"
Write-Host "    1. Restart your AI client (OpenCode/Claude Code)"
Write-Host "    2. Configure browser to use proxy localhost:8080"
Write-Host "    3. Start audit: /team https://target.com"
Write-Host ""
Write-Host "  Locations:"
Write-Host "    Skills:   $SkillsDir"
Write-Host "    Agents:   $AgentsDir"
Write-Host "    Commands: $CommandsDir"
Write-Host "    MCP:      $McpDir"
Write-Host "    Proxy:    $ProxyInstallDir"
Write-Host ""
Write-Host "  Note: Proxy server not built (Go not available)." -ForegroundColor Yellow
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
