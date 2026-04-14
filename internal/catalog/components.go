package catalog

import "orquestador-auditor/internal/model"

type Component struct {
	ID          model.ComponentID
	Name        string
	Description string
}

var components = []Component{
	{ID: model.ComponentAssets, Name: "Assets", Description: "Embedded command and prompt assets"},
	{ID: model.ComponentMCP, Name: "MCP", Description: "Model Context Protocol injection"},
	{ID: model.ComponentPrompts, Name: "Prompts", Description: "System prompt injection"},
	{ID: model.ComponentCommands, Name: "Commands", Description: "Slash command injection"},
	{ID: model.ComponentSkills, Name: "Skills", Description: "Security analysis skill packs"},
	{ID: model.ComponentOutputStyles, Name: "Output Styles", Description: "Structured reporting styles for clients that support them"},
	{ID: model.ComponentSubAgents, Name: "Subagents", Description: "Security-specialized delegated agents"},
	{ID: model.ComponentBackup, Name: "Backup", Description: "Config backup and restore before sync operations"},
	{ID: model.ComponentVerify, Name: "Verify", Description: "Post-sync integrity verification"},
	{ID: model.ComponentSystem, Name: "System", Description: "Platform detection and runtime guards"},
}

func AllComponents() []Component {
	items := make([]Component, len(components))
	copy(items, components)
	return items
}
