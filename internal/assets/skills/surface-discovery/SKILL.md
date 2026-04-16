# Skill: Surface Discovery & Recon

**Categoría:** recon  
**Metodología base:** PTES Intelligence Gathering, OWASP Testing Guide OTG-INFO  
**Cuándo activar:** inicio de cualquier engagement antes de cualquier análisis activo

---

## Protocolo

### Fase 1 — Definición de perímetro

Antes de tocar el target, establecé el mapa mental del sistema:

```
¿Qué está explícitamente en scope?
¿Hay subdominios, APIs secundarias, o entornos (staging, dev) incluidos?
¿Hay restricciones explícitas (no tocar /admin, no autenticarse)?
```

Ejecutá `memory.search` con `scope`, `authorized`, `engagement` para recuperar contexto previo. Si no existe, solicitalo antes de continuar.

### Fase 2 — Stack fingerprinting pasivo

Identificá el stack tecnológico sin interacción activa:

**Headers HTTP a inspeccionar:**
- `Server`, `X-Powered-By`, `X-Generator`, `X-Frame-Options` (ausencia es un hallazgo)
- `Set-Cookie` — flags `HttpOnly`, `Secure`, `SameSite`, nombre de la cookie (PHPSESSID, JSESSIONID revelan stack)
- `Content-Security-Policy` — ausencia o política débil (`unsafe-inline`, `*`) es MEDIO
- `Strict-Transport-Security` — ausencia es BAJO, mal configurado es MEDIO

**Archivos de fingerprinting:**
- `/robots.txt` — rutas excluidas son superficie de ataque
- `/security.txt` (RFC 9116) — presencia indica madurez; ausencia es informativo
- `/.well-known/` — puede exponer configuración ACME, WebFinger, OAuth metadata
- `/sitemap.xml` — enumera endpoints sin autenticación
- `/.git/` — si accesible, CRÍTICO (extracción de código fuente)
- `/api/`, `/api/v1/`, `/api/docs`, `/swagger.json`, `/openapi.yaml` — superficie API

**CMS y frameworks:**
- WordPress: `/wp-login.php`, `/wp-json/wp/v2/users`, `readme.html`
- Django: páginas de error con traceback, `/admin/`
- Laravel: `.env` expuesto, `storage/logs/laravel.log`
- Next.js/Nuxt: `/_next/static/`, `/__nuxt/`

### Fase 3 — Enumeración de assets

Construí un inventario estructurado:

```
ASSET INVENTORY
───────────────
[URL]           https://target.com
[STACK]         Nginx 1.24 / Node.js / React
[ENDPOINTS]     /api/v2/* (REST), /graphql (GraphQL)
[AUTH]          JWT Bearer tokens, /auth/login
[ADMIN]         /admin (responde 403 — confirmar bypass)
[EXPOSURES]     /robots.txt lista /internal, /backup
[COOKIES]       session= (falta Secure flag)
```

### Fase 4 — Priorización de superficie

Clasifica cada asset por potencial de impacto:

| Prioridad | Criterio | Acción |
|-----------|----------|--------|
| P0 | Admin panels, autenticación, API keys expuestos | Investigar inmediatamente |
| P1 | APIs sin documentación, endpoints de upload, GraphQL | Delegar a security-web |
| P2 | Endpoints autenticados, funciones de negocio | Queue para web-triage |
| P3 | Assets estáticos, páginas públicas | Registrar, baja prioridad |

### Fase 5 — Subdomain enumeration (pasiva)

Sin herramientas activas:
- Revisar certificado SSL/TLS: Subject Alternative Names revelan subdominios
- DNS histórico: consultar Certificate Transparency logs (crt.sh)
- Archivos JS del frontend: URLs hardcodeadas de APIs internas
- Respuestas de error: stack traces con hostnames internos

### Fase 6 — Output estructurado

Al finalizar esta skill, producí:

```markdown
## Surface Discovery Report

**Target:** [target]
**Fecha:** [fecha]
**Stack detectado:** [stack]

### Assets identificados
[lista priorizada]

### Hallazgos inmediatos
[findings con severidad]

### Próximos pasos recomendados
- Delegar [X] a security-web
- Delegar [Y] a security-supply
```

---

## Anti-patterns a evitar

- ❌ Asumir que un endpoint en `/robots.txt` no existe porque está excluido
- ❌ Ignorar headers de respuesta — contienen más información que el body en el recon inicial
- ❌ Marcar "sin hallazgos" sin haber revisado los archivos de fingerprinting estándar
- ❌ Enumerar subdominios con herramientas activas sin autorización explícita
