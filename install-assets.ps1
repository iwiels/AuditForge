#Requires -Version 5.1
<#
.SYNOPSIS
    AuditForge Assets Installer - PowerShell Edition
    Installs skills, agents, MCP config, and builds the proxy server

.DESCRIPTION
    This script automates the installation of AuditForge assets including:
    - 29 security skills
    - Agent definitions
    - MCP server configurations
    - AuditForge Proxy Server (requires Go)
    - SSL certificates for HTTPS interception

.PARAMETER Skills
    Install skills only

.PARAMETER Agents
    Install agents only

.PARAMETER Mcp
    Install MCP config only

.PARAMETER Proxy
    Build and install proxy server only

.PARAMETER All
    Full install (default)

.PARAMETER Uninstall
    Uninstall all assets

.EXAMPLE
    .\install-assets.ps1
    Full installation

.EXAMPLE
    .\install-assets.ps1 -Proxy
    Build proxy server only

.NOTES
    Version: 2.0
    Author: AuditForge Team
    Requires: Windows PowerShell 5.1 or PowerShell 7+
#>

[CmdletBinding()]
param(
    [switch]$Skills,
    [switch]$Agents,
    [switch]$Mcp,
    [switch]$Proxy,
    [switch]$All = $true,
    [switch]$Uninstall,
    [switch]$Help
)

# Override $All if any specific option is selected
if ($Skills -or $Agents -or $Mcp -or $Proxy) {
    $All = $false
}

if ($Help) {
    Get-Help $MyInvocation.MyCommand.Path -Detailed
    exit 0
}

# Configuration
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = $ScriptDir
$OpenCodeDir = Join-Path $env:USERPROFILE ".config\opencode"
$SkillsDir = Join-Path $env:USERPROFILE ".auditforge\skills"
$AuditForgeDir = Join-Path $env:USERPROFILE ".auditforge"
$ProxyInstallDir = Join-Path $AuditForgeDir "proxy"
$AgentsDir = Join-Path $OpenCodeDir "agents"
$CommandsDir = Join-Path $OpenCodeDir "commands"
$McpDir = Join-Path $OpenCodeDir "mcp"

# Colors for output
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

# Banner
function Show-Banner {
    Write-Host ""
    Write-Host "╔═══════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "║     AuditForge - Assets Installer v2.0 (PowerShell)     ║" -ForegroundColor Cyan
    Write-Host "╚═══════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
    Write-Host ""
}

