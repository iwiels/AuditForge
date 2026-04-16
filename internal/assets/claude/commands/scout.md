Ejecutá este comando de seguridad autorizado:

**Objetivo:** Recon pasivo + surface discovery del target autorizado.

**Pasos a seguir:**
1. Verificá la autorización via `memory.search("authorized engagement")` — si no existe, pedila antes de continuar.
2. Ejecutá `memory.search` con el nombre del target para recuperar contexto de sesiones previas.
3. Aplicá la skill `osint-passive`: certificate transparency, DNS histórico, Shodan, repositorios públicos, Google dorks. Todo sin tocar el target.
4. Aplicá la skill `surface-discovery`: fingerprinting de stack vía headers HTTP, archivos de recon estándar (`/robots.txt`, `/.git/`, `/api/docs`), enumeración de endpoints.
5. Construí el **Asset Inventory**: clasificá cada asset encontrado como P0/P1/P2/P3.
6. Identificá los vectores de mayor interés para web-triage y delegá en consecuencia.

**Output esperado:**
- Asset inventory priorizado
- Stack tecnológico identificado
- Hallazgos inmediatos con severidad
- Lista de vectores para investigación activa

**Limitaciones:** No ejecutes herramientas activas sin autorización explícita. Esta fase es observacional.
