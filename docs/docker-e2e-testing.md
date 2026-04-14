# Docker E2E Testing

This repository is ready for containerized integration tests once scanner images are prepared.

Recommended pattern:

1. Build a target container.
2. Mount the workspace.
3. Run the orchestrator in dry-run or real scanner mode.
4. Compare `audit_report.md` with expected findings.
