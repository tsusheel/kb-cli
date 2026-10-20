package plugins

import (
	"fmt"
	"sync"

	"github.com/tsusheel/kb-cli/plugins/api"
)

type Registry struct {
	mu      sync.RWMutex
	plugins map[string]api.Plugin
}

var defaultRegistry = &Registry{
	plugins: make(map[string]api.Plugin),
}

// Register registers a plugin into the global registry.
func Register(p api.Plugin) error {
	return defaultRegistry.Register(p)
}

// Get retrieves a registered plugin by ID.
func Get(id string) (api.Plugin, bool) {
	return defaultRegistry.Get(id)
}

// List returns all currently registered plugins.
func List() []api.Plugin {
	return defaultRegistry.List()
}

// Unregister removes a plugin from the registry.
func Unregister(id string) {
	defaultRegistry.Unregister(id)
}

func (r *Registry) Register(p api.Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	manifest := p.Manifest()
	if manifest.ID == "" {
		return fmt.Errorf("plugin manifest ID cannot be empty")
	}

	if _, exists := r.plugins[manifest.ID]; exists {
		return fmt.Errorf("plugin %q is already registered", manifest.ID)
	}

	r.plugins[manifest.ID] = p
	return nil
}

func (r *Registry) Get(id string) (api.Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.plugins[id]
	return p, ok
}

func (r *Registry) List() []api.Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []api.Plugin
	for _, p := range r.plugins {
		list = append(list, p)
	}
	return list
}

func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.plugins, id)
}
