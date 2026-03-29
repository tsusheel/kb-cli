package models

type Status string

const (
	Active     Status = "active"
	Raw        Status = "raw"
	Refined    Status = "refined"
	InProgress Status = "in-progress"
	Completed  Status = "completed"
	Archived   Status = "archived"
)
