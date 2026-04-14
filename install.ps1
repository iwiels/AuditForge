$ErrorActionPreference = 'Stop'

$Repo = if ($env:ORQUESTADOR_AUDITOR_REPO) { $env:ORQUESTADOR_AUDITOR_REPO } else { 'victo/orquestador_auditor' }
$Version = if ($env:ORQUESTADOR_AUDITOR_VERSION) { $env:ORQUESTADOR_AUDITOR_VERSION } else { 'latest' }
$InstallDir = if ($env:ORQUESTADOR_AUDITOR_INSTALL_DIR) { $env:ORQUESTADOR_AUDITOR_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\orquestador-auditor' }
$Bundle = $env:ORQUESTADOR_AUDITOR_BUNDLE
$SyncAll = $env:ORQUESTADOR_AUDITOR_SYNC_ALL
$UseWSL = $env:ORQUESTADOR_AUDITOR_USE_WSL

# ============================================================
# WSL Detection & Configuration
# ============================================================
function Test-WSLAvailable {
    <#
    .SYNOPSIS
        Detects if WSL2 is available and functional on the system.
    #>
    try {
        $wslCheck = wsl --status 2>$null
        if ($LASTEXITCODE -eq 0) {
            $wslList = wsl -l -q 2>$null
            if ($LASTEXITCODE -eq 0 -and $wslList) {
                return $true
            }
        }
        return $false
    } catch {
        return $false
    }
}

function Test-WSLDistroReady {
    <#
    .SYNOPSIS
        Checks if a default WSL distro is set and accessible.
    #>
    try {
        $defaultDistro = wsl -l -q 2>$null | Select-Object -First 1
        if ($defaultDistro -and $defaultDistro.Trim()) {
            # Test if we can execute commands in WSL
            $testCmd = wsl echo "WSL_READY" 2>$null
            if ($testCmd -eq 'WSL_READY') {
                return $true
            }
        }
        return $false
    } catch {
        return $false
    }
}

function Invoke-WSLCommand {
    <#
    .SYNOPSIS
        Executes a command inside WSL and returns the output.
    #>
    param([string]$Command)
    try {
        $output = wsl bash -c $Command 2>&1
        return $output
    } catch {
        throw "WSL command failed: $_"
    }
}

$WSLAvailable = Test-WSLAvailable
$WSLReady = if ($WSLAvailable) { Test-WSLDistroReady } else { $false }

# Auto-detect: use WSL if available and not explicitly disabled
if ($UseWSL -eq $null) {
    $UseWSL = $WSLReady
}

if ($UseWSL -and -not $WSLReady) {
    Write-Warning "WSL installation requested but WSL is not ready. Falling back to native Windows installation."
    Write-Host "To enable WSL, run: wsl --install"
    $UseWSL = $false
}

if ($UseWSL) {
    Write-Host "============================================" -ForegroundColor Cyan
    Write-Host " WSL Mode: Installing tools via WSL" -ForegroundColor Cyan
    Write-Host "============================================" -ForegroundColor Cyan
}

$ReleaseFound = $true
if ($Version -eq 'latest') {
  try {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $release.tag_name
  } catch {
    Write-Warning "No GitHub release found for $Repo. Skipping main binary download."
    $ReleaseFound = $false
  }
}

if (-not $Version -and $ReleaseFound) {
  throw "Could not resolve release version for $Repo"
}

$arch = switch ($env:PROCESSOR_ARCHITECTURE.ToLower()) {
  'amd64' { 'amd64' }
  'arm64' { 'arm64' }
  default { throw "Unsupported arch: $env:PROCESSOR_ARCHITECTURE" }
}

if ($ReleaseFound) {
  $archive = "orquestador-auditor_$($Version.TrimStart('v'))_windows_$arch.zip"
  $url = "https://github.com/$Repo/releases/download/$Version/$archive"
  $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
  New-Item -ItemType Directory -Path $tmp | Out-Null
  New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null

  Invoke-WebRequest -Uri $url -OutFile (Join-Path $tmp $archive)
  Expand-Archive -Path (Join-Path $tmp $archive) -DestinationPath $tmp -Force
  Copy-Item (Join-Path $tmp 'orquestador-auditor.exe') (Join-Path $InstallDir 'orquestador-auditor.exe') -Force
  Write-Host "installed orquestador-auditor to $(Join-Path $InstallDir 'orquestador-auditor.exe')"
} else {
  Write-Host "Running in local development mode. Tools will be installed to portable bin."
  $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
  New-Item -ItemType Directory -Path $tmp | Out-Null
}

# Portable Tools Setup
$PortableBinDir = Join-Path $env:USERPROFILE '.orquestador-auditor\bin'
New-Item -ItemType Directory -Path $PortableBinDir -Force | Out-Null
$Env:Path = "$PortableBinDir;$Env:Path"

function Download-GitHubBinary {
    param($Repo, $AssetPattern, $BinaryName)
    Write-Host "Downloading $BinaryName from $Repo..."
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
        # Adjusted pattern matching for various release styles
        $asset = $release.assets | Where-Object { $_.name -like "*$AssetPattern*" -and $_.name -like "*windows*" -and ($_.name -like "*amd64*" -or $_.name -like "*x86_64*") -and $_.name -notlike "*checksums*" } | Select-Object -First 1
        if ($asset) {
            $dlPath = Join-Path $tmp "$($asset.name)"
            Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $dlPath
            if ($dlPath -like "*.zip") {
                Expand-Archive -Path $dlPath -DestinationPath (Join-Path $tmp $BinaryName) -Force
                $exe = Get-ChildItem -Path (Join-Path $tmp $BinaryName) -Filter "$BinaryName.exe" -Recurse | Select-Object -First 1
                if ($exe) { Copy-Item $exe.FullName (Join-Path $PortableBinDir "$BinaryName.exe") -Force }
            } else {
                Copy-Item $dlPath (Join-Path $PortableBinDir "$BinaryName.exe") -Force
            }
            Write-Host "Successfully installed ${BinaryName}"
        }
    } catch {
        Write-Warning "Failed to download ${BinaryName}: $_"
    }
}

