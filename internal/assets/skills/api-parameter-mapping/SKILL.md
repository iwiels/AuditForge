# Skill: api-parameter-mapping

## Mission
Build a parameter-aware model of the API surface before active validation.

## Method
1. Harvest OpenAPI or proxy-derived schemas.
2. Enumerate visible parameters from docs, traffic, and JS.
3. Use bounded discovery to identify hidden parameters.
4. Normalize endpoint -> method -> auth -> params -> evidence.
5. Highlight authz, input validation, and mass-assignment candidates.

## Output
- `api_inventory`
- `observed_params`
- `suspected_hidden_params`
- `hypothesis_candidates`
