# Skill: js-deobfuscation-intel

## Mission
Turn large or obfuscated JavaScript bundles into readable, evidence-bearing intelligence.

## Method
1. Beautify or pretty-print the bundle when readability is poor.
2. Look for source maps and original source references.
3. Use `jsluice` or AST-aware extraction to collect endpoints, secrets, tokens, and auth flows.
4. Correlate static extraction with runtime browser traffic when available.

## Output
- `observed_js_artifacts`
- `extracted_endpoints`
- `suspected_client_secrets`
- `runtime_correlation_notes`
