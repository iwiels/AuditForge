Generá el reporte de auditoría final con todos los findings de esta sesión:

**Objetivo:** Reporte completo, estructurado y accionable para el cliente.

**Pasos a seguir:**
1. Ejecutá `memory.context` para recuperar todos los findings de la sesión actual.
2. Consolidá y deduplicá findings de todos los agentes (scout, web, supply-chain).
3. Aplicá la skill `evidence-reporting`:
   - Verificá que cada finding CRÍTICO/ALTO tiene evidencia concreta (no teórica)
   - Calculá CVSS v3.1 para cada finding crítico/alto
   - Mapeá a CWE y OWASP Top 10 2021
   - Redactá remediaciones accionables con código cuando sea posible
4. Aplicá la skill `compliance-check`: mapeá los hallazgos a controles ASVS/NIST CSF si aplica.
5. Estructurá el reporte completo:
   - Resumen ejecutivo (para management no técnico)
   - Tabla de hallazgos por severidad
   - Detalle de cada finding
   - Recomendaciones estratégicas
   - Superficie analizada y alcance negativo

**Output:** Reporte en Markdown listo para entregar. Guardalo en memoria con `memory.save`.

**Calidad mínima:**
- Sin findings sin evidencia en la sección "confirmados"
- Remediación con pasos concretos en cada finding
- Resumen ejecutivo sin jerga técnica
- CVSS calculado para todos los CRÍTICO y ALTO
