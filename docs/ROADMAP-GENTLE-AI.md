# Roadmap: Gentle AI-Level Cybersecurity Orchestrator

> **Objetivo:** Elevar orquestador-auditor al nivel de Gentle AI en ciberseguridad, manteniendo el enfoque defensivo y OpenCode-first.

---

## maturity Model

| Nivel | Estado | Descripción |
|-------|--------|-------------|
| **L1 - CLI Tool** | ✅ ALCANZADO | Comandos funcionales, tests, CI/CD |
| **L2 - Profile-Aware** | ✅ ALCANZADO | Perfiles, políticas, memory, MCP |
| **L3 - Orchestrated** | 🟡 EN PROGRESO | Agentes coordinados, drift control, validation |
| **L4 - Platform** | ❌ PENDIENTE | Dashboard, reporting, integrations, lifecycle |
| **L5 - Autonomous** | ❌ FUTURO | Agentes autónomos con supervisión humana |

---

## Phase 1: Foundation Hardening (Sprint 1-2)

### 1.1 Marker-Based Injection (Drift Control)
**Problema:** `sync` repetido puede duplicar contenido en configs de AI clients.  
**Solución:** Sistema de content markers con reemplazo atómico.

```go
// internal/orchestrator/injector.go
// Nuevo método:
func (i *Injector) InjectWithMarkers(adapter agents.Adapter) error {
    // 1. Detectar markers existentes <!-- ORQUESTADOR:commands:start -->
    // 2. Si existe, reemplazar entre markers en vez de append
    // 3. Si no existe, insertar con markers nuevos
    // 4. Verificar integridad post-inyección
}
```

**Markers por tipo:**
- `<!-- ORQUESTADOR:commands:start/end -->`
- `<!-- ORQUESTADOR:skills:start/end -->`
- `<!-- ORQUESTADOR:mcp:start/end -->`
- `<!-- ORQUESTADOR:agents:start/end -->`

**Tests requeridos:**
- Sync repetido no duplica contenido
- Sync parcial (solo commands) no toca skills
- Rollback restaura versión anterior

### 1.2 Engram Protocol (Cross-Session Memory Injection)
**Problema:** La memoria existe pero no se usa automáticamente.  
**Solución:** Al iniciar un agente, inyectar contexto histórico del target.

```markdown
## Contexto Histórico (auto-inyectado)
Target: example.com
Última auditoría: 2026-04-01
Hallazgos previos: 3 (1 Alto - IDOR en /api/users, 2 Medios)
Campaña: bug-bounty-q1
Notas del operador: "El equipo de dev ya parcheó el IDOR, verificar"
```

**Implementación:**
1. `internal/memory/engram.go` - Nuevo paquete
2. Método `BuildContextPrompt(target, campaign string) string`
3. Se inyecta en `InjectPrompts()` como preamble del system prompt
4. Cacheable por 24h para no saturar el prompt

### 1.3 Validation Phase (Judgment Day)
**Problema:** `correlation` consolida pero no valida calidad.  
**Solución:** Nueva fase `judgment` post-correlation.

```go
// internal/runtime/types.go
const PhaseJudgment PhaseID = "judgment"

type ValidationCheck struct {
    Check     string `json:"check"`
    Passed    bool   `json:"passed"`
    FindingID string `json:"finding_id,omitempty"`
    Detail    string `json:"detail"`
}

type JudgmentResult struct {
    TotalChecks     int                `json:"total_checks"`
    Passed          int                `json:"passed"`
    Failed          int                `json:"failed"`
    FailedChecks    []ValidationCheck  `json:"failed_checks"`
    QualityScore    float64            `json:"quality_score"` // 0-100
    ReadyToReport   bool               `json:"ready_to_report"`
}
```

**Checks de validación:**
- [ ] Todo finding tiene al menos 1 evidencia
- [ ] Todo finding CRÍTICO/ALTO tiene vector confirmado
- [ ] CWE asignado en todo finding validado
- [ ] OWASP Top 10 mapeado donde aplica
- [ ] Remediation existe y es accionable (no genérica)
- [ ] No hay findings duplicados
- [ ] Pipeline completo (no fases saltadas sin justificación)

