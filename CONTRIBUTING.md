# Contributing

## Development

1. Run `go test ./...` before proposing changes.
2. Keep security behavior defensive and authorization-aware.
3. Do not introduce automatic exploitation flows.
4. When touching client integration code, preserve existing user config whenever possible.

## Pull Requests

- Describe the risk reduction or operator value of the change.
- Mention config, backup, or compatibility impacts.
- Include test coverage for new integration behavior.
