# Skill: sqli-hypothesis-validation

## Mission
Validate SQL injection hypotheses safely and only when justified.

## Method
1. Start from evidence-backed hypotheses, not blind automation.
2. Classify the signal: error-based, boolean-based, time-based, differential response.
3. Capture a reproducible baseline.
4. If authorization allows, run targeted `sqlmap` confirmation against the specific parameter.
5. Record whether the result is `suspected`, `validated`, or `blocked-by-policy`.

## Guardrails
- No broad sqlmap against whole sites.
- No destructive payloads.
- No data extraction unless the engagement explicitly authorizes it.
