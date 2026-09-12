package db

import (
	"database/sql"
	"time"
)

// GetLastSyncedAt returns the timestamp of the last successful sync for a provider.
func GetLastSyncedAt(provider string) (time.Time, error) {
	query := `SELECT last_synced_at FROM sync_state WHERE provider = ?`
	row := DB.QueryRow(query, provider)

	var lastSynced sql.NullTime
	if err := row.Scan(&lastSynced); err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, nil // never synced
		}
		return time.Time{}, err
	}

	if lastSynced.Valid {
		return lastSynced.Time, nil
	}
	return time.Time{}, nil
}

// SetLastSyncedAt updates or creates the sync timestamp for a provider.
func SetLastSyncedAt(provider string, syncedAt time.Time) error {
	now := time.Now()
	query := `
		INSERT INTO sync_state (provider, last_synced_at, created_at)
		VALUES (?, ?, ?)
		ON CONFLICT(provider) DO UPDATE SET last_synced_at = excluded.last_synced_at
	`
	_, err := DB.Exec(query, provider, syncedAt, now)
	return err
}

// GetUnsyncedCounts returns counts of local notes and logs modified since a given timestamp.
func GetUnsyncedCounts(since time.Time) (int, int, error) {
	var notesCount int
	var logsCount int

	if since.IsZero() {
		// All rows
		if err := DB.QueryRow(`SELECT COUNT(*) FROM notes`).Scan(&notesCount); err != nil {
			return 0, 0, err
		}
		if err := DB.QueryRow(`SELECT COUNT(*) FROM daily_logs`).Scan(&logsCount); err != nil {
			return 0, 0, err
		}
		return notesCount, logsCount, nil
	}

	noteQuery := `SELECT COUNT(*) FROM notes WHERE updated_at > ? OR (deleted_at IS NOT NULL AND deleted_at > ?)`
	if err := DB.QueryRow(noteQuery, since, since).Scan(&notesCount); err != nil {
		return 0, 0, err
	}

	logQuery := `SELECT COUNT(*) FROM daily_logs WHERE created_at > ? OR (deleted_at IS NOT NULL AND deleted_at > ?)`
	if err := DB.QueryRow(logQuery, since, since).Scan(&logsCount); err != nil {
		return 0, 0, err
	}

	return notesCount, logsCount, nil
}
