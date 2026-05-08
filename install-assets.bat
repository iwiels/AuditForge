@echo off
setlocal EnableDelayedExpansion

:: AuditForge Assets Installer - Windows Batch
:: Installs skills, agents, MCP config, and builds the proxy server

set "OPENCODE_DIR=%USERPROFILE%\.config\opencode"
set "SKILLS_DIR=%USERPROFILE%\.auditforge\skills"
set "AUDITFORGE_DIR=%USERPROFILE%\.auditforge"
set "PROJECT_ROOT=%~dp0"

if "%~1"=="" goto install
if "%~1"=="--skills" goto install_skills
if "%~1"=="--agents" goto install_agents
if "%~1"=="--mcp" goto install_mcp
if "%~1"=="--proxy" goto install_proxy
if "%~1"=="--help" goto help
goto help

:help
echo.
echo ============================================
echo   AuditForge - Assets Installer
echo ============================================
echo.
echo Usage: %~nx0 [OPTIONS]
echo.
echo Options:
echo   --skills    Install skills only
echo   --agents    Install agents only
echo   --mcp       Install MCP config only
echo   --proxy     Build and install proxy server only
echo   (no args)   Install all
echo   --help      Show this help
echo.
echo This script will:
echo   1. Install 29 security skills
echo   2. Install agent definitions
echo   3. Configure MCP servers
echo   4. Build the AuditForge Proxy (requires Go)
echo   5. Generate SSL certificates for HTTPS interception
echo.
exit /b 0

:install
echo.
echo ============================================
echo   AuditForge - Assets Installer
echo ============================================
echo.
echo [INFO] Platform: Windows
echo [INFO] Skills: %SKILLS_DIR%
echo [INFO] Target: %OPENCODE_DIR%
echo [INFO] Proxy:  %AUDITFORGE_DIR%\proxy
echo.

:: Install everything
call :install_skills
call :install_agents
call :install_mcp
call :install_proxy
call :show_completion
exit /b 0

:install_skills
echo [STEP] Installing skills...

set "SKILLS_SRC=%PROJECT_ROOT%internal\assets\skills"

if not exist "%SKILLS_DIR%" mkdir "%SKILLS_DIR%"

if not exist "%SKILLS_SRC%" (
    echo [ERROR] Skills source not found: %SKILLS_SRC%
    exit /b 1
)

set "skill_count=0"
for /d %%D in ("%SKILLS_SRC%\*") do (
    set "skill_name=%%~nxD"
    if exist "%%D\SKILL.md" (
        if not exist "%SKILLS_DIR%\!skill_name!" mkdir "%SKILLS_DIR%\!skill_name!"
        copy /Y "%%D\SKILL.md" "%SKILLS_DIR%\!skill_name!\" >nul
        echo   [OK] !skill_name!
        set /a skill_count+=1
    )
)
echo.
echo [OK] Installed !skill_count! skills
echo.
if "%~1"=="--skills" goto show_completion
exit /b 0

:install_agents
echo [STEP] Installing agents and commands...

if not exist "%OPENCODE_DIR%" mkdir "%OPENCODE_DIR%"
set "AGENTS_DIR=%OPENCODE_DIR%\agents"
set "CMDS_DIR=%OPENCODE_DIR%\commands"

if not exist "%AGENTS_DIR%" mkdir "%AGENTS_DIR%"
if not exist "%CMDS_DIR%" mkdir "%CMDS_DIR%"

set "AGENTS_SRC=%PROJECT_ROOT%internal\assets\opencode\agents"
set "CMDS_SRC=%PROJECT_ROOT%internal\assets\opencode\commands"

set "agent_count=0"
set "cmd_count=0"

if exist "%AGENTS_SRC%" (
    for %%F in ("%AGENTS_SRC%\*.md") do (
        set "agent_name=%%~nF"
        copy /Y "%%F" "%AGENTS_DIR%\!agent_name!.md" >nul
        echo   [OK] agent:!agent_name!
        set /a agent_count+=1
    )
)

