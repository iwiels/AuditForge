# Skill: API Schema Harvest

### Protocol
1. **Discovery**: Identify public API specs (Swagger/OpenAPI), Telerik/WSDL files, or GraphiQL endpoints.
2. **Traffic Synthesis**: Harvest schemas from HAR files or intercepted traffic if public specs are missing.
3. **Normalization**: Convert inconsistent schemas into a standard OpenAPI 3.x format for the audit report.
4. **Validation**: Validate that the schema matches the actual runtime behavior of the API.
5. **Team Lead**: Delegate deep parameter fuzzing of discovered endpoints to `security-web`.
