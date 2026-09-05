package models

import "time"

type Link struct {
	ID          string
	FromNote    string
	ToNote      string
	Type        LinkType
	CreatedAt   time.Time
	DeletedAt   time.Time
	DeletedNote string
}
