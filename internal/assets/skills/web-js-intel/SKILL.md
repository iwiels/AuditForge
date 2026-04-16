# Skill: Web JS Intel

### Protocol
1. **Extraction**: Use `jsluice` or AST-based parsers to extract secrets, endpoints, and logic from JS bundles.
2. **Deobfuscation**: Identify obfuscated or minified code sections that hide sensitive functionality.
3. **Sensitive Token Hunting**: Search for API keys, AWS credentials, or internal Dev URL patterns.
4. **Endpoint Synthesis**: Feed discovered endpoints into `api-schema-harvest` for further mapping.
