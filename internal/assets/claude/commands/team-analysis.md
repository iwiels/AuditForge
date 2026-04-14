Lanzá el Security Audit Team completo contra el target autorizado usando Task delegation paralela.

**Preparación:**

Primero verificá autorización:
```
memory.search("authorized engagement [TARGET]")
```
Si no existe → pedí autorización al usuario antes de continuar.

Generá SESSION_ID: `[target]-[YYYYMMDD]-[4chars]`

Cargá contexto histórico:
```
memory.search("[TARGET] findings")
memory.search("[TARGET] session")
```

**Lanzar el equipo en pipeline coordinado:**

Usá Task() para lanzar cada agente en secuencia, pasando el contexto acumulado:

```
Task("Security Scout", "
  Target: [TARGET]
  Session: [SESSION_ID]
  Scope: [SCOPE]
  
  Ejecutá recon completo siguiendo la skill surface-discovery y osint-passive.
  
  Para cada hallazgo, guardalo en memoria con:
  - kind: 'finding', agent: 'security-scout', session: '[SESSION_ID]'
  - severity, status (observed/suspected/validated), evidence, vector
  - needs_followup_by si otro agente debe profundizar
  
  Al terminar escribí a memoria: kind: 'phase-complete', agent: 'security-scout'
  
  Skills disponibles: ~/.claude/skills/surface-discovery/SKILL.md y osint-passive/SKILL.md
")

# Cuando scout complete:
Task("Security Web", "
  Target: [TARGET]
  Session: [SESSION_ID]
  
  Primero: memory.search('[SESSION_ID] findings agent:security-scout')
  Priorizar vectores con needs_followup_by que te incluyan.
  
  Luego ejecutá análisis web siguiendo web-triage y web-js-intel.
  
  Escribir cada hallazgo a memoria con: kind: 'finding', agent: 'security-web'
  Al terminar: kind: 'phase-complete', agent: 'security-web'
  
  Skills: ~/.claude/skills/web-triage/SKILL.md y web-js-intel/SKILL.md
")

Task("Security Code", "
  Target: [TARGET]
  Session: [SESSION_ID]
  
  Primero: memory.search('[SESSION_ID] findings') — leer todos los hallazgos previos
  Enfocar el análisis en los componentes señalados por los hallazgos de scout y web.
  
  Ejecutá code-review y supply-chain-triage.
  
  Escribir hallazgos con: kind: 'finding', agent: 'security-code'
  Al terminar: kind: 'phase-complete', agent: 'security-code'
  
  Skills: ~/.claude/skills/code-review/SKILL.md y supply-chain-triage/SKILL.md
")

Task("Security Ops", "
  Target: [TARGET]
  Session: [SESSION_ID]
  
  Primero: memory.search('[SESSION_ID] findings') — leer TODOS los hallazgos del equipo
  
  Ejecutá secure-design-review y compliance-check sobre los hallazgos acumulados.
  Mapear a OWASP ASVS v4 y NIST CSF. Identificar gaps de diseño sistémicos.
  
  Escribir hallazgos con: kind: 'finding', agent: 'security-ops'
  Al terminar: kind: 'phase-complete', agent: 'security-ops'
  
  Skills: ~/.claude/skills/secure-design-review/SKILL.md y compliance-check/SKILL.md
")

Task("Security Report", "
  Session: [SESSION_ID]
  Target: [TARGET]
  
  memory.search('[SESSION_ID] findings') — leer TODOS los hallazgos del equipo completo
  
  Deduplicar y sintetizar siguiendo evidence-reporting:
  - CVSS v3.1 para CRÍTICO y ALTO
  - Mapeo CWE y OWASP Top 10 2021
  - Solo status: validated/observed con evidencia al reporte principal
  - status: suspected → sección 'requiere validación adicional'
  
  Producir reporte completo en Markdown.
  Guardar: memory.save({ kind: 'report', session: '[SESSION_ID]', status: 'complete' })
  
  Skill: ~/.claude/skills/evidence-reporting/SKILL.md
")
```

**Si cualquier agente reporta CRÍTICO:** interrumpí el pipeline y notificá al usuario.

**Output:** Reporte final consolidado de los 4 agentes especializados.
