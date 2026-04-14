# orquestador-auditor

Plataforma OpenCode-first para orquestar auditorias defensivas asistidas por AI.

El objetivo no es ser "otro launcher de herramientas", sino el **Gentle AI de ciberseguridad sobre OpenCode**:

- **methodology-first**: OWASP, PTES, OSSTMM, threat modeling y secure design review
- **authorization-first**: todo engagement requiere alcance y autorizacion explicita
- **evidence-first**: cada hallazgo debe quedar respaldado por evidencia verificable
- **defensive-by-default**: prioriza reconocimiento, analisis y remediacion sobre explotacion
- **memory-enabled**: conserva contexto, hallazgos y decisiones operativas entre sesiones
- **policy-driven profiles**: cada perfil define que assets se instalan y que acciones quedan permitidas

## Pipeline metodologico integrado

El orquestador ahora se orienta a este flujo por fases:

1. **Scope** - autorizacion, target kind, agresividad permitida
2. **Network Recon** - puertos, servicios, versiones, TLS/vhosts
3. **Surface Discovery** - crawling, historial, fingerprint, tecnologias
4. **JS / Client-Side Intel** - beautify, jsluice, source maps, browser capture
5. **API / Parameter Discovery** - OpenAPI, arjun, ffuf, schema normalization
6. **Vuln Hypothesis** - authz, IDOR, SQLi, XSS, SSRF, command injection, leaks
7. **Authorized Validation** - sqlmap acotado y confirmaciones puntuales
8. **Correlation** - severidad, CWE, OWASP mapping, remediation

Hoy el proyecto ya cubre tres capacidades base:

1. **`install`** - resuelve como instalar herramientas de seguridad en la maquina.
2. **`sync`** - inyecta en **OpenCode** prompts, skills, comandos, agentes y MCP orientados a ciberseguridad.
3. **`run`** - crea artefactos estructurados por fase, aplica policy gating y deja correlacion reproducible.
4. **`memory` / `--mcp`** - expone memoria persistente y herramientas MCP para que los agentes trabajen con contexto reutilizable.

> Estado del producto: **OpenCode es el runtime principal**. La compatibilidad con otros clientes queda como modo avanzado/legacy, no como foco del producto.

---

## Bootstrap recomendado

```powershell
orquestador-auditor install --bundle full
orquestador-auditor sync --profile web-triage
orquestador-auditor run start --target https://example.com --profile web-triage --authorized --aggressiveness bounded --approved-tools nmap,katana
```

---

## Sync

`sync` apunta a **OpenCode por defecto** y aplica un **perfil de auditoria**.

```bash
orquestador-auditor sync --profile recon
orquestador-auditor sync --profile web-triage
orquestador-auditor sync --profile supply-chain
orquestador-auditor sync --profile reporting
orquestador-auditor sync --profile memory-only
```

Cada perfil controla dos cosas:

1. **Que assets se instalan**
   - prompts
   - commands
   - skills
   - MCP
   - overlays / agentes

2. **Que acciones estan permitidas**
   - read / write / edit / bash en agentes OpenCode
   - guardrails en prompts
   - metadata de riesgo en MCP

Los agentes especialistas del orquestador quedan **siempre registrados** en OpenCode; el perfil ya no los elimina, solo les cambia permisos y conducta.

#### Perfiles disponibles

| Perfil | Modo | Que permite |
|--------|------|-------------|
| `recon` | passive-first | alcance, red, superficie y enumeracion pasiva |
| `web-triage` | bounded-active-analysis | analisis web profundo, JS/API intel y validacion acotada |
| `supply-chain` | read-heavy-code-audit | revision de codigo, dependencias, CI/CD y correlation |
| `reporting` | non-operational-reporting | consolidacion, severidad, CWE/OWASP y remediacion |
| `memory-only` | context-only | continuidad operativa, memoria y handoff |

---

## OpenCode runtime

Despues de `sync`, OpenCode recibe:

- `AGENTS.md` con politica del perfil activo y el pipeline metodologico
- commands por fase como `network-recon`, `deep-web`, `js-intel`, `api-discovery`, `sqli-validate`, `correlate-findings`
- skills filtradas por perfil
- overlay de agentes con permisos `read/write/edit/bash` alineados al perfil
- MCP local con metadata de riesgo y perfiles

## Runtime operativo por fases

El salto nuevo ya no es solo metodologico: ahora existe un runtime que guarda artefactos por fase en `.orquestador/runs/<run-id>/`.

```bash
go run ./cmd/orquestador-auditor run start --target https://example.com --profile web-triage --authorized --aggressiveness bounded --approved-tools katana,arjun
go run ./cmd/orquestador-auditor run phase --run-id 20260409-120000-example-com --phase network-recon --status observed --requested-tools nmap
go run ./cmd/orquestador-auditor run correlate --run-id 20260409-120000-example-com
```

### Que hace `run`

- crea `run.json` con autorizacion, target kind, perfil y agresividad
- genera artefactos JSON por fase (`scope`, `network-recon`, `surface-discovery`, `js-intel`, `api-discovery`, `vuln-hypothesis`, `authorized-validation`, `correlation`)
- bloquea herramientas como `sqlmap`, `ffuf` o `arjun` si no hay policy explicita
- fuerza que `authorized-validation` tenga hipotesis previas antes de aceptar validacion
- hace que la fase de `correlation` consuma findings estructurados y complete CWE / OWASP / remediation

---

## Documentacion relacionada

- [Architecture](docs/architecture.md)
- [Usage](docs/usage.md)
- [Tool Matrix](docs/tool-matrix.md)
- [Security Ops](docs/security-ops.md)

## Tests

```bash
go test ./...
```
