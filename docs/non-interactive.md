# Non-Interactive Mode

Use `--dry-run` to validate orchestration without executing scanners.

Use `--mcp` when embedding the orchestrator behind a client that speaks MCP over stdio.

## CI Pattern

```bash
go test ./...
go run ./cmd/orquestador-auditor --target ./repo --dry-run
```
