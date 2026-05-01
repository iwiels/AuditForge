# 🎯 AuditForge Proxy + Smart Replay Engine

## Implementación Completa de Intercepción Proxy y Análisis Diferencial

Esta implementación agrega dos capacidades críticas a AuditForge:

1. **MCP Proxy Server**: Proxy HTTP/HTTPS nativo expuesto como MCP server
2. **Smart Replay Engine**: Motor de análisis diferencial semántico para detección automática de vulnerabilidades

---

## 📁 Estructura de Archivos

```
cmd/proxy-server/
├── main.go              # Entry point del servidor MCP
├── proxy_core.go        # Lógica del proxy MITM
├── storage.go           # Persistencia SQLite
├── mcp_tools.go         # Tools MCP expuestos
└── smart_replay.go      # Motor de análisis diferencial

internal/assets/skills/
├── auditforge-proxy/        # Skill de integración con proxy
│   └── SKILL.md
└── smart-replay-engine/     # Skill del motor de replay
    └── SKILL.md

internal/assets/mcp-config/
└── auditforge-proxy.json    # Configuración MCP para OpenCode
```

---

## 🚀 Instalación y Setup

### 1. Instalar Dependencias

```bash
# Instalar dependencias de Go
cd cmd/proxy-server
go mod init auditforge-proxy 2>/dev/null || true
go get github.com/google/uuid
go get github.com/mark3labs/mcp-go
go get github.com/mattn/go-sqlite3
```

### 2. Generar Certificado CA

```bash
# Crear directorio para certificados
mkdir -p certs

# Generar certificado raíz (MITM para HTTPS)
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
  -keyout certs/ca.key -out certs/ca.crt \
  -subj "/CN=AuditForge Proxy CA" \
  -addext "basicConstraints=critical,CA:TRUE"

# Generar certificado para el proxy
openssl req -newkey rsa:2048 -nodes -keyout certs/proxy.key \
  -out certs/proxy.csr -subj "/CN=localhost"

openssl x509 -req -in certs/proxy.csr -CA certs/ca.crt -CAkey certs/ca.key \
  -CAcreateserial -out certs/proxy.crt -days 365 -sha256
```

### 3. Instalar Certificado CA en el Sistema

**macOS:**
```bash
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain certs/ca.crt
```

**Windows (PowerShell Admin):**
```powershell
Import-Certificate -FilePath ".\certs\ca.crt" `
  -CertStoreLocation Cert:\LocalMachine\Root
```

**Linux:**
```bash
sudo cp certs/ca.crt /usr/local/share/ca-certificates/auditforge.crt
sudo update-ca-certificates
```

**Firefox:**
- Preferences → Privacy & Security → View Certificates
- Import → Seleccionar `certs/ca.crt`
- Trust this CA to identify websites

---

## 🎮 Uso

### Iniciar el Servidor Proxy

```bash
cd cmd/proxy-server
go run .
```

Verás output similar a:
```
🚀 Starting AuditForge Proxy Server...
[PROXY] 2024/01/15 10:30:45 Proxy server listening on localhost:8080
📡 MCP Server ready. Waiting for connections...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔧 MCP Tools Available:
   • proxy.intercept.enable
   • proxy.intercept.disable
   • proxy.history.search
   • proxy.request.get
   • proxy.request.modify
   • proxy.request.forward
   • proxy.request.drop
   • proxy.replay.execute
   • proxy.stats.get
   • proxy.findings.list
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🌐 Proxy listening on: http://localhost:8080
   Configure your browser/app to use this proxy
```

### Configurar OpenCode

Agregar a tu configuración de OpenCode (`~/.config/opencode/opencode.json`):

```json
{
  "mcpServers": {
    "auditforge-proxy": {
      "command": "go",
      "args": ["run", "C:/Users/victo/GitHub/orquestador_auditor/cmd/proxy-server"],
      "env": {
        "PROXY_PORT": "8080"
      }
    }
  }
}
```

### Configurar Browser

**Chrome:**
```bash
# Iniciar Chrome con proxy
chrome --proxy-server=http://localhost:8080
```

O configurar en Settings → System → Open proxy settings

**Firefox:**
- Settings → Network Settings → Manual proxy configuration
- HTTP Proxy: `localhost` Port: `8080`
- HTTPS Proxy: `localhost` Port: `8080`

### Uso con cURL

```bash
export HTTP_PROXY=http://localhost:8080
export HTTPS_PROXY=http://localhost:8080

curl https://api.target.com/users
```

---

## 🧪 Flujo de Trabajo Completo

### 1. Habilitar Interceptación

```javascript
// En OpenCode, con el MCP proxy activo:

proxy.intercept.enable({
  filters: {
    host_pattern: "api.target.com",
    path_pattern: "/api/v1"
  }
});
```

### 2. Navegar y Capturar

Navegar normalmente por la aplicación. El proxy capturará automáticamente el tráfico.

### 3. Revisar Historial

```javascript
// Ver últimos requests
proxy.history.search({ limit: 20 });

// Buscar específicos
proxy.history.search({
  host: "api.target.com",
  method: "GET",
  path: "/users"
});
```

### 4. Interceptar y Modificar

```javascript
// Cuando un request es interceptado, obtener su ID
const intercepted = await proxy.history.search({
  intercepted_only: true,
  limit: 1
});

// Modificar y enviar
await proxy.request.modify({
  request_id: intercepted[0].id,
  headers: {
    "Authorization": "Bearer ADMIN_TOKEN",
    "X-User-Role": "admin"
  },
  body: JSON.stringify({
    user_id: 999,
    role: "superadmin"
  })
});
```

### 5. Smart Replay

```javascript
// Ejecutar análisis diferencial automático
proxy.replay.execute({
  request_id: "req-abc-123",
  smart_mode: true,
  max_variations: 20
});
```

### 6. Revisar Hallazgos

```javascript
// Ver todos los hallazgos
proxy.findings.list({});

