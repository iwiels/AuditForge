# Skill: Evidence Reporting

**Categoría:** reporting  
**Metodología base:** CVSS v3.1, OWASP Risk Rating Methodology, PTES Reporting  
**Cuándo activar:** al finalizar triage, antes de entregar resultados al cliente

---

## Protocolo

### Fase 1 — Recolección y normalización de findings

Antes de escribir el reporte, consolidá todos los hallazgos de todos los agentes:

```
FUENTES A CONSOLIDAR:
□ Findings de security-scout (surface discovery)
□ Findings de security-web (web triage, JS intel)
□ Findings de security-supply (supply chain, código)
□ Findings de security-osint (OSINT pasivo)
□ Threat model (vectores identificados)

PROCESO DE NORMALIZACIÓN:
1. Deduplicar: mismo vector reportado por múltiples agentes → un solo finding
2. Validar evidencia: todo finding sin evidencia concreta → "requiere validación"
3. Clasificar por tipo: authn / authz / injection / exposure / config / supply-chain
```

### Fase 2 — Scoring CVSS v3.1

Para cada finding crítico o alto, calculá el score CVSS:

**Vector String:** `CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:N`

**Métricas base:**
```
Attack Vector (AV):      N=Network / A=Adjacent / L=Local / P=Physical
Attack Complexity (AC):  L=Low / H=High
Privileges Required (PR): N=None / L=Low / H=High
User Interaction (UI):   N=None / R=Required
Scope (S):               U=Unchanged / C=Changed
Confidentiality (C):     N=None / L=Low / H=High
Integrity (I):           N=None / L=Low / H=High
Availability (A):        N=None / L=Low / H=High
```

**Rangos de severidad:**
| Score | Severidad |
|-------|-----------|
| 9.0 - 10.0 | CRÍTICO |
| 7.0 - 8.9 | ALTO |
| 4.0 - 6.9 | MEDIO |
| 0.1 - 3.9 | BAJO |
| 0.0 | INFORMATIVO |

### Fase 3 — Estructura de cada finding

Cada hallazgo debe seguir esta estructura sin excepción:

```markdown
### [SEVERIDAD] CVSS:X.X — Título del hallazgo

**CWE:** CWE-XXX — Nombre
**OWASP:** A0X:2021 — Categoría
**Afecta:** [componente, endpoint, o archivo específico]

#### Descripción
[Explicación técnica de qué es la vulnerabilidad y por qué existe.
Sin jerga innecesaria. Máximo 3 párrafos.]

#### Evidencia
[Comportamiento observable que confirma el hallazgo. Puede ser:
- Fragmento de código fuente con línea específica
- Request/response HTTP (con datos sensibles redactados)
- Configuración expuesta
- Respuesta de error que revela información]

```
GET /api/users/1337/profile HTTP/1.1
Host: target.com
Authorization: Bearer eyJ... (usuario ID 9999)

HTTP/1.1 200 OK
{"id": 1337, "email": "victim@example.com", "role": "admin"}
```

#### Impacto
[Qué puede hacer un atacante si explota esto. Ser específico:
"Un atacante autenticado puede leer el perfil y credenciales de cualquier usuario
de la plataforma iterando IDs numéricos secuenciales."]

#### Remediación
[Pasos concretos para corregir. Incluir código cuando sea posible.]

**Corto plazo (1-7 días):**
- [acción inmediata]

**Mediano plazo (1-4 semanas):**
- [cambio estructural]

**Referencias:**
- [CWE link]
- [OWASP link]
- [documentación relevante]
```

### Fase 4 — Estructura del reporte completo

```markdown
# Reporte de Auditoría de Seguridad

**Cliente:** [nombre]
**Target:** [target]
**Fecha:** [fecha]
**Auditor:** Security Audit Orchestrator v0.2.0
**Metodología:** OWASP Testing Guide v4.2 / PTES / OSSTMM 3
**Alcance:** [descripción del scope autorizado]

---

## Resumen ejecutivo

[3-5 párrafos. Audiencia: CTO, CISO, management no técnico.
¿Qué se revisó? ¿Cuál es el estado general de seguridad?
¿Cuáles son los 2-3 riesgos más importantes?
¿Qué debe hacerse primero?]

## Hallazgos por severidad

| ID | Severidad | Título | CVSS | Estado |
|----|-----------|--------|------|--------|
| F-001 | CRÍTICO | ... | 9.8 | Confirmado |
| F-002 | ALTO | ... | 7.5 | Confirmado |

## Detalle de hallazgos

[Cada finding en formato Fase 3, ordenado: CRÍTICO → ALTO → MEDIO → BAJO → INFO]

## Superficie analizada

[Lista de endpoints, componentes y archivos revisados]

## Hallazgos que requieren validación adicional

[Vectores identificados sin evidencia suficiente para confirmar]

## Recomendaciones estratégicas

[Mejoras de proceso, arquitectura y cultura de seguridad.
No son fixes puntuales — son cambios estructurales.]

## Metodología

[Descripción de qué se hizo y qué no se hizo, con justificación]
```

### Fase 5 — Control de calidad pre-entrega

Checklist antes de finalizar:

```
□ Todos los findings CRÍTICO y ALTO tienen evidencia concreta (no teórica)
□ Todos los findings tienen remediación accionable
□ No hay duplicados
□ Los CVSS scores son coherentes con el impacto descrito
□ El resumen ejecutivo no usa términos que el management no entienda
□ Los findings "requiere validación" están separados de los confirmados
□ Las referencias CWE/OWASP están linkeadas correctamente
□ Los datos sensibles del cliente están redactados en los ejemplos
```

---

## Anti-patterns

- ❌ Reportar hallazgos teóricos sin evidencia como "confirmados"
- ❌ CVSS score sin justificación de las métricas elegidas
- ❌ Remediación vaga: "mejorar la validación de inputs" — debe ser código o configuración específica
- ❌ Resumen ejecutivo técnico — el management no sabe qué es un JWT
- ❌ Findings ordenados por tipo en vez de por severidad — el cliente quiere saber qué es urgente
- ❌ Olvidar documentar qué NO se revisó — el alcance negativo es tan importante como el positivo
