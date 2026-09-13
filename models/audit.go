package models

import "time"

type AuditAction string

const (
	ActionCreated  AuditAction = "created"
	ActionUpdated  AuditAction = "updated"
	ActionDeleted  AuditAction = "deleted"
	ActionRestored AuditAction = "restored"
	ActionPromoted AuditAction = "promoted"
)

type AuditEntry struct {
	ID             string      `json:"id"`
	EntityType     string      `json:"entity_type"` // "note" | "daily_log"
	EntityID       string      `json:"entity_id"`
	Action         AuditAction `json:"action"`
	ChangesSummary string      `json:"changes_summary"`
	SnapshotJSON   string      `json:"snapshot_json"`
	CreatedAt      time.Time   `json:"created_at"`
}