function Download-PythonTool {
    param($Repo, $BinaryName, $MainScript)
    Write-Host "Downloading Python tool $BinaryName from $Repo..."
    try {
        $zipUrl = "https://github.com/$Repo/archive/refs/heads/master.zip"
        $dlPath = Join-Path $tmp "$BinaryName.zip"
        Invoke-WebRequest -Uri $zipUrl -OutFile $dlPath
        Expand-Archive -Path $dlPath -DestinationPath $PortableBinDir -Force

        # Create a PowerShell shim
        $repoFolder = (Get-ChildItem -Path $PortableBinDir -Directory | Where-Object { $_.Name -like "*$BinaryName*" })[0].Name
        $shimContent = "python `"`$(Join-Path `$PSScriptRoot '$repoFolder\$MainScript')`" `$args"
        Set-Content -Path (Join-Path $PortableBinDir "$BinaryName.ps1") -Value $shimContent

        # Also create a .bat shim just in case
        $batContent = "@echo off`npython `"%~dp0$repoFolder\$MainScript%`" %*"
        Set-Content -Path (Join-Path $PortableBinDir "$BinaryName.bat") -Value $batContent

        Write-Host "Successfully installed ${BinaryName} (via Python shim)"
    } catch {
        Write-Warning "Failed to install ${BinaryName}: $_"
    }
}

function Install-ToolViaWSL {
    <#
    .SYNOPSIS
        Installs a security tool inside WSL using apt-get.
    .PARAMETER ToolName
        Name of the package to install via apt.
    .PARAMETER DisplayName
        Friendly name for display in output.
    #>
    param(
        [string]$ToolName,
        [string]$DisplayName
    )
    if (-not $DisplayName) { $DisplayName = $ToolName }
    
    Write-Host "[WSL] Installing ${DisplayName} via apt..."
    try {
        $updateOutput = Invoke-WSLCommand "sudo apt-get update -qq"
        $installOutput = Invoke-WSLCommand "sudo apt-get install -y ${ToolName}"
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "[WSL] Successfully installed ${DisplayName}"
            return $true
        } else {
            Write-Warning "[WSL] Failed to install ${DisplayName}: $installOutput"
            return $false
        }
    } catch {
        Write-Warning "[WSL] Exception installing ${DisplayName}: $_"
        return $false
    }
}

function Install-BundleViaWSL {
    <#
    .SYNOPSIS
        Installs a complete bundle of security tools inside WSL.
    .PARAMETER Bundle
        Bundle name: core-web, supply-chain, advanced-web, full
    #>
    param([string]$Bundle)
    
    Write-Host "[WSL] Installing bundle: ${Bundle}" -ForegroundColor Green
    
    # Update apt first
    Write-Host "[WSL] Updating package lists..."
    Invoke-WSLCommand "sudo apt-get update -qq" | Out-Null
    
    switch ($Bundle) {
        'core-web' {
            $tools = @(
                @{Name='nmap'; Display='Nmap'},
                @{Name='whatweb'; Display='WhatWeb'},
                @{Name='sqlmap'; Display='SQLMap'},
                @{Name='nikto'; Display='Nikto'}
            )
        }
        'supply-chain' {
            # These require Go or Rust, install via WSL
            $tools = @()
            Write-Host "[WSL] Installing Go for supply-chain tools..."
            Invoke-WSLCommand "sudo apt-get install -y golang-go" | Out-Null
            
            # Install trivy via GitHub
            Write-Host "[WSL] Installing Trivy..."
            Invoke-WSLCommand "curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin" 2>&1 | Out-Null
            
            # Install gitleaks via GitHub
            Write-Host "[WSL] Installing Gitleaks..."
            Invoke-WSLCommand "curl -sSfL https://raw.githubusercontent.com/gitleaks/gitleaks/master/install.sh | sh -s -- -b /usr/local/bin" 2>&1 | Out-Null
        }
        'advanced-web' {
            $tools = @(
                @{Name='mitmproxy'; Display='mitmproxy'},
                @{Name='python3-pip'; Display='Python3 pip'}
            )
            
            # Install ffuf via GitHub releases in WSL
            Write-Host "[WSL] Installing ffuf..."
            Invoke-WSLCommand "curl -sSfL https://raw.githubusercontent.com/ffuf/ffuf/master/install.sh | sh -s -- -b /usr/local/bin" 2>&1 | Out-Null
            
            # Install Python-based tools
            Write-Host "[WSL] Installing arjun and waymore via pip..."
            Invoke-WSLCommand "pip3 install arjun waymore" 2>&1 | Out-Null
            
            # Install jsluice via npm or go
            Write-Host "[WSL] Installing jsluice via npm..."
            Invoke-WSLCommand "npm install -g jsluice" 2>&1 | Out-Null
        }
        'full' {
            # Install everything
            Install-BundleViaWSL 'core-web'
            Install-BundleViaWSL 'supply-chain'
            Install-BundleViaWSL 'advanced-web'
            return
        }
        default {
            Write-Warning "[WSL] Unknown bundle: ${Bundle}"
            return
        }
    }
    
    # Install individual tools
    foreach ($tool in $tools) {
        Install-ToolViaWSL -ToolName $tool.Name -DisplayName $tool.Display
    }
    
    Write-Host "[WSL] Bundle ${Bundle} installation complete!" -ForegroundColor Green
}

if ($Bundle -eq 'full' -or $Bundle -eq 'core-web' -or $Bundle -eq 'advanced-web' -or $Bundle -eq 'supply-chain') {
    if ($UseWSL) {
        # Use WSL-based installation
        Install-BundleViaWSL $Bundle
    } else {
        # Fallback to native Windows installation
        Write-Host "Installing tools natively on Windows..." -ForegroundColor Yellow
        
        # Go/Binary Tools
    Download-GitHubBinary "projectdiscovery/nuclei" "nuclei" "nuclei"
    Download-GitHubBinary "projectdiscovery/katana" "katana" "katana"
    Download-GitHubBinary "ffuf/ffuf" "ffuf" "ffuf"
    Download-GitHubBinary "gitleaks/gitleaks" "gitleaks" "gitleaks"
    Download-GitHubBinary "anchore/grype" "grype" "grype"
    Download-GitHubBinary "aquasecurity/trivy" "trivy" "trivy"

    # Go-based Tools without GitHub releases (Install via go and move to bin)
    Write-Host "Installing jsluice via go install..."
    try {
        go install github.com/BishopFox/jsluice/cmd/jsluice@latest
        $goPath = go env GOPATH
        $goBin = Join-Path $goPath "bin\jsluice.exe"
        if (Test-Path $goBin) {
            Copy-Item $goBin (Join-Path $PortableBinDir "jsluice.exe") -Force
            Write-Host "✅ Successfully installed jsluice"
        }
    } catch {
        Write-Warning "⚠️ Failed to install jsluice via go install. Ensure Go is in your PATH."
    }

    # Python Tools (Require python installed in system)
    Download-PythonTool "sqlmapproject/sqlmap" "sqlmap" "sqlmap.py"
    Download-PythonTool "s0md3v/Arjun" "Arjun" "arjun.py"
    Download-PythonTool "xnl-h4ck3r/waymore" "waymore" "waymore.py"
    }
}

if ($Bundle) {
  if ($ReleaseFound) {
    & (Join-Path $InstallDir 'orquestador-auditor.exe') install --bundle $Bundle --execute
  } else {
    go run ./cmd/orquestador-auditor install --bundle $Bundle --execute
  }
}

if ($SyncAll -eq 'true') {
  if ($ReleaseFound) {
    & (Join-Path $InstallDir 'orquestador-auditor.exe') sync --all
  } else {
    go run ./cmd/orquestador-auditor sync --all
  }
}
