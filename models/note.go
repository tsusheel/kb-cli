package models

import "time"

type Note struct {
	ID      string
	Title   string
	Content string
	Type    NoteType

	Status string // raw, refined, in-progress, completed, archived
	Area   Area

	CreatedAt time.Time
	UpdatedAt time.Time

	// Optional metadata
	Importance int // 1-5
	Clarity    int // 1-5
	Source     string
}