---

## Phase 2: Agent Orchestration (Sprint 3-4)

### 2.1 Real Agent Orchestrator
**Problema:** AGENTS.md documenta el equipo pero no hay motor de ejecución.  
**Solución:** runtime interno de equipo dentro del orquestador

```go
type AgentOrchestrator struct {
    Memory    *memory.Store
    Runtime   *auditruntime.Store
    Adapter   agents.Adapter
}

func (o *AgentOrchestrator) RunTeam(input TeamInput) error {
    // 1. security-memory → carga contexto histórico
    // 2. security-scout   → recon pasivo
    //    → escribe findings a memoria
    //    → señales needs_followup_by
    // 3. security-web     → lee findings de scout, prioriza pendientes
    //    → escribe findings a memoria
    // 4. security-code    → lee findings de scout + web
    // 5. security-ops     → lee todos
    // 6. security-report  → síntesis final
    
    // Cada agente:
    //   a. Lee memoria con su session_id
    //   b. Lee needs_followup_by dirigidos a él
    //   c. Ejuta su análisis
    //   d. Escribe findings estructurados
    //   e. Señala followups para otros agentes
}
```

**Cada agente es un prompt especializado + herramientas MCP:**
- `security-scout.md` → tools: nmap, subfinder, httpx
- `security-web.md` → tools: katana, jsluice, arjun
- `security-code.md` → tools: semgrep, trivy, gosec
- `security-ops.md` → tools: memory search, run inspection
- `security-report.md` → tools: run correlation, CVSS calculator

### 2.2 Inter-Agent Signal System
**Problema:** Los agentes no se comunican en runtime.  
**Solución:** Sistema de señales via memoria compartida.

```json
{
  "kind": "signal",
  "from": "security-scout",
  "to": "security-web",
  "session": "example-com-20260413",
  "type": "followup",
  "priority": "high",
  "finding_id": "finding-001",
  "message": "GraphQL endpoint descubierto en /graphql - requiere análisis de schema",
  "created_at": "2026-04-13T10:30:00Z"
}
```

**Tipos de señal:**
- `followup` → "investigá esto"
- `escalation` → "esto es crítico, priorizalo"
- `blocker` → "no puedo continuar sin X"
- `complete` → "terminé mi parte"

### 2.3 Critical Finding Interruption
**Problema:** El flujo es lineal, no reacciona a hallazgos críticos.  
**Solución:** Si un agente escribe un finding CRÍTICO, el orchestrator puede:

1. Notificar al operador (output destacado en TUI)
2. Redirigir al siguiente agente para priorizar ese vector
3. Ofrecer pausar el pipeline para revisión humana

---

## Phase 3: TUI Evolution (Sprint 5-6)

### 3.1 De Menú a Dashboard

**Current:**
```
Security Audit Orchestrator

Elegí una acción:
  Install bundle
  Sync AI clients
  Search memory
  Self-update
  Quit
```

