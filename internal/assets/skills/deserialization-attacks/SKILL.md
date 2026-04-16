# Skill: Deserialization Attacks

**Categoría:** deserialization, injection
**Metodología base:** OWASP Deserialization Cheat Sheet, CWE-502
**Cuándo activar:** cuando se identifican datos serializados en requests, cookies, o APIs (Java, PHP, Python, .NET, Node.js)

---

## Protocolo

### Fase 1 — Identification of Serialization Formats

**Java Serialization:**
```
1. Detection Markers:
   - Binary: AC ED 00 05 (magic bytes)
   - Base64: rO0AB... (decoded starts with AC ED)
   - Content-Type: application/x-java-serialized-object
   - Headers: X-Java-Serialization, java-serialized-object

2. Where Found:
   - HttpSession cookies (JSESSIONID sometimes serialized)
   - Request/response bodies
   - Hidden form fields
   - URL parameters (base64 encoded)
   - JMX/RMI interfaces
   - Cache files

3. Frameworks Using Serialization:
   - Spring WebFlow
   - Apache Commons Collections
   - Hibernate
   - Flex/L BlazeDS
```

**PHP Serialization:**
```
1. Detection Markers:
   - Text format: O:4:"User":2:{s:8:"username";s:5:"admin";}
   - Pattern: O:[length]:"[class]":[attrs]:{...}
   - Pattern: a:[length]:{[key-value pairs]}
   - Pattern: s:[length]:"[string]";

2. Where Found:
   - $_SESSION data (PHP session files)
   - Cookies (sometimes serialized)
   - URL parameters
   - POST data in legacy APIs
   - Cache/metadata files

3. Common Targets:
   - __wakeup() magic method
   - __destruct() magic method
   - __toString() magic method
   - Monolog, SwiftMailer, Guzzle libraries
```

**Python Serialization:**
```
1. Pickle Format:
   - Binary: \x80\x03\x95... (protocol 3)
   - Base64: gAN9cQAo...
   - Markers: c__main__, S'string', Vunicode

2. Where Found:
   - Flask session cookies (if not using signing)
   - Celery task queues
   - Django signed cookies (if SECRET_KEY known)
   - API request/response bodies
   - Cache backends

3. YAML Serialization:
   - !!python/object/apply:os.system
   - !!python/object/new:subprocess
   - PyYAML unsafe load()

4. JSON-like with exec:
   - Custom serializers using eval()
   - marshal.loads() on user input
```

**.NET Serialization:**
```
1. Detection Markers:
   - Binary: FF 01 00 00 00 00 00 00 0C 01 00 00
   - XmlSerializer: XML with xsi:type attributes
   - DataContractSerializer: specific XML structure
   - LosFormatter: base64 starting with /wEx

2. Where Found:
   - ViewState (__VIEWSTATE hidden field)
   - Cookies (ASP.NET session data)
   - TempData in ASP.NET MVC
   - WebService responses
   - Cache/session state

3. Frameworks:
   - Newtonsoft.Json (TypeNameHandling = All → vulnerable)
   - FastJson
   - JavaScriptTypeResolver
```

**Node.js Serialization:**
```
1. Detection Patterns:
   - JSON with __proto__ or constructor
   - node-serialize package output
   - func: function(){} in JSON (non-standard)

2. Where Found:
   - Session stores (Redis, MongoDB)
   - API payloads
   - Custom serialization functions

3. Vulnerable Libraries:
   - node-serialize
   - serialize-to-js
   - Custom JSON.parse with reviver misuse
```

### Fase 2 — Gadget Chain Analysis

**Java Gadget Chains:**
```
1. CommonsCollections (ysoserial):
   - CC1-CC7 chains
   - InvokerTransformer exploitation
   - LazyMap trigger
   - Requires: commons-collections ≤ 3.2.1

2. Spring AOP:
   - Spring Core ≤ 4.3.7
   - ReflectiveMethodInvocation
   - Requires: spring-aop dependency

3. Rome (XStream alternative):
   - Rome ≤ 1.6.0
   - ToStringBean exploitation
   - JdbcRowSetImpl RCE

4. CommonsBeanutils:
   - BeanComparator with PropertyUtils
   - Requires: commons-beanutils

Detection Strategy:
- Identify libraries in dependencies (SAST/SCA)
- Check for deserialization endpoints
- Map available gadget chains
```

