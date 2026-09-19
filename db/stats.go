package db

import (
	"time"

	"github.com/tsusheel/kb-cli/models"
)

// GetKnowledgeBaseStats computes summary statistics across notes, logs, tags, links, and activity streaks.
func GetKnowledgeBaseStats() (*models.KBStats, error) {
	stats := &models.KBStats{
		NotesByType:   make(map[string]int),
		NotesByStatus: make(map[string]int),
		NotesByArea:   make(map[string]int),
	}

	// 1. Total active and deleted notes
	err := DB.QueryRow("SELECT COUNT(*) FROM notes WHERE deleted_at IS NULL").Scan(&stats.TotalActiveNotes)
	if err != nil {
		return nil, err
	}
	_ = DB.QueryRow("SELECT COUNT(*) FROM notes WHERE deleted_at IS NOT NULL").Scan(&stats.TotalDeletedNotes)

	// 2. Notes by Type
	typeRows, err := DB.Query("SELECT type, COUNT(*) FROM notes WHERE deleted_at IS NULL GROUP BY type ORDER BY COUNT(*) DESC")
	if err == nil {
		defer typeRows.Close()
		for typeRows.Next() {
			var t string
			var c int
			if err := typeRows.Scan(&t, &c); err == nil {
				stats.NotesByType[t] = c
			}
		}
		_ = typeRows.Err()
	}

	// 3. Notes by Status
	statusRows, err := DB.Query("SELECT status, COUNT(*) FROM notes WHERE deleted_at IS NULL GROUP BY status ORDER BY COUNT(*) DESC")
	if err == nil {
		defer statusRows.Close()
		for statusRows.Next() {
			var s string
			var c int
			if err := statusRows.Scan(&s, &c); err == nil {
				stats.NotesByStatus[s] = c
			}
		}
		_ = statusRows.Err()
	}

	// 4. Notes by Area
	areaRows, err := DB.Query("SELECT area, COUNT(*) FROM notes WHERE deleted_at IS NULL AND area != '' GROUP BY area ORDER BY COUNT(*) DESC")
	if err == nil {
		defer areaRows.Close()
		for areaRows.Next() {
			var a string
			var c int
			if err := areaRows.Scan(&a, &c); err == nil {
				stats.NotesByArea[a] = c
			}
		}
		_ = areaRows.Err()
	}

	// 5. Daily logs total and today
	_ = DB.QueryRow("SELECT COUNT(*) FROM daily_logs WHERE deleted_at IS NULL").Scan(&stats.TotalDailyLogs)

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1)
	_ = DB.QueryRow("SELECT COUNT(*) FROM daily_logs WHERE deleted_at IS NULL AND created_at >= ? AND created_at < ?", startOfDay, endOfDay).Scan(&stats.TodayDailyLogs)

	// 6. Total tags and top tags
	_ = DB.QueryRow("SELECT COUNT(*) FROM tags").Scan(&stats.TotalTags)
	tagRows, err := DB.Query(`
		SELECT t.name, COUNT(nt.tag_id) as count
		FROM tags t
		JOIN note_tags nt ON t.id = nt.tag_id
		JOIN notes n ON nt.note_id = n.id AND n.deleted_at IS NULL
		GROUP BY t.id, t.name
		ORDER BY count DESC
		LIMIT 5
	`)
	if err == nil {
		defer tagRows.Close()
		for tagRows.Next() {
			var tc models.TagCount
			if err := tagRows.Scan(&tc.Name, &tc.Count); err == nil {
				stats.TopTags = append(stats.TopTags, tc)
			}
		}
		_ = tagRows.Err()
	}

	// 7. Top Hub notes (most linked)
	hubRows, err := DB.Query(`
		SELECT n.id, n.note, COUNT(l.id) as link_count
		FROM notes n
		JOIN links l ON (n.id = l.from_note OR n.id = l.to_note) AND l.deleted_at IS NULL
		WHERE n.deleted_at IS NULL
		GROUP BY n.id, n.note
		ORDER BY link_count DESC
		LIMIT 5
	`)
	if err == nil {
		defer hubRows.Close()
		for hubRows.Next() {
			var hn models.HubNote
			if err := hubRows.Scan(&hn.ID, &hn.Title, &hn.LinkCount); err == nil {
				stats.TopHubNotes = append(stats.TopHubNotes, hn)
			}
		}
		_ = hubRows.Err()
	}

	// 8. Daily Streak calculation (consecutive days with activity)
	stats.ConsecutiveStreak = calculateStreak()

	return stats, nil
}

func calculateStreak() int {
	// Query unique dates with notes created or daily logs recorded
	query := `
		SELECT DISTINCT DATE(created_at) as act_date
		FROM (
			SELECT created_at FROM notes WHERE deleted_at IS NULL
			UNION ALL
			SELECT created_at FROM daily_logs WHERE deleted_at IS NULL
		)
		ORDER BY act_date DESC
	`
	rows, err := DB.Query(query)
	if err != nil {
		return 0
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err == nil {
			if t, err := time.Parse("2006-01-02", dateStr); err == nil {
				dates = append(dates, t)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return 0
	}

	if len(dates) == 0 {
		return 0
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)

	firstDate := dates[0]
	firstNorm := time.Date(firstDate.Year(), firstDate.Month(), firstDate.Day(), 0, 0, 0, 0, time.UTC)

	// Streak must include today or yesterday to be active
	if !firstNorm.Equal(today) && !firstNorm.Equal(yesterday) {
		return 0
	}

	streak := 1
	expected := firstNorm.AddDate(0, 0, -1)

	for i := 1; i < len(dates); i++ {
		currNorm := time.Date(dates[i].Year(), dates[i].Month(), dates[i].Day(), 0, 0, 0, 0, time.UTC)
		if currNorm.Equal(expected) {
			streak++
			expected = expected.AddDate(0, 0, -1)
		} else if currNorm.After(expected) {
			continue // duplicate day
		} else {
			break // gap in streak
		}
	}

	return streak
}

// GetKnowledgeMap returns a lightweight bird's-eye topology of the knowledge base.
func GetKnowledgeMap() (*models.KnowledgeMap, error) {
	stats, err := GetKnowledgeBaseStats()
	if err != nil {
		return nil, err
	}
	return &models.KnowledgeMap{
		TotalActiveNotes:  stats.TotalActiveNotes,
		TotalDailyLogs:    stats.TotalDailyLogs,
		NotesByArea:       stats.NotesByArea,
		NotesByType:       stats.NotesByType,
		NotesByStatus:     stats.NotesByStatus,
		TopTags:           stats.TopTags,
		TopHubNotes:       stats.TopHubNotes,
		ConsecutiveStreak: stats.ConsecutiveStreak,
	}, nil
}