// Filtrar por severidad
proxy.findings.list({ severity: "CRITICAL" });
```

---

## 📊 Detección Automática

El Smart Replay Engine detecta automáticamente:

| Vulnerabilidad | Trigger | Severidad |
|----------------|---------|-----------|
| **Auth Bypass** | 403 → 200 | CRITICAL |
| **IDOR** | 404 → 200 con ID diferente | HIGH |
| **Privilege Escalation** | Campos extra en respuesta | MEDIUM |
| **Info Disclosure** | Stack traces, paths en errores | MEDIUM |
| **Timing Attack** | Diferencia >500ms | LOW |

---

## 🔧 Arquitectura Técnica

### Componentes Principales

```
┌─────────────────────────────────────────────────────┐
│                  MCP Server (stdio)                 │
│  ┌─────────────────┐  ┌──────────────────────────┐  │
│  │   MCPServerTools │  │    SmartReplayEngine     │  │
│  │   (mcp_tools.go) │  │    (smart_replay.go)     │  │
│  └────────┬────────┘  └────────────┬─────────────┘  │
└───────────┼────────────────────────┼────────────────┘
            │                        │
            ▼                        ▼
┌─────────────────┐          ┌──────────────────┐
│  ProxyServer    │          │ MutationGenerator│
│  (proxy_core.go)│          │                  │
└────────┬────────┘          └──────────────────┘
         │
         ▼
┌─────────────────┐     ┌──────────────────┐
│  Interceptor    │────→│  ProxyStorage    │
│                 │     │  (SQLite)        │
└─────────────────┘     └──────────────────┘
```

### Base de Datos SQLite

**Tablas:**

```sql
-- Requests capturados
requests (
  id, timestamp, method, url, host, path, query,
  request_headers, request_body,
  response_status, response_headers, response_body,
  duration_ms, is_intercepted, intercept_action, tags
);

-- Hallazgos de seguridad
findings (
  id, request_id, finding_type, severity,
  description, evidence, cwe, created_at
);
```

### Detectores Implementados

En `smart_replay.go`, los detectores evalúan:

```go
// Ejemplo: Status Code Bypass
detectors = []DetectionRule{
    {
        Name: "Status Code Bypass",
        Type: "auth_bypass",
        Severity: "CRITICAL",
        CWE: "CWE-287",
        Condition: func(baseline, variation *ReplayResult) *DetectionFinding {
            if baseline.Response.StatusCode >= 400 && 
               variation.Response.StatusCode < 300 {
                return &DetectionFinding{
                    Type: "auth_bypass_status_change",
                    Severity: "CRITICAL",
                    // ...
                }
            }
            return nil
        },
    },
    // ... más detectores
}
```

---

## 🧪 Testing

### Tests Unitarios

```bash
cd cmd/proxy-server

go test -v ./...
```

### Test Manual

```bash
# 1. Iniciar proxy
go run . &

# 2. Configurar proxy
curl -x http://localhost:8080 -I https://httpbin.org/get

# 3. Verificar captura
# (En OpenCode)
proxy.history.search({ host: "httpbin.org" });
```

---

## 🔒 Seguridad

### Consideraciones Importantes

1. **Certificado CA**: Nunca compartir la clave privada (`ca.key`)
2. **Scope**: Solo interceptar targets autorizados explícitamente
3. **Datos Sensibles**: La base de datos SQLite puede contener tokens y cookies
4. **Limpieza**: Ejecutar `proxy.storage.delete_old_requests({ older_than: "24h" })` regularmente

### Configuración de Seguridad

```javascript
// Desactivar interceptación después de usar
proxy.intercept.disable();

// Limpiar requests antiguos
proxy.storage.delete_old_requests({
  older_than: "1h"
});

// Exportar y luego borrar DB
proxy.export.har({ output_path: "./evidence.har" });
// rm auditforge-proxy.db
```

---

## 🚀 Roadmap Futuro

### Corto Plazo
- [ ] WebSocket interception completo
- [ ] Response modification (no solo request)
- [ ] Exportación a formato Burp Suite
- [ ] Filtros avanzados con regex

### Mediano Plazo
- [ ] Plugin system para detectores personalizados
- [ ] Integración con Burp Collaborator para OOB
- [ ] Análisis de tráfico en tiempo real con ML
- [ ] Soporte para gRPC

### Largo Plazo
- [ ] Distributed proxy (múltiples nodos)
- [ ] Replay automático en CI/CD
- [ ] Integración con herramientas de fuzzing
- [ ] Dashboard web para visualización

---

## 📚 Recursos Adicionales

- **Skills de integración**: Ver `internal/assets/skills/auditforge-proxy/SKILL.md`
- **Smart Replay Engine**: Ver `internal/assets/skills/smart-replay-engine/SKILL.md`
- **OWASP Testing Guide**: https://owasp.org/www-project-web-security-testing-guide/

---

## 🤝 Contribución

Para agregar nuevos detectores al Smart Replay Engine:

1. Editar `smart_replay.go`
2. Agregar `DetectionRule` al slice `detectors`
3. Implementar función `Condition`
4. Testear con tráfico de ejemplo
5. Documentar en el skill correspondiente

---

## 📄 Licencia

MIT - Ver LICENSE en el proyecto principal.

---

**AuditForge Proxy** - *Interceptación inteligente, análisis diferencial, auditorías más profundas.*
