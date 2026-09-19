package models

import "time"

type KBStats struct {
	TotalActiveNotes  int
	TotalDeletedNotes int
	NotesByType       map[string]int
	NotesByStatus     map[string]int
	NotesByArea       map[string]int
	TotalDailyLogs    int
	TodayDailyLogs    int
	TotalTags         int
	TopTags           []TagCount
	TopHubNotes       []HubNote
	ConsecutiveStreak int
	LastActivityAt    time.Time
}

type TagCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type HubNote struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	LinkCount int    `json:"link_count"`
}

// KnowledgeMap is a token-efficient (~100 tokens) bird's-eye topology of the knowledge base.
type KnowledgeMap struct {
	TotalActiveNotes  int            `json:"total_active_notes"`
	TotalDailyLogs    int            `json:"total_daily_logs"`
	NotesByArea       map[string]int `json:"notes_by_area"`
	NotesByType       map[string]int `json:"notes_by_type"`
	NotesByStatus     map[string]int `json:"notes_by_status"`
	TopTags           []TagCount     `json:"top_tags"`
	TopHubNotes       []HubNote      `json:"top_hub_notes"`
	ConsecutiveStreak int            `json:"consecutive_streak_days"`
}
