package db

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tsusheel/kb-cli/models"
)


var ErrNotFound = errors.New("note not found")
var ErrAmbiguous = errors.New("ambiguous short id, multiple notes found")

func CreateNote(n *models.Note) error {
	if n.ID == "" {
		n.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	} else {
		n.ID = strings.ReplaceAll(n.ID, "-", "")
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	if n.UpdatedAt.IsZero() {
		n.UpdatedAt = time.Now()
	}

	var targetDT sql.NullTime
	if !n.TargetDateTime.IsZero() {
		targetDT = sql.NullTime{Time: n.TargetDateTime, Valid: true}
	}

	query := `INSERT INTO notes (
		id, note, note_flesh, type, status, area, importance, clarity, source, target_date_time, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.Exec(query, n.ID, n.Note, n.NoteFlesh, n.Type, n.Status, n.Area, n.Importance, n.Clarity, n.Source, targetDT, n.CreatedAt, n.UpdatedAt)
	if err != nil {
		return err
	}

	ftsQuery := `INSERT INTO notes_fts (note_id, note, note_flesh) VALUES (?, ?, ?)`
	_, err = tx.Exec(ftsQuery, n.ID, n.Note, n.NoteFlesh)
	if err != nil {
		return err
	}

	// Record audit log
	if err := RecordAudit(tx, "note", n.ID, models.ActionCreated, "created note", n); err != nil {
		return err
	}

	return tx.Commit()
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

const noteColumns = "id, note, note_flesh, type, status, area, importance, clarity, source, target_date_time, created_at, updated_at, deleted_at, deleted_note"

func scanNote(s rowScanner) (*models.Note, error) {
	var n models.Note
	var targetDT sql.NullTime
	var deletedDT sql.NullTime
	var deletedNote sql.NullString

	err := s.Scan(&n.ID, &n.Note, &n.NoteFlesh, &n.Type, &n.Status, &n.Area, &n.Importance, &n.Clarity, &n.Source, &targetDT, &n.CreatedAt, &n.UpdatedAt, &deletedDT, &deletedNote)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if targetDT.Valid {
		n.TargetDateTime = targetDT.Time
	}
	if deletedDT.Valid {
		n.DeletedAt = deletedDT.Time
	}
	if deletedNote.Valid {
		n.DeletedNote = deletedNote.String
	}
	return &n, nil
}

func prefixedNoteColumns(prefix string) string {
	cols := []string{
		"id", "note", "note_flesh", "type", "status", "area",
		"importance", "clarity", "source", "target_date_time",
		"created_at", "updated_at", "deleted_at", "deleted_note",
	}
	var prefixed []string
	for _, c := range cols {
		prefixed = append(prefixed, prefix+"."+c)
	}
	return strings.Join(prefixed, ", ")
}

func ResolveID(id string) (string, error) {
	cleanID := strings.ReplaceAll(id, "-", "")
	if len(cleanID) == 32 {
		var exists string
		err := DB.QueryRow("SELECT id FROM notes WHERE id = ?", cleanID).Scan(&exists)
		if err == sql.ErrNoRows {
			return "", ErrNotFound
		} else if err != nil {
			return "", err
		}
		return cleanID, nil
	}

	query := `SELECT id FROM notes WHERE id LIKE ?`
	rows, err := DB.Query(query, cleanID+"%")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var matchedID string
	count := 0
	for rows.Next() {
		count++
		if err := rows.Scan(&matchedID); err != nil {
			return "", err
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	if count == 0 {
		return "", ErrNotFound
	}
	if count > 1 {
		return "", ErrAmbiguous
	}

	return matchedID, nil
}

func GetNote(id string) (*models.Note, error) {
	fullID, err := ResolveID(id)
	if err != nil {
		return nil, err
	}

	query := `SELECT ` + noteColumns + ` FROM notes WHERE id = ?`
	row := DB.QueryRow(query, fullID)
	return scanNote(row)
}

func UpdateNote(n *models.Note) error {
	fullID, err := ResolveID(n.ID)
	if err != nil {
		return err
	}
	n.ID = fullID

	oldNote, err := GetNote(n.ID)
	var diffSummary string
	if err == nil {
		diffSummary = ComputeNoteDiffSummary(oldNote, n)
	} else {
		diffSummary = "updated note"
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	n.UpdatedAt = time.Now()

	var targetDT sql.NullTime
	if !n.TargetDateTime.IsZero() {
		targetDT = sql.NullTime{Time: n.TargetDateTime, Valid: true}
	}

	query := `UPDATE notes SET 
		note = ?, 
		note_flesh = ?, 
		type = ?, 
		status = ?, 
		area = ?, 
		importance = ?, 
		clarity = ?, 
		source = ?, 
		target_date_time = ?, 
		updated_at = ?
	WHERE id = ?`

	_, err = tx.Exec(query, n.Note, n.NoteFlesh, n.Type, n.Status, n.Area, n.Importance, n.Clarity, n.Source, targetDT, n.UpdatedAt, n.ID)
	if err != nil {
		return err
	}

	// Update FTS table
	_, err = tx.Exec(`DELETE FROM notes_fts WHERE note_id = ?`, n.ID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO notes_fts (note_id, note, note_flesh) VALUES (?, ?, ?)`, n.ID, n.Note, n.NoteFlesh)
	if err != nil {
		return err
	}

	// Record audit log
	if err := RecordAudit(tx, "note", n.ID, models.ActionUpdated, diffSummary, n); err != nil {
		return err
	}

	return tx.Commit()
}

