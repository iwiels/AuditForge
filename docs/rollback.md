# Rollback

The engine persists state to `*.state.json` and the final report to `audit_report.md`.

To roll back an audit run:

1. Remove the output directory for that run.
2. Re-run with `--resume` only if you want to continue from saved state.
3. Re-run scanners after fixing the underlying target or config.
