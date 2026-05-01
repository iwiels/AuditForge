# Skill: AuditForge Proxy Integration

**Categoría:** proxy, interception, replay
**Metodología base:** OWASP WSTG-CRYP, WSTG-ATHN, WSTG-ATHZ
**Cuándo activar:** cuando se necesita interceptación fuera del browser, análisis de APIs nativas, o pruebas multi-aplicación
**MCP requerido:** `auditforge-proxy` (servidor MCP standalone)

---

## Arquitectura

```
┌─────────────────┐     ┌─────────────────────┐     ┌──────────────┐
│   Aplicación    │────→│  auditforge-proxy   │────→│    Target    │
│  (Browser/App)  │←────│   (MCP Server)      │←────│   (API/Web)  │
└─────────────────┘     └─────────────────────┘     └──────────────┘
         ↑                       ↓
  HTTP_PROXY=localhost:8080   MCP Tools:
                                - proxy.intercept.enable()
                                - proxy.request.modify()
                                - proxy.replay.execute()
                                - proxy.findings.list()
```

---

## Setup Inicial

### 1. Iniciar el Servidor Proxy

```bash
# Terminal 1: Iniciar el servidor MCP proxy
cd cmd/proxy-server
go run .

# El servidor iniciará:
# - Proxy HTTP/HTTPS en localhost:8080
# - MCP server en stdio para integración con OpenCode
```

### 2. Configurar Certificado CA

```bash
# Generar certificado raíz (solo una vez)
auditforge-proxy init-certs

# Instalar certificado en el sistema/browser
# macOS:
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ./certs/ca.crt

# Windows (PowerShell admin):
Import-Certificate -FilePath ".\certs\ca.crt" -CertStoreLocation Cert:\LocalMachine\Root

# Firefox: Preferences → Privacy & Security → View Certificates → Import
```

### 3. Configurar Aplicación para usar Proxy

**Browser:**
```
Settings → Network → Proxy
HTTP Proxy: localhost:8080
HTTPS Proxy: localhost:8080
```

**cURL:**
```bash
curl -x http://localhost:8080 https://target.com/api/users
```

**Aplicación Node.js:**
```javascript
process.env.HTTP_PROXY = 'http://localhost:8080';
process.env.HTTPS_PROXY = 'http://localhost:8080';
```

**Mobile App (con redirección):**
```bash
# Usar con mitmproxy o herramientas de redirección
adb reverse tcp:8080 tcp:8080
```

---

## Fase 1: Captura Pasiva

### Habilitar Interceptación Selectiva

```javascript
// Interceptar solo requests a APIs específicas
proxy.intercept.enable({
  filters: {
    host_pattern: "api.target.com",
    path_pattern: "/api/v1",
    method_pattern: "POST"
  }
})
```

### Revisar Historial Capturado

```javascript
// Listar últimos requests
proxy.history.search({
  limit: 20
})

// Buscar requests específicos
proxy.history.search({
  host: "api.target.com",
  path: "/users",
  method: "GET"
})
```

### Obtener Detalles de un Request

```javascript
proxy.request.get({
  request_id: "abc-123-def"
})
```

---

## Fase 2: Intercepción Activa (Burp-style)

### Flujo de Interceptación

```
Browser → Proxy → [PAUSA] → OpenCode decide → Servidor
                     ↑
              proxy.request.modify()
              proxy.request.forward()
              proxy.request.drop()
```

### Modificar Request antes de enviar

```javascript
// 1. Esperar a que un request sea interceptado
// (El proxy pausa automáticamente cuando interception está activa)

// 2. Obtener el ID del request interceptado
const intercepted = await proxy.history.search({
  intercepted_only: true,
  limit: 1
});

// 3. Modificar y forward
await proxy.request.modify({
  request_id: intercepted[0].id,
  headers: {
    ...intercepted[0].request_headers,
    "X-User-Role": "admin",
    "X-Debug-Mode": "true"
  },
  body: JSON.stringify({
    ...JSON.parse(intercepted[0].request_body),
    role: "superadmin"
  })
});
```

### Secuencia de Pruebas Comunes

```javascript
// Test 1: IDOR - Cambiar ID de usuario
await proxy.request.modify({
  request_id: reqId,
  headers: { /* original */ },
  body: JSON.stringify({ user_id: 999 })  // ID diferente
});

// Test 2: Auth Bypass - Remover token
await proxy.request.modify({
  request_id: reqId,
  headers: {
    // Authorization removido
    "Content-Type": "application/json"
  }
});

// Test 3: Privilege Escalation - Agregar header de rol
await proxy.request.modify({
  request_id: reqId,
  headers: {
    ...originalHeaders,
    "X-Admin-Token": "true",
    "X-User-Permissions": "admin"
  }
});
```

---

## Fase 3: Smart Replay Engine

### Replay Simple con Modificaciones