func SoftDeleteNote(id string, reason string) error {
	fullID, err := ResolveID(id)
	if err != nil {
		return err
	}

	if reason == "" {
		reason = "deleted"
	}

	n, _ := GetNote(fullID)

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	query := `UPDATE notes SET deleted_at = ?, deleted_note = ?, updated_at = ? WHERE id = ?`
	_, err = tx.Exec(query, now, reason, now, fullID)
	if err != nil {
		return err
	}

	if n != nil {
		n.DeletedAt = now
		n.DeletedNote = reason
		n.UpdatedAt = now
		if err := RecordAudit(tx, "note", fullID, models.ActionDeleted, fmt.Sprintf("soft-deleted: %s", reason), n); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func RestoreNote(id string) (*models.Note, error) {
	fullID, err := ResolveID(id)
	if err != nil {
		return nil, err
	}

	n, err := GetNote(fullID)
	if err != nil {
		return nil, err
	}

	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now()
	query := `UPDATE notes SET deleted_at = NULL, deleted_note = NULL, updated_at = ? WHERE id = ?`
	if _, err := tx.Exec(query, now, fullID); err != nil {
		return nil, err
	}

	n.DeletedAt = time.Time{}
	n.DeletedNote = ""
	n.UpdatedAt = now
	if err := RecordAudit(tx, "note", fullID, models.ActionRestored, "restored note", n); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return n, nil
}

func ListNotes(filterType string) ([]models.Note, error) {
	return ListNotesExtended(filterType, "", "", false)
}

func ListNotesExtended(filterType, filterStatus, filterArea string, includeDeleted bool) ([]models.Note, error) {
	var whereClauses []string
	var args []interface{}

	if !includeDeleted {
		whereClauses = append(whereClauses, "deleted_at IS NULL")
	}
	if filterType != "" {
		whereClauses = append(whereClauses, "type = ?")
		args = append(args, filterType)
	}
	if filterStatus != "" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, filterStatus)
	}
	if filterArea != "" {
		whereClauses = append(whereClauses, "area = ?")
		args = append(args, filterArea)
	}

	query := "SELECT " + noteColumns + " FROM notes"
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	query += " ORDER BY updated_at DESC"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, *n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notes, nil
}

// SanitizeFTS5Query escapes and formats search tokens into safe SQLite FTS5 expressions.
func SanitizeFTS5Query(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	// If query is an explicit phrase enclosed in quotes, sanitize and preserve phrase matching
	if strings.HasPrefix(query, `"`) && strings.HasSuffix(query, `"`) && len(query) >= 2 {
		inner := strings.Trim(query, `"`)
		inner = strings.ReplaceAll(inner, `"`, `""`)
		if strings.TrimSpace(inner) == "" {
			return ""
		}
		return fmt.Sprintf(`"%s"`, inner)
	}

	words := strings.Fields(query)
	if len(words) == 0 {
		return ""
	}

	var tokens []string
	for _, w := range words {
		escaped := strings.ReplaceAll(w, `"`, `""`)
		escaped = strings.Trim(escaped, `*^:+-/()`)
		if escaped == "" {
			continue
		}
		tokens = append(tokens, fmt.Sprintf(`"%s"*`, escaped))
	}

	if len(tokens) == 0 {
		return ""
	}
	return strings.Join(tokens, " ")
}

