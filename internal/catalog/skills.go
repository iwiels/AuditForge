package catalog

import "orquestador-auditor/internal/model"

type Skill struct {
	ID       model.SkillID
	Name     string
	Category string
	Priority string
}

var skills = []Skill{
	{ID: model.SkillAuthorizationGuard, Name: "authorization-guard", Category: "safety", Priority: "p0"},
	{ID: model.SkillNetworkRecon, Name: "network-recon", Category: "network", Priority: "p0"},
	{ID: model.SkillTLSVHostEnum, Name: "tls-vhost-enum", Category: "network", Priority: "p1"},
	{ID: model.SkillSurfaceDiscovery, Name: "surface-discovery", Category: "recon", Priority: "p0"},
	{ID: model.SkillOSINTPassive, Name: "osint-passive", Category: "recon", Priority: "p1"},
	{ID: model.SkillArchiveIntel, Name: "archive-intel", Category: "recon", Priority: "p1"},
	{ID: model.SkillWebTriage, Name: "web-triage", Category: "web", Priority: "p0"},
	{ID: model.SkillWebJSIntel, Name: "web-js-intel", Category: "web-dynamic", Priority: "p0"},
	{ID: model.SkillJSDeobfuscationIntel, Name: "js-deobfuscation-intel", Category: "web-dynamic", Priority: "p1"},
	{ID: model.SkillBrowserAPIMapping, Name: "browser-api-mapping", Category: "web-dynamic", Priority: "p0"},
	{ID: model.SkillProxyCaptureReplay, Name: "proxy-capture-replay", Category: "web-dynamic", Priority: "p1"},
	{ID: model.SkillRequestInterceptionManipulation, Name: "request-interception-manipulation", Category: "web-dynamic", Priority: "p0"},
	{ID: model.SkillAPISchemaHarvest, Name: "api-schema-harvest", Category: "api", Priority: "p0"},
	{ID: model.SkillAPIParameterMapping, Name: "api-parameter-mapping", Category: "api", Priority: "p0"},
	{ID: model.SkillParamDiscoveryFuzz, Name: "param-discovery-fuzz", Category: "api", Priority: "p1"},
	{ID: model.SkillSQLIHypothesisValidate, Name: "sqli-hypothesis-validation", Category: "validation", Priority: "p1"},
	{ID: model.SkillSupplyChainTriage, Name: "supply-chain-triage", Category: "supply-chain", Priority: "p0"},
	{ID: model.SkillCodeReview, Name: "code-review", Category: "code", Priority: "p0"},
	{ID: model.SkillSecureDesignReview, Name: "secure-design-review", Category: "architecture", Priority: "p1"},
	{ID: model.SkillThreatModeling, Name: "threat-modeling", Category: "analysis", Priority: "p1"},
	{ID: model.SkillVulnerabilityCorrelation, Name: "vulnerability-correlation", Category: "analysis", Priority: "p0"},
	{ID: model.SkillEvidenceReporting, Name: "evidence-reporting", Category: "reporting", Priority: "p1"},
	{ID: model.SkillComplianceCheck, Name: "compliance-check", Category: "governance", Priority: "p1"},
	{ID: model.SkillIncidentResponse, Name: "incident-response", Category: "operations", Priority: "p1"},
	{ID: model.SkillAdvancedAuthBypass, Name: "advanced-auth-bypass", Category: "authentication", Priority: "p1"},
	{ID: model.SkillFileUploadAttacks, Name: "file-upload-attacks", Category: "input-validation", Priority: "p1"},
	{ID: model.SkillDeserializationAttacks, Name: "deserialization-attacks", Category: "injection", Priority: "p1"},
	{ID: model.SkillWebSocketSecurity, Name: "websocket-security", Category: "web-dynamic", Priority: "p1"},
	{ID: model.SkillJWTJWKSAnalysis, Name: "jwt-jwks-analysis", Category: "authentication", Priority: "p0"},
}

func AllSkills() []Skill {
	items := make([]Skill, len(skills))
	copy(items, skills)
	return items
}
