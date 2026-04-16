# Audit Scenarios

## Scenario: Leak Hunt

Input: repository path

Tools: Gitleaks, Semgrep, Trivy, Grype

Expected output: secrets, code smells, and package vulnerabilities.

## Scenario: Web Triage

Input: URL or host

Tools: WhatWeb, Katana, Nuclei, Nikto, Sqlmap

Expected output: fingerprinted technologies, discovered routes, and structured issues.

## Scenario: Service Correlation

Input: host with open ports

Tools: Nmap, Searchsploit

Expected output: service versions linked to local exploit references.
