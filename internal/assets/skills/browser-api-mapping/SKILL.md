# Skill: Browser API Mapping

**Categoría:** web, api, reconnaissance
**Metodología base:** OWASP WSTG-CLIENT, WSTG-SESS
**Cuándo activar:** después de surface-discovery, cuando se necesita mapear la superficie API completa de una SPA o aplicación con tráfico XHR/fetch dinámico
**MCP requerido:** `chrome-devtools`

---

## Protocol

### Objetivo

Mapear la superficie de API expuesta por el cliente (endpoints, parámetros, headers, esquemas) mediante ejecución de navegador e interceptación de tráfico. Generar un inventario estructurado para correlación con hallazgos de security-scout y para alimentación de security-code.

### Diferencia con validación activa

| Aspecto | browser-api-mapping | validación activa |
|---------|---------------------|-------------------|
| Objetivo | Inventario y descubrimiento | Validación de hipótesis |
| Modificación | Solo lectura y captura | Manipulación controlada |
| Resultado | Mapa de API (OpenAPI/HAR) | Evidencia de vulnerabilidad |
| Timing | temprano en el análisis | cuando ya hay hipótesis confirmada |

---

## Fase 1 — Setup e Inventario Inicial

### 1.1 Listar Páginas Abiertas

```javascript
chrome-devtools_list_pages()
// Resultado: [{id: 1, url: "https://target.com/app"}, ...]
```

### 1.2 Mapear Endpoints en localStorage/sessionStorage

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    const results = {
      localStorage: {},
      sessionStorage: {},
      indexedDB: [],
      window_globals: []
    };

    // LocalStorage — tokens, preferencias, estado
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      results.localStorage[key] = localStorage.getItem(key);
    }

    // SessionStorage
    for (let i = 0; i < sessionStorage.length; i++) {
      const key = sessionStorage.key(i);
      results.sessionStorage[key] = sessionStorage.getItem(key);
    }

    // IndexedDB — bases de datos client-side
    if (indexedDB.databases) {
      results.indexedDB = await indexedDB.databases();
    }

    // Window globals que parecen API keys o tokens
    const suspicious = ['apiKey', 'api_key', 'authToken', 'token', 'accessToken', 'csrfToken'];
    for (const key of suspicious) {
      if (window[key]) results.window_globals.push({ key, value: window[key] });
    }

    return JSON.stringify(results, null, 2);
  }`
})
```

### 1.3 Detectar Endpoints Hardcodeados en JavaScript

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    // Buscar patrones de URL en el JavaScript cargado
    const scripts = Array.from(document.querySelectorAll('script[src]'));
    const patterns = [
      /\/api\/[a-zA-Z0-9_\/-]+/g,
      /https?:\/\/[a-zA-Z0-9.-]+\/api\/[a-zA-Z0-9_\/-]+/g,
      /endpoint[s]?[:\s]+["'][^"']+["']/gi,
      /baseURL[:\s]+["'][^"']+["']/gi
    ];

    const results = scripts.map(s => ({
      src: s.src,
      patterns: patterns.map(p => ({
        pattern: p.source,
        matches: (s.textContent || '').match(p) || []
      })).filter(m => m.matches.length > 0)
    })).filter(r => r.patterns.some(p => p.matches.length > 0));

    return JSON.stringify(results, null, 2);
  }`
})
```

---

## Fase 2 — Interceptación de Tráfico

### 2.1 Fetch Interceptor para Inventario Pasivo

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    if (window.__apiMapper) return 'Ya instalado';

    window.__apiMapper = {
      originalFetch: window.fetch,
      endpoints: new Set(),
      requests: []
    };

    window.fetch = async function(url, options = {}) {
      const entry = {
        timestamp: Date.now(),
        url: typeof url === 'string' ? url : url.toString(),
        method: options.method || 'GET',
        headers: Object.fromEntries(options.headers || []),
        body: options.body ? JSON.parse(options.body) : null
      };

      // Extraer endpoint y parámetros
      try {
        const parsed = new URL(entry.url);
        entry.endpoint = parsed.pathname;
        entry.params = Object.fromEntries(parsed.searchParams);
        window.__apiMapper.endpoints.add(parsed.pathname);
      } catch (e) {
        entry.raw = entry.url;
      }

      window.__apiMapper.requests.push(entry);
      return window.__apiMapper.originalFetch(url, options);
    };

    // Interceptar también a XMLHttpRequest
    const OriginalXHR = window.XMLHttpRequest;
    window.XMLHttpRequest = function() {
      const xhr = new OriginalXHR();
      const origSend = xhr.send.bind(xhr);
      xhr.send = function(body) {
        xhr.addEventListener('load', function() {
          try {
            const parsed = new URL(xhr.responseURL || xhr._url);
            window.__apiMapper.endpoints.add(parsed.pathname);
          } catch (e) {}
        });
        return origSend(body);
      };
      return xhr;
    };

    return 'API Mapper instalado. Navegá la app para mapear endpoints.';
  }`
})
```

### 2.2 Listar y Exportar Endpoints Capturados

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    const endpoints = Array.from(window.__apiMapper?.endpoints || []);
    const requests = window.__apiMapper?.requests || [];

    return JSON.stringify({
      endpoints: endpoints.sort(),
      total_requests: requests.length,
      by_method: requests.reduce((acc, req) => {
        acc[req.method] = (acc[req.method] || 0) + 1;
        return acc;
      }, {}),
      sample_requests: requests.slice(-20)  // últimos 20
    }, null, 2);
  }`
})
```

