# Ejemplo Práctico: Auditoría Completa con AuditForge Proxy

Este ejemplo demuestra un flujo de trabajo completo de auditoría usando el MCP Proxy y Smart Replay Engine.

## Escenario

**Target**: `https://api.ecommerce-demo.com`
**Objetivo**: Encontrar vulnerabilidades de IDOR y auth bypass en el API de órdenes

---

## Paso 1: Iniciar el Entorno

### Terminal 1: Iniciar Proxy
```bash
cd cmd/proxy-server
./start-proxy.sh

# Output esperado:
# 🚀 Starting AuditForge Proxy Server...
# [PROXY] Proxy server listening on localhost:8080
# 📡 MCP Server ready...
```

### Terminal 2: Configurar Browser
```bash
# Chrome con proxy
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome \
  --proxy-server=http://localhost:8080 \
  --ignore-certificate-errors \
  --user-data-dir=/tmp/auditforge-chrome
```

---

## Paso 2: Configurar Interceptación

En OpenCode, con el MCP auditforge-proxy activo:

```javascript
// Habilitar interceptación selectiva
proxy.intercept.enable({
  filters: {
    host_pattern: "api.ecommerce-demo.com",
    path_pattern: "/api/v1/orders",
    method_pattern: "GET"
  }
});

// Output:
// ✅ Request interception enabled
//    Filters: host="api.ecommerce-demo.com", path="/api/v1/orders"
```

---

## Paso 3: Capturar Tráfico Baseline

En el browser autenticado como usuario normal:
1. Ir a "Mis Órdenes"
2. Click en "Ver detalle" de una orden

En OpenCode:

```javascript
// Ver requests capturados
proxy.history.search({
  host: "api.ecommerce-demo.com",
  limit: 5
});

// Output:
// Found 3 requests:
//
// 🆔 a1b2c3d4 | GET /api/v1/orders [INTERCEPTED] | Status: 200 | 10:45:22
// 🆔 e5f6g7h8 | GET /api/v1/orders/12345 | Status: 200 | 10:45:23
// 🆔 i9j0k1l2 | GET /api/v1/orders/12345/items | Status: 200 | 10:45:24
```

---

## Paso 4: Análisis del Request

```javascript
// Obtener detalles del request con ID
proxy.request.get({
  request_id: "e5f6g7h8"
});

// Output:
// 📋 REQUEST DETAILS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// 🆔 ID: e5f6g7h8
// 🕐 Timestamp: 2024-01-15 10:45:23
// ⏱️  Duration: 145ms
//
// 📤 REQUEST
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// GET https://api.ecommerce-demo.com/api/v1/orders/12345
// Host: api.ecommerce-demo.com
//
// Headers:
//   Authorization: Bearer eyJhbGciOiJIUzI1NiJ9...
//   Content-Type: application/json
//   X-User-ID: 9876
//
// 📥 RESPONSE
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// Status: 200
//
// Body (458 bytes):
// {
//   "order_id": 12345,
//   "user_id": 9876,
//   "total": 149.99,
//   "status": "shipped",
//   "items": [...]
// }
```

---

## Paso 5: Smart Replay con Detección Automática

```javascript
// Ejecutar análisis diferencial automático
proxy.replay.execute({
  request_id: "e5f6g7h8",
  smart_mode: true,
  max_variations: 15
});

// El engine generará automáticamente:
// - ID mutations: 12345 → 1, 2, 999, 9999, 12344, 12346
// - Auth mutations: remove Authorization header
// - Role mutations: add X-User-Role: admin
```

---

## Paso 6: Revisar Hallazgos

```javascript
// Ver hallazgos detectados
proxy.findings.list({});

// Output:
// 🚨 SECURITY FINDINGS (2)
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//
// [CRITICAL] auth_bypass_status_change
//    Request: e5f6g7h8
//    Authorization bypass: 401 → 200
//    CWE: CWE-287
//    Evidence: {
//      "baseline_status": 401,
//      "variation_status": 200,
//      "mutation_applied": "Remover header Authorization"
//    }
//    Recomendación: Verificar que todos los endpoints validen la autorización
//
// [HIGH] idor_data_access
//    Request: e5f6g7h8
//    Posible IDOR: acceso a recursos con ID modificado
//    CWE: CWE-639
//    Evidence: {
//      "accessible_id": "12346",
//      "response_size": 458,
//      "mutation": "id_fuzz_order_id_12346"
//    }
//    Recomendación: Implementar verificación de ownership en todos los endpoints
```

---

## Paso 7: Validación Manual del IDOR

