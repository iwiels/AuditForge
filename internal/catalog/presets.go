package catalog

import "orquestador-auditor/internal/model"

var presets = []model.Profile{
	{
		Name:       "surface-audit",
		Preset:     model.PresetSurfaceAudit,
		Persona:    model.PersonaAnalyst,
		Skills:     []model.SkillID{model.SkillSurfaceDiscovery, model.SkillAuthorizationGuard},
		Components: []model.ComponentID{model.ComponentMCP, model.ComponentPrompts, model.ComponentCommands, model.ComponentBackup, model.ComponentVerify, model.ComponentSystem},
		Mode:       model.AuditModeSingleTarget,
	},
	{
		Name:       "deep-audit",
		Preset:     model.PresetDeepAudit,
		Persona:    model.PersonaRedTeamBounded,
		Skills:     []model.SkillID{model.SkillSurfaceDiscovery, model.SkillWebTriage, model.SkillSupplyChainTriage, model.SkillThreatModeling, model.SkillEvidenceReporting, model.SkillAuthorizationGuard, model.SkillCodeReview, model.SkillOSINTPassive, model.SkillSecureDesignReview, model.SkillComplianceCheck},
		Components: []model.ComponentID{model.ComponentMCP, model.ComponentPrompts, model.ComponentCommands, model.ComponentSkills, model.ComponentOutputStyles, model.ComponentSubAgents, model.ComponentBackup, model.ComponentVerify, model.ComponentSystem},
		Mode:       model.AuditModeSingleTarget,
	},
	{
		Name:       "forensics",
		Preset:     model.PresetForensics,
		Persona:    model.PersonaExecutiveReporter,
		Skills:     []model.SkillID{model.SkillThreatModeling, model.SkillEvidenceReporting, model.SkillAuthorizationGuard, model.SkillIncidentResponse},
		Components: []model.ComponentID{model.ComponentPrompts, model.ComponentSkills, model.ComponentBackup, model.ComponentVerify, model.ComponentSystem},
		Mode:       model.AuditModeCampaign,
	},
}

func AllPresets() []model.Profile {
	items := make([]model.Profile, len(presets))
	copy(items, presets)
	return items
}
