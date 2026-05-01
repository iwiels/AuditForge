# 🎯 AuditForge Proxy - Cheatsheet Rápido

## 🚀 Inicio Rápido

```bash
# 1. Setup (una vez)
cd cmd/proxy-server && ./setup.sh

# 2. Iniciar proxy
./start-proxy.sh

# 3. Configurar browser
# Chrome: --proxy-server=http://localhost:8080
# Firefox: Network Settings → Manual proxy → localhost:8080

# 4. En OpenCode, usar tools MCP
```

---

## 🛠️ Tools MCP

### Interceptación

```javascript
// Habilitar interceptación (todo)
proxy.intercept.enable({})

// Habilitar con filtros
proxy.intercept.enable({
  filters: {
    host_pattern: "api.target.com",
    path_pattern: "/api/v1",
    method_pattern: "POST"
  }
})

// Deshabilitar
proxy.intercept.disable()
```

### Historial

```javascript
// Listar últimos 20 requests
proxy.history.search({ limit: 20 })

// Buscar específicos
proxy.history.search({
  host: "api.target.com",
  path: "/users",
  method: "GET"
})

// Solo interceptados
proxy.history.search({
  intercepted_only: true
})

// Obtener detalles completos
proxy.request.get({
  request_id: "abc-123-def"
})
```

### Modificación de Requests

```javascript
// Modificar y enviar
proxy.request.modify({
  request_id: "abc-123",
  headers: {
    "Authorization": "Bearer NEW_TOKEN",
    "X-User-Role": "admin"
  },
  body: JSON.stringify({
    user_id: 999,
    role: "superadmin"
  })
})

// Enviar sin cambios
proxy.request.forward({ request_id: "abc-123" })

// Dropear (error al cliente)
proxy.request.drop({
  request_id: "abc-123",
  status_code: 403
})
```

### Smart Replay

```javascript
// Ejecutar con mutaciones automáticas
proxy.replay.execute({
  request_id: "abc-123",
  smart_mode: true,
  max_variations: 20
})

// Replay con mutaciones personalizadas
proxy.replay.execute({
  request_id: "abc-123",
  mutations: [
    {
      type: "header",
      target: "X-User-ID",
      operation: "replace",
      value: "999"
    },
    {
      type: "param",
      target: "order_id",
      operation: "replace",
      value: "12346"
    }
  ]
})

// Replay masivo (race condition)
proxy.replay.execute({
  request_id: "coupon-redeem",
  times: 50,
  concurrent: true
})
```

### Hallazgos

```javascript
// Listar todos
proxy.findings.list({})

// Filtrar por severidad
proxy.findings.list({ severity: "CRITICAL" })
proxy.findings.list({ severity: "HIGH" })

// Filtrar por tipo
proxy.findings.list({ type: "idor" })
proxy.findings.list({ type: "auth_bypass" })
```

### Estadísticas y Exportación

```javascript
// Ver estadísticas
proxy.stats.get({})

// Exportar a HAR
proxy.export.har({
  output_path: "./evidence/session.har",
  filters: {
    host: "api.target.com"
  }
})
```

---

## 🎭 Flujos de Trabajo Comunes

### Flujo 1: IDOR Testing

```javascript
// 1. Capturar tráfico navegando
proxy.intercept.enable({})

// 2. Buscar request con ID
const reqs = await proxy.history.search({
  path: "/orders/",
  limit: 5
})

// 3. Smart replay para detectar IDOR
await proxy.replay.execute({
  request_id: reqs[0].id,
  smart_mode: true
})

// 4. Ver hallazgos
await proxy.findings.list({
  type: "idor"
})
```

### Flujo 2: Auth Bypass Testing

```javascript
// 1. Capturar request autenticado
const req = await proxy.history.search({
  path: "/api/admin",
  limit: 1
})

// 2. Probar sin auth
await proxy.replay.execute({
  request_id: req[0].id,
  mutations: [
    {
      type: "header",
      target: "Authorization",
      operation: "remove"
    }
  ]
})

// 3. Verificar hallazgos
await proxy.findings.list({
  severity: "CRITICAL"
})
```

### Flujo 3: Intercepción Manual

```javascript
// 1. Habilitar interceptación selectiva
proxy.intercept.enable({
  filters: {
    host_pattern: "api.target.com",
    method_pattern: "POST"
  }
})

// 2. En la app, hacer acción que dispara POST
// El request se pausa automáticamente

// 3. Obtener request interceptado
const intercepted = await proxy.history.search({
  intercepted_only: true,
  limit: 1
})

// 4. Modificar y enviar
await proxy.request.modify({
  request_id: intercepted[0].id,
  headers: {
    ...intercepted[0].request_headers,
    "X-Debug": "true"
  }
})
```

---

## 🔍 Patrones de Búsqueda

### Buscar APIs interesantes

```javascript
// Endpoints de administración
proxy.history.search({
  path: "/admin",
  limit: 50
})

// Requests con IDs numéricos (candidatos IDOR)
proxy.history.search({
  path: "/\\d+",  // regex en implementación real
  limit: 50
})

// Requests POST a endpoints sensibles
proxy.history.search({
  method: "POST",
  path: "/api/",
  limit: 50
})
```

