package models

import "time"

type DailyLog struct {
	ID          string
	Content     string
	NoteID      string // Non-empty if promoted to a Note
	CreatedAt   time.Time
	DeletedAt   time.Time
	DeletedNote string
}
