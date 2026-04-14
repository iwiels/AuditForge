# Usage

## CLI

Modelo recomendado:

- instalar backend y bundles
- sincronizar **OpenCode** con el perfil correcto
- trabajar desde OpenCode con el orquestador y sus agentes especializados

## Sync

Baseline de reconocimiento:

```bash
go run ./cmd/orquestador-auditor sync --profile recon
```

Pipeline web profundo recomendado:

```bash
go run ./cmd/orquestador-auditor sync --profile web-triage
```

Review de supply chain:

```bash
go run ./cmd/orquestador-auditor sync --profile supply-chain
```

Reporting/correlation:

```bash
go run ./cmd/orquestador-auditor sync --profile reporting
```

### Fases integradas en OpenCode

Con el perfil `web-triage`, OpenCode ya queda preparado para este recorrido:

1. `network-recon`
2. `security-scout`
3. `js-intel`
4. `api-discovery`
5. `deep-web`
6. `sqli-validate`
7. `correlate-findings`
8. `security-report`

`sync` ahora:

- apunta a OpenCode por defecto
- filtra prompts, commands y skills segun el perfil
- deja siempre presentes los agentes especialistas del orchestrator
- aplica permisos policy-driven por perfil
- poda assets administrados que no correspondan al perfil activo

## OpenCode-first UX

After `sync`, OpenCode gets:

- slash commands relevantes al perfil
- skills filtradas por perfil
- MCP
- named agents in `opencode.json` (siempre presentes para evitar incompatibilidades de delegacion)
- policy-driven tool permissions for those agents

## Runtime de auditoria

El runtime deja de ser solo prompting: ahora produce artefactos estructurados por fase y aplica gating explicito.

### 1. Crear la corrida

```bash
go run ./cmd/orquestador-auditor run start \
  --target https://example.com \
  --target-kind web \
  --profile web-triage \
  --authorized \
  --authorization-ref ENG-2026-001 \
  --aggressiveness bounded \
  --approved-tools nmap,katana,jsluice
```

### 2. Registrar outputs de una fase

```bash
go run ./cmd/orquestador-auditor run phase \
  --run-id 20260409-120000-example-com \
  --phase network-recon \
  --status observed \
  --summary "Se observo 443/tcp y 8443/tcp con superficie HTTPS." \
  --requested-tools nmap
```

Si queres findings estructurados, pasalos con `--findings-file` apuntando a un JSON array.

### 3. Correlacionar

```bash
go run ./cmd/orquestador-auditor run correlate --run-id 20260409-120000-example-com
```

### 4. Inspeccionar el estado

```bash
go run ./cmd/orquestador-auditor run inspect --run-id 20260409-120000-example-com
```

## Gating de policy

- `nmap`, `katana`, `chromedp`, `mitmproxy`, `arjun`, `ffuf`, `sqlmap` no quedan automaticamente permitidos
- `sqlmap` y `ffuf` requieren agresividad suficiente **y** aprobacion explicita
- `authorized-validation` se bloquea si no existe antes un `vuln-hypothesis` con findings
- `correlation` deduplica findings, completa taxonomia (`CWE`, `OWASP`) y arma recomendaciones
