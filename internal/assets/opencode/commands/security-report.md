---
description: Síntesis de hallazgos del equipo y reporte final
agent: security-report
---

Sintetizá todos los hallazgos del Security Audit Team y producí el reporte final.

**Al arrancar:**
1. `memory.search("[SESSION_ID] findings")` — leer TODOS los hallazgos de todos los agentes
2. `memory.search("[SESSION_ID] phase-complete")` — verificar qué agentes completaron
3. Deduplicar: mismo vector detectado por múltiples agentes → un solo finding consolidado

**Proceso:**

**Verificación de calidad:**
- Todo CRÍTICO/ALTO con `status: suspected` → moverlo a "requiere validación adicional"
- Solo `status: validated` u `observed` con evidencia concreta van al reporte principal
- Calcular CVSS v3.1 para todos los CRÍTICO y ALTO

**Mapeo:**
- CWE para cada finding técnico
- OWASP Top 10 2021 para findings web
- Controles ASVS/NIST si security-ops los incluyó

**Estructura del reporte:**

```markdown
# Reporte de Auditoría de Seguridad

**Target:** [TARGET]
**Session:** [SESSION_ID]
**Equipo:** security-scout · security-web · security-code · security-ops
**Metodología:** OWASP Testing Guide v4.2 / PTES / OSSTMM 3

## Resumen ejecutivo
[Para management. Sin jerga técnica. ¿Qué se revisó? ¿Estado general? ¿Qué es urgente?]

## Hallazgos consolidados

| ID | Agente | Severidad | CVSS | Título | Estado |
|----|--------|-----------|------|--------|--------|

## Detalle de hallazgos
[CRÍTICO → ALTO → MEDIO → BAJO → INFO]
[Para cada uno: descripción, evidencia, impacto de negocio, remediación con código]

## Requiere validación adicional
[Hipótesis con status: suspected]

## Recomendaciones estratégicas
[Cambios de arquitectura sistémicos, no solo fixes puntuales]

## Superficie analizada y alcance negativo
[Qué se revisó y qué quedó explícitamente fuera]
```

**Al finalizar:**
```
memory.save({ kind: "report", session: "[SESSION_ID]", target: "[TARGET]",
  status: "complete", finding_count: N, critical: N, high: N, medium: N })
```

Skills: evidence-reporting, compliance-check, authorization-guard
