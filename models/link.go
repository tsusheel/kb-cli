package models

import "time"

type Link struct {
	ID          string    `json:"id"`
	FromNote    string    `json:"from_note"`
	ToNote      string    `json:"to_note"`
	Type        LinkType  `json:"type"`
	CreatedAt   time.Time `json:"created_at"`
	DeletedAt   time.Time `json:"deleted_at,omitempty"`
	DeletedNote string    `json:"deleted_note,omitempty"`
}

type GraphRelation struct {
	ID                 string   `json:"id"`
	Direction          string   `json:"direction"` // "incoming" | "outgoing"
	RelationType       LinkType `json:"relation_type"`
	ConnectedNoteID    string   `json:"connected_note_id"`
	ConnectedShortID   string   `json:"connected_short_id"`
	ConnectedNoteTitle string   `json:"connected_note_title"`
	ConnectedNoteType  NoteType `json:"connected_note_type"`
}

type TagCluster struct {
	TagName   string `json:"tag_name"`
	NoteCount int    `json:"note_count"`
}

type GraphNeighborhood struct {
	FocalNote         *Note           `json:"focal_note"`
	FocalTags         []string        `json:"focal_tags,omitempty"`
	Relations         []GraphRelation `json:"relations"`
	SharedTagClusters []TagCluster    `json:"shared_tag_clusters"`
}

type LinkCandidate struct {
	TargetID          string   `json:"target_id"`
	TargetShortID     string   `json:"target_short_id"`
	TargetTitle       string   `json:"target_title"`
	TargetType        NoteType `json:"target_type"`
	TargetArea        Area     `json:"target_area,omitempty"`
	SharedTags        []string `json:"shared_tags,omitempty"`
	SuggestedRelation LinkType `json:"suggested_relation"`
	ConfidenceScore   float64  `json:"confidence_score"`
	MatchReason       string   `json:"match_reason"`
}
