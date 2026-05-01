# Skill: Smart Replay Engine

**Categoría:** replay, differential-analysis, automated-testing
**Metodología base:** OWASP WSTG-BUSL, WSTG-ATHZ, WSTG-INPV
**Cuándo activar:** después de capturar tráfico baseline, cuando se necesita validar hipótesis de vulnerabilidad a escala
**MCP requerido:** `auditforge-proxy`

---

## Concepto

El Smart Replay Engine automatiza la detección de vulnerabilidades mediante **análisis diferencial semántico**:

1. **Captura Baseline**: Request legítimo del usuario
2. **Generación de Variaciones**: Mutaciones automáticas (IDOR, auth bypass, etc.)
3. **Ejecución Paralela**: Replays con diferentes modificaciones
4. **Análisis Diferencial**: Comparación semántica de respuestas
5. **Detección Automática**: Reglas que identifican vulnerabilidades

---

## Tipos de Detección

### 1. Auth Bypass Detection

```javascript
// Baseline: Usuario normal accede a /api/user/123 → 200 OK
// Variación: Mismo request sin Authorization header

// Si la variación devuelve 200 OK:
// → DETECCIÓN: Authentication Bypass (CWE-287)
```

**Detectado cuando:**
- Baseline: Status ≥ 400 (requiere auth)
- Variación: Status < 300 (acceso concedido sin auth)

### 2. IDOR Detection

```javascript
// Baseline: GET /api/user/9999 (ID inexistente) → 404 Not Found
// Variación: GET /api/user/1 (ID diferente) → 200 OK con datos

// → DETECCIÓN: Insecure Direct Object Reference (CWE-639)
```

**Detectado cuando:**
- Baseline: 404 Not Found
- Variación: 200 OK con body sustancial
- Mutación: Cambio de ID numérico

### 3. Privilege Escalation

```javascript
// Baseline: Usuario normal → respuesta con campos básicos
// Variación: Mismo request con header X-Role: admin

// Si la variación tiene campos adicionales (is_admin, permissions):
// → DETECCIÓN: Schema Field Leak (CWE-200)
```

**Detectado cuando:**
- Variación tiene campos JSON que baseline no tiene
- Especialmente campos sensibles: `role`, `permissions`, `is_admin`

### 4. Timing Attack Detection

```javascript
// Baseline: GET /api/user/existente → 200 OK (150ms)
// Variación: GET /api/user/no-existente → 404 (50ms)

// Diferencia significativa (>500ms):
// → DETECCIÓN: Timing Side Channel (CWE-208)
```

**Detectado cuando:**
- Diferencia de tiempo > 500ms entre baseline y variación

### 5. Error Information Disclosure

```javascript
// Variación devuelve error con:
// - Stack trace
// - Ruta de archivo (/var/www/app/...)
// - Query SQL completa
// - Nombres de tablas/columnas

// → DETECCIÓN: Information Disclosure in Error Messages (CWE-209)
```

---

## Mutaciones Automáticas

El engine genera automáticamente variaciones basadas en patrones:

### ID Mutations (IDOR Testing)

```javascript
// Si detecta parámetros como id, user_id, order_id:
const idMutations = [
  { param: "id", value: "1" },
  { param: "id", value: "2" },
  { param: "id", value: "999" },
  { param: "id", value: "9999" },
  { param: "id", value: "-1" },
  { param: "id", value: "../admin" }
];
```

### Role Header Injection

```javascript
// Agrega headers comunes de control de acceso:
const roleHeaders = [
  { "X-User-Role": "admin" },
  { "X-User-Role": "superadmin" },
  { "X-Admin-Token": "true" },
  { "X-Forwarded-For": "127.0.0.1" }
];
```

### Auth Bypass Mutations

```javascript
// Pruebas de bypass de autenticación:
const authMutations = [
  { remove: "Authorization" },
  { remove: "Cookie" },
  { replace: "Authorization", value: "Bearer null" },
  { replace: "Authorization", value: "" }
];
```

### Path Mutations

```javascript
// Pruebas de acceso a rutas administrativas:
const pathMutations = [
  { path: "/api/admin/users" },
  { path: "/internal/api/config" },
  { path: "/debug" }
];
```

---

## Flujo de Trabajo

### Paso 1: Capturar Baseline

```javascript
// Navegar por la aplicación normalmente
// Capturar requests legítimos

// En el proxy:
proxy.history.search({
  host: "api.target.com",
  method: "GET",
  limit: 10
});
```

### Paso 2: Ejecutar Smart Replay

```javascript
// Seleccionar un request interesante (con ID o auth)
const targetRequestId = "req-abc-123";

// El engine generará automáticamente mutaciones
// y ejecutará todas las combinaciones

proxy.replay.execute({
  request_id: targetRequestId,
  smart_mode: true,  // Activa mutaciones automáticas
  max_variations: 20  // Limitar a 20 variaciones
});
```

### Paso 3: Revisar Hallazgos

