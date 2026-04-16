@echo off
setlocal EnableDelayedExpansion

set "OPENCODE_DIR=%USERPROFILE%\.config\opencode"
set "SKILLS_DIR=%USERPROFILE%\.auditforge\skills"

if "%~1"=="" goto install
if "%~1"=="--skills" goto install_skills
if "%~1"=="--agents" goto install_agents
if "%~1"=="--mcp" goto install_mcp
if "%~1"=="--help" goto help
goto help

:help
echo Usage: %~nx0 [OPTIONS]
echo.
echo Options:
echo   --skills    Install skills only
echo   --agents    Install agents only
echo   --mcp       Install MCP config only
echo   (no args)   Install all
echo   --help      Show this help
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
echo.

:install_skills
echo [STEP] Installing skills...

set "SKILLS_SRC=%~dp0internal\assets\skills"

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

:install_agents
echo [STEP] Installing agents and commands...

if not exist "%OPENCODE_DIR%" mkdir "%OPENCODE_DIR%"
set "AGENTS_DIR=%OPENCODE_DIR%\agents"
set "CMDS_DIR=%OPENCODE_DIR%\commands"

if not exist "%AGENTS_DIR%" mkdir "%AGENTS_DIR%"
if not exist "%CMDS_DIR%" mkdir "%CMDS_DIR%"

set "AGENTS_SRC=%~dp0internal\assets\opencode\agents"
set "CMDS_SRC=%~dp0internal\assets\opencode\commands"

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

:install_mcp
echo [STEP] Installing MCP configuration...

if not exist "%OPENCODE_DIR%\mcp" mkdir "%OPENCODE_DIR%\mcp"

echo { > "%OPENCODE_DIR%\mcp\chrome-devtools.json"
echo   "mcpServers": { >> "%OPENCODE_DIR%\mcp\chrome-devtools.json"
echo     "chrome-devtools": { >> "%OPENCODE_DIR%\mcp\chrome-devtools.json"
echo       "command": "npx", >> "%OPENCODE_DIR%\mcp\chrome-devtools.json"
echo       "args": ["-y", "@modelcontextprotocol/server-chrome-devtools"] >> "%OPENCODE_DIR%\mcp\chrome-devtools.json"
echo     } >> "%OPENCODE_DIR%\mcp\chrome-devtools.json"
echo   } >> "%OPENCODE_DIR%\mcp\chrome-devtools.json"
echo } >> "%OPENCODE_DIR%\mcp\chrome-devtools.json"

echo [OK] MCP config: %OPENCODE_DIR%\mcp\chrome-devtools.json
echo.

echo ============================================
echo   Installation complete!
echo ============================================
echo.
echo   Next steps:
echo   1. Restart your AI client
echo   2. Verify: /memory-search test
echo   3. Start audit: /team https://target.com
echo.
exit /b 0
