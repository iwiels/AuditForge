# 🎯 Resumen de Implementación: AuditForge Proxy + Smart Replay Engine

## 📦 Qué se ha creado

### 1. MCP Proxy Server (cmd/proxy-server/)

**Archivos principales:**
- `main.go` - Entry point del servidor MCP
- `proxy_core.go` - Lógica del proxy MITM con interceptación
- `storage.go` - Persistencia SQLite para requests/hallazgos
- `mcp_tools.go` - 12 tools MCP expuestos a OpenCode
- `smart_replay.go` - Motor de análisis diferencial

**Funcionalidades:**
- ✅ Proxy HTTP/HTTPS en localhost:8080
- ✅ Interceptación tipo Burp (pause/modify/forward/drop)
- ✅ MITM HTTPS con certificados dinámicos
- ✅ Persistencia en SQLite
- ✅ Exportación HAR

### 2. Smart Replay Engine

**Capacidades de detección automática:**

```
┌─────────────────────────────────────────────────────────────┐
│                    SMART REPLAY ENGINE                       │
├─────────────────────────────────────────────────────────────┤
│  Baseline Request                                            │
│    GET /api/orders/12345 → 200 OK                           │
│         ↓                                                    │
│  Mutations automáticas                                       │
│    • ID: 12345 → 1, 2, 999, -1, ../admin                    │
│    • Auth: Remover Authorization header                     │
│    • Role: Agregar X-User-Role: admin                       │
│         ↓                                                    │
│  Análisis diferencial                                        │
│    ┌─────────────────┬─────────────────┐                    │
│    │   Variación     │   Detección     │                    │
│    ├─────────────────┼─────────────────┤                    │
│    │ 401 → 200       │ Auth Bypass     │ ← CRITICAL        │
│    │ 404 → 200       │ IDOR            │ ← HIGH            │
│    │ Campos extra    │ Privilege Esc   │ ← MEDIUM          │
│    │ Δtime >500ms    │ Timing Attack   │ ← LOW             │
│    └─────────────────┴─────────────────┘                    │
└─────────────────────────────────────────────────────────────┘
```

### 3. Skills de Integración

**`auditforge-proxy/SKILL.md`**:
- Guía completa de uso del proxy
- Flujo de trabajo paso a paso
- Integración con el equipo de agents
- Troubleshooting

**`smart-replay-engine/SKILL.md`**:
- Documentación del motor de análisis
- Tipos de detección
- Mutaciones automáticas
- Uso avanzado

### 4. Configuración y Setup

- `setup.sh` - Script automatizado de instalación
- `go.mod` - Dependencias de Go
- `auditforge-proxy.json` - Config MCP para OpenCode

---

## 🚀 Cómo usarlo

### Paso 1: Setup (Una vez)

```bash
cd cmd/proxy-server
./setup.sh
```

Esto hará:
- ✅ Verificar Go y OpenSSL
- ✅ Generar certificados CA
- ✅ Instalar certificados en el sistema
- ✅ Instalar dependencias
- ✅ Compilar el binario
- ✅ Crear scripts de inicio

### Paso 2: Iniciar Proxy

```bash
./start-proxy.sh
```

Output:
```
🚀 Starting AuditForge Proxy Server...
📡 MCP Server ready. Waiting for connections...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔧 MCP Tools Available:
   • proxy.intercept.enable
   • proxy.history.search
   • proxy.request.modify
   • proxy.replay.execute
   • proxy.findings.list
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🌐 Proxy listening on: http://localhost:8080
```

### Paso 3: Configurar Browser

**Opción A - Chrome:**
```bash
chrome --proxy-server=http://localhost:8080
```

**Opción B - Firefox:**
- Settings → Network Settings → Manual proxy
- HTTP Proxy: `localhost` Port: `8080`
- HTTPS Proxy: `localhost` Port: `8080`

**Opción C - Variables de entorno:**
```bash
export HTTP_PROXY=http://localhost:8080
export HTTPS_PROXY=http://localhost:8080
curl https://api.target.com/users
```

### Paso 4: Usar en OpenCode

```javascript
// 1. Habilitar interceptación
proxy.intercept.enable({
  filters: {
    host_pattern: "api.target.com"
  }
});

// 2. Navegar por la app (se captura automáticamente)

// 3. Ver historial
proxy.history.search({ limit: 20 });

// 4. Ejecutar Smart Replay
proxy.replay.execute({
  request_id: "abc-123",
  smart_mode: true
});

// 5. Revisar hallazgos
proxy.findings.list({});
```

---

## 📊 Comparativa: Antes vs Después

