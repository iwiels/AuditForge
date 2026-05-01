# 📦 Integración Completa: Proxy en Scripts de Instalación

## ✅ Qué se ha integrado

Ahora el **AuditForge Proxy Server** y **Smart Replay Engine** se instalan automáticamente con los scripts de instalación existentes. No se necesitan comandos manuales adicionales.

---

## 🚀 Scripts de Instalación Actualizados

### 1. PowerShell (Windows) - `install-assets.ps1` ⭐ RECOMENDADO

**Características:**
- ✅ Detección automática de Go
- ✅ Compilación automática del proxy
- ✅ Generación de certificados SSL
- ✅ Instalación MCP config
- ✅ Mensajes de estado con colores
- ✅ Manejo de errores robusto

**Uso:**
```powershell
# Instalar todo (skills + agents + MCP + proxy)
.\install-assets.ps1

# Solo el proxy
.\install-assets.ps1 -Proxy

# Solo skills
.\install-assets.ps1 -Skills

# Desinstalar
.\install-assets.ps1 -Uninstall
```

### 2. Bash (Linux/macOS/WSL) - `install-assets.sh`

**Características:**
- ✅ Detección de OS (Linux/macOS/WSL)
- ✅ Compilación automática si Go está disponible
- ✅ Generación de certificados
- ✅ Scripts de inicio automáticos
- ✅ Colores en terminal

**Uso:**
```bash
# Instalar todo
./install-assets.sh --all

# Solo el proxy
./install-assets.sh --proxy

# Solo skills
./install-assets.sh --skills

# Desinstalar
./install-assets.sh --uninstall
```

### 3. Batch (Windows Legacy) - `install-assets.bat`

**Características:**
- ✅ Compatible con cmd.exe
- ✅ Compilación automática
- ✅ Sin dependencias de PowerShell

**Uso:**
```cmd
install-assets.bat
```

---

## 📋 Flujo de Instalación Automática

```
Usuario ejecuta: ./install-assets.sh (o .ps1 / .bat)
         │
         ▼
┌─────────────────────────────────────────────┐
│ 1. Instalar Skills (29 skills)              │
│    └── Copia a ~/.auditforge/skills/        │
└──────────────┬──────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────┐
│ 2. Instalar Agents & Commands               │
│    └── Copia a ~/.config/opencode/          │
└──────────────┬──────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────┐
│ 3. Configurar MCP                           │
│    ├── chrome-devtools.json                 │
│    └── auditforge-proxy.json ← NEW!         │
└──────────────┬──────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────┐
│ 4. Build AuditForge Proxy ← NEW!            │
│    ├── Verificar Go instalado               │
│    ├── Compilar: go build                   │
│    ├── Generar certificados SSL             │
│    └── Crear scripts de inicio              │
└──────────────┬──────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────┐
│ 5. Mostrar resumen con instrucciones        │
│    └── Proxy listo para usar!               │
└─────────────────────────────────────────────┘
```

---

## 📁 Estructura después de instalar

```
~/.auditforge/                    # NEW: AuditForge home
├── skills/                       # 29 skills instalados
│   ├── auditforge-proxy/
│   ├── smart-replay-engine/
│   ├── surface-discovery/
│   └── ... (26 más)
│
└── proxy/                        # NEW: Proxy server
    ├── auditforge-proxy          # Binario compilado
    ├── auditforge-proxy.exe      # (Windows)
    ├── start-proxy.sh            # Script de inicio
    ├── start-proxy.bat           # Script de inicio
    ├── certs/
    │   ├── ca.crt               # Certificado CA
    │   └── ca.key               # Clave privada CA
    └── auditforge-proxy.db      # Base de datos SQLite

~/.config/opencode/               # OpenCode config
├── agents/                       # 6 agentes
├── commands/                     # 10 comandos
└── mcp/                          # MCP configs
    ├── chrome-devtools.json
    └── auditforge-proxy.json     # NEW: Proxy MCP config
```

---

## 🎯 Uso después de instalar

### 1. Iniciar el Proxy

