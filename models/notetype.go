package models

type NoteType string

const (
	DefaultNote NoteType = "note"
	Idea       NoteType = "idea"
	Person     NoteType = "person"
	Concept    NoteType = "concept"
	Project    NoteType = "project"
	TIL        NoteType = "til"
	Resource   NoteType = "resource"
	Question   NoteType = "question"
	Experiment NoteType = "experiment"
	Decision   NoteType = "decision"
)
