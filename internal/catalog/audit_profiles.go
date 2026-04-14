package catalog

import (
	"fmt"
	"strings"

	"orquestador-auditor/internal/model"
)

var allSpecialistSubAgents = []string{"security-scout", "security-web", "security-report", "security-memory", "security-supply-chain"}
var allSpecialistNativeAgents = []string{"security-orchestrator", "security-scout", "security-web", "security-report", "security-memory", "security-code", "security-compliance", "security-ops"}

var auditProfiles = []model.AuditProfile{
	{
		ID:          model.AuditProfileRecon,
		Name:        "Recon",
		Description: "Scope validation, network recon, passive discovery, and early technology mapping.",
		FocusAreas:  []string{"authorization validation", "target kind classification", "port and service discovery", "passive recon", "surface inventory", "initial stack detection"},
		Components: []model.ComponentID{
			model.ComponentMCP,
			model.ComponentPrompts,
			model.ComponentCommands,
			model.ComponentSkills,
			model.ComponentSubAgents,
			model.ComponentBackup,
			model.ComponentVerify,
			model.ComponentSystem,
		},
		Skills:       []model.SkillID{model.SkillAuthorizationGuard, model.SkillNetworkRecon, model.SkillTLSVHostEnum, model.SkillSurfaceDiscovery, model.SkillOSINTPassive, model.SkillArchiveIntel},
		Commands:     []string{"security-scout", "network-recon", "memory-search", "orquestador-auditor"},
		SubAgents:    append([]string(nil), allSpecialistSubAgents...),
		NativeAgents: append([]string(nil), allSpecialistNativeAgents...),
		Risk: model.RiskPolicy{
			Mode:           "passive-first",
			Summary:        "Read-heavy reconnaissance with bounded enumeration only; prioritize ports, TLS/vhosts, crawl seeds, and stack evidence before any active validation.",
			AllowedActions: []string{"leer contexto y archivos", "buscar memoria", "enumerar puertos y servicios autorizados", "mapear TLS/vhosts", "hacer crawling y fingerprint pasivo", "construir asset inventory"},
			BlockedActions: []string{"explotacion destructiva", "payloads activos intrusivos", "fuzzing intensivo", "persistencia o weaponization"},
			Permissions:    model.ToolPermissions{Read: true, Write: false, Edit: false, Bash: true},
		},
	},
	{
		ID:          model.AuditProfileWebTriage,
		Name:        "Web Triage",
		Description: "Deep web analysis including JS intel, API/parameter discovery, and vulnerability hypothesis generation.",
		FocusAreas:  []string{"authentication and authorization", "input validation", "browser capture", "JavaScript and source maps", "OpenAPI/schema harvest", "parameter discovery", "vulnerability hypothesis"},
		Components: []model.ComponentID{
			model.ComponentMCP,
			model.ComponentPrompts,
			model.ComponentCommands,
			model.ComponentSkills,
			model.ComponentSubAgents,
			model.ComponentBackup,
			model.ComponentVerify,
			model.ComponentSystem,
		},
		Skills:       []model.SkillID{model.SkillAuthorizationGuard, model.SkillNetworkRecon, model.SkillTLSVHostEnum, model.SkillSurfaceDiscovery, model.SkillWebTriage, model.SkillWebJSIntel, model.SkillJSDeobfuscationIntel, model.SkillBrowserAPIMapping, model.SkillAPISchemaHarvest, model.SkillAPIParameterMapping, model.SkillParamDiscoveryFuzz, model.SkillProxyCaptureReplay, model.SkillArchiveIntel, model.SkillThreatModeling, model.SkillSQLIHypothesisValidate, model.SkillAdvancedAuthBypass, model.SkillFileUploadAttacks, model.SkillDeserializationAttacks},
		Commands:     []string{"security-scout", "network-recon", "deep-web", "js-intel", "api-discovery", "sqli-validate", "memory-search", "orquestador-auditor"},
		SubAgents:    append([]string(nil), allSpecialistSubAgents...),
		NativeAgents: append([]string(nil), allSpecialistNativeAgents...),
		Risk: model.RiskPolicy{
			Mode:           "bounded-active-analysis",
			Summary:        "Allows deeper web, JS, API, and parameter analysis plus bounded SQLi validation, but still forbids destructive exploitation or uncontrolled fuzzing.",
			AllowedActions: []string{"captura de trafico del navegador", "analisis de JavaScript y source maps", "harvest y normalizacion de OpenAPI", "descubrimiento de parametros", "heuristicas de SQLi/XSS/SSRF/IDOR", "validacion acotada autorizada"},
			BlockedActions: []string{"explotacion destructiva", "payloads persistentes", "cambios en infraestructura del target", "fuzzing sin limites", "weaponization"},
			Permissions:    model.ToolPermissions{Read: true, Write: false, Edit: false, Bash: true},
		},
	},
	{
		ID:          model.AuditProfileSupplyChain,
		Name:        "Supply Chain",
		Description: "Dependency, source code, CI/CD, secret exposure, and secure design review.",
		FocusAreas:  []string{"dependency risk", "code review", "pipeline hardening", "secret handling", "design review", "vulnerability correlation"},
		Components: []model.ComponentID{
			model.ComponentMCP,
			model.ComponentPrompts,
			model.ComponentCommands,
			model.ComponentSkills,
			model.ComponentSubAgents,
			model.ComponentBackup,
			model.ComponentVerify,
			model.ComponentSystem,
		},
		Skills:       []model.SkillID{model.SkillAuthorizationGuard, model.SkillSupplyChainTriage, model.SkillCodeReview, model.SkillSecureDesignReview, model.SkillComplianceCheck, model.SkillThreatModeling, model.SkillVulnerabilityCorrelation},
		Commands:     []string{"supply-chain", "correlate-findings", "memory-search", "orquestador-auditor"},
		SubAgents:    append([]string(nil), allSpecialistSubAgents...),
		NativeAgents: append([]string(nil), allSpecialistNativeAgents...),
		Risk: model.RiskPolicy{
			Mode:           "read-heavy-code-audit",
			Summary:        "Focused on code, dependencies, CI/CD, and secrets; correlation-heavy, non-destructive, and evidence-first.",
			AllowedActions: []string{"leer repositorios y manifests", "analizar dependencias y CVEs", "buscar secretos", "revisar CI/CD y diseno seguro", "correlacionar hallazgos con CWE/OWASP"},
			BlockedActions: []string{"modificar codigo del target", "ejecutar payloads contra servicios", "cambiar pipelines o secretos", "acciones destructivas"},
			Permissions:    model.ToolPermissions{Read: true, Write: false, Edit: false, Bash: true},
		},
	},
	{
		ID:          model.AuditProfileReporting,
		Name:        "Reporting",
		Description: "Evidence consolidation, severity mapping, OWASP/CWE correlation, and remediation output.",
		FocusAreas:  []string{"finding consolidation", "evidence quality", "risk communication", "severity mapping", "OWASP/CWE mapping", "remediation guidance"},
		Components: []model.ComponentID{
			model.ComponentMCP,
			model.ComponentPrompts,
			model.ComponentCommands,
			model.ComponentSkills,
			model.ComponentOutputStyles,
			model.ComponentSubAgents,
			model.ComponentBackup,
			model.ComponentVerify,
			model.ComponentSystem,
		},
		Skills:       []model.SkillID{model.SkillAuthorizationGuard, model.SkillEvidenceReporting, model.SkillThreatModeling, model.SkillComplianceCheck, model.SkillVulnerabilityCorrelation},
		Commands:     []string{"security-report", "correlate-findings", "memory-search", "orquestador-auditor"},
		SubAgents:    append([]string(nil), allSpecialistSubAgents...),
		NativeAgents: append([]string(nil), allSpecialistNativeAgents...),
		Risk: model.RiskPolicy{
			Mode:           "non-operational-reporting",
			Summary:        "Consolidates evidence, maps severity/CWE/OWASP, and writes remediation without active target interaction.",
			AllowedActions: []string{"leer hallazgos previos", "correlacionar evidencias", "mapear severidad/CWE/OWASP", "redactar reportes y remediacion", "guardar artefactos locales de salida"},
			BlockedActions: []string{"escaneo activo", "fuzzing", "modificaciones del target", "investigacion fuera del alcance del reporte"},
			Permissions:    model.ToolPermissions{Read: true, Write: true, Edit: true, Bash: false},
		},
	},
	{
		ID:          model.AuditProfileMemoryOnly,
		Name:        "Memory Only",
		Description: "Long-term context retrieval, engagement continuity, and methodological handoff.",
		FocusAreas:  []string{"engagement recall", "finding history", "campaign continuity", "handoff preparation"},
		Components: []model.ComponentID{
			model.ComponentMCP,
			model.ComponentPrompts,
			model.ComponentCommands,
			model.ComponentSubAgents,
			model.ComponentBackup,
			model.ComponentVerify,
			model.ComponentSystem,
		},
		Skills:       []model.SkillID{},
		Commands:     []string{"memory-search", "correlate-findings", "orquestador-auditor"},
		SubAgents:    append([]string(nil), allSpecialistSubAgents...),
		NativeAgents: append([]string(nil), allSpecialistNativeAgents...),
		Risk: model.RiskPolicy{
			Mode:           "context-only",
			Summary:        "Only continuity and recall; no active analysis, but enough context to hand work to the next phase cleanly.",
			AllowedActions: []string{"buscar memoria", "leer contexto", "resumir campanas previas", "preparar handoff", "correlacionar hallazgos existentes"},
			BlockedActions: []string{"bash operativo", "fuzzing", "captura dinamica", "edicion de codigo o configs del target"},
			Permissions:    model.ToolPermissions{Read: true, Write: false, Edit: false, Bash: false},
		},
	},
}

func AllAuditProfiles() []model.AuditProfile {
	items := make([]model.AuditProfile, len(auditProfiles))
	copy(items, auditProfiles)
	return items
}

func DefaultAuditProfile() model.AuditProfile {
	profile, _ := AuditProfileByID(string(model.AuditProfileRecon))
	return profile
}

func AuditProfileByID(raw string) (model.AuditProfile, error) {
	id := strings.TrimSpace(strings.ToLower(raw))
	if id == "" {
		return DefaultAuditProfile(), nil
	}
	for _, profile := range auditProfiles {
		if string(profile.ID) == id {
			return profile, nil
		}
	}
	return model.AuditProfile{}, fmt.Errorf("unsupported audit profile %q", raw)
}
