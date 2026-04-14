# Skill: network-recon

## Mission
Enumerate authorized network exposure before assuming the target is only web-facing.

## Method
1. Confirm authorization, target kind, and aggressiveness.
2. Enumerate TCP ports and service banners with bounded `nmap` usage.
3. Capture service/version evidence and map likely protocols.
4. Identify TLS endpoints, alternate HTTP ports, and admin surfaces.
5. Return a structured inventory: port, service, version, evidence, next phase.

## Output
- `observed_services`
- `suspected_web_surfaces`
- `next_recommended`

## Guardrails
- No destructive NSE use by default.
- No UDP spray or aggressive scans unless explicitly allowed.
