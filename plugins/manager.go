package plugins

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/plugins/api"
)

type Manager struct {
	mu           sync.RWMutex
	rootCmd      *cobra.Command
	httpMux      *http.ServeMux
	syncRegistry api.SyncRegistry
	running      map[string]api.Plugin
}

var defaultManager *Manager

// InitManager initializes the global plugin manager.
func InitManager(rootCmd *cobra.Command, mux *http.ServeMux, syncRegistry api.SyncRegistry) *Manager {
	defaultManager = &Manager{
		rootCmd:      rootCmd,
		httpMux:      mux,
		syncRegistry: syncRegistry,
		running:      make(map[string]api.Plugin),
	}
	return defaultManager
}

// GetManager returns the global plugin manager.
func GetManager() *Manager {
	return defaultManager
}

// StartEnabledPlugins discovers and initializes all enabled plugins.
func (m *Manager) StartEnabledPlugins(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	records, err := db.ListPluginRecords()
	if err != nil {
		return fmt.Errorf("failed listing plugin records: %w", err)
	}

	for _, rec := range records {
		if !rec.Enabled {
			continue
		}

		plugin, ok := Get(rec.PluginID)
		if !ok {
			// External plugin loading handled via loader package
			continue
		}

		pCtx := NewPluginContext(rec.PluginID, m.rootCmd, m.httpMux, m.syncRegistry)

		if err := plugin.Register(pCtx); err != nil {
			return fmt.Errorf("failed registering plugin %q: %w", rec.PluginID, err)
		}

		if err := plugin.Start(ctx); err != nil {
			return fmt.Errorf("failed starting plugin %q: %w", rec.PluginID, err)
		}

		m.running[rec.PluginID] = plugin
	}

	return nil
}

// EnablePlugin enables a plugin, grants declared capabilities if approved, and starts it.
func (m *Manager) EnablePlugin(ctx context.Context, pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, err := db.GetPluginRecord(pluginID)
	if err != nil {
		return err
	}

	plugin, ok := Get(pluginID)
	if !ok {
		return fmt.Errorf("plugin %q not found in registry", pluginID)
	}

	if err := db.SetPluginEnabled(pluginID, true); err != nil {
		return err
	}

	pCtx := NewPluginContext(pluginID, m.rootCmd, m.httpMux, m.syncRegistry)

	if err := plugin.Register(pCtx); err != nil {
		_ = db.SetPluginEnabled(pluginID, false)
		return fmt.Errorf("failed registering plugin %q: %w", pluginID, err)
	}

	if err := plugin.Start(ctx); err != nil {
		_ = db.SetPluginEnabled(pluginID, false)
		return fmt.Errorf("failed starting plugin %q: %w", pluginID, err)
	}

	m.running[pluginID] = plugin
	rec.Enabled = true
	return nil
}

// DisablePlugin stops an active plugin and marks it disabled.
func (m *Manager) DisablePlugin(ctx context.Context, pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if plugin, active := m.running[pluginID]; active {
		_ = plugin.Stop(ctx)
		delete(m.running, pluginID)
	}

	return db.SetPluginEnabled(pluginID, false)
}

// InstallBuiltin registers and records a compiled-in plugin.
func (m *Manager) InstallBuiltin(p api.Plugin, autoEnable bool) error {
	manifest := p.Manifest()
	if err := ValidateManifestCapabilities(manifest); err != nil {
		return err
	}

	if err := Register(p); err != nil {
		return err
	}

	rec := &models.PluginRecord{
		PluginID:      manifest.ID,
		Version:       manifest.Version,
		APIVersion:    manifest.APIVersion,
		SchemaVersion: manifest.SchemaVersion,
		Enabled:       autoEnable,
		Source:        "builtin",
		ManifestJSON:  fmt.Sprintf(`{"id":%q,"name":%q,"version":%q}`, manifest.ID, manifest.Name, manifest.Version),
	}

	if err := db.RegisterPluginRecord(rec); err != nil {
		return err
	}

	// Grant declared capabilities for built-in plugins
	for _, cap := range manifest.Capabilities {
		_ = GrantCapability(manifest.ID, cap)
	}

	return nil
}
