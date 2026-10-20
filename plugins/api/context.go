package api

import (
	"context"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

// ConfigReader provides read-only scoped access to plugin configuration.
type ConfigReader interface {
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetStringSlice(key string) []string
	IsSet(key string) bool
}

// AuditEvent represents a single audit log mutation.
type AuditEvent struct {
	ID             string
	PluginID       string
	EntityType     string
	EntityID       string
	Revision       int64
	Action         string // created, updated, deleted, restored
	ActorType      string // user, plugin, ai, system
	ActorID        string
	RequestID      string
	ChangesSummary string
	DiffJSON       []byte
	CreatedAt      time.Time
}

// AuditRecorder records immutable audit events.
type AuditRecorder interface {
	Record(ctx context.Context, event AuditEvent) error
}

// RevisionInfo gives high-level metadata about an entity snapshot.
type RevisionInfo struct {
	RevisionID    string    `json:"revision_id"`
	Revision      int64     `json:"revision"`
	SchemaVersion int       `json:"schema_version"`
	CreatedAt     time.Time `json:"created_at"`
}

// RevisionStore provides access to point-in-time entity snapshots.
type RevisionStore interface {
	GetRevision(ctx context.Context, pluginID, entityType, entityID string, revision int64) ([]byte, error)
	ListRevisions(ctx context.Context, pluginID, entityType, entityID string, limit int) ([]RevisionInfo, error)
	RestoreRevision(ctx context.Context, pluginID, entityType, entityID string, revision int64, expectedCurrentRevision int64) error
}

// SecretStore provides secure hardware-backed secret storage.
type SecretStore interface {
	Get(key string) (string, error)
	Set(key string, val string) error
	Delete(key string) error
}

// CommandRegistry allows mounting subcommands onto the main kb CLI tree.
type CommandRegistry interface {
	AddCommand(cmd *cobra.Command)
}

// HTTPRegistry allows mounting REST endpoints under /api/plugins/<plugin_id>/*.
type HTTPRegistry interface {
	HandleFunc(pattern string, handler http.HandlerFunc)
}

// MCPToolDef declares a tool callable by AI assistants via the Model Context Protocol.
type MCPToolDef struct {
	Name        string
	Description string
	SchemaJSON  []byte
	Handler     func(ctx context.Context, args []byte) (any, error)
}

// MCPRegistry allows registering AI tools dynamically.
type MCPRegistry interface {
	RegisterTool(tool MCPToolDef) error
}

// PluginContext is injected into plugins during registration.
type PluginContext struct {
	PluginID  string
	Config    ConfigReader
	Store     EntryStore
	Audit     AuditRecorder
	Revisions RevisionStore
	Sync      SyncRegistry
	Secrets   SecretStore
	Commands  CommandRegistry
	HTTP      HTTPRegistry
	MCP       MCPRegistry
}
