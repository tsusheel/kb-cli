package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tsusheel/kb-cli/models"
)

func AddLink(fromID string, toID string, linkType models.LinkType) error {
	fullFromID, err := ResolveID(fromID)
	if err != nil {
		return err
	}

	fullToID, err := ResolveID(toID)
	if err != nil {
		return err
	}

	linkID := uuid.New().String()
	now := time.Now()
	query := `INSERT INTO links (id, from_note, to_note, type, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err = DB.Exec(query, linkID, fullFromID, fullToID, linkType, now)
	return err
}

func SoftDeleteLink(linkID string, reason string) error {
	if reason == "" {
		reason = "deleted"
	}
	now := time.Now()
	query := `UPDATE links SET deleted_at = ?, deleted_note = ? WHERE id = ?`
	_, err := DB.Exec(query, now, reason, linkID)
	return err
}

func RemoveLink(fromID, toID string, reason string) error {
	fullFromID, err := ResolveID(fromID)
	if err != nil {
		return err
	}
	fullToID, err := ResolveID(toID)
	if err != nil {
		return err
	}
	if reason == "" {
		reason = "deleted"
	}
	now := time.Now()
	query := `UPDATE links SET deleted_at = ?, deleted_note = ? WHERE from_note = ? AND to_note = ? AND deleted_at IS NULL`
	res, err := DB.Exec(query, now, reason, fullFromID, fullToID)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		fromDisplay := fullFromID
		if len(fromDisplay) > 7 {
			fromDisplay = fromDisplay[:7]
		}
		toDisplay := fullToID
		if len(toDisplay) > 7 {
			toDisplay = toDisplay[:7]
		}
		return fmt.Errorf("no active link found from [%s] to [%s]", fromDisplay, toDisplay)
	}
	return nil
}

func scanLink(s rowScanner) (*models.Link, error) {
	var l models.Link
	var createdAt sql.NullTime
	var deletedAt sql.NullTime
	var deletedNote sql.NullString
	if err := s.Scan(&l.ID, &l.FromNote, &l.ToNote, &l.Type, &createdAt, &deletedAt, &deletedNote); err != nil {
		return nil, err
	}
	if createdAt.Valid {
		l.CreatedAt = createdAt.Time
	}
	if deletedAt.Valid {
		l.DeletedAt = deletedAt.Time
	}
	if deletedNote.Valid {
		l.DeletedNote = deletedNote.String
	}
	return &l, nil
}

func GetLinksForNote(noteID string) ([]models.Link, error) {
	fullNoteID, err := ResolveID(noteID)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, from_note, to_note, type, created_at, deleted_at, deleted_note FROM links WHERE (from_note = ? OR to_note = ?) AND deleted_at IS NULL`
	rows, err := DB.Query(query, fullNoteID, fullNoteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.Link
	for rows.Next() {
		l, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, *l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return links, nil
}

func GetIncomingLinks(noteID string) ([]models.Link, error) {
	fullNoteID, err := ResolveID(noteID)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, from_note, to_note, type, created_at, deleted_at, deleted_note FROM links WHERE to_note = ? AND deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := DB.Query(query, fullNoteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.Link
	for rows.Next() {
		l, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, *l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return links, nil
}

func GetOutgoingLinks(noteID string) ([]models.Link, error) {
	fullNoteID, err := ResolveID(noteID)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, from_note, to_note, type, created_at, deleted_at, deleted_note FROM links WHERE from_note = ? AND deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := DB.Query(query, fullNoteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.Link
	for rows.Next() {
		l, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, *l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return links, nil
}

func GetAllActiveLinks() ([]models.Link, error) {
	query := `SELECT id, from_note, to_note, type, created_at, deleted_at, deleted_note FROM links WHERE deleted_at IS NULL`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.Link
	for rows.Next() {
		l, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, *l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return links, nil
}

// GetGraphNeighborhood returns the focal note, 1-hop incoming and outgoing relations with titles and types, and shared tag clusters.
func GetGraphNeighborhood(noteID string) (*models.GraphNeighborhood, error) {
	fullNoteID, err := ResolveID(noteID)
	if err != nil {
		return nil, err
	}

	focalNote, err := GetNote(fullNoteID)
	if err != nil {
		return nil, err
	}

	focalTags, _ := GetTagsForNote(fullNoteID)
	var focalTagNames []string
	for _, t := range focalTags {
		focalTagNames = append(focalTagNames, t.Name)
	}

	// 1-hop relations (incoming & outgoing)
	relQuery := `
		SELECT 
			l.id, 
			'outgoing' AS direction,
			l.type, 
			n.id, 
			n.note, 
			n.type
		FROM links l
		JOIN notes n ON l.to_note = n.id
		WHERE l.from_note = ? AND l.deleted_at IS NULL AND n.deleted_at IS NULL

		UNION ALL

		SELECT 
			l.id, 
			'incoming' AS direction,
			l.type, 
			n.id, 
			n.note, 
			n.type
		FROM links l
		JOIN notes n ON l.from_note = n.id
		WHERE l.to_note = ? AND l.deleted_at IS NULL AND n.deleted_at IS NULL
	`
	rows, err := DB.Query(relQuery, fullNoteID, fullNoteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []models.GraphRelation
	for rows.Next() {
		var r models.GraphRelation
		var relTypeStr, noteTypeStr string
		if err := rows.Scan(&r.ID, &r.Direction, &relTypeStr, &r.ConnectedNoteID, &r.ConnectedNoteTitle, &noteTypeStr); err == nil {
			r.RelationType = models.LinkType(relTypeStr)
			r.ConnectedNoteType = models.NoteType(noteTypeStr)
			r.ConnectedShortID = r.ConnectedNoteID
			if len(r.ConnectedShortID) > 7 {
				r.ConnectedShortID = r.ConnectedShortID[:7]
			}
			relations = append(relations, r)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Shared tag clusters
	clusterQuery := `
		SELECT t.name, COUNT(DISTINCT nt2.note_id) as cnt
		FROM note_tags nt1
		JOIN tags t ON nt1.tag_id = t.id
		JOIN note_tags nt2 ON nt1.tag_id = nt2.tag_id AND nt2.note_id != ?
		JOIN notes n ON nt2.note_id = n.id AND n.deleted_at IS NULL
		WHERE nt1.note_id = ?
		GROUP BY t.id, t.name
		ORDER BY cnt DESC
	`
	var clusters []models.TagCluster
	clusterRows, err := DB.Query(clusterQuery, fullNoteID, fullNoteID)
	if err == nil {
		defer clusterRows.Close()
		for clusterRows.Next() {
			var tc models.TagCluster
			if err := clusterRows.Scan(&tc.TagName, &tc.NoteCount); err == nil {
				clusters = append(clusters, tc)
			}
		}
		_ = clusterRows.Err()
	}

	return &models.GraphNeighborhood{
		FocalNote:         focalNote,
		FocalTags:         focalTagNames,
		Relations:         relations,
		SharedTagClusters: clusters,
	}, nil
}
