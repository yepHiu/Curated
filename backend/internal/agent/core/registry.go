package core

import (
	"fmt"
	"sort"
	"sync"
)

// Registry holds the unique toolset. It is safe for concurrent read after freeze.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]ToolDefinition
}

func NewRegistry() *Registry {
	return &Registry{tools: map[string]ToolDefinition{}}
}

func (r *Registry) Register(def ToolDefinition) error {
	if r == nil {
		return fmt.Errorf("nil registry")
	}
	name := def.Name
	if name == "" {
		return fmt.Errorf("tool name is required")
	}
	if def.Handler == nil {
		return fmt.Errorf("tool %q is missing a handler", name)
	}
	switch def.Permission {
	case PermissionRead, PermissionWritePreview, PermissionWriteApply:
	default:
		return fmt.Errorf("tool %q has invalid permission %q", name, def.Permission)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %q is already registered", name)
	}
	if def.Version == "" {
		def.Version = ToolsetVersion
	}
	r.tools[name] = def
	return nil
}

func (r *Registry) Get(name string) (ToolDefinition, bool) {
	if r == nil {
		return ToolDefinition{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.tools[name]
	return def, ok
}

func (r *Registry) List() []ToolDefinition {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ToolDefinition, 0, len(r.tools))
	for _, def := range r.tools {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (r *Registry) ListByPermission(allowed map[string]bool) []ToolDefinition {
	all := r.List()
	out := make([]ToolDefinition, 0, len(all))
	for _, def := range all {
		if allowed[def.Permission] {
			out = append(out, def)
		}
	}
	return out
}
