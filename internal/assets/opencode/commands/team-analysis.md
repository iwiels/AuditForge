---
description: Lanzar el Security Audit Team completo en pipeline coordinado
agent: security-orchestrator
---

Coordiná el Security Audit Team completo contra el target autorizado.

**Paso 1 — Preparación:**
Generá un SESSION_ID único: `[target-sin-espacios]-[YYYYMMDD]-[4chars aleatorios]` (ej: `app-example-com-20250409-a3f1`).

Verificá autorización:
```
memory.search("authorized engagement [target]")
```
Si no existe → pedí autorización explícita antes de continuar. Registrala en memoria antes de lanzar el equipo.

**Paso 2 — Contexto histórico (security-memory):**
Invocá a `security-memory` con el target y SESSION_ID. Pedile que cargue hallazgos de sesiones previas y te devuelva un resumen. Pasá ese contexto a los agentes siguientes.

**Paso 3 — Recon y superficie (security-scout):**
Invocá a `security-scout` pasándole:
- TARGET, SESSION_ID, SCOPE
- Contexto histórico de security-memory
- Instrucción: "Escribir cada hallazgo a memoria con el SESSION_ID. Marcar needs_followup_by cuando corresponda."

Esperá que confirme phase-complete antes de continuar.

**Paso 4 — Web, JS y APIs (security-web):**
Invocá a `security-web` pasándole:
- TARGET, SESSION_ID, SCOPE
- "Leer los hallazgos de security-scout de esta sesión primero. Priorizar vectores con needs_followup_by que te incluyan."

Esperá phase-complete.

**Paso 5 — Código y supply chain (security-code):**
Invocá a `security-code` pasándole:
- TARGET, SESSION_ID, SCOPE
- "Leer todos los hallazgos de scout y web de esta sesión. Enfocar el análisis en los componentes señalados por los hallazgos previos."

Esperá phase-complete. Si no hay código fuente disponible, indicalo y salteá este paso.

**Paso 6 — Compliance y arquitectura (security-ops):**
Invocá a `security-ops` pasándole:
- TARGET, SESSION_ID
- "Leer todos los hallazgos acumulados. Mapear a ASVS/NIST. Identificar gaps de diseño sistémicos."

Esperá phase-complete.

**Paso 7 — Síntesis (security-report):**
Invocá a `security-report` con SESSION_ID:
- "Leer todos los hallazgos del equipo para la sesión [SESSION_ID]. Deduplicar, calcular CVSS, y producir el reporte final."

**Paso 8 — Coordinación de hallazgos críticos:**
Durante el pipeline, si cualquier agente reporta un hallazgo CRÍTICO con `needs_followup_by`, interrumpí el flujo normal y notificá al usuario antes de continuar.

**Output esperado:**
Reporte completo con hallazgos de los 4 agentes especializados, deduplicado y priorizado por security-report.