---

## 🚨 Detecciones Automáticas

| Tipo | Descripción | Severidad |
|------|-------------|-----------|
| `auth_bypass_status_change` | 401→200 | CRITICAL |
| `idor_data_access` | 404→200 con ID diferente | HIGH |
| `schema_field_leak` | Campos nuevos en respuesta | MEDIUM |
| `error_info_disclosure` | Stack traces en errores | MEDIUM |
| `timing_side_channel` | Δ > 500ms | LOW |

---

## 🔧 Configuración

### Variables de Entorno

```bash
# Terminal con proxy
cd cmd/proxy-server
./start-proxy.sh

# Terminal para curl
export HTTP_PROXY=http://localhost:8080
export HTTPS_PROXY=http://localhost:8080

# Verificar
curl -I https://httpbin.org/get
```

### Config OpenCode

`~/.config/opencode/opencode.json`:

```json
{
  "mcpServers": {
    "auditforge-proxy": {
      "command": "./auditforge-proxy",
      "args": [],
      "env": {
        "PROXY_PORT": "8080"
      }
    }
  }
}
```

---

## 🐛 Troubleshooting

### Problema: Certificado no confiable

```bash
# Reinstalar certificado
./setup.sh

# O manualmente:
# macOS
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain certs/ca.crt

# Windows (PowerShell admin)
Import-Certificate -FilePath ".\certs\ca.crt" `
  -CertStoreLocation Cert:\LocalMachine\Root
```

### Problema: Requests no aparecen

```bash
# Verificar proxy está corriendo
curl -x http://localhost:8080 -I https://httpbin.org/get

# Verificar browser tiene proxy configurado
# Chrome: chrome://settings/?search=proxy
# Firefox: about:preferences#general → Network Settings

# Verificar filtros no son muy restrictivos
proxy.intercept.enable({})  // Sin filtros = todo
```

### Problema: Puerto 8080 en uso

```bash
# macOS/Linux
lsof -ti:8080 | xargs kill -9

# Windows
netstat -ano | findstr :8080
taskkill /PID <PID> /F
```

---

## 📊 Métricas Útiles

```javascript
// Ver cobertura de testing
const stats = await proxy.stats.get({})
console.log(`
  Total requests: ${stats.total_requests}
  Intercepted: ${stats.intercepted_requests}
  Findings: ${stats.total_findings}
  Critical: ${stats.findings_by_severity.CRITICAL || 0}
  High: ${stats.findings_by_severity.HIGH || 0}
`)

// Calcular tasa de detección
const detectionRate = (stats.total_findings / stats.total_requests * 100).toFixed(2)
console.log(`Detection rate: ${detectionRate}%`)
```

---

## 🔗 Integración con Equipo

```javascript
// security-scout descubre endpoints
// security-web usa proxy para:

// 1. Capturar tráfico real
const traffic = await proxy.history.search({
  host: "api.target.com"
})

// 2. Ejecutar smart replay
const findings = await proxy.replay.execute({
  request_id: traffic[0].id,
  smart_mode: true
})

// 3. Guardar en memoria compartida
// para security-code y security-report
```

---

## 💡 Tips Avanzados

### Tip 1: Filtros por URL

```javascript
// Interceptar solo APIs de usuarios
proxy.intercept.enable({
  filters: {
    path_pattern: "/api/users|/api/orders|/api/admin"
  }
})
```

### Tip 2: Análisis de tiempo

```javascript
// Detectar time-based vulnerabilities
const results = await proxy.replay.execute({
  request_id: "search-req",
  mutations: [
    { type: "param", target: "q", value: "normal" },
    { type: "param", target: "q", value: "' OR SLEEP(5) --" }
  ]
})

// Comparar tiempos de respuesta
```

### Tip 3: Batch Replay

```javascript
// Probar múltiples requests
const requests = await proxy.history.search({ limit: 10 })

for (const req of requests) {
  await proxy.replay.execute({
    request_id: req.id,
    smart_mode: true,
    max_variations: 10  // Limitar por request
  })
}
```

### Tip 4: Exportar evidencia

```javascript
// Después de una sesión
await proxy.export.har({
  output_path: `./evidence/${new Date().toISOString()}.har`
})

// Limpiar DB antigua
await proxy.storage.delete_old_requests({
  older_than: "24h"
})
```

---

## 📚 Recursos

- **Setup completo**: `cmd/proxy-server/README.md`
- **Ejemplo práctico**: `cmd/proxy-server/EXAMPLE.md`
- **Skill Proxy**: `internal/assets/skills/auditforge-proxy/SKILL.md`
- **Skill Replay**: `internal/assets/skills/smart-replay-engine/SKILL.md`
- **Diagrama**: `ARCHITECTURE_DIAGRAM.txt`

---

**Recuerda**: Siempre obtener autorización explícita antes de interceptar tráfico de aplicaciones en producción.
