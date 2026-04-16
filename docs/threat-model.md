# Threat Model

## Trust Boundaries

- CLI arguments are untrusted.
- Scanner output is untrusted.
- Report content is derived from untrusted data and must be treated as evidence, not truth.

## Main Risks

- prompt injection through target content
- command execution against untrusted inputs
- false positives from scanners
- accidental weaponization of weak evidence
