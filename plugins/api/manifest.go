package api

import (
	"context"
	"encoding/json"
)

// Manifest contains the static declaration and metadata for a plugin.
type Manifest struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Version        string          `json:"version"`
	APIVersion     string          `json:"api_version"`
	SchemaVersion  int             `json:"schema_version"`
	MinCoreVersion string          `json:"min_core_version"`
	Capabilities   []string        `json:"capabilities"`
	Dependencies   []Dependency    `json:"dependencies,omitempty"`
	EntryPoint     string          `json:"entry_point,omitempty"`
	ConfigSchema   json.RawMessage `json:"config_schema,omitempty"`
}

// Dependency specifies another plugin required by this plugin.
type Dependency struct {
	PluginID string `json:"plugin_id"`
	Version  string `json:"version"`
}

// Plugin is the primary lifecycle contract for any domain module.
type Plugin interface {
	Manifest() Manifest
	Register(ctx *PluginContext) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Migrator is an optional interface plugins can implement if they own typed schema migrations.
type Migrator interface {
	Migrate(ctx context.Context, from, to int) error
}
