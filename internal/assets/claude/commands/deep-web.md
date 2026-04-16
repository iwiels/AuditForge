Ejecutá este análisis web profundo y metodológico del target autorizado:

**Objetivo:** Triage completo de la superficie web con enfoque en autenticación, autorización, validación de inputs y headers de seguridad.

**Pasos a seguir:**
1. Verificá la autorización via `memory.search("authorized engagement")`.
2. Recuperá el Asset Inventory del scout previo via `memory.search("[target] asset inventory")`.
3. Aplicá la skill `web-triage` sobre los assets P0 y P1:
   - Análisis de mecanismo de autenticación (JWT, cookies, tokens)
   - Control de acceso: IDOR, privilege escalation, mass assignment
   - Validación de inputs: SQL Injection, XSS, SSRF (conceptual, sin explotar)
   - Headers de seguridad: CSP, HSTS, X-Frame-Options, etc.
4. Si hay código fuente disponible, aplicá la skill `code-review` sobre los archivos de mayor riesgo.
5. Aplicá la skill `web-js-intel` si hay JavaScript accesible: endpoints hardcodeados, tokens, lógica de negocio expuesta.
6. Documentá cada vector con: parámetro específico, comportamiento observado, severidad, CWE.

**Output esperado:**
- Findings con severidad, CWE, y evidencia observable
- Vectores de SQL/XSS/SSRF identificados (sin explotar)
- Estado completo de headers de seguridad
- Lista para delegación a security-report

**Reglas:**
- No ejecutes payloads activos — identificá vectores, documentá evidencia observable
- Si un vector requiere validación activa, marcalo como "requiere confirmación" y escalá al usuario
