package api

import (
	"context"
	"time"
)

// Entry represents an item in the universal storage envelope.
type Entry struct {
	ID            string
	PluginID      string
	EntityType    string
	SchemaVersion int
	ParentID      *string

	Status     string
	OccurredAt *time.Time
	EntryDate  *string

	NumVal     float64
	SearchText string
	Data       []byte // JSON, validated by plugin

	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	DeletedNote *string

	Revision int64
}

// EntryFilter contains parameters for querying universal entries.
type EntryFilter struct {
	PluginID   string
	EntityType string
	ParentID   *string
	FromDate   *time.Time
	ToDate     *time.Time
	EntryDate  *string
	Status     string
	SearchText string
	Limit      int
	Offset     int
}

// EntryStore exposes universal persistence capabilities to plugins.
type EntryStore interface {
	Create(ctx context.Context, entry Entry) error
	Get(ctx context.Context, pluginID, id string) (Entry, error)
	Update(ctx context.Context, entry Entry, expectedRevision int64) error
	Query(ctx context.Context, filter EntryFilter) ([]Entry, error)
	SoftDelete(ctx context.Context, pluginID, id, reason string) error
	Restore(ctx context.Context, pluginID, id string, expectedRevision int64) error
}
