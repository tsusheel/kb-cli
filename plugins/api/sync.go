package api

import "context"

type ConflictPolicy string

const (
	ConflictReject        ConflictPolicy = "reject"
	ConflictLastWriteWins ConflictPolicy = "lww"
	ConflictMerge         ConflictPolicy = "merge"
	ConflictImmutable     ConflictPolicy = "immutable"
)

// Change represents a serialized unit of change across distributed replicas.
type Change struct {
	ChangeID      string
	PluginID      string
	EntityType    string
	EntityID      string
	Operation     string // "upsert" or "tombstone"
	BaseVersion   int64
	EntityVersion int64
	SchemaVersion int
	Payload       []byte
}

// SyncAdapter defines how a plugin handles serialization, validation, and domain-specific merge.
type SyncAdapter interface {
	PluginID() string
	EntityType() string
	SchemaVersion() int
	ConflictPolicy() ConflictPolicy

	Export(ctx context.Context, change Change) ([]byte, error)
	Import(ctx context.Context, change Change) error
	Validate(ctx context.Context, change Change) error
	Merge(ctx context.Context, local, remote Change) (Change, error)
}

// SyncRegistry manages registered entity adapters for synchronization.
type SyncRegistry interface {
	Register(adapter SyncAdapter) error
	Get(pluginID, entityType string) (SyncAdapter, bool)
	List() []SyncAdapter
}