if exist "%CMDS_SRC%" (
    for %%F in ("%CMDS_SRC%\*.md") do (
        set "cmd_name=%%~nF"
        copy /Y "%%F" "%CMDS_DIR%\!cmd_name!.md" >nul
        echo   [OK] cmd:!cmd_name!
        set /a cmd_count+=1
    )
)
echo.
echo [OK] Installed !agent_count! agents, !cmd_count! commands
echo.
if "%~1"=="--agents" goto show_completion
exit /b 0

:install_mcp
echo [STEP] Installing MCP configuration...

if not exist "%OPENCODE_DIR%\mcp" mkdir "%OPENCODE_DIR%\mcp"

:: Chrome DevTools MCP
echo { > "%OPENCODE_DIR%\mcp\chrome-devtools.json"
echo   "mcpServers": { >> "%OPENCODE_DIR%\mcp\chrome-devtools.json"
echo     "chrome-devtools": { >> "%OPENCODE_DIR%\mcp\chrome-devtools.json"
echo       "command": "npx", >> "%OPENCODE_DIR%\mcp\chrome-devtools.json"
echo       "args": ["-y", "@modelcontextprotocol/server-chrome-devtools"] >> "%OPENCODE_DIR%\mcp\chrome-devtools.json"
echo     } >> "%OPENCODE_DIR%\mcp\chrome-devtools.json"
echo   } >> "%OPENCODE_DIR%\mcp\chrome-devtools.json"
echo } >> "%OPENCODE_DIR%\mcp\chrome-devtools.json"

echo   [OK] chrome-devtools.json

:: AuditForge Proxy MCP
echo { > "%OPENCODE_DIR%\mcp\auditforge-proxy.json"
echo   "mcpServers": { >> "%OPENCODE_DIR%\mcp\auditforge-proxy.json"
echo     "auditforge-proxy": { >> "%OPENCODE_DIR%\mcp\auditforge-proxy.json"
echo       "command": "%AUDITFORGE_DIR:\=\\%\\proxy\\auditforge-proxy.exe", >> "%OPENCODE_DIR%\mcp\auditforge-proxy.json"
echo       "args": [], >> "%OPENCODE_DIR%\mcp\auditforge-proxy.json"
echo       "env": { >> "%OPENCODE_DIR%\mcp\auditforge-proxy.json"
echo         "PROXY_PORT": "8080", >> "%OPENCODE_DIR%\mcp\auditforge-proxy.json"
echo         "DB_PATH": "%AUDITFORGE_DIR:\=\\%\\proxy\\auditforge-proxy.db" >> "%OPENCODE_DIR%\mcp\auditforge-proxy.json"
echo       } >> "%OPENCODE_DIR%\mcp\auditforge-proxy.json"
echo     } >> "%OPENCODE_DIR%\mcp\auditforge-proxy.json"
echo   } >> "%OPENCODE_DIR%\mcp\auditforge-proxy.json"
echo } >> "%OPENCODE_DIR%\mcp\auditforge-proxy.json"

echo   [OK] auditforge-proxy.json
echo.
echo [OK] MCP configs installed to %OPENCODE_DIR%\mcp
echo.
if "%~1"=="--mcp" goto show_completion
exit /b 0

:install_proxy
echo [STEP] Building AuditForge Proxy Server...

set "PROXY_DIR=%PROJECT_ROOT%cmd\proxy-server"
set "PROXY_INSTALL_DIR=%AUDITFORGE_DIR%\proxy"

if not exist "%PROXY_INSTALL_DIR%" mkdir "%PROXY_INSTALL_DIR%"

:: Check for Go
where go >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [WARN] Go not found. Proxy server will not be built.
    echo [WARN] Install Go from https://golang.org/dl/ and re-run this script.
    echo.
    if "%~1"=="--proxy" goto show_completion
    exit /b 0
)

echo   [INFO] Go version:
for /f "tokens=*" %%a in ('go version') do echo     %%a

:: Check for OpenSSL
where openssl >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [WARN] OpenSSL not found. SSL certificates will not be generated.
    echo [WARN] Install OpenSSL or use the pre-built binary.
    echo.
)

