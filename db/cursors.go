package db

import (
	"database/sql"
	"time"
)

// GetSyncCursor retrieves the stored high-water mark / change feed cursor for a remote stream.
func GetSyncCursor(remoteID, streamID string) (string, error) {
	query := `SELECT cursor_value FROM sync_cursors WHERE remote_id = ? AND stream_id = ?`
	row := DB.QueryRow(query, remoteID, streamID)

	var cursorVal sql.NullString
	if err := row.Scan(&cursorVal); err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No cursor established yet (start from beginning)
		}
		return "", err
	}

	if cursorVal.Valid {
		return cursorVal.String, nil
	}
	return "", nil
}

// SetSyncCursor saves or advances the sync cursor for a remote stream.
func SetSyncCursor(remoteID, streamID, cursorValue string) error {
	now := time.Now().UTC()
	query := `INSERT INTO sync_cursors (remote_id, stream_id, cursor_value, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(remote_id, stream_id) DO UPDATE SET
			cursor_value = excluded.cursor_value,
			updated_at = excluded.updated_at`

	_, err := DB.Exec(query, remoteID, streamID, cursorValue, now)
	return err
}
