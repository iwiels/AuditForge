# Skill: Proxy Capture Replay

**Categoría:** web, api, evidence
**Metodología base:** OWASP WSTG-SESS, WSTG-AUTHN, manual testing methodology
**Cuándo activar:** cuando se necesita persistir tráfico autenticado para reproducción offline, auditoría de compliance, o correlación con hallazgos
**MCP opcional:** `chrome-devtools` para captura nativa; herramientas externas (mitmproxy, Burp Suite) para captura completa de tráfico
**Herramientas externas mencionadas:** mitmproxy, Burp Suite Professional

---

## Protocol

### Objetivo

Capturar sesiones autenticadas completas (tráfico HTTP/WebSocket) y convertirlas en templates de replay reproducibles para validación de hipótesis, auditoría de compliance, o evidencia documentary. El replay debe preservar la fidelidad del request original incluyendo tokens, cookies, y timing.

### Flujo General

```
1. Captura de tráfico (browser o proxy)
2. Normalización y sanitización de datos sensibles
3. Generación de templates de replay
4. Verificación de reproducibilidad
5. Persistencia en artefactos de auditoría
```

---

## Fase 1 — Captura de Tráfico

### 1.1 Captura vía Chrome DevTools (Browser-Nativo)

```javascript
// Capturar todos los requests de la sesión
chrome-devtools_list_network_requests({
  resourceTypes: ["xhr", "fetch", "websocket", "document"]
})

// Para cada request capturado, obtener detalle completo
chrome-devtools_get_network_request({
  reqid: N,
  requestFilePath: "evidence/raw/req-N-request.json",
  responseFilePath: "evidence/raw/req-N-response.json"
})
```

### 1.2 Dump Completo de Estado de Sesión

```javascript
chrome-devtools_evaluate_script({
  function: `() => JSON.stringify({
    cookies: document.cookie,
    localStorage: Object.assign({}, localStorage),
    sessionStorage: Object.assign({}, sessionStorage),
    indexedDB: Array.from(indexedDB.databases()).map(d => d.name),
    cache: window.caches ? await window.caches.keys() : []
  }, null, 2)`
})
```

### 1.3 Captura vía mitmproxy (Proxy Externo)

Para captura completa (incluyendo headers modificados por TLS, redirects, etc.):

```bash
# Iniciar mitmproxy en modo dump para captura
mitmdump -w capture.mitmproxy --flow-detail 3

# Captura con script personalizado para sanitización en tiempo real
mitmdump -s sanitize.py -w capture.mitmproxy

# Convertir a HAR
mitmproxy2har -c capture.mitmproxy -o capture.har
```

### 1.4 Script de Sanitización mitmproxy

```python
# sanitize.py — mitmproxy script para sanitización en tiempo real
from mitmproxy import http

def response(flow: http.HTTPFlow):
    # Sanitizar Authorization headers
    if "authorization" in flow.request.headers:
        flow.request.headers["authorization"] = "[REDACTED]"

    # Sanitizar cookies
    if "cookie" in flow.request.headers:
        flow.request.headers["cookie"] = "[REDACTED]"

    # Sanitizar set-cookie
    if "set-cookie" in flow.response.headers:
        flow.response.headers["set-cookie"] = "[REDACTED]"

    # Sanitizar PII en query params conocidos
    pii_params = ["email", "phone", "ssn", "credit_card", "password"]
    for param in pii_params:
        if param in flow.request.query:
            flow.request.query[param] = "[REDACTED]"

    # Sanitizar PII en JSON body
    if flow.request.content:
        import json
        try:
            body = json.loads(flow.request.content)
            for key in list(body.keys()):
                if any(pii in key.lower() for pii in pii_params):
                    body[key] = "[REDACTED]"
            flow.request.content = json.dumps(body).encode()
        except:
            pass
```

---

## Fase 2 — Normalización de Captura

### 2.1 Convertir mitmproxy dump a HAR

```bash
mitmproxy2har -c capture.mitmproxy -o capture.har

# O con python api
python -c "
from mitmproxy2har import convert
convert('capture.mitmproxy', 'capture.har')
"
```

### 2.2 Normalizar HAR para Auditoría

