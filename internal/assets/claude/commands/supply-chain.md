Analizá la supply chain y el código fuente del target:

**Objetivo:** Auditoría de dependencias, secrets en código, pipeline CI/CD, y revisión de código estático.

**Pasos a seguir:**
1. Verificá la autorización via `memory.search("authorized engagement")`.
2. Aplicá la skill `supply-chain-triage`:
   - Inventario de dependencias por ecosistema (npm, pip, go, maven)
   - Identificación de CVEs en dependencias directas y transitivas
   - Búsqueda de secrets en código fuente e historial de git
   - Revisión de pipeline CI/CD por misconfigurations
3. Si hay código fuente disponible, aplicá la skill `code-review`:
   - SQL Injection, XSS, Command Injection, Path Traversal
   - Insecure Deserialization, Hardcoded Secrets
   - Broken Access Control, lógica de negocio insegura
4. Documentá hallazgos con: archivo, línea, patrón vulnerable, impacto, remediación.

**Output esperado:**
- Lista de dependencias con CVEs (tabla: paquete, versión, CVE, CVSS, fix disponible)
- Secrets detectados (sin mostrar el valor — solo tipo y ubicación)
- Findings de código con ubicación específica
- Riesgos en pipeline CI/CD

**Regla crítica:** Nunca mostrés el valor completo de un secret en el reporte. Solo el tipo y la ubicación.