# Ensure directory exists
function Ensure-Dir($Path) {
    if (!(Test-Path $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

# Check dependencies
function Test-Dependencies {
    $deps = @()
    
    if (!(Get-Command git -ErrorAction SilentlyContinue)) {
        $deps += "git"
    }
    
    if ($deps.Count -gt 0) {
        Write-Error "Missing dependencies: $($deps -join ', ')"
        exit 1
    }
}

# Install Skills
function Install-Skills {
    Write-Step "Installing skills..."
    
    Ensure-Dir $SkillsDir
    
    $skillsSrc = Join-Path $ProjectRoot "internal\assets\skills"
    if (!(Test-Path $skillsSrc)) {
        Write-Error "Skills source not found: $skillsSrc"
        return
    }
    
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
}

# Install Agents and Commands
function Install-Agents {
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
}

# Install MCP Configurations
function Install-Mcp {
    Write-Step "Installing MCP configuration..."
    
    Ensure-Dir $McpDir
    
    # Chrome DevTools MCP
    $chromeConfig = @{
        mcpServers = @{
            "chrome-devtools" = @{
                command = "npx"
                args = @("-y", "@modelcontextprotocol/server-chrome-devtools")
            }
        }
    }
    
    $chromeConfig | ConvertTo-Json -Depth 10 | Set-Content -Path (Join-Path $McpDir "chrome-devtools.json")
    Write-Success "  chrome-devtools.json"
    
    # AuditForge Proxy MCP
    $proxyPath = $ProxyInstallDir -replace '\\', '\\'
    $proxyConfig = @{
        mcpServers = @{
            "auditforge-proxy" = @{
                command = (Join-Path $ProxyInstallDir "auditforge-proxy.exe")
                args = @()
                env = @{
                    PROXY_PORT = "8080"
                    DB_PATH = (Join-Path $ProxyInstallDir "auditforge-proxy.db")
                }
            }
        }
    }
    
    $proxyConfig | ConvertTo-Json -Depth 10 | Set-Content -Path (Join-Path $McpDir "auditforge-proxy.json")
    Write-Success "  auditforge-proxy.json"
    
    # Try to verify npx
    if (Get-Command npx -ErrorAction SilentlyContinue) {
        Write-Info "Verifying chrome-devtools MCP server..."
        try {
            $null = npx -y @modelcontextprotocol/server-chrome-devtools --version 2>&1
            Write-Success "MCP server ready"
        }
        catch {
            Write-Warning "MCP server not available - will be installed on first use"
        }
    }
    
    Write-Success "MCP configuration installed to $McpDir"
}

# Build and Install Proxy Server
function Install-Proxy {
    Write-Step "Building AuditForge Proxy Server..."
    
    Ensure-Dir $ProxyInstallDir
    Ensure-Dir (Join-Path $ProxyInstallDir "certs")
    
    $proxySrc = Join-Path $ProjectRoot "cmd\proxy-server"
    
    # Check for Go
    $goCmd = Get-Command go -ErrorAction SilentlyContinue
    if (!$goCmd) {
        Write-Warning "Go not found. Proxy server will not be built."
        Write-Warning "Install Go from https://golang.org/dl/ and re-run with -Proxy"
        Write-Host ""
        return
    }
    
    Write-Info "Go version:"
    go version | ForEach-Object { Write-Host "    $_" }
    
    # Check for OpenSSL
    $hasOpenSSL = $null -ne (Get-Command openssl -ErrorAction SilentlyContinue)
    
    # Build proxy
    Write-Info "Building proxy server..."
    Push-Location $proxySrc
    
    try {
        # Initialize go.mod if not exists
        if (!(Test-Path "go.mod")) {
            go mod init auditforge-proxy 2>$null
            go get github.com/google/uuid 2>$null
            go get github.com/mark3labs/mcp-go 2>$null
            go get github.com/mattn/go-sqlite3 2>$null
        }
        
        # Build
        $output = Join-Path $ProxyInstallDir "auditforge-proxy.exe"
        go build -o $output . 2>&1
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Proxy built: $output"
        }
        else {
            Write-Error "Failed to build proxy server"
            Write-Info "You may need to install a C compiler for SQLite3 support"
            Write-Info "Or download pre-built binaries from releases"
            return
        }
    }
    finally {
        Pop-Location
    }
    
    # Generate certificates
    Write-Info "Generating SSL certificates..."
    
    if ($hasOpenSSL) {
        $caKey = Join-Path $ProxyInstallDir "certs\ca.key"
        $caCert = Join-Path $ProxyInstallDir "certs\ca.crt"
        
        openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes `
            -keyout $caKey -out $caCert `
            -subj "/CN=AuditForge Proxy CA" `
            -addext "basicConstraints=critical,CA:TRUE" 2>$null
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "CA certificate generated"
        }
        else {
            Write-Warning "Failed to generate CA certificate"
        }
    }
    else {
        Write-Warning "OpenSSL not available - certificates must be generated manually"
    }
    
    # Copy setup script
    $setupSh = Join-Path $proxySrc "setup.sh"
    if (Test-Path $setupSh) {
        Copy-Item -Path $setupSh -Destination $ProxyInstallDir -Force
    }
    
    # Create start script
    $startScript = @"
@echo off
echo Starting AuditForge Proxy Server...
echo Proxy: http://localhost:8080
echo Press Ctrl+C to stop
echo.
"%~dp0auditforge-proxy.exe"
"@
    
    Set-Content -Path (Join-Path $ProxyInstallDir "start-proxy.bat") -Value $startScript
    Write-Success "Start script created: start-proxy.bat"
    
    Write-Success "Proxy server installed to $ProxyInstallDir"
}

# Uninstall
function Uninstall-All {
    Write-Warning "Uninstalling AuditForge assets..."
    
    $paths = @($SkillsDir, $AgentsDir, $CommandsDir, $ProxyInstallDir)
    
    foreach ($path in $paths) {
        if (Test-Path $path) {
            Remove-Item -Path $path -Recurse -Force
            Write-Success "Removed: $path"
        }
    }
    
    Write-Success "Uninstall complete"
}

# Show completion message
function Show-Completion {
    Write-Host ""
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
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
    
    $proxyExe = Join-Path $ProxyInstallDir "auditforge-proxy.exe"
    if (Test-Path $proxyExe) {
        Write-Host "  🎯 Proxy server is ready to use!" -ForegroundColor Green
        Write-Host "     To start: $ProxyInstallDir\start-proxy.bat"
        Write-Host ""
        Write-Host "  🔧 MCP Tools Available:"
        Write-Host "     • proxy.intercept.enable"
        Write-Host "     • proxy.history.search"
        Write-Host "     • proxy.request.modify"
        Write-Host "     • proxy.replay.execute"
        Write-Host "     • proxy.findings.list"
        Write-Host ""
    }
    else {
        Write-Host "  ⚠️  Proxy server not built. Install Go and re-run with -Proxy" -ForegroundColor Yellow
    }
    
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
}

# Main execution
Show-Banner

Write-Info "Platform: Windows PowerShell"
Write-Info "Project:  $ProjectRoot"
Write-Info "Target:   $OpenCodeDir"
Write-Info "Proxy:    $ProxyInstallDir"
Write-Host ""

if ($Uninstall) {
    Uninstall-All
    exit 0
}

Test-Dependencies

if ($All -or $Skills) {
    Install-Skills
}

if ($All -or $Agents) {
    Install-Agents
}

if ($All -or $Mcp) {
    Install-Mcp
}

if ($All -or $Proxy) {
    Install-Proxy
}

Show-Completion
