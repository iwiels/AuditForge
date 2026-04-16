package agents

import (
	"fmt"
	"slices"

	"orquestador-auditor/internal/model"
)

type Registry struct {
	adapters map[model.AgentID]Adapter
}

func NewRegistry(adapters ...Adapter) (*Registry, error) {
	r := &Registry{adapters: map[model.AgentID]Adapter{}}
	for _, adapter := range adapters {
		if err := r.Register(adapter); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Registry) Register(adapter Adapter) error {
	if adapter == nil {
		return fmt.Errorf("adapter is nil")
	}
	if _, exists := r.adapters[adapter.ID()]; exists {
		return fmt.Errorf("duplicate adapter %q", adapter.ID())
	}
	r.adapters[adapter.ID()] = adapter
	return nil
}

func (r *Registry) Get(agent model.AgentID) (Adapter, bool) {
	adapter, ok := r.adapters[agent]
	return adapter, ok
}

func (r *Registry) SupportedAgents() []model.AgentID {
	ids := make([]model.AgentID, 0, len(r.adapters))
	for id := range r.adapters {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func NewDefaultRegistry() (*Registry, error) {
	adapters := []Adapter{
		&ClaudeCodeAdapter{},
		&CursorAdapter{},
		&OpenCodeAdapter{},
		&GeminiAdapter{},
	}
	return NewRegistry(adapters...)
}
