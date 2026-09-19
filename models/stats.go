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
	Name  string
	Count int
}

type HubNote struct {
	ID        string
	Title     string
	LinkCount int
}