```python
# normalize_har.py
import json

def normalize_har(input_file, output_file):
    with open(input_file) as f:
        har = json.load(f)

    normalized = {
        'log': {
            'version': har['log']['version'],
            'creator': har['log']['creator'],
            'entries': []
        }
    }

    for entry in har['log']['entries']:
        # Filtrar entries irrelevantes (imágenes, fonts, CSS estático)
        mime = entry['request']['mimeType'].lower()
        if any(x in mime for x in ['image', 'font', 'css']):
            continue

        normalized_entry = {
            'startedDateTime': entry['startedDateTime'],
            'time': entry['time'],
            'request': {
                'method': entry['request']['method'],
                'url': entry['request']['url'],
                'headers': [h for h in entry['request']['headers']
                           if h['name'].lower() not in
                           ['cookie', 'authorization', 'x-api-key']],
                'queryString': entry['request']['queryString'],
                'postData': entry['request'].get('postData', {})
            },
            'response': {
                'status': entry['response']['status'],
                'headers': [h for h in entry['response']['headers']
                           if h['name'].lower() not in ['set-cookie', 'x-session-token']],
                'content': entry['response']['content']
            },
            'timings': entry['timings']
        }

        normalized['log']['entries'].append(normalized_entry)

    with open(output_file, 'w') as f:
        json.dump(normalized, f, indent=2)

    print(f'Normalized: {len(normalized["log"]["entries"])} entries')
```

### 2.3 Extraer Requests como Templates curl

```python
# har_to_curl.py
import json
import sys

def har_to_curl(har_file, output_file):
    with open(har_file) as f:
        har = json.load(f)

    curls = []
    for entry in har['log']['entries']:
        req = entry['request']
        curl = ['curl']

        # Method
        curl.append(f"-X {req['method']}")

        # URL
        curl.append(f"'{req['url']}'")

        # Headers
        for h in req['headers']:
            if h['name'].lower() not in ['cookie', 'authorization']:
                curl.append(f"-H '{h['name']}: {h['value']}'")

        # Body
        if req.get('postData') and req['postData'].get('text'):
            body = req['postData']['text']
            curl.append(f"-d '{body}'")

        curls.append(' \\\\\n'.join(curl))

    with open(output_file, 'w') as f:
        f.write('\n\n---\n\n'.join(curls))

    print(f'Generated {len(curls)} curl commands')
```

---

## Fase 3 — Replay y Verificación

### 3.1 Replay Básico via curl

```bash
# Replay de un request específico
curl -X POST 'https://target/api/endpoint' \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer [TOKEN]' \
  -d '{"field": "value"}' \
  -v

# Replay con cookies
curl -X GET 'https://target/api/user/profile' \
  --cookie "session=[SESSION_COOKIE]" \
  -v
```

### 3.2 Replay Automatizado con Python

```python
# replay.py
import requests
import json
import time

class RequestReplay:
    def __init__(self, har_file, token=None, cookie=None):
        with open(har_file) as f:
            self.har = json.load(f)
        self.session = requests.Session()
        self.token = token
        self.cookie = cookie

    def replay_entry(self, entry, delay=0):
        if delay:
            time.sleep(delay)

        req = entry['request']
        url = req['url']
        method = req['method']

        headers = {h['name']: h['value'] for h in req['headers']}
        if self.token:
            headers['Authorization'] = f'Bearer {self.token}'
        if self.cookie:
            headers['Cookie'] = self.cookie

        body = None
        if req.get('postData') and req['postData'].get('text'):
            body = req['postData']['text']

        response = self.session.request(
            method=method,
            url=url,
            headers=headers,
            data=body,
            timeout=30
        )

        return {
            'expected_status': entry['response']['status'],
            'actual_status': response.status_code,
            'match': response.status_code == entry['response']['status']
        }

    def replay_all(self, delay=0.5):
        results = []
        for entry in self.har['log']['entries']:
            result = self.replay_entry(entry, delay)
            result['url'] = entry['request']['url']
            results.append(result)
            print(f"{'✓' if result['match'] else '✗'} {result['url']}: {result['actual_status']}")
        return results
```

### 3.3 Replay con Modificación de Headers (Auth Bypass Test)