### 2.3 Captura Completa de Red (XHR/Fetch)

```javascript
// Capturar TODOS los requests del tipo especificado
chrome-devtools_list_network_requests({
  resourceTypes: ["xhr", "fetch"]
})
```

---

## Fase 3 — Mapeo de Esquemas y Parámetros

### 3.1 Inferir Esquema desde Requests

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    const requests = window.__apiMapper?.requests || [];

    // Agrupar por endpoint
    const byEndpoint = requests.reduce((acc, req) => {
      const ep = req.endpoint || req.url;
      if (!acc[ep]) acc[ep] = { GET: [], POST: [], PUT: [], PATCH: [], DELETE: [] };
      if (req.method === 'POST' || req.method === 'PUT' || req.method === 'PATCH') {
        acc[ep][req.method].push(req.body);
      } else {
        acc[ep][req.method].push(req.params);
      }
      return acc;
    }, {});

    // Inferir esquema de body para cada endpoint
    const schemas = {};
    for (const [ep, methods] of Object.entries(byEndpoint)) {
      schemas[ep] = {};
      for (const [method, bodies] of Object.entries(methods)) {
        if (bodies.length === 0) continue;

        const allKeys = new Set();
        bodies.forEach(b => Object.keys(b || {}).forEach(k => allKeys.add(k)));

        schemas[ep][method] = {
          sample_count: bodies.length,
          fields: Array.from(allKeys).map(key => ({
            name: key,
            types: [...new Set(bodies.map(b => typeof (b || {})[key]))],
            examples: bodies.map(b => (b || {})[key]).filter(v => v !== undefined).slice(0, 3)
          }))
        };
      }
    }

    return JSON.stringify(schemas, null, 2);
  }`
})
```

### 3.2 Identificar GraphQL Operations

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    const gql = window.__apiMapper?.requests?.filter(r =>
      r.url.includes('/graphql') || r.url.includes('/api')
    ).map(r => {
      try {
        const body = typeof r.body === 'string' ? JSON.parse(r.body) : r.body;
        if (body.query) {
          return {
            operationName: body.operationName || 'anonymous',
            query: body.query.substring(0, 500),
            variables: body.variables
          };
        }
      } catch (e) {}
      return null;
    }).filter(Boolean) || [];

    return JSON.stringify(gql, null, 2);
  }`
})
```

### 3.3 Identificar WebSocket Channels

```javascript
// Listar requests filtrando por websocket
chrome-devtools_list_network_requests({
  resourceTypes: ["websocket"]
})
```

---

## Fase 4 — Exportar Inventario

### 4.1 Generar OpenAPI 3.0 desde Tráfico

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    const requests = window.__apiMapper?.requests || [];
    const byEndpoint = {};

    requests.forEach(req => {
      const url = typeof req.url === 'string' ? req.url : req.url.toString();
      try {
        const parsed = new URL(url);
        const path = parsed.pathname;
        if (!byEndpoint[path]) byEndpoint[path] = {};
        byEndpoint[path][req.method] = byEndpoint[path][req.method] || [];
        byEndpoint[path][req.method].push({
          params: req.params,
          body: req.body,
          headers: req.headers
        });
      } catch (e) {}
    });

    const paths = {};
    for (const [path, methods] of Object.entries(byEndpoint)) {
      paths[path] = {};
      for (const [method, calls] of Object.entries(methods)) {
        const firstCall = calls[0];
        paths[path][method.toLowerCase()] = {
          summary: \`Auto-generated from \${calls.length} observations\`,
          parameters: Object.keys(firstCall.params || {}).map(k => ({
            name: k,
            in: 'query',
            example: firstCall.params[k]
          })),
          requestBody: firstCall.body ? {
            content: {
              'application/json': {
                example: firstCall.body
              }
            }
          } : undefined,
          responses: { '200': { description: 'OK' } }
        };
      }
    }

    const openapi = {
      openapi: '3.0.3',
      info: { title: 'Discovered API', version: '1.0.0' },
      paths
    };

    return JSON.stringify(openapi, null, 2);
  }`
})
```

### 4.2 Exportar HAR Completo

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    const reqs = window.__apiMapper?.requests || [];
    return JSON.stringify({
      log: {
        version: '1.2',
        creator: { name: 'Browser API Mapper', version: '1.0' },
        entries: reqs.map(r => ({
          startedDateTime: new Date(r.timestamp).toISOString(),
          request: {
            method: r.method,
            url: r.url,
            headers: Object.entries(r.headers || {}).map(([n,v]) => ({ name: n, value: v })),
            queryString: Object.entries(r.params || {}).map(([n,v]) => ({ name: n, value: v })),
            postData: r.body ? { mimeType: 'application/json', text: JSON.stringify(r.body) } : undefined
          },
          response: { status: 0, headers: [] }  // incompleto - solo para inventario
        }))
      }
    }, null, 2);
  }`
})
```

---

## Integración con Team

- **security-scout** → pasa URL inicial y archivos JS descubiertos
- **security-web** → usa este skill para enriquecer superficie y pasar endpoints a validación acotada posterior
- **security-code** → recibe inventario de endpoints y puede identificar handlers correspondientes en código fuente

## Checklist de Cierre

```
[ ] Fetch interceptor instalado
[ ] Navegación completa de la app ejecutada
[ ] Endpoints mapeados: N total
[ ] GraphQL operations identificadas (si aplica)
[ ] WebSocket channels identificados (si aplica)
[ ] OpenAPI schema generado
[ ] HAR exportado para replay externo
[ ] Endpoints críticos para security-web priorizados
```
