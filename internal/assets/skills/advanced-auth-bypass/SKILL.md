# Skill: Advanced Authentication Bypass

**Categoría:** authentication
**Metodología base:** OWASP WSTG-AUTHN, PTES Authentication Testing
**Cuándo activar:** cuando se identifican mecanismos de autenticación débiles o mal implementados

---

## Protocolo

### Fase 1 — JWT Attack Patterns

**Algorithm Confusion Attacks:**
```
1. RS256 → HS256 Conversion:
   - Cambiar header de RS256 a HS256
   - Firmar con la clave pública del servidor (disponible en /.well-known/jwks.json)
   - El servidor usa la clave pública como secreto HMAC → bypass exitoso

2. None Algorithm Exploitation:
   - Header: {"alg":"none","typ":"JWT"}
   - Payload: claims arbitrarios
   - Sin firma (string vacía después del segundo punto)
   - Algunos parsers aceptan tokens sin validación

3. PS256/ES256 Confusion:
   - Cambiar entre familias de algoritmos
   - Algunos parsers solo verifican "family" no algoritmo específico
```

**Key Identification Attacks:**
```
1. kid (Key ID) Manipulation:
   - Path traversal: {"kid":"../../../etc/passwd"}
   - SQL injection: {"kid":"1' OR 1=1--"}
   - Command injection: {"kid":";cat /etc/passwd"}
   - Null kid: {"kid":null} → some servers use first available key

2. jku (JWK Set URL) Attacks:
   - Apuntar a servidor controlado: {"jku":"http://attacker.com/keys.json"}
   - El servidor descarga y usa tu clave pública
   - Firmar token con tu clave privada correspondiente

3. x5u (X.509 URL) Similar to jku but with X.509 certificates
```

**Claim Exploitation:**
```
1. Role/Privilege Escalation:
   - Agregar: {"role":"admin","isAdmin":true,"scope":["read","write","admin"]}
   - Claims ocultos: {"_role":"admin","x-admin":true}
   - Nested roles: {"permissions":{"admin":true}}

2. Token Confusion:
   - Cambiar 'sub' (subject) a otro usuario
   - Modificar 'aud' (audience) para acceder a otros servicios
   - Alterar 'iss' (issuer) para trust confusion

3. Time-based Attacks:
   - exp en futuro lejano (año 2099)
   - iat en pasado (token "viejo" pero válido)
   - nbf omitido (some parsers don't check "not before")
```

### Fase 2 — Session Management Attacks

**Session Fixation:**
```
1. Pre-set Session IDs:
   - Obtener session ID antes de login
   - Forzar uso de ese ID: ?session_id=ATTACKER_CONTROLLED
   - Después de login de víctima, usar mismo ID

2. Session ID Predictability:
   - Analizar patrón: ¿secuencial? ¿timestamp-based?
   - ¿UUID v1 (MAC address + time) vs UUID v4 (random)?
   - Generar múltiples sessions y buscar patrones
```

**Cookie Manipulation:**
```
1. Cookie Tossing:
   - Setear cookies de dominio amplio: .example.com
   - Overflow de cookies (4096 bytes por cookie, 180 cookies por dominio)
   - Cookies con mismo nombre, diferentes paths → cual prevalece?

2. Signature Stripping:
   - Cookies firmadas: value.signature
   - Remover .signature y enviar solo value
   - Algunos sistemas validan solo si signature está presente, no su valor

3. Parameter Pollution:
   - Múltiple cookies: session=ABC; session=XYZ
   - Diferentes parsers toman diferente valor (primero vs último)
```

### Fase 3 — OAuth/OpenID Connect Attacks

**OAuth Flow Manipulation:**
```
1. Redirect URI Bypass:
   - redirect_uri=http://attacker.com/callback
   - redirect_uri=http://localhost:8080/callback (localhost relay)
   - redirect_uri=http://app.example.com.evil.com/callback (subdomain take-over)
   - redirect_uri=urn:ietf:wg:oauth:2.0:oob (OOB attack)

2. State Parameter Omission:
   - Sin state parameter → CSRF possible
   - State predecible → generar y usar antes que víctima
   - State no validado en callback → attack succeeds

3. Authorization Code Leakage:
   - Referer header en redirect a terceros
   - Logging en servidor de atacante
   - Code reuse before legitimate client exchanges it
```