```javascript
// Ver resultados del análisis diferencial
proxy.findings.list({});

// Ejemplo de output:
// 🚨 SECURITY FINDINGS (3)
// [CRITICAL] auth_bypass_status_change
//    Request: abc-123
//    Authorization bypass: 401 → 200
//    CWE: CWE-287
//
// [HIGH] idor_data_access
//    Request: abc-123
//    Posible IDOR: acceso a recursos con ID modificado
//    CWE: CWE-639
//
// [MEDIUM] schema_field_leak
//    Request: abc-123
//    Campos adicionales: ["is_admin", "api_key"]
//    CWE: CWE-200
```

### Paso 4: Validación Manual

```javascript
// Para hallazgos críticos, validar manualmente:

proxy.request.get({
  request_id: "req-con-finding"
});

// Revisar evidencia y replicar para confirmar
```

---

## Uso Avanzado

### Mutaciones Personalizadas

```javascript
// Definir mutaciones específicas para tu app
proxy.replay.execute({
  request_id: "req-abc",
  mutations: [
    {
      type: "header",
      target: "X-Tenant-ID",
      operation: "replace",
      value: "other-tenant-123"
    },
    {
      type: "body",
      target: "organization_id",
      operation: "replace",
      value: "competitor-org"
    }
  ]
});
```

### Race Condition Testing

```javascript
// Ejecutar múltiples requests concurrentes
proxy.replay.execute({
  request_id: "coupon-redeem",
  times: 50,
  concurrent: true,
  delay_ms: 0  // Sin delay entre requests
});

// Analizar si múltiples requests tienen éxito
// (debería fallar por limitación de uso único)
```

### Time-Based Blind Detection

```javascript
// Comparar tiempos de respuesta
// Útil para blind SQL injection, blind XSS, etc.

const baseline = await proxy.replay.execute({
  request_id: "search-req",
  modifications: { body: { query: "normal" } }
});

const injection = await proxy.replay.execute({
  request_id: "search-req",
  modifications: { body: { query: "' OR SLEEP(5) --" } }
});

// Si injection.response_time > baseline.response_time + 4000ms
// → Posible blind SQL injection
```

---

## Configuración de Detectores

Los detectores son reglas que evalúan baseline vs variación:

```go
// Ejemplo de detector personalizado
{
    Name: "Custom API Key Leak",
    Type: "api_key_exposure",
    Severity: "HIGH",
    CWE: "CWE-798",
    Condition: func(baseline, variation *ReplayResult) *DetectionFinding {
        // Detectar si la respuesta contiene API keys
        apiKeyPattern := regexp.MustCompile(`[a-zA-Z0-9]{32,}`)
        
        if apiKeyPattern.Match(variation.ResponseBody) {
            return &DetectionFinding{
                Type: "api_key_exposure",
                Severity: "HIGH",
                Description: "API key expuesta en respuesta",
                CWE: "CWE-798",
            }
        }
        return nil
    },
}
```

---

## Métricas y Reportes

### Cobertura de Testing

```javascript
// El engine trackea:
// - Requests testeados
// - Mutaciones aplicadas
// - Hallazgos por categoría
// - Tasa de éxito de detección

proxy.stats.get({});

// Output:
// {
//   total_requests: 150,
//   smart_replay_executions: 45,
//   total_variations: 890,
//   findings: {
//     critical: 3,
//     high: 8,
//     medium: 15,
//     low: 7
//   },
//   detection_rate: "4.0%"  // hallazgos / variaciones
// }
```

### Exportar Reporte

```javascript
// Generar reporte estructurado
proxy.replay.export_report({
  output_path: "./reports/smart-replay-findings.json",
  format: "sarif"  // SARIF, JSON, HTML
});
```

---

## Integración con CI/CD

```yaml
# .github/workflows/security-scan.yml
name: Smart Replay Security Scan

on: [pull_request]

jobs:
  smart-replay:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Start AuditForge Proxy
        run: |
          go run cmd/proxy-server &
          sleep 5
      
      - name: Run integration tests
        run: |
          HTTP_PROXY=http://localhost:8080 npm test
      
      - name: Execute Smart Replay
        run: |
          auditforge-proxy replay --auto --fail-on-critical
      
      - name: Upload findings
        uses: actions/upload-artifact@v3
        with:
          name: security-findings
          path: ./reports/
```

---

## Limitaciones y Consideraciones

### Performance
- Smart replay puede generar muchas peticiones
- Usar `max_variations` para limitar
- Ejecutar en ambientes de staging, no producción

### Falsos Positivos
- Algunos cambios de status pueden ser legítimos
- Revisar siempre hallazgos antes de reportar
- Validar con evidencia reproducible

### Alcance
- Solo detecta vulnerabilidades con respuesta observable
- No detecta blind attacks sin time delay
- Requiere tráfico baseline representativo

---

## Checklist de Smart Replay

```
[ ] Tráfico baseline capturado y representativo
[ ] Requests con IDs o autenticación identificados
[ ] Smart replay ejecutado con mutaciones automáticas
[ ] Hallazgos revisados manualmente
[ ] Evidencia de cada hallazgo guardada
[ ] Falsos positivos descartados
[ ] Hallazgos validados exportados a memoria
[ ] Reporte generado con métricas
```
