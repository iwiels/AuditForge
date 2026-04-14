@echo off
set REPO_PATH=C:\Users\victo\GitHub\orquestador_auditor
cd /d %REPO_PATH%
orquestador-auditor.exe --mcp 2>> %REPO_PATH%\mcp-error.log