:: Build proxy
echo   [INFO] Building proxy server...
cd /d "%PROXY_DIR%"

:: Initialize go.mod if not exists
if not exist "go.mod" (
    go mod init auditforge-proxy 2>nul
    go get github.com/google/uuid 2>nul
    go get github.com/mark3labs/mcp-go 2>nul
    go get github.com/mattn/go-sqlite3 2>nul
)

:: Build
go build -o "%PROXY_INSTALL_DIR%\auditforge-proxy.exe" . 2>nul
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Failed to build proxy server
    echo [INFO] You may need to install a C compiler for SQLite3 support
    echo [INFO] Or download pre-built binaries from releases
    cd /d "%PROJECT_ROOT%"
    if "%~1"=="--proxy" goto show_completion
    exit /b 0
)

echo   [OK] Proxy built: %PROXY_INSTALL_DIR%\auditforge-proxy.exe

:: Generate certificates
if exist "%PROXY_DIR%\setup.sh" (
    echo   [INFO] Copying certificate setup script...
    copy /Y "%PROXY_DIR%\setup.sh" "%PROXY_INSTALL_DIR%\setup.sh" >nul
)

echo   [INFO] Generating SSL certificates...
if not exist "%PROXY_INSTALL_DIR%\certs" mkdir "%PROXY_INSTALL_DIR%\certs"

:: Generate CA cert with OpenSSL if available
where openssl >nul 2>nul
if %ERRORLEVEL% equ 0 (
    openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes ^
        -keyout "%PROXY_INSTALL_DIR%\certs\ca.key" ^
        -out "%PROXY_INSTALL_DIR%\certs\ca.crt" ^
        -subj "/CN=AuditForge Proxy CA" ^
        -addext "basicConstraints=critical,CA:TRUE" 2>nul
    
    if %ERRORLEVEL% equ 0 (
        echo   [OK] CA certificate generated
    ) else (
        echo   [WARN] Failed to generate CA certificate
    )
) else (
    echo   [WARN] OpenSSL not available - certificates must be generated manually
)

:: Create start script
echo @echo off > "%PROXY_INSTALL_DIR%\start-proxy.bat"
echo echo Starting AuditForge Proxy Server... >> "%PROXY_INSTALL_DIR%\start-proxy.bat"
echo echo Proxy: http://localhost:8080 >> "%PROXY_INSTALL_DIR%\start-proxy.bat"
echo echo. >> "%PROXY_INSTALL_DIR%\start-proxy.bat"
echo "%~dp0auditforge-proxy.exe" >> "%PROXY_INSTALL_DIR%\start-proxy.bat"

echo   [OK] Start script created: start-proxy.bat
echo.
echo [OK] Proxy server installed to %PROXY_INSTALL_DIR%
echo.
if "%~1"=="--proxy" goto show_completion
exit /b 0

:show_completion
echo ============================================
echo   Installation Complete!
echo ============================================
echo.
echo   Next steps:
echo   1. Restart your AI client (OpenCode)
echo   2. Configure browser to use proxy localhost:8080
echo   3. Start audit: /team https://target.com
echo.
echo   Locations:
echo   - Skills:   %SKILLS_DIR%
echo   - Agents:   %OPENCODE_DIR%\agents
echo   - MCP:      %OPENCODE_DIR%\mcp
echo   - Proxy:    %AUDITFORGE_DIR%\proxy
echo.
if exist "%AUDITFORGE_DIR%\proxy\auditforge-proxy.exe" (
    echo   Proxy server is ready to use!
    echo   To start: %AUDITFORGE_DIR%\proxy\start-proxy.bat
    echo.
    echo   MCP Tools Available:
    echo     • proxy.intercept.enable	echo     • proxy.history.search	echo     • proxy.request.modify	echo     • proxy.findings.list	echo.
) else (
    echo   [WARNING] Proxy server not built. Install Go and re-run with --proxy
)
echo ============================================
echo.
exit /b 0