**PHP Gadget Chains:**
```
1. Monolog (Logging Library):
   - Chain: RCE via system() call
   - Requires: Monolog ≤ 1.25.1
   - Payload triggers on __destruct()

2. SwiftMailer:
   - Chain: File write via __toString()
   - Requires: SwiftMailer dependency
   - Arbitrary file creation

3. Guzzle (HTTP Client):
   - Chain: SSRF via __destruct()
   - Requires: Guzzle ≤ 6.3.2
   - Arbitrary HTTP requests

4. Laravel/Illuminate:
   - Framework-specific chains
   - PendingResourceRegistration RCE
   - Requires: specific Laravel versions

Detection Strategy:
- Identify PHP framework and version
- Check composer.json for vulnerable libraries
- Look for unserialize() calls in code
```

**Python Pickle Gadgets:**
```
1. os.system() Execution:
   - cos\nsystem\n(S'command'\ntR.
   - Direct command execution
   - Most basic pickle payload

2. subprocess.Popen:
   - csubprocess\nPopen\n...
   - More flexible than os.system
   - Can capture output

3. __import__() Abuse:
   - c__builtin__\n__import__\n(S'os'\ntR.
   - Module import then execution
   - Bypasses some filters

4. YAML-specific:
   - !!python/object/apply:os.system [cmd]
   - !!python/object/new:subprocess.Popen [cmd, shell:true]
   - Requires: yaml.load() not yaml.safe_load()

Detection Strategy:
- Look for pickle.loads(), yaml.load()
- Check for marshal.loads()
- Identify eval() on user input
```

### Fase 3 — Exploitation Concepts (Without Active Exploitation)

**Detection Without Execution:**
```
1. Fingerprint Serialization:
   - Send malformed data → error messages?
   - Error reveals library/version?
   - Stack trace shows deserialization?

2. Version Analysis:
   - Check dependencies (SCA tools)
   - Identify vulnerable library versions
   - Map to known CVEs

3. Behavioral Analysis:
   - Response timing differences
   - Error patterns with malformed input
   - Out-of-band interaction indicators
```

**Blind Deserialization Detection:**
```
1. Time-based Detection:
   - Sleep/delay gadgets (if available)
   - Measure response time differences
   - Confirms processing without RCE

2. DNS/HTTP Interaction:
   - Gadgets that trigger DNS lookup
   - HTTP requests to external server
   - Use DNSCanary or Burp Collaborator

3. Error-based Detection:
   - Class not found errors
   - Malformed data exceptions
   - Stack traces revealing internals
```

### Fase 4 — Language-Specific Vectors

**ViewState Exploitation (.NET):**
```
1. ViewState Analysis:
   - Is ViewState signed? (enableViewStateMac)
   - ValidationKey/DecryptionKey known?
   - Machine.config defaults?

2. ViewState Deserialization:
   - LosFormatter.Parse()
   - ObjectStateFormatter.Deserialize()
   - Exploit with ysoserial.net

3. Detection:
   - __VIEWSTATE value in forms
   - Base64 decoded: check for serialization markers
   - Error messages on tampering
```

**JWT Claims Deserialization:**
```
1. JWT Payload Objects:
   - JWTs may contain serialized objects
   - Deserialized during validation
   - Framework-specific behavior

2. Custom Claims:
   - Objects in claims (not just primitives)
   - Deserialized by recipient
   - May trigger gadget chains
```

**Cache Poisoning via Serialization:**
```
1. Session Store Manipulation:
   - Redis/Memcached session injection
   - Malicious serialized object
   - Deserialized on session load

2. CDN Cache Poisoning:
   - Cache serialized responses
   - Include malicious payloads
   - Deserialized by backend
```

### Fase 5 — Output

```markdown
## Deserialization Security — [Target]

### Serialization Endpoints Identified
[list with format, location, risk level]

### Library/Version Analysis
[vulnerable dependencies found]

### Potential Gadget Chains
[applicable chains based on stack]

### Exploitation Feasibility
[assessment per endpoint]

### Detection Evidence
[errors, behavior, OOB indicators]

### Recommended Testing
[vectors requiring authorized validation]
```

---

## Guardrails

- ⚠️ NO ejecutar gadgets de RCE sin autorización explícita
- ⚠️ NO modificar serialized objects en producción
- ⚠️ NO usar ysoserial/ysoserial.net activamente sin aprobación
- ✅ Identificar endpoints y formatos de serialización
- ✅ Analizar dependencias y versiones (SAST/SCA)
- ✅ Documentar vectores como `suspected` con evidencia observable
- ✅ Recomendar validación controlada con herramientas especializadas

## Anti-patterns

- ❌ Asumir "solo datos" sin analizar si son objetos serializados
- ❌ Ignorar que JSON/XML pueden contener objetos peligrosos (TypeNameHandling)
- ❌ No revisar dependencias del proyecto para gadget chains
- ❌ Concluir "seguro" sin verificar todas las rutas de deserialización
- ❌ Olvidar que caché/sessions también deserializan datos