func SearchNotesExtended(searchTerm, filterType, filterStatus, filterArea string) ([]models.Note, error) {
	sanitizedQuery := SanitizeFTS5Query(searchTerm)
	if sanitizedQuery == "" {
		// If search term has no valid alphanumeric tokens, return empty list
		return []models.Note{}, nil
	}

	whereClauses := []string{"notes_fts MATCH ?", "n.deleted_at IS NULL"}
	args := []interface{}{sanitizedQuery}

	if filterType != "" {
		whereClauses = append(whereClauses, "n.type = ?")
		args = append(args, filterType)
	}
	if filterStatus != "" {
		whereClauses = append(whereClauses, "n.status = ?")
		args = append(args, filterStatus)
	}
	if filterArea != "" {
		whereClauses = append(whereClauses, "n.area = ?")
		args = append(args, filterArea)
	}

	query := fmt.Sprintf(`
		SELECT %s 
		FROM notes_fts fts
		JOIN notes n ON n.id = fts.note_id
		WHERE %s
		ORDER BY rank
	`, prefixedNoteColumns("n"), strings.Join(whereClauses, " AND "))

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, *n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notes, nil
}

func SearchNotes(searchTerm string) ([]models.Note, error) {
	return SearchNotesExtended(searchTerm, "", "", "")
}

