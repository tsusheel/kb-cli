package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tsusheel/kb-cli/plugins/api"
)

type AdapterRegistry struct {
	mu       sync.RWMutex
	adapters map[string]api.SyncAdapter
}

func NewAdapterRegistry() *AdapterRegistry {
	r := &AdapterRegistry{
		adapters: make(map[string]api.SyncAdapter),
	}
	// Register default universal entries adapter
	_ = r.Register(&UniversalEntriesAdapter{})
	return r
}

func adapterKey(pluginID, entityType string) string {
	return fmt.Sprintf("%s:%s", pluginID, entityType)
}

func (r *AdapterRegistry) Register(adapter api.SyncAdapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := adapterKey(adapter.PluginID(), adapter.EntityType())
	if _, exists := r.adapters[key]; exists {
		return fmt.Errorf("sync adapter already registered for %s", key)
	}
	r.adapters[key] = adapter
	return nil
}

func (r *AdapterRegistry) Get(pluginID, entityType string) (api.SyncAdapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Exact match first
	if a, ok := r.adapters[adapterKey(pluginID, entityType)]; ok {
		return a, true
	}
	// Fallback to wildcard plugin match (e.g., "*:entry")
	if a, ok := r.adapters[adapterKey("*", entityType)]; ok {
		return a, true
	}
	// Universal default fallback
	if a, ok := r.adapters[adapterKey("*", "*")]; ok {
		return a, true
	}
	return nil, false
}

func (r *AdapterRegistry) List() []api.SyncAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []api.SyncAdapter
	for _, a := range r.adapters {
		list = append(list, a)
	}
	return list
}

// UniversalEntriesAdapter provides default sync serialization and LWW conflict resolution for universal entries.
type UniversalEntriesAdapter struct{}

func (a *UniversalEntriesAdapter) PluginID() string {
	return "*"
}

func (a *UniversalEntriesAdapter) EntityType() string {
	return "*"
}

func (a *UniversalEntriesAdapter) SchemaVersion() int {
	return 1
}

func (a *UniversalEntriesAdapter) ConflictPolicy() api.ConflictPolicy {
	return api.ConflictLastWriteWins
}

func (a *UniversalEntriesAdapter) Export(ctx context.Context, change api.Change) ([]byte, error) {
	return change.Payload, nil
}

func (a *UniversalEntriesAdapter) Import(ctx context.Context, change api.Change) error {
	if len(change.Payload) == 0 {
		return fmt.Errorf("empty payload in sync change")
	}
	var js json.RawMessage
	if err := json.Unmarshal(change.Payload, &js); err != nil {
		return fmt.Errorf("invalid json payload: %w", err)
	}
	return nil
}

func (a *UniversalEntriesAdapter) Validate(ctx context.Context, change api.Change) error {
	if change.PluginID == "" || change.EntityID == "" {
		return fmt.Errorf("change plugin_id and entity_id must not be empty")
	}
	return nil
}

func (a *UniversalEntriesAdapter) Merge(ctx context.Context, local, remote api.Change) (api.Change, error) {
	// Last-Write-Wins: higher version takes precedence
	if remote.EntityVersion >= local.EntityVersion {
		return remote, nil
	}
	return local, nil
}