**Target:**
```
┌─ Security Audit Orchestrator ───────────────────────────────┐
│                                                             │
│  ACTIVE RUNS                              FINDINGS          │
│  ● example.com (web-triage)    ████████░░  Phase 5/8   CRITICAL  1  ▲
│  ○ api.internal (recon)        ████░░░░░░  Phase 2/8   HIGH      3  ━
│                                                      MEDIUM    7  ━
│  MEMORY: 142 observations                    LOW      12  ━
│                                                             │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │
│  │ New Run     │ │ Sync        │ │ Memory      │           │
│  │             │ │             │ │             │           │
│  │             │ │             │ │             │           │
│  └─────────────┘ └─────────────┘ └─────────────┘           │
│                                                             │
│  [q] quit  [n] new run  [s] sync  [m] memory  [r] refresh  │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 Engagement Wizard

Flujo guiado para crear un audit run:

```
Step 1/4: Target
  URL/IP/Repo: [https://example.com                    ]

Step 2/4: Authorization
  ☑ Confirmed authorized engagement
  Authorization ref: [ticket-123                       ]

Step 3/4: Profile & Aggressiveness
  Profile:    [web-triage ▼]
  Aggression: [bounded     ▼]

Step 4/4: Tool Approval
  Approved tools:
  ☑ nmap   ☑ katana   ☐ sqlmap   ☑ arjun   ☐ ffuf

  → Launch engagement? [Y/n]
```

### 3.3 Live Finding Feed

Durante un run, mostrar findings a medida que aparecen:

```
┌─ Run: example.com (web-triage) ────────────────────────────┐
│ Phase: api-discovery (5/8)                                  │
│                                                             │
│ FINDINGS:                                                   │
│ [HIGH]   IDOR en /api/users/{id}     ← security-web        │
│          evidence: user_id=100 retorna datos de user 101   │
│          → followup: security-code (revisar resolver)      │
│                                                             │
│ [MEDIUM] API sin rate limiting       ← security-web         │
│          evidence: 1000 requests en 60s sin 429            │
│                                                             │
│ [LOW]    Header X-Powered-Expo        ← security-scout      │
│          evidence: Express expuesto en response headers    │
│                                                             │
│ [ctrl+c] abort  [p] pause  [j] jump to phase  [e] escalate │
└─────────────────────────────────────────────────────────────┘
```

---

## Phase 4: Platform Features (Sprint 7-8)

### 4.1 Report Generation

**Output formats:**
- JSON (ya existe como artifacts)
- Markdown (reporte consolidado)
- SARIF (integración con GitHub/GitLab)
- CSV (import a trackers)

```bash
orquestador-auditor run report --run-id <id> --format markdown
orquestador-auditor run report --run-id <id> --format sarif --output findings.sarif
```

**Markdown report structure:**
```markdown
# Security Audit Report: example.com
## Executive Summary
- Target: https://example.com
- Profile: web-triage
- Period: 2026-04-13
- Findings: 11 (1 Critical, 3 High, 7 Medium)
- Overall Risk: HIGH

## Findings
### [CRITICAL] IDOR in /api/users/{id}
- **CWE-639**: Authorization Bypass Through User-Controlled Key
- **Evidence**: GET /api/users/100 returns data for user 101
- **Impact**: Full access to other users' PII
- **Remediation**: Implement owner-based authorization check

## Methodology Compliance
- ✅ Scope defined
- ✅ Network recon completed
- ✅ Surface discovery completed
- ⚠️ Authorized validation skipped (no hypothesis generated)
```

### 4.2 Issue Tracker Integration

```bash
# Create GitHub issues for each finding
orquestador-auditor run export --run-id <id> --to github --repo owner/repo

# Jira integration
orquestador-auditor run export --run-id <id> --to jira --project SEC
```

### 4.3 Lifecycle Management

**Rollback:**
```bash
# Si un sync rompió la config de OpenCode
orquestador-auditor rollback --before 2026-04-13T10:00:00
```

**Health check:**
```bash
orquestador-auditor health
# Verifica:
# - Binary version matches config version
# - All managed paths exist and are valid
# - MCP server responds
# - Memory store accessible
```

**Migration:**
```bash
# Al actualizar de v0.1 a v0.2
orquestador-auditor migrate
# Detecta cambios breaking y aplica migraciones
```

---

## Phase 5: Autonomous Features (Future)

### 5.1 Self-Healing Sync
Detectar si un AI client cambió su formato de config y auto-adapt el injector.

### 5.2 Campaign Tracking
Seguir un target a través de múltiples auditorías y detectar:
- Findings recurrentes (mismo vector, diferente sesión)
- Remediation effectiveness (¿se parcheó realmente?)
- Trend analysis (¿mejora o empeora?)

### 5.3 Policy Learning
Aprender de decisiones del operador:
- Si siempre aprueba `nmap` en recon, auto-aprobar
- Si siempre bloquea `sqlmap`, requerir confirmación extra
- Sugerir perfil basado en target kind (web app → web-triage, API → api-discovery)

---

## Implementation Priority Matrix

| Feature | Impact | Effort | Priority |
|---------|--------|--------|----------|
| Marker-Based Injection | 🔴 Critical | Medium | **P0 - Sprint 1** |
| Engram Memory Protocol | 🔴 Critical | Medium | **P0 - Sprint 1** |
| Validation Phase | 🟡 High | Low | **P0 - Sprint 1** |
| Agent Orchestrator | 🔴 Critical | High | **P1 - Sprint 3** |
| Inter-Agent Signals | 🟡 High | Medium | **P1 - Sprint 3** |
| TUI Dashboard | 🟡 High | High | **P1 - Sprint 5** |
| Engagement Wizard | 🟢 Medium | Medium | **P2 - Sprint 5** |
| Report Generation | 🟡 High | Medium | **P2 - Sprint 7** |
| SARIF Export | 🟢 Medium | Low | **P2 - Sprint 7** |
| Rollback/Health | 🟡 High | Medium | **P2 - Sprint 7** |
| Issue Tracker Integration | 🟢 Medium | High | **P3 - Sprint 8** |
| Campaign Tracking | 🟢 Medium | High | **P3 - Future** |

---

## Quick Wins (se pueden hacer ya, sin romper nada)

1. **Add `run report` command** → consolidar findings en markdown (1-2 días)
2. **Add validation checks to correlation** → validar que findings tengan evidencia (1 día)
3. **Inject memory context in prompts** → buscar memoria del target al hacer sync (2-3 días)
4. **Add markers to injected files** → prevenir drift en syncs repetidos (2-3 días)
5. **TUI: show active runs** → leer `.orquestador/runs/` y mostrar estado (2 días)

---

## Comparativa Final

| Feature | Gentle AI | Orquestador (actual) | Orquestador (target) |
|---------|-----------|---------------------|---------------------|
| Spec-driven pipeline | ✅ 9 phases | ✅ 8 phases | ✅ 9 phases (+judgment) |
| Drift-free injection | ✅ Markers | ❌ Append | ✅ Markers |
| Cross-session memory | ✅ Engram | ⚠️ Store only | ✅ Engram protocol |
| Multi-agent orchestration | ✅ Orchestrator | ❌ Doc only | ✅ Real orchestrator |
| Inter-agent communication | ✅ Signals | ❌ None | ✅ Signal system |
| TUI Dashboard | ✅ Installer | ⚠️ Basic menu | ✅ Dashboard + wizard |
| Validation phase | ✅ Judgment day | ❌ None | ✅ Judgment phase |
| Self-update | ✅ | ✅ | ✅ |
| Backup/Rollback | ✅ | ⚠️ Paths only | ✅ Full rollback |
| Report generation | ✅ PR auto | ❌ JSON only | ✅ Multi-format |
| Issue tracker integration | ✅ GitHub | ❌ None | ✅ GitHub/Jira |
| Tool policy engine | ⚠️ Basic | ✅ Advanced | ✅ Advanced |
| Authorization gating | ⚠️ Manual | ✅ Built-in | ✅ Built-in |
| MCP server | ⚠️ Limited | ✅ 10 tools | ✅ 10+ tools |

---

## Next Steps Recomendados

1. **Sprint planning:** Empezar con Phase 1 (Foundation Hardening) — son cambios de bajo riesgo con alto impacto
2. **Crear issues en GitHub:** Cada feature como issue con labels `enhancement`, `priority: P0/P1/P2`
3. **TDD:** Seguir la práctica actual de tests para cada nuevo módulo
4. **OpenSpec:** Usar el framework openspec para propuestas de cambio grandes
5. **No romper compatibilidad:** Todo cambio debe ser backwards-compatible con runs existentes
