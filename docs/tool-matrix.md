# Tool Matrix

> [!NOTE]
> Most of the tools below are automatically managed by the orchestrator in the portable bin directory (`~/.orquestador-auditor/bin`).

## Phase 1 - Network Recon

- `nmap` - port discovery, service enumeration, and version detection
- TLS/certificate inspection - certificate SANs, alternate vhosts, and HTTPS posture

## Phase 2 - Surface Discovery

- `whatweb` - web fingerprinting
- `katana` - crawling and URL discovery
- `waymore` - historical URL and archived response intelligence
- `robots.txt` / `sitemap.xml` / standard recon paths - low-noise surface hints

## Phase 3 - JS / Client-Side Intel

- `js-beautify` - beautify/minified JS readability support
- `jsluice` - AST-aware JavaScript endpoint and secret extraction
- `chromedp` - runtime browser traffic capture for XHR, fetch, cookies, and websocket activity
- source maps - original source recovery when exposed

## Phase 4 - API / Parameter Discovery

- `mitmproxy` - proxy-based authenticated traffic capture and replay workflows
- `mitmproxy2swagger` - convert HAR or mitmproxy captures into OpenAPI
- `kin-openapi` - load, validate, and normalize OpenAPI schemas
- `arjun` - hidden parameter discovery
- `ffuf` - bounded path, param, and body fuzzing

## Phase 5 - Vulnerability Hypothesis

- manual authz / IDOR / SQLi / XSS / SSRF / command injection reasoning
- static + dynamic evidence correlation across previous phases

## Phase 6 - Authorized Validation

- `sqlmap` - authorized SQL injection validation against scoped parameters only
- targeted confirmation workflows - reproduce only the already-supported hypothesis

## Phase 7 - Correlation

- `searchsploit` - local exploit reference lookup for service versions
- CWE / OWASP mapping - severity and remediation contextualization

## Source and Supply Chain

- `semgrep` - source code static analysis
- `gitleaks` - secret scanning
- `trivy` - filesystem and dependency vulnerability scanning
- `grype` - package and directory vulnerability scanning
