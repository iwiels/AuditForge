# Skill: tls-vhost-enum

## Mission
Map TLS posture, certificates, and virtual host hints that can expand the attack surface.

## Method
1. Inspect certificate names and SANs.
2. Capture issuer, expiration, protocol/cipher support when available.
3. Derive candidate vhosts and alternate HTTPS surfaces.
4. Feed discovered hostnames back into surface discovery.

## Output
- `observed_cert_hosts`
- `suspected_vhosts`
- `tls_findings`