**Token Endpoint Attacks:**
```
1. Client ID Impersonation:
   - Usar client_id de otra aplicación (si no requiere client_secret)
   - Public clients sin autenticación

2. Refresh Token Abuse:
   - Refresh tokens sin expiración o rotación
   - Refresh token usado después de revocado (sin detección)
   - Refresh token de otro usuario (si no hay binding)

3. Scope Creep:
   - Solicitar scopes no autorizados: scope=admin:read admin:write
   - Escalar privilegios via scope manipulation
```

### Fase 4 — Multi-Factor Authentication Bypass

**MFA Logic Flaws:**
```
1. MFA Skip via Direct Access:
   - Acceso directo a post-MFA endpoints
   - Saltar paso de verificación MFA en flujo
   - API endpoints sin protección MFA (solo UI protegida)

2. Code Bruteforce:
   - OTP de 4-6 dígitos sin rate limiting
   - Codes con validez extendida (>5 min)
   - Reuse de codes (sin invalidación post-use)
   - No lockout after N failed attempts

3. Backup Code Abuse:
   - Backup codes sin limitación de uso
   - Backup codes predecibles o débilmente generados
   - Backup codes expuestos en código fuente o backups
```

**MFA Fatigue/Push Bombing:**
```
1. Push Notification Flooding:
   - Enviar múltiples solicitudes MFA push
   - Usuario acepta por fatiga o accidente
   - Sin throttling en envío de push notifications

2. SMS/Email Interception:
   - SMS no cifrado (interceptación posible)
   - Email como MFA channel (email comprometido = MFA bypass)
```

### Fase 5 — Password Reset & Account Recovery Attacks

**Password Reset Token Abuse:**
```
1. Token Prediction:
   - Tokens basados en timestamp
   - Tokens basados en user_id
   - Tokens cortos o débilmente aleatorios

2. Host Header Injection:
   - Host: attacker.com en request de reset
   - Link de reset enviado a servidor de atacante
   - Password reset token leak via Referer

3. Parameter Manipulation:
   - Cambiar email en request: {"email":"victim@target.com"} → {"email":"attacker@evil.com"}
   - User enumeration via timing differences en reset response
```

**Account Recovery Logic Flaws:**
```
1. Security Question Bypass:
   - Questions con respuestas públicas (OSINT)
   - Respuestas case-insensitive o sin normalización
   - Múltiples attempts sin lockout

2. Knowledge-based Authentication:
   - Preguntas basadas en datos personales (address, phone)
   - Datos disponibles en breaches públicas
   - Respuestas inferibles de redes sociales
```

### Fase 6 — Output

```markdown
## Advanced Auth Bypass — [Target]

### JWT Vulnerabilities
[vectores identificados con evidencia]

### Session Management Issues
[findings con CWE mapping]

### OAuth/OpenID Misconfigurations
[flujos vulnerables y impacto]

### MFA Bypass Vectors
[lógica comprometida y alternativas]

### Account Recovery Flaws
[riesgo de toma de control de cuentas]

### Exploitability Assessment
[vectores más críticos y recomendación de validación]
```

---

## Guardrails

- ⚠️ NO ejecutar ataques activos sin autorización explícita
- ⚠️ NO usar tokens reales de usuarios legítimos
- ⚠️ NO realizar push bombing o MFA fatigue attacks sin autorización específica
- ✅ Documentar vectores conceptualmente con evidencia observable
- ✅ Marcar hallazgos como `suspected` hasta validación autorizada

## Anti-patterns

- ❌ Asumir "JWT seguro" sin analizar claims y algoritmo
- ❌ Ignorar OAuth misconfigurations — son el vector #1 en apps modernas
- ❌ Tratar MFA como infalible — la lógica puede tener bypasses
- ❌ Olvidar password reset flows —常见 vector de account takeover
- ❌ Reportar sin evidencia de que el mecanismo es explotable