```python
# replay_modify.py
def test_auth_bypass(replay, endpoint_hint):
    """Probar si el endpoint es accesible sin auth"""

    results = []
    for entry in replay.har['log']['entries']:
        if endpoint_hint not in entry['request']['url']:
            continue

        req = entry['request']

        # Original
        orig = replay.replay_entry(entry)
        results.append(('original', orig))

        # Sin Authorization
        replay.session.headers.pop('Authorization', None)
        no_auth = replay.replay_entry(entry)
        results.append(('no_auth', no_auth))

        # Sin Cookie
        replay.session.headers.pop('Cookie', None)
        replay.session.cookies.clear()
        no_cookie = replay.replay_entry(entry)
        results.append(('no_cookie', no_cookie))

    return results
```

---

## Fase 4 — Detección de Diferencias (Diffing)

### 4.1 Diffing de Respuestas

```python
# diff.py
import requests
import json

def diff_responses(entry, token_modifications):
    """Comparar respuestas variando tokens/headers"""

    req = entry['request']
    base_response = requests.request(
        method=req['method'],
        url=req['url'],
        headers={h['name']: h['value'] for h in req['headers']},
        data=req.get('postData', {}).get('text')
    )

    results = {'base': base_response.status_code}

    for name, modified_headers in token_modifications.items():
        headers = {h['name']: h['value'] for h in req['headers']}
        headers.update(modified_headers)

        response = requests.request(
            method=req['method'],
            url=req['url'],
            headers=headers,
            data=req.get('postData', {}).get('text')
        )

        results[name] = {
            'status': response.status_code,
            'diff': response.text != base_response.text
        }

    return results
```

### 4.2 Timing Analysis para Race Conditions

```python
# timing_analysis.py
import time
import statistics

def analyze_timing(entry, iterations=10):
    """Medir varianza de timing para detectar race conditions"""

    req = entry['request']
    timings = []

    for _ in range(iterations):
        start = time.time()
        response = requests.request(
            method=req['method'],
            url=req['url'],
            headers={h['name']: h['value'] for h in req['headers']}
        )
        elapsed = (time.time() - start) * 1000  # ms
        timings.append(elapsed)

    return {
        'mean': statistics.mean(timings),
        'stdev': statistics.stdev(timings) if len(timings) > 1 else 0,
        'min': min(timings),
        'max': max(timings),
        'samples': timings
    }
```

---

## Fase 5 — Persistencia de Artefactos

### 5.1 Estructura de Directorios de Evidencia

```
evidence/
├── capture-[session]-[date]/
│   ├── raw/
│   │   ├── capture.mitmproxy
│   │   ├── capture.har
│   │   └── requests/
│   │       ├── req-001.json
│   │       ├── req-002.json
│   │       └── ...
│   ├── normalized/
│   │   ├── capture-normalized.har
│   │   └── requests/
│   ├── replay/
│   │   ├── curl-commands.sh
│   │   ├── replay-results.json
│   │   └── diff-results.json
│   └── session-state.json
```

### 5.2 Metadata de la Captura

```json
{
  "capture_id": "session-001-20240115",
  "captured_at": "2024-01-15T10:30:00Z",
  "capture_method": "mitmproxy",
  "target": "https://target.com",
  "authenticated_user": "user@target.com",
  "session_cookie": "[REDACTED]",
  "total_entries": 142,
  "filtered_entries": 38,
  "replay_verified": true,
  "notes": "Captura durante flujo de checkout. Posible IDOR en /api/orders/{id}."
}
```

---

## Integración con Team

- **security-scout** → pasa URLs y contexto inicial
- **security-web** → usa este skill para capturar tráfico real antes de validación activa
- **security-ops** → recibe HAR sanitizado para compliance audit
- **security-report** → recibe curl templates y replay results como evidencia documentada

## Checklist de Cierre

```
[ ] Tráfico capturado completo
[ ] Sesión autenticada verificada
[ ] Datos sensibles sanitizados
[ ] HAR normalizado generado
[ ] curl templates exportados
[ ] Replay verificado (status codes match)
[ ] Diffing completado (si aplica)
[ ] Timing analysis completado (si race condition sospechada)
[ ] Artefactos persistidos en evidencia/
[ ] Metadata documentada
```
