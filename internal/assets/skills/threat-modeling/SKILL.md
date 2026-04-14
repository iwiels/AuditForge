# Skill: Threat Modeling

**Categoría:** análisis  
**Metodología base:** STRIDE (Microsoft), PASTA, OWASP Threat Model Manifesto  
**Cuándo activar:** después de surface-discovery, antes o durante web-triage; o cuando el cliente pide "¿qué tan expuesto estoy?"

---

## Protocolo

### Fase 1 — Definir el sistema bajo análisis

Antes de modelar amenazas, construí el mapa del sistema:

```
SISTEMA: [nombre]
COMPONENTES:
  - Frontend: [tecnología, hosting]
  - Backend: [lenguaje, framework, versión]
  - Base de datos: [tipo, ¿expuesta directamente?]
  - APIs externas: [listado de integraciones third-party]
  - Infraestructura: [cloud, on-prem, containers]
  - CI/CD: [pipeline, acceso al repo]

USUARIOS:
  - Anónimos: ¿qué pueden hacer?
  - Autenticados (rol básico): ¿qué pueden hacer?
  - Administradores: ¿qué pueden hacer?
  - APIs externas: ¿qué acceso tienen?
```

### Fase 2 — Identificar Trust Boundaries

Un trust boundary es donde los datos cruzan de un contexto de confianza a otro. Cada cruce es un vector de ataque potencial.

**Boundaries comunes:**
```
Internet → WAF/Reverse Proxy         → ¿el WAF valida todo o solo filtra IP?
Frontend → Backend API               → ¿el backend confía en el token sin validar claims?
Backend → Base de datos              → ¿usa ORM con queries parametrizadas?
Backend → Servicios externos (APIs)  → ¿las credenciales están hardcodeadas?
CI/CD → Producción                   → ¿quién puede hacer deploy? ¿hay aprobación?
Admin → Sistema                      → ¿hay MFA? ¿logs de auditoría?
```

Para cada boundary, anotá:
- ¿Qué datos cruzan?
- ¿Cómo se validan?
- ¿Qué pasa si el componente origen está comprometido?

### Fase 3 — Aplicar STRIDE

Para cada componente y trust boundary, evaluá las 6 categorías:

**S — Spoofing (Suplantación)**
```
¿Puede un atacante hacerse pasar por otro usuario o sistema?
Vectores: JWT sin firma correcta, sesiones predecibles, CSRF, OAuth misconfiguration
Preguntas: ¿Cómo verifica la identidad el sistema? ¿Hay MFA? ¿Hay API keys compartidas?
```

**T — Tampering (Manipulación)**
```
¿Puede un atacante modificar datos en tránsito o en reposo?
Vectores: MITM (sin HTTPS forzado), Mass Assignment, SQL Injection, insecure deserialization
Preguntas: ¿Los datos críticos tienen checksums? ¿Las APIs validan tipos y rangos?
```

**R — Repudiation (Repudio)**
```
¿Puede un atacante negar haber realizado una acción?
Vectores: logs insuficientes, falta de audit trail, tokens compartidos
Preguntas: ¿Hay logs de quién hizo qué y cuándo? ¿Los logs son inmutables?
```

**I — Information Disclosure (Exposición de información)**
```
¿Puede un atacante acceder a información que no debería ver?
Vectores: IDOR, mensajes de error verbosos, directorios listables, endpoints no autenticados
Preguntas: ¿Las respuestas de error revelan stack traces? ¿Hay datos sensibles en logs?
```

**D — Denial of Service (Denegación de servicio)**
```
¿Puede un atacante hacer el sistema inaccesible?
Vectores: endpoints sin rate limiting, queries costosas sin paginación, file uploads sin límite
Preguntas: ¿Hay límites de recursos? ¿Las queries tienen timeout? ¿Hay caching?
```

**E — Elevation of Privilege (Escalada de privilegios)**
```
¿Puede un atacante obtener más permisos de los que debería tener?
Vectores: IDOR, broken access control, admin endpoints sin protección, JWT con roles manipulables
Preguntas: ¿Se verifica el rol en cada endpoint o solo en el login?
```

### Fase 4 — Priorización con Attack Trees

Para los vectores más críticos, construí un árbol de ataque:

```
OBJETIVO: Acceso a datos de todos los usuarios

├── Via autenticación
│   ├── Bypass de login (SQLi en formulario)
│   ├── Credential stuffing (sin rate limiting)
│   └── Robo de sesión (XSS + cookie sin HttpOnly)
│
├── Via autorización
│   ├── IDOR en /api/users/{id} (IDs secuenciales)
│   └── Escalada de privilegios (role param en JWT)
│
└── Via infraestructura
    ├── Acceso al backup expuesto en S3 público
    └── Credenciales en repositorio público
```

Para cada rama: **¿Cuán difícil es? ¿Qué impacto tiene? ¿Hay controles?**

### Fase 5 — Matriz de riesgo

| Amenaza STRIDE | Vector | Probabilidad | Impacto | Riesgo | Control existente |
|----------------|--------|-------------|---------|--------|-------------------|
| Spoofing | JWT alg:none | Alta | Crítico | CRÍTICO | Ninguno detectado |
| Information Disclosure | IDOR en /api/orders | Media | Alto | ALTO | Sin validación ownership |
| Tampering | Mass assignment | Baja | Alto | MEDIO | Documentar para validar |

### Fase 6 — Output

```markdown
## Threat Model — [Sistema]

### Arquitectura analizada
[diagrama textual de componentes y trust boundaries]

### Amenazas identificadas por categoría STRIDE
[tabla de amenazas]

### Attack trees para vectores críticos
[árboles para P0/P1]

### Recomendaciones de controles
[por categoría, priorizadas]

### Gaps en la arquitectura de seguridad
[lo que falta, no solo lo que está mal]
```

---

## Anti-patterns

- ❌ Hacer threat modeling sin entender el negocio — el impacto de un finding depende del contexto
- ❌ Listar todas las amenazas STRIDE sin priorizarlas — produce ruido, no valor
- ❌ Modelar amenazas sin considerar el atacante real (script kiddie vs nation state vs insider)
- ❌ Threat model sin control existente — siempre documentar qué hay, no solo qué falta
