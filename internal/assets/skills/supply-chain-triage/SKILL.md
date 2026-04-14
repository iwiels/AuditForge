# Skill: Supply Chain Triage

**Categoría:** supply-chain  
**Metodología base:** OWASP A06:2021, SLSA Framework, CIS Software Supply Chain Security Guide  
**Cuándo activar:** cuando el target es un repositorio de código, aplicación con dependencias, o pipeline de CI/CD

---

## Protocolo

### Fase 1 — Inventario de dependencias

**Identificar el ecosistema:**
```
Node.js:   package.json, package-lock.json, yarn.lock
Python:    requirements.txt, Pipfile.lock, pyproject.toml
Go:        go.mod, go.sum
Java:      pom.xml, build.gradle
Ruby:      Gemfile.lock
Rust:      Cargo.toml, Cargo.lock
PHP:       composer.json, composer.lock
```

**Señales de riesgo inmediatas:**
- Lockfile ausente o desactualizado respecto al manifest
- Dependencias fijadas con `*` o `>=` sin upper bound
- Dependencias instaladas directamente desde GitHub sin tag ni hash
- `postinstall` scripts en `package.json` que ejecutan código arbitrario

### Fase 2 — Análisis de CVEs conocidos

Sin ejecutar herramientas externas, analizá el inventario:

**Para cada dependencia crítica, verificar:**
```
1. ¿La versión instalada tiene CVEs conocidos?
   → Consultar: https://osv.dev, https://nvd.nist.gov, GitHub Advisory Database
   
2. ¿La dependencia sigue siendo mantenida?
   → Última release hace más de 2 años + issues sin respuesta = riesgo alto
   
3. ¿La dependencia es transitiva o directa?
   → Transitivas comprometidas son más difíciles de controlar (dependency confusion)
   
4. ¿Tiene permisos excesivos?
   → npm packages con postinstall que ejecutan scripts de red
   → pip packages que importan subprocess o socket en el __init__
```

**Patrones de dependency confusion:**
```
- Paquetes internos con nombres genéricos también publicados en npm/pypi
- Versiones en registros privados menores a las públicas (el resolver elige la pública)
- Namespaces sin verificar (@empresa/paquete si el namespace no está registrado)
```

### Fase 3 — Revisión de secretos en código

Buscar en el repositorio completo, incluyendo historial de git:

**Patrones de alto valor:**
```
Claves AWS:         AKIA[0-9A-Z]{16}
Tokens GitHub:      ghp_[A-Za-z0-9]{36}
Claves Stripe:      sk_live_[A-Za-z0-9]{24}
JWT secrets:        "secret": "...", JWT_SECRET=, SECRET_KEY=
Conexiones DB:      DATABASE_URL=, POSTGRES_PASSWORD=, mongodb://user:pass@
Claves privadas:    -----BEGIN RSA PRIVATE KEY-----
Google API Keys:    AIza[0-9A-Za-z\-_]{35}
Slack tokens:       xox[baprs]-[0-9a-zA-Z]
```

**Lugares críticos a revisar:**
- `.env` (¿está en .gitignore?)
- `config/`, `settings/`, `app.config.*`
- Archivos de CI/CD: `.github/workflows/*.yml`, `.gitlab-ci.yml`, `Jenkinsfile`
- Código fuente en comentarios (credentials "temporales")
- Historial de git: `git log -p --all | grep -E "(password|secret|key|token)"`

### Fase 4 — Análisis de pipeline CI/CD

**Superficie de ataque del pipeline:**
```
¿Quién puede hacer push a main/master?        → ¿branch protection activo?
¿Los workflows ejecutan código de PRs externos? → CRÍTICO si sí (GitHub Actions pwn)
¿Los secrets están en variables de entorno?    → ¿accesibles desde forks?
¿Hay pasos de deploy automático sin aprobación? → ¿quién aprueba?
¿Las imágenes Docker tienen tag fijo o :latest? → :latest = no reproducible
¿Se verifica la integridad de artifacts?       → checksum, firma SLSA
```

**Misconfiguraciones comunes en GitHub Actions:**
```yaml
# RIESGO: Ejecuta código de PRs de externos con acceso a secrets
on:
  pull_request_target:    # ← peligroso si el workflow hace checkout del PR
    types: [opened]

# RIESGO: Usa versiones flotantes de actions
uses: actions/checkout@main    # ← debe ser @v4 o @sha-hash

# RIESGO: Secrets accesibles en PRs de forks
env:
  TOKEN: ${{ secrets.TOKEN }}   # ← combinado con pull_request_target = comprometido
```

### Fase 5 — Revisión de código estático (manual)

Sin herramientas de SAST, revisá los patrones más impactantes:

**Inyección de comandos:**
```python
# Python — RIESGO
import subprocess
subprocess.run(f"git clone {user_input}", shell=True)  # shell=True + input = RCE

# Correcto
subprocess.run(["git", "clone", validated_url], shell=False)
```

**Deserialización insegura:**
```python
# Python — CRÍTICO
import pickle
data = pickle.loads(user_input)  # RCE garantizado

# Node.js — CRÍTICO  
eval(userInput)
Function(userInput)()
```

**Path traversal:**
```javascript
// Node.js — RIESGO
const file = fs.readFileSync(`/app/files/${req.params.name}`)
// Input: ../../etc/passwd → lee archivos del sistema
```

**Logging de datos sensibles:**
```javascript
// RIESGO
console.log("User login:", { username, password })
logger.info({ body: req.body })  // puede contener passwords, tokens
```

### Fase 6 — Output

```markdown
## Supply Chain Triage — [Repositorio/Target]

### Inventario de dependencias
[ecosistema, cantidad, fechas de última actualización]

### CVEs identificados
[tabla: dependencia, versión, CVE, CVSS, remediación]

### Secretos detectados
[location, tipo, severidad — NO mostrar el valor real]

### Riesgos en pipeline CI/CD
[findings con referencia al archivo y línea]

### Hallazgos de revisión de código
[patrones inseguros con ubicación]

### Recomendaciones
[priorizadas]
```

---

## Anti-patterns

- ❌ Reportar "dependencia desactualizada" sin verificar si tiene CVEs explotables
- ❌ Mostrar el valor completo de un secret en el reporte — siempre redactar
- ❌ Ignorar las dependencias transitivas — la mayoría de compromisos reales son transitivos
- ❌ No revisar el historial de git — el 80% de los secrets "eliminados" siguen en el historial