**Automático (recomendado):**
El proxy se inicia automáticamente cuando OpenCode carga el MCP.

**Manual (si es necesario):**
```bash
# Linux/macOS
~/.auditforge/proxy/start-proxy.sh

# Windows
%USERPROFILE%\.auditforge\proxy\start-proxy.bat
```

### 2. Configurar Browser

**Chrome:**
```bash
chrome --proxy-server=http://localhost:8080
```

**Firefox:**
- Settings → Network Settings
- Manual proxy configuration
- HTTP/HTTPS Proxy: `localhost` Port: `8080`

### 3. Usar en OpenCode

```javascript
// El proxy ya está listo, solo usar:
proxy.intercept.enable({})
proxy.history.search({limit: 20})
proxy.replay.execute({request_id: "abc-123", smart_mode: true})
```

---

## ⚠️ Requisitos

### Opcionales pero recomendados:

1. **Go 1.21+** - Para compilar el proxy
   - Si no está instalado, el proxy no se construye
   - Instalar desde: https://golang.org/dl/

2. **OpenSSL** - Para generar certificados SSL
   - Incluido en macOS y la mayoría de Linux
   - Windows: Instalar Git for Windows (incluye OpenSSL)

### Si Go NO está instalado:
- ✅ Skills, agents y MCP config se instalan normalmente
- ⚠️ El proxy no se compila
- 💡 El usuario puede descargar un binario pre-compilado

---

## 🔧 Solución de problemas

### "Go not found"
```bash
# Instalar Go y re-ejecutar con --proxy
./install-assets.sh --proxy
```

### "Proxy binary not found" después de instalar
```bash
# Compilar manualmente
cd cmd/proxy-server
go build -o ~/.auditforge/proxy/auditforge-proxy .
```

### Certificados SSL
```bash
# Si no se generaron automáticamente
cd ~/.auditforge/proxy
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
    -keyout certs/ca.key -out certs/ca.crt \
    -subj "/CN=AuditForge Proxy CA"
```

---

## 📊 Comparación: Antes vs Ahora

| Aspecto | Antes | Ahora |
|---------|-------|-------|
| **Instalación proxy** | Comandos manuales separados | Integrado en `./install-assets.sh` |
| **Build** | `cd cmd/proxy-server && go build` | Automático si Go está instalado |
| **Certificados** | `./setup.sh` | Automático durante instalación |
| **MCP config** | Manual | Automático en `install-assets.sh` |
| **Scripts de inicio** | Crear manualmente | Creados automáticamente |
| **Pasos totales** | ~5-7 comandos | **1 comando** |

---

## 📝 Cambios en archivos

### Archivos modificados:
1. ✅ `install-assets.sh` - Agregada función `install_proxy`
2. ✅ `install-assets.bat` - Agregada sección `:install_proxy`
3. ✅ `README.md` - Actualizada documentación de instalación

### Archivos nuevos:
1. ✅ `install-assets.ps1` - Script PowerShell completo
2. ✅ `cmd/proxy-server/` - Código fuente del proxy
3. ✅ `internal/assets/skills/auditforge-proxy/SKILL.md`
4. ✅ `internal/assets/skills/smart-replay-engine/SKILL.md`
5. ✅ Documentación adicional (CHEATSHEET, ARCHITECTURE, etc.)

---

## 🎉 Resumen

**Antes:**
```bash
# 5+ pasos manuales
./install-assets.sh
cd cmd/proxy-server
./setup.sh
go build -o auditforge-proxy .
./start-proxy.sh
# Configurar MCP manualmente
```

**Ahora:**
```bash
# 1 solo comando
./install-assets.sh
# Listo! El proxy está instalado y configurado
```

**O en Windows:**
```powershell
.\install-assets.ps1
# Listo!
```

---

## 🚀 Próximos pasos para el usuario

1. Ejecutar el instalador
2. Configurar browser con proxy localhost:8080
3. Reiniciar OpenCode
4. Empezar a auditar con `/team https://target.com`

¡Todo integrado en un solo flujo! 🎯
