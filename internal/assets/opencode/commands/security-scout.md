---
description: Recon y surface discovery — primer agente del pipeline
agent: security-scout
---

Ejecutá recon completo del target autorizado. Sos el primer agente — tus hallazgos guían a security-web y security-code.

**Antes de empezar:**
1. Verificá `SESSION_ID` y `TARGET` del contexto (el orquestador los pasó)
2. `memory.search("authorized engagement [target]")` — si no hay autorización, detenete
3. `memory.search("[SESSION_ID] findings agent:security-memory")` — cargar contexto histórico

**Qué analizar:**

OSINT pasivo (sin tocar el target):
- Certificate Transparency: subdominios via crt.sh
- DNS: SPF/DMARC/DKIM, MX records, historial
- Shodan/Censys: puertos e infra ya indexada
- GitHub: `"[target]" password`, `org:[empresa] filename:.env`
- Wayback Machine: endpoints históricos

Surface discovery (pasivo sobre el target):
- Headers HTTP: Server, X-Powered-By, Set-Cookie flags, CSP, HSTS
- Archivos de fingerprinting: /robots.txt, /security.txt, /.well-known/, /sitemap.xml
- Rutas API obvias: /api/, /api/v1/, /graphql, /swagger.json, /openapi.yaml
- Error pages: ¿revelan stack o versiones?

**Escribir cada hallazgo a memoria:**
```
memory.save({
  kind: "finding", agent: "security-scout", session: "[SESSION_ID]",
  target: "[TARGET]", title: "...", severity: "...", status: "observed|suspected",
  evidence: "...", vector: "...",
  needs_followup_by: ["security-web"]  // si aplica
})
```

**Al terminar:**
```
memory.save({ kind: "phase-complete", agent: "security-scout", session: "[SESSION_ID]",
  summary: "[N hallazgos, assets identificados, stack detectado]" })
```

Skills: surface-discovery, osint-passive, authorization-guard