```javascript
// Validar manualmente el hallazgo IDOR

// Interceptar y modificar
proxy.request.modify({
  request_id: "e5f6g7h8",
  headers: {
    "Authorization": "Bearer eyJhbGciOiJIUzI1NiJ9...",
    "Content-Type": "application/json"
  }
});

// Después modificar el path para cambiar ID
proxy.request.modify({
  request_id: "e5f6g7h8",
  headers: { /* originales */ },
  // Modificar el cuerpo implícitamente cambia la URL
  // Nota: En implementación real, necesitamos modificar el path
});

// Alternativa: Replay específico
proxy.replay.execute({
  request_id: "e5f6g7h8",
  mutations: [
    {
      type: "path",
      target: "path",
      operation: "replace",
      value: "/api/v1/orders/12346"
    }
  ]
});
```

---

## Paso 8: Exportar Evidencia

```javascript
// Exportar a HAR
proxy.export.har({
  output_path: "./evidence/ecommerce-idor-session.har",
  filters: {
    host: "api.ecommerce-demo.com"
  }
});

// Ver estadísticas
proxy.stats.get({});

// Output:
// 📊 PROXY STATISTICS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// 📨 Total Requests: 47
// ⏸️  Intercepted: 12
// 🌐 Unique Hosts: 3
// 🚨 Security Findings: 2
//
// Severity Distribution:
//    CRITICAL: 1
//    HIGH: 1
//    MEDIUM: 0
```

---

## Paso 9: Reporte Final

```javascript
// Generar reporte completo
const report = {
  engagement: "ecommerce-demo-audit",
  date: "2024-01-15",
  target: "api.ecommerce-demo.com",
  findings: [
    {
      id: "FND-001",
      type: "Authentication Bypass",
      severity: "CRITICAL",
      cwe: "CWE-287",
      description: "El endpoint /api/v1/orders/{id} permite acceso sin autenticación válida",
      evidence: {
        request_id: "e5f6g7h8",
        baseline_status: 401,
        bypass_status: 200,
        reproduction_steps: [
          "1. Capturar request autenticado a /api/v1/orders/12345",
          "2. Remover header Authorization",
          "3. Enviar request → Recibe 200 OK con datos de la orden"
        ]
      },
      remediation: "Implementar middleware de autenticación obligatorio en todos los endpoints de API"
    },
    {
      id: "FND-002",
      type: "IDOR - Insecure Direct Object Reference",
      severity: "HIGH",
      cwe: "CWE-639",
      description: "Es posible acceder a órdenes de otros usuarios modificando el ID en la URL",
      evidence: {
        request_id: "e5f6g7h8",
        original_id: 12345,
        accessed_id: 12346,
        proof: "Datos de orden de otro usuario expuestos"
      },
      remediation: "Verificar que el usuario autenticado sea el dueño del recurso antes de servirlo"
    }
  ],
  tools_used: ["auditforge-proxy", "smart-replay-engine"],
  confidence: "validated"  // observed | suspected | validated
};

// Guardar en memoria compartida
// (Integración con security-memory)
```

---

## Resumen del Flujo

```
1. Iniciar Proxy         → Terminal: ./start-proxy.sh
2. Configurar Browser    → Chrome/Firefox con proxy localhost:8080
3. Habilitar Intercepción → proxy.intercept.enable({filters})
4. Navegar Normalmente   → Captura automática
5. Smart Replay          → proxy.replay.execute({smart_mode: true})
6. Revisar Hallazgos     → proxy.findings.list({})
7. Validación Manual     → proxy.request.modify({})
8. Exportar Evidencia    → proxy.export.har({})
9. Generar Reporte       → Integración con security-report
```

---

## Comandos Rápidos

```bash
# Setup inicial
./setup.sh

# Iniciar proxy
./start-proxy.sh

# Ver logs en tiempo real
tail -f logs/proxy.log

# Limpiar base de datos
rm auditforge-proxy.db

# Generar nuevos certificados
rm certs/*
./setup.sh
```

---

## Troubleshooting

### Error: "certificate signed by unknown authority"
**Solución**: Instalar el certificado CA en el sistema/browser
```bash
# macOS
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain certs/ca.crt

# Verificar
openssl s_client -connect localhost:8080 -showcerts
```

### Error: "address already in use"
**Solución**: Matar proceso en el puerto 8080
```bash
# macOS/Linux
lsof -ti:8080 | xargs kill -9

# Windows
netstat -ano | findstr :8080
taskkill /PID <PID> /F
```

### Requests no aparecen en history
**Verificar**:
1. ¿El proxy está corriendo? `curl -x http://localhost:8080 https://httpbin.org/get`
2. ¿El browser tiene configurado el proxy?
3. ¿Hay filtros activos muy restrictivos?

---

## Próximos Pasos

- [ ] Automatizar replay en CI/CD
- [ ] Agregar detectores personalizados para tu API
- [ ] Integrar con Burp Suite para colaboración
- [ ] Exportar hallazgos a JIRA/Trello
