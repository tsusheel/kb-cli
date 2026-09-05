package models

import "time"

type Tag struct {
	ID        string
	Name      string // e.g. "finance/investing"
	CreatedAt time.Time
}
