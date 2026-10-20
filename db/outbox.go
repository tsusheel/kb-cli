package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/tsusheel/kb-cli/models"
)

// GetPendingOutboxChanges retrieves unacknowledged changes in sequential order.
func GetPendingOutboxChanges(limit int) ([]models.OutboxChange, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	query := `SELECT sequence, change_id, plugin_id, entity_type, entity_id, operation,
		base_version, entity_version, schema_version, payload_json, created_at, attempts,
		last_error, acknowledged_at
		FROM sync_outbox
		WHERE acknowledged_at IS NULL
		ORDER BY sequence ASC
		LIMIT ?`

	rows, err := DB.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed querying pending outbox changes: %w", err)
	}
	defer rows.Close()

	var changes []models.OutboxChange
	for rows.Next() {
		var c models.OutboxChange
		var lastErr sql.NullString
		var ackAt sql.NullTime

		err := rows.Scan(
			&c.Sequence, &c.ChangeID, &c.PluginID, &c.EntityType, &c.EntityID, &c.Operation,
			&c.BaseVersion, &c.EntityVersion, &c.SchemaVersion, &c.PayloadJSON, &c.CreatedAt,
			&c.Attempts, &lastErr, &ackAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed scanning outbox change: %w", err)
		}

		if lastErr.Valid {
			c.LastError = &lastErr.String
		}
		if ackAt.Valid {
			c.AcknowledgedAt = &ackAt.Time
		}

		changes = append(changes, c)
	}

	return changes, rows.Err()
}

// AcknowledgeOutboxChanges marks a batch of changes as successfully accepted by the remote sync authority.
func AcknowledgeOutboxChanges(changeIDs []string) error {
	if len(changeIDs) == 0 {
		return nil
	}

	now := time.Now().UTC()
	placeholders := make([]string, len(changeIDs))
	args := make([]any, len(changeIDs)+1)
	args[0] = now

	for i, id := range changeIDs {
		placeholders[i] = "?"
		args[i+1] = id
	}

	query := fmt.Sprintf(`UPDATE sync_outbox SET acknowledged_at = ? WHERE change_id IN (%s)`, strings.Join(placeholders, ","))
	_, err := DB.Exec(query, args...)
	return err
}

// RecordOutboxFailure increments the retry attempt count and records the last error.
func RecordOutboxFailure(changeID, errMsg string) error {
	query := `UPDATE sync_outbox SET attempts = attempts + 1, last_error = ? WHERE change_id = ?`
	_, err := DB.Exec(query, errMsg, changeID)
	return err
}

// PruneAcknowledgedOutbox removes changes that have been acknowledged past a retention duration.
func PruneAcknowledgedOutbox(retention time.Duration) (int64, error) {
	threshold := time.Now().UTC().Add(-retention)
	res, err := DB.Exec(`DELETE FROM sync_outbox WHERE acknowledged_at IS NOT NULL AND acknowledged_at < ?`, threshold)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
