package models

import "time"

type NoteTag struct {
	NoteID    string
	TagID     string
	CreatedAt time.Time
}

