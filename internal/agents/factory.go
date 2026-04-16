package agents

import (
	"fmt"
	"strings"

	"orquestador-auditor/internal/model"
)

// NewAdapter returns the Adapter implementation for the given agent ID.
func NewAdapter(agent model.AgentID) (Adapter, error) {
	switch agent {
	case model.AgentClaudeCode:
		return &ClaudeCodeAdapter{}, nil
	case model.AgentClaude:
		return &ClaudeAdapter{}, nil
	case model.AgentCursor:
		return &CursorAdapter{}, nil
	case model.AgentOpenCode:
		return &OpenCodeAdapter{}, nil
	case model.AgentGemini:
		return &GeminiAdapter{}, nil
	default:
		return nil, fmt.Errorf("agent %q is not supported", agent)
	}
}

// AgentIDFromString normalizes a raw string into an AgentID.
func AgentIDFromString(s string) model.AgentID {
	return model.AgentID(strings.ToLower(strings.TrimSpace(s)))
}
