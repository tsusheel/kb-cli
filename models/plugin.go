package models

import "time"

// PluginEntry represents a strongly-typed record in the universal storage envelope.
type PluginEntry struct {
	ID            string     `json:"id"`
	PluginID      string     `json:"plugin_id"`
	EntityType    string     `json:"entity_type"`
	SchemaVersion int        `json:"schema_version"`
	ParentID      *string    `json:"parent_id,omitempty"`
	Status        string     `json:"status"`
	OccurredAt    *time.Time `json:"occurred_at,omitempty"`
	EntryDate     *string    `json:"entry_date,omitempty"` // YYYY-MM-DD
	NumVal        float64    `json:"num_val"`
	SearchText    string     `json:"search_text"`
	Data          string     `json:"data"` // Validated JSON payload
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
	DeletedNote   *string    `json:"deleted_note,omitempty"`
	Revision      int64      `json:"revision"`
}

// EntryFilter provides multi-dimensional querying across universal plugin entries.
type EntryFilter struct {
	PluginID   string     `json:"plugin_id"`
	EntityType string     `json:"entity_type,omitempty"`
	ParentID   *string    `json:"parent_id,omitempty"`
	Status     string     `json:"status,omitempty"`
	EntryDate  *string    `json:"entry_date,omitempty"` // Specific date YYYY-MM-DD
	FromDate   *time.Time `json:"from_date,omitempty"`
	ToDate     *time.Time `json:"to_date,omitempty"`
	SearchText string     `json:"search_text,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
}

// ActorContext records who initiated an action for auditing and attribution.
type ActorContext struct {
	ActorType string `json:"actor_type"` // "user", "plugin", "ai", "system"
	ActorID   string `json:"actor_id,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// DefaultUserActor returns a default user actor context.
func DefaultUserActor() ActorContext {
	return ActorContext{
		ActorType: "user",
		ActorID:   "local_user",
	}
}

// DefaultAIActor returns an AI actor context.
func DefaultAIActor(agentName string) ActorContext {
	if agentName == "" {
		agentName = "mcp_agent"
	}
	return ActorContext{
		ActorType: "ai",
		ActorID:   agentName,
	}
}

// EntityRevision captures a point-in-time snapshot of an entity state.
type EntityRevision struct {
	RevisionID    string    `json:"revision_id"`
	PluginID      string    `json:"plugin_id"`
	EntityType    string    `json:"entity_type"`
	EntityID      string    `json:"entity_id"`
	Revision      int64     `json:"revision"`
	SchemaVersion int       `json:"schema_version"`
	SnapshotJSON  string    `json:"snapshot_json"`
	CreatedAt     time.Time `json:"created_at"`
}

// OutboxChange represents an enqueued local mutation waiting to be pushed to PostgreSQL.
type OutboxChange struct {
	Sequence       int64      `json:"sequence"`
	ChangeID       string     `json:"change_id"`
	PluginID       string     `json:"plugin_id"`
	EntityType     string     `json:"entity_type"`
	EntityID       string     `json:"entity_id"`
	Operation      string     `json:"operation"` // "upsert" or "tombstone"
	BaseVersion    int64      `json:"base_version"`
	EntityVersion  int64      `json:"entity_version"`
	SchemaVersion  int        `json:"schema_version"`
	PayloadJSON    string     `json:"payload_json"`
	CreatedAt      time.Time  `json:"created_at"`
	Attempts       int        `json:"attempts"`
	LastError      *string    `json:"last_error,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
}

// PluginRecord represents a registered plugin installation in the database.
type PluginRecord struct {
	PluginID      string    `json:"plugin_id"`
	Version       string    `json:"version"`
	APIVersion    string    `json:"api_version"`
	SchemaVersion int       `json:"schema_version"`
	Enabled       bool      `json:"enabled"`
	Source        string    `json:"source"`
	ManifestJSON  string    `json:"manifest_json"`
	InstalledAt   time.Time `json:"installed_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