// GetRawNotes returns raw unrefined notes for today (todayOnly=true) or all time (todayOnly=false).
func GetRawNotes(todayOnly bool) ([]models.Note, error) {
	var query string
	var args []interface{}

	if todayOnly {
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfDay := startOfDay.AddDate(0, 0, 1)

		query = fmt.Sprintf("SELECT %s FROM notes WHERE status = 'raw' AND deleted_at IS NULL AND created_at >= ? AND created_at < ? ORDER BY updated_at DESC", noteColumns)
		args = []interface{}{startOfDay, endOfDay}
	} else {
		query = fmt.Sprintf("SELECT %s FROM notes WHERE status = 'raw' AND deleted_at IS NULL ORDER BY updated_at DESC", noteColumns)
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, *n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notes, nil
}

// GetOrphanNotes returns active notes that have no associated tags and no incoming or outgoing links.
func GetOrphanNotes() ([]models.Note, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM notes n
		WHERE n.deleted_at IS NULL
		  AND n.id NOT IN (SELECT note_id FROM note_tags)
		  AND n.id NOT IN (SELECT from_note FROM links WHERE deleted_at IS NULL)
		  AND n.id NOT IN (SELECT to_note FROM links WHERE deleted_at IS NULL)
		ORDER BY n.updated_at DESC
	`, prefixedNoteColumns("n"))

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, *n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notes, nil
}

type candidateScore struct {
	note       models.Note
	score      float64
	reasons    []string
	sharedTags []string
}

// SuggestLinkCandidates analyzes note content, tags, area, and graph topology to propose ranked link candidates.
func SuggestLinkCandidates(noteID, text string, limit int) ([]models.LinkCandidate, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	alreadyLinked := make(map[string]bool)
	var focalNote *models.Note
	var focalTags []models.Tag

	if noteID != "" {
		fullID, err := ResolveID(noteID)
		if err == nil {
			alreadyLinked[fullID] = true
			if n, err := GetNote(fullID); err == nil {
				focalNote = n
			}
			focalTags, _ = GetTagsForNote(fullID)
			existingLinks, _ := GetLinksForNote(fullID)
			for _, l := range existingLinks {
				alreadyLinked[l.FromNote] = true
				alreadyLinked[l.ToNote] = true
			}
		}
	}

	analysisText := strings.TrimSpace(text)
	if analysisText == "" && focalNote != nil {
		analysisText = focalNote.Note + " " + focalNote.NoteFlesh
	}

	candidateMap := make(map[string]*candidateScore)

	getOrCreateCandidate := func(n models.Note) *candidateScore {
		if cs, ok := candidateMap[n.ID]; ok {
			return cs
		}
		cs := &candidateScore{
			note:  n,
			score: 0.10, // baseline active presence
		}
		candidateMap[n.ID] = cs
		return cs
	}

	// 1. FTS5 Lexical Search with extracted keywords
	if analysisText != "" {
		keywords := extractKeywords(analysisText, 8)
		if keywords != "" {
			matchedNotes, err := SearchNotes(keywords)
			if err == nil {
				for rankIdx, mn := range matchedNotes {
					if alreadyLinked[mn.ID] {
						continue
					}
					cs := getOrCreateCandidate(mn)
					rankBoost := 0.40 - float64(rankIdx)*0.03
					if rankBoost < 0.15 {
						rankBoost = 0.15
					}
					cs.score += rankBoost
					cs.reasons = append(cs.reasons, "matched keywords in content")
				}
			}
		}
	}

	// 2. Tag overlap signal
	if len(focalTags) > 0 {
		for _, ft := range focalTags {
			query := `
				SELECT ` + prefixedNoteColumns("n") + `
				FROM note_tags nt
				JOIN tags t ON nt.tag_id = t.id
				JOIN notes n ON nt.note_id = n.id AND n.deleted_at IS NULL
				WHERE t.name = ?
			`
			rows, err := DB.Query(query, ft.Name)
			if err == nil {
				for rows.Next() {
					n, err := scanNote(rows)
					if err == nil && !alreadyLinked[n.ID] {
						cs := getOrCreateCandidate(*n)
						cs.score += 0.30
						cs.sharedTags = append(cs.sharedTags, ft.Name)
						cs.reasons = append(cs.reasons, fmt.Sprintf("shares tag #%s", ft.Name))
					}
				}
				rows.Close()
			}
		}
	}

	// 3. Same Area boost
	if focalNote != nil && focalNote.Area != "" {
		areaNotes, err := ListNotesExtended("", "", string(focalNote.Area), false)
		if err == nil {
			for _, an := range areaNotes {
				if !alreadyLinked[an.ID] {
					if cs, exists := candidateMap[an.ID]; exists {
						cs.score += 0.15
						cs.reasons = append(cs.reasons, fmt.Sprintf("in same area '%s'", focalNote.Area))
					}
				}
			}
		}
	}

	// 4. Hub Note Centrality signal
	stats, err := GetKnowledgeBaseStats()
	if err == nil {
		for _, hn := range stats.TopHubNotes {
			if !alreadyLinked[hn.ID] {
				if cs, exists := candidateMap[hn.ID]; exists {
					cs.score += 0.10
					cs.reasons = append(cs.reasons, "frequently referenced hub note")
				}
			}
		}
	}

	// If candidateMap is still small/empty and we have analysis text, backfill with recent active notes
	if len(candidateMap) == 0 {
		recentNotes, _ := ListNotesExtended("", "", "", false)
		for _, rn := range recentNotes {
			if !alreadyLinked[rn.ID] {
				cs := getOrCreateCandidate(rn)
				cs.reasons = append(cs.reasons, "recent active note")
				if len(candidateMap) >= limit {
					break
				}
			}
		}
	}

	// Format results
	var candidates []models.LinkCandidate
	for _, cs := range candidateMap {
		score := cs.score
		if score > 0.99 {
			score = 0.99
		}
		if score < 0.10 {
			score = 0.10
		}
		score = math.Round(score*100) / 100

		shortID := cs.note.ID
		if len(shortID) > 7 {
			shortID = shortID[:7]
		}

		// Suggest appropriate relation type
		suggestedType := models.RelatedTo
		if focalNote != nil {
			if focalNote.Type == models.Todo && cs.note.Type == models.Project {
				suggestedType = models.PartOf
			} else if focalNote.Type == models.Project && cs.note.Type == models.Todo {
				suggestedType = models.PartOf
			} else if focalNote.Type == models.Decision || cs.note.Type == models.Decision {
				suggestedType = models.Supports
			}
		}

		uniqueReasons := deduplicateStrings(cs.reasons)
		reasonText := strings.Join(uniqueReasons, "; ")
		if reasonText == "" {
			reasonText = "relevant topic association"
		}

		candidates = append(candidates, models.LinkCandidate{
			TargetID:          cs.note.ID,
			TargetShortID:     shortID,
			TargetTitle:       cs.note.Note,
			TargetType:        cs.note.Type,
			TargetArea:        cs.note.Area,
			SharedTags:        deduplicateStrings(cs.sharedTags),
			SuggestedRelation: suggestedType,
			ConfidenceScore:   score,
			MatchReason:       reasonText,
		})
	}

	// Sort by confidence score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ConfidenceScore > candidates[j].ConfidenceScore
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return candidates, nil
}

func extractKeywords(text string, maxWords int) string {
	words := strings.Fields(text)
	var filtered []string
	stopwords := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "this": true,
		"that": true, "from": true, "have": true, "what": true, "about": true,
		"http": true, "https": true, "true": true, "false": true, "null": true,
		"note": true, "task": true, "todo": true, "will": true, "been": true,
		"when": true, "where": true, "which": true, "there": true, "their": true,
	}
	for _, w := range words {
		cleaned := strings.ToLower(strings.Trim(w, `.,!?:;'"()[]{}<>-/*#`))
		if len(cleaned) >= 3 && !stopwords[cleaned] {
			filtered = append(filtered, cleaned)
			if len(filtered) >= maxWords {
				break
			}
		}
	}
	return strings.Join(filtered, " ")
}

func deduplicateStrings(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
