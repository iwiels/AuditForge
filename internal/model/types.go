package model

type AgentID string

const (
	AgentClaudeCode AgentID = "claude-code"
	AgentClaude     AgentID = "claude"
	AgentCursor     AgentID = "cursor"
	AgentOpenCode   AgentID = "opencode"
	AgentGemini     AgentID = "gemini"
)

type ComponentID string

const (
	ComponentAssets       ComponentID = "assets"
	ComponentMCP          ComponentID = "mcp"
	ComponentPrompts      ComponentID = "prompts"
	ComponentCommands     ComponentID = "commands"
	ComponentSkills       ComponentID = "skills"
	ComponentOutputStyles ComponentID = "output-styles"
	ComponentSubAgents    ComponentID = "subagents"
	ComponentBackup       ComponentID = "backup"
	ComponentVerify       ComponentID = "verify"
	ComponentSystem       ComponentID = "system"
)

type SkillID string

const (
	SkillSurfaceDiscovery                SkillID = "surface-discovery"
	SkillNetworkRecon                    SkillID = "network-recon"
	SkillTLSVHostEnum                    SkillID = "tls-vhost-enum"
	SkillWebTriage                       SkillID = "web-triage"
	SkillSupplyChainTriage               SkillID = "supply-chain-triage"
	SkillThreatModeling                  SkillID = "threat-modeling"
	SkillEvidenceReporting               SkillID = "evidence-reporting"
	SkillAuthorizationGuard              SkillID = "authorization-guard"
	SkillWebJSIntel                      SkillID = "web-js-intel"
	SkillJSDeobfuscationIntel            SkillID = "js-deobfuscation-intel"
	SkillBrowserAPIMapping               SkillID = "browser-api-mapping"
	SkillProxyCaptureReplay              SkillID = "proxy-capture-replay"
	SkillAPISchemaHarvest                SkillID = "api-schema-harvest"
	SkillAPIParameterMapping             SkillID = "api-parameter-mapping"
	SkillParamDiscoveryFuzz              SkillID = "param-discovery-fuzz"
	SkillArchiveIntel                    SkillID = "archive-intel"
	SkillSQLIHypothesisValidate          SkillID = "sqli-hypothesis-validation"
	SkillVulnerabilityCorrelation        SkillID = "vulnerability-correlation"
	SkillCodeReview                      SkillID = "code-review"
	SkillOSINTPassive                    SkillID = "osint-passive"
	SkillSecureDesignReview              SkillID = "secure-design-review"
	SkillIncidentResponse                SkillID = "incident-response"
	SkillComplianceCheck                 SkillID = "compliance-check"
	SkillAdvancedAuthBypass              SkillID = "advanced-auth-bypass"
	SkillFileUploadAttacks               SkillID = "file-upload-attacks"
	SkillDeserializationAttacks          SkillID = "deserialization-attacks"
)

type AuditProfileID string

const (
	AuditProfileRecon       AuditProfileID = "recon"
	AuditProfileWebTriage   AuditProfileID = "web-triage"
	AuditProfileRedTeam     AuditProfileID = "red-team"
	AuditProfileSupplyChain AuditProfileID = "supply-chain"
	AuditProfileReporting   AuditProfileID = "reporting"
	AuditProfileMemoryOnly  AuditProfileID = "memory-only"
)

type ToolPermissions struct {
	Read  bool
	Write bool
	Edit  bool
	Bash  bool
}

type RiskPolicy struct {
	Mode           string
	Summary        string
	AllowedActions []string
	BlockedActions []string
	Permissions    ToolPermissions
}

type AuditProfile struct {
	ID           AuditProfileID
	Name         string
	Description  string
	FocusAreas   []string
	Components   []ComponentID
	Skills       []SkillID
	Commands     []string
	SubAgents    []string
	NativeAgents []string
	Risk         RiskPolicy
}

func (p AuditProfile) IncludesComponent(component ComponentID) bool {
	for _, item := range p.Components {
		if item == component {
			return true
		}
	}
	return false
}

func (p AuditProfile) AllowsSkill(skill SkillID) bool {
	for _, item := range p.Skills {
		if item == skill {
			return true
		}
	}
	return false
}

func (p AuditProfile) AllowsCommand(name string) bool {
	for _, item := range p.Commands {
		if item == name {
			return true
		}
	}
	return false
}

func (p AuditProfile) AllowsSubAgent(name string) bool {
	for _, item := range p.SubAgents {
		if item == name {
			return true
		}
	}
	return false
}

func (p AuditProfile) AllowsNativeAgent(name string) bool {
	for _, item := range p.NativeAgents {
		if item == name {
			return true
		}
	}
	return false
}

type SystemPromptStrategy int

const (
	StrategyMarkdownSections SystemPromptStrategy = iota
	StrategyFileReplace
	StrategyAppendToFile
	StrategyInstructionsFile
)

type MCPStrategy int

const (
	StrategySeparateMCPFiles MCPStrategy = iota
	StrategyMergeIntoSettings
	StrategyMCPConfigFile
)
