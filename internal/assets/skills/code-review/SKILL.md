# Skill: Code Review

**Categoría:** code-analysis
**Metodología base:** OWASP Code Review Guide v2, CWE Top 25, SANS Top 25
**Cuándo activar:** cuando se tiene acceso al código fuente del target

---

## Protocolo

### Fase 1 — Orientación en el codebase
Entender entry points, middleware de seguridad, ORM, auth flow y manejo de inputs.

### Fase 2 — Análisis de Vulnerabilidades (CWE)
Buscar SQLi (CWE-89), XSS (CWE-79), Injection (CWE-502), Path Traversal (CWE-22), Command Injection (CWE-78), Broken Access Control (CWE-639), Hardcoded Secrets (CWE-798).

### Fase 3 — Output
Reportar con severidad, evidencia (línea de código), impacto y remediación.
