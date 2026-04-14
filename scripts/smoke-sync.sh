#!/usr/bin/env bash
set -eu
go test ./...
go run ./cmd/orquestador-auditor sync --all