| Capacidad | Antes (Chrome DevTools) | Ahora (AuditForge Proxy) |
|-----------|------------------------|--------------------------|
| **Interceptación** | Solo Chrome | Cualquier app/browser/CLI |
| **HTTPS** | Nativo | Con certificado CA |
| **Pause/Modify** | Via JS injection | Nativo en proxy |
| **Persistencia** | En memoria (sesión) | SQLite permanente |
| **Análisis** | Manual | Automático (Smart Replay) |
| **Detección IDOR** | Manual | Automático |
| **Detección Auth Bypass** | Manual | Automático |
| **Mutaciones** | Scripts manuales | Generador automático |
| **Exportación** | HAR manual | HAR programático |

---

## 🎭 Casos de Uso

### Caso 1: API Móvil Nativa
```
App móvil → Proxy (localhost:8080) → API backend
   ↑
   └── Captura requests nativos que Chrome DevTools no ve
```

### Caso 2: CLI Tools
```bash
export HTTP_PROXY=localhost:8080
aws s3 ls
kubectl get pods
terraform plan
   ↑
   └── Todo capturado en el proxy
```

### Caso 3: Smart Replay Automático
```javascript
// 1 request base → 15+ variaciones automáticas
// Detección de vulnerabilidades sin escribir código
```

---

## 🔧 Arquitectura

```
┌──────────────────────────────────────────────────────────────┐
│                     OPENCODE / CLI                           │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  MCP Tools:                                            │  │
│  │  • proxy.intercept.enable()                            │  │
│  │  • proxy.request.modify()                              │  │
│  │  • proxy.replay.execute()                              │  │
│  │  • proxy.findings.list()                               │  │
│  └────────────────────┬───────────────────────────────────┘  │
└───────────────────────┼──────────────────────────────────────┘
                        │ stdio
                        ▼
┌──────────────────────────────────────────────────────────────┐
│                AUDITFORGE PROXY SERVER                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   MCP Server │  │Proxy Listener│  │Smart Replay  │       │
│  │   (Tools)    │  │(localhost:8080)│ │  Engine      │       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘       │
│         │                 │                  │               │
│         └─────────────────┼──────────────────┘               │
│                           │                                  │
│  ┌────────────────────────┼────────────────────────┐         │
│  │    Interceptor         │    Storage (SQLite)   │         │
│  │  - Pause requests      │  - requests table     │         │
│  │  - Queue actions       │  - findings table     │         │
│  │  - Modify & forward    │  - HAR export         │         │
│  └────────────────────────┴────────────────────────┘         │
└──────────────────────────────────────────────────────────────┘
                        │
                        ▼ HTTP/HTTPS
┌──────────────────────────────────────────────────────────────┐
│           APPLICATIONS (Browser, Mobile, CLI)                │
└──────────────────────────────────────────────────────────────┘
```

---

## 📈 Métricas de Detección

El Smart Replay Engine detecta automáticamente:

| Tipo | Condición | Precisión |
|------|-----------|-----------|
| Auth Bypass | 403→200 | Alta |
| IDOR | 404→200 + cambio de ID | Alta |
| Privilege Escalation | Campos nuevos en JSON | Media |
| Info Disclosure | Stack traces en errores | Alta |
| Timing Attack | Δ > 500ms | Media |

---

## 🎯 Próximos Pasos Sugeridos

### Para el usuario:
1. Ejecutar `./setup.sh` para instalar
2. Probar con `curl -x http://localhost:8080 https://httpbin.org/get`
3. Navegar una app real y capturar tráfico
4. Ejecutar Smart Replay sobre requests con IDs

### Para mejorar el proyecto:
1. **WebSocket support** - Interceptar WS/WSS
2. **Response modification** - Modificar respuestas, no solo requests
3. **Plugin system** - Detectores personalizables en YAML
4. **Burp integration** - Exportar a formato Burp
5. **gRPC support** - Interceptar tráfico gRPC

---

## 📚 Documentación Completa

| Documento | Descripción |
|-----------|-------------|
| `cmd/proxy-server/README.md` | Guía completa de instalación y uso |
| `cmd/proxy-server/EXAMPLE.md` | Ejemplo práctico paso a paso |
| `internal/assets/skills/auditforge-proxy/SKILL.md` | Skill de integración |
| `internal/assets/skills/smart-replay-engine/SKILL.md` | Skill del motor de replay |
| `CHANGELOG.md` | Historial de cambios |

---

## ✅ Checklist de Implementación

- [x] Servidor proxy HTTP/HTTPS en Go
- [x] MITM con certificados dinámicos
- [x] 12 tools MCP para OpenCode
- [x] Persistencia SQLite
- [x] Smart Replay Engine con 5 detectores
- [x] Generador de mutaciones automáticas
- [x] Skills de integración (2)
- [x] Script de setup automatizado
- [x] Documentación completa
- [x] Ejemplo práctico
- [x] Configuración MCP
- [x] CHANGELOG

---

**🎉 Implementación completa y lista para usar!**

El proyecto ahora tiene capacidades de interceptación proxy nativas y análisis diferencial automático que lo diferencian significativamente de un simple orquestador pasivo.
