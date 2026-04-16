package catalog

import "orquestador-auditor/internal/model"

type Agent struct {
	ID         model.AgentID
	Name       string
	Tier       model.SupportTier
	ConfigPath string
}

var allAgents = []Agent{
	{ID: model.AgentClaudeCode, Name: "Claude Code", Tier: model.TierFull, ConfigPath: "~/.claude"},
	{ID: model.AgentClaude, Name: "Claude", Tier: model.TierFull, ConfigPath: "~/.claude"},
	{ID: model.AgentOpenCode, Name: "OpenCode", Tier: model.TierFull, ConfigPath: "~/.config/opencode"},
	{ID: model.AgentCursor, Name: "Cursor", Tier: model.TierFull, ConfigPath: "~/.cursor"},
}

func AllAgents() []Agent {
	items := make([]Agent, len(allAgents))
	copy(items, allAgents)
	return items
}