```javascript
// Replicar un request con cambios
proxy.replay.execute({
  request_id: "abc-123",
  modifications: {
    headers: {
      "Authorization": "Bearer ADMIN_TOKEN"
    },
    body_params: {
      "user_id": "999"
    }
  },
  times: 1
})
```

### Replay Masivo para Race Conditions

```javascript
// Enviar 10 requests concurrentes para test de race condition
proxy.replay.execute({
  request_id: "coupon-redeem-req",
  modifications: {},
  times: 10,
  concurrent: true
})
```

### Generar Variaciones Automáticas

```javascript
// El Smart Replay Engine puede generar mutaciones automáticas:
// - ID fuzzing (IDOR)
// - Role header injection
// - Auth token removal
// - Path traversal

// Esto se activa automáticamente cuando hay múltiples replays
// con diferentes mutaciones
```

---

## Fase 4: Análisis Diferencial

### Comparar Respuestas

El Smart Replay Engine detecta automáticamente:

| Indicador | Descripción | Severidad |
|-----------|-------------|-----------|
| `status_code_change` | 403→200 indica bypass | CRITICAL |
| `idor_data_access` | Acceso a datos con ID diferente | HIGH |
| `schema_field_leak` | Campos extra en respuesta | MEDIUM |
| `timing_side_channel` | Diferencia de tiempo >500ms | LOW |
| `error_info_disclosure` | Stack traces o paths en errores | MEDIUM |

### Revisar Hallazgos

```javascript
// Listar todos los hallazgos detectados
proxy.findings.list({})

// Filtrar por severidad
proxy.findings.list({
  severity: "CRITICAL"
})

// Filtrar por tipo
proxy.findings.list({
  type: "auth_bypass"
})
```

---

## Fase 5: Exportación y Evidencia

### Exportar a HAR

```javascript
proxy.export.har({
  output_path: "./evidence/session-2024-01-15.har",
  filters: {
    host: "api.target.com"
  }
})
```

### Estadísticas de Sesión

```javascript
proxy.stats.get({})

// Retorna:
// {
//   total_requests: 150,
//   intercepted_requests: 23,
//   unique_hosts: 5,
//   total_findings: 8,
//   findings_by_severity: {
//     CRITICAL: 2,
//     HIGH: 3,
//     MEDIUM: 3
//   }
// }
```

---

## Casos de Uso Específicos

### Caso 1: API Nativa Móvil

```javascript
// Configurar app móvil para usar proxy
// Capturar tráfico de la app nativa

proxy.intercept.enable({
  filters: {
    host_pattern: "mobile-api.target.com"
  }
});

// La app nativa no pasa por Chrome DevTools
// pero sí pasa por este proxy
```

### Caso 2: CLI Tools y Scripts

```bash
# Configurar proxy en variables de entorno
export HTTP_PROXY=http://localhost:8080
export HTTPS_PROXY=http://localhost:8080

# Cualquier herramienta CLI ahora pasa por el proxy
curl https://api.target.com/users
http GET https://api.target.com/admin
python script.py  # Requests en Python
```

### Caso 3: WebSockets

```javascript
// El proxy captura tráfico WebSocket también
// (Implementación futura)

proxy.websocket.intercept({
  host_pattern: "ws.target.com"
});
```

---

## Integración con el Equipo de Agentes

```
security-scout → Identifica endpoints de API
       ↓
security-web → Usa auditforge-proxy para:
               - Capturar tráfico real de la app
               - Interceptar y modificar requests
               - Ejecutar smart replay
       ↓
security-code → Recibe findings del replay engine
                - Busca handlers en el código
                - Valida fix de vulnerabilidades
       ↓
security-report → Consolida findings del proxy
```

---

## Checklist de Uso

```
[ ] Certificado CA instalado en browser/sistema
[ ] Servidor proxy iniciado (go run cmd/proxy-server)
[ ] Aplicación configurada para usar proxy
[ ] Interceptación habilitada con filtros apropiados
[ ] Requests capturados revisados en history
[ ] Smart replay ejecutado con mutaciones
[ ] Hallazgos revisados y clasificados
[ ] Evidencia exportada a HAR
[ ] Findings guardados en memoria compartida
```

---

## Anti-Patterns

- **NO** interceptar tráfico de aplicaciones no autorizadas
- **NO** dejar el proxy interceptando todo indefinidamente (usa filtros)
- **NO** modificar requests en producción sin autorización explícita
- **NO** olvidar remover el certificado CA después de la auditoría
- **NO** compartir el certificado CA privado

---

## Troubleshooting

### Certificado no confiable
```bash
# Verificar que el certificado está instalado
openssl s_client -connect localhost:8080 -showcerts
```

### Requests no aparecen en history
```bash
# Verificar que el proxy está corriendo
curl -x http://localhost:8080 -I https://httpbin.org/get

# Verificar configuración de proxy en la app
```

### Interceptación no funciona
```javascript
// Verificar que interception está habilitada
proxy.intercept.enable({});  // Sin filtros = interceptar todo
```
