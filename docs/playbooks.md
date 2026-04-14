# Playbooks

## 1. Project Repository Audit

Goal: find code issues, leaked secrets, vulnerable dependencies, and container hardening gaps.

Flow:
1. `Scout` detects stack and manifests.
2. `Fingerprint` runs Semgrep.
3. `Secrets` runs Gitleaks.
4. `Supply Chain` runs Trivy and Grype.
5. `Report` produces a triaged Markdown summary.

## 2. Web Application Audit

Goal: fingerprint tech, crawl URLs, probe surface, and collect evidence for controlled review.

Flow:
1. `Scout` resolves URL or host surface.
2. `WhatWeb` fingerprints technologies.
3. `Katana` discovers URLs.
4. `Nuclei` checks discovered endpoints.
5. `Nikto` and `Sqlmap` validate exposed issues under authorization.

## 3. Host Audit

Goal: identify exposed services, correlate versions, and create evidence-backed follow-up items.

Flow:
1. `Nmap` enumerates open services.
2. `Searchsploit` correlates versions.
3. `Report` summarizes attack surface and follow-up steps.
