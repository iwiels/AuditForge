# Tasks: Go Security Orchestrator

## Phase 1: Foundation & Universal Schema
- [x] 1.1 Create `internal/model/audit.go` with `AuditState` and `Finding` (UFS) model.
- [x] 1.2 Implement `internal/tools/kali_bin.go` as a generic runner for any binary.
- [x] 1.3 Setup project structure with `go.mod` and idiomatic Go packages.

## Phase 2: Parsers & Data Normalization
- [x] 2.1 Implement parser contracts via Go package boundaries.
- [x] 2.2 Create `internal/parsers/nmap.go` to map Nmap XML results into UFS assets.
- [x] 2.3 Create config scanning helpers for web vulnerability normalization.
- [x] 2.4 Create Searchsploit result parsing helpers.

## Phase 3: Specialized Agent Nodes
- [x] 3.1 Implement recon step using Nmap and project scanning.
- [x] 3.2 Implement web auditor step triggered by HTTP/HTTPS exposure.
- [x] 3.3 Implement correlator step using `searchsploit`.

## Phase 4: AI Reasoning & Filtering
- [x] 4.1 Create `internal/llm/client.go`: Wrapper for pluggable review calls.
- [x] 4.2 Implement critical review step for false-positive detection.
- [x] 4.3 Design review prompt/default remediation heuristics.

## Phase 5: Weaponization (Metasploit)
- [x] 5.1 Implement Metasploit RC generation for confirmed findings.
- [x] 5.2 Create resource file storage logic in `outputs/msf/`.

## Phase 6: Core Graph & CLI
- [x] 6.1 Define the final Go pipeline including Review and MSF phases.
- [x] 6.2 Implement CLI flags for choosing review provider (e.g., `--provider static`).
- [x] 6.3 Update report generation to include review insights and MSF script paths.
