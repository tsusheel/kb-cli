type DailyNote struct {
	ID   string
	Date time.Time
	NoteID string // reference to main note
}

