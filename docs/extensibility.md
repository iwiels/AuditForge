# Extensibility

Add new tools by following the existing pattern:

1. Add a wrapper in `internal/tools`.
2. Add a parser in `internal/parsers`.
3. Wire the stage into `internal/orchestrator/stages`.
4. Document the capability in `docs/tool-matrix.md`.
