package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/utils"
)

func NewServer() *server.MCPServer {
	s := server.NewMCPServer("kb-knowledge-base", "1.0.0",
		server.WithResourceCapabilities(true, false),
		server.WithPromptCapabilities(false),
	)

	registerTools(s)
	return s
}

func StartServer() error {
	s := NewServer()
	return server.ServeStdio(s)
}

func registerTools(s *server.MCPServer) {
	// 1. list_notes
	listNotesTool := mcp.NewTool("list_notes",
		mcp.WithDescription("List notes with optional filters (by type, status, or area). Returns note summaries."),
		mcp.WithString("type", mcp.Description("Optional note type (e.g., 'todo', 'note', 'project', 'idea', 'decision')")),
		mcp.WithString("status", mcp.Description("Optional status (e.g., 'active', 'raw', 'refined', 'in-progress', 'completed', 'archived')")),
		mcp.WithString("area", mcp.Description("Optional area (e.g., 'work', 'finance', 'personal')")),
		mcp.WithBoolean("include_deleted", mcp.Description("Whether to include soft-deleted notes (default false)")),
	)
	s.AddTool(listNotesTool, handleListNotes)

	// 2. get_note
	getNoteTool := mcp.NewTool("get_note",
		mcp.WithDescription("Get the full details of a specific note by ID (full or short ID), including note body (note_flesh), tags, links, and metadata."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Full UUID or short ID prefix of the note")),
	)
	s.AddTool(getNoteTool, handleGetNote)

	// 3. search_notes
	searchNotesTool := mcp.NewTool("search_notes",
		mcp.WithDescription("Full-text search across note titles and content using SQLite FTS5."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search terms or keywords")),
	)
	s.AddTool(searchNotesTool, handleSearchNotes)

	// 4. create_note
	createNoteTool := mcp.NewTool("create_note",
		mcp.WithDescription("Create a new note, task, project, concept, or idea in the knowledge base."),
		mcp.WithString("note", mcp.Required(), mcp.Description("Title or summary of the note")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Detailed note content or markdown body (note_flesh)")),
		mcp.WithString("type", mcp.Description("Note type: 'note', 'todo', 'project', 'idea', 'person', 'concept', 'til', 'resource', 'question', 'experiment', 'decision' (default 'note')")),
		mcp.WithString("status", mcp.Description("Status: 'active', 'raw', 'refined', 'in-progress', 'completed', 'archived' (default 'active')")),
		mcp.WithString("area", mcp.Description("Area: 'work', 'finance', 'personal' or custom")),
		mcp.WithString("due", mcp.Description("Optional due or target date (e.g. 'today', 'tomorrow', 'monday', '+3d', or '2026-09-10')")),
		mcp.WithNumber("importance", mcp.Description("Importance rating 1-5")),
		mcp.WithNumber("clarity", mcp.Description("Clarity rating 1-5")),
		mcp.WithString("source", mcp.Description("Source of note (e.g. 'AI Agent', 'Web', 'Book')")),
	)
	s.AddTool(createNoteTool, handleCreateNote)

	// 5. update_note
	updateNoteTool := mcp.NewTool("update_note",
		mcp.WithDescription("Update an existing note's fields (title, content, type, status, area, due date, etc.)."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Full UUID or short ID of note to update")),
		mcp.WithString("note", mcp.Description("New title/summary")),
		mcp.WithString("content", mcp.Description("New body content (note_flesh)")),
		mcp.WithString("type", mcp.Description("New note type")),
		mcp.WithString("status", mcp.Description("New status")),
		mcp.WithString("area", mcp.Description("New area")),
		mcp.WithString("due", mcp.Description("New due date/time ('clear' to remove, or formatted/relative date)")),
		mcp.WithNumber("importance", mcp.Description("Importance rating 1-5")),
		mcp.WithNumber("clarity", mcp.Description("Clarity rating 1-5")),
		mcp.WithString("source", mcp.Description("Updated source")),
	)
	s.AddTool(updateNoteTool, handleUpdateNote)

	// 6. delete_note
	deleteNoteTool := mcp.NewTool("delete_note",
		mcp.WithDescription("Soft-delete a note by marking it with deleted timestamp and deletion attribution."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Full UUID or short ID of note to soft delete")),
		mcp.WithString("reason", mcp.Description("Reason or attribution for deletion (e.g. 'deleted by AI: completed task', defaults to 'deleted by AI')")),
	)
	s.AddTool(deleteNoteTool, handleDeleteNote)

	// 7. add_tag
	addTagTool := mcp.NewTool("add_tag",
		mcp.WithDescription("Add a tag to a note (e.g., 'tech/golang', 'project/kb')."),
		mcp.WithString("note_id", mcp.Required(), mcp.Description("Full UUID or short ID of note")),
		mcp.WithString("tag", mcp.Required(), mcp.Description("Tag name")),
	)
	s.AddTool(addTagTool, handleAddTag)

	// 8. link_notes
	linkNotesTool := mcp.NewTool("link_notes",
		mcp.WithDescription("Create a directional relation link between two notes."),
		mcp.WithString("from_id", mcp.Required(), mcp.Description("Source note ID")),
		mcp.WithString("to_id", mcp.Required(), mcp.Description("Target note ID")),
		mcp.WithString("type", mcp.Description("Link type: 'related_to', 'part_of', 'inspired_by', 'depends_on', 'supports', 'contradicts', 'about', 'created_by' (default 'related_to')")),
	)
	s.AddTool(linkNotesTool, handleLinkNotes)
}

func handleListNotes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	noteType := req.GetString("type", "")
	status := req.GetString("status", "")
	area := req.GetString("area", "")
	includeDeleted := req.GetBool("include_deleted", false)

	notes, err := db.ListNotesExtended(noteType, status, area, includeDeleted)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list notes: %v", err)), nil
	}

	type NoteSummary struct {
		ID             string     `json:"id"`
		ShortID        string     `json:"short_id"`
		Note           string     `json:"note"`
		Type           string     `json:"type"`
		Status         string     `json:"status"`
		Area           string     `json:"area,omitempty"`
		TargetDateTime *time.Time `json:"target_date_time,omitempty"`
		UpdatedAt      time.Time  `json:"updated_at"`
		DeletedAt      *time.Time `json:"deleted_at,omitempty"`
		DeletedNote    string     `json:"deleted_note,omitempty"`
	}

	var summaries []NoteSummary
	for _, n := range notes {
		shortID := n.ID
		if len(shortID) > 7 {
			shortID = shortID[:7]
		}
		ns := NoteSummary{
			ID:        n.ID,
			ShortID:   shortID,
			Note:      n.Note,
			Type:      string(n.Type),
			Status:    string(n.Status),
			Area:      string(n.Area),
			UpdatedAt: n.UpdatedAt,
		}
		if !n.TargetDateTime.IsZero() {
			ns.TargetDateTime = &n.TargetDateTime
		}
		if !n.DeletedAt.IsZero() {
			ns.DeletedAt = &n.DeletedAt
			ns.DeletedNote = n.DeletedNote
		}
		summaries = append(summaries, ns)
	}

	data, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize notes: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	n, err := db.GetNote(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("note not found: %v", err)), nil
	}

	tags, _ := db.GetTagsForNote(n.ID)
	links, _ := db.GetLinksForNote(n.ID)

	var tagNames []string
	for _, t := range tags {
		tagNames = append(tagNames, t.Name)
	}

	type LinkDetail struct {
		ID          string `json:"id"`
		Direction   string `json:"direction"`
		OtherNoteID string `json:"other_note_id"`
		OtherNote   string `json:"other_note,omitempty"`
		Type        string `json:"type"`
	}

	var linkDetails []LinkDetail
	for _, l := range links {
		otherID := l.ToNote
		dir := "outgoing"
		if l.ToNote == n.ID {
			otherID = l.FromNote
			dir = "incoming"
		}
		otherTitle := otherID
		if len(otherTitle) > 7 {
			otherTitle = otherTitle[:7]
		}
		if lNote, err := db.GetNote(otherID); err == nil && lNote.Note != "" {
			otherTitle = lNote.Note
		}
		linkDetails = append(linkDetails, LinkDetail{
			ID:          l.ID,
			Direction:   dir,
			OtherNoteID: otherID,
			OtherNote:   otherTitle,
			Type:        string(l.Type),
		})
	}

	res := map[string]interface{}{
		"id":         n.ID,
		"note":       n.Note,
		"note_flesh": n.NoteFlesh,
		"type":       n.Type,
		"status":     n.Status,
		"area":       n.Area,
		"importance": n.Importance,
		"clarity":    n.Clarity,
		"source":     n.Source,
		"created_at": n.CreatedAt,
		"updated_at": n.UpdatedAt,
		"tags":       tagNames,
		"links":      linkDetails,
	}
	if !n.TargetDateTime.IsZero() {
		res["target_date_time"] = n.TargetDateTime
	}
	if !n.DeletedAt.IsZero() {
		res["deleted_at"] = n.DeletedAt
		res["deleted_note"] = n.DeletedNote
	}

	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize note: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func handleSearchNotes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	notes, err := db.SearchNotes(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	type SearchResult struct {
		ID             string     `json:"id"`
		ShortID        string     `json:"short_id"`
		Note           string     `json:"note"`
		Type           string     `json:"type"`
		Status         string     `json:"status"`
		Area           string     `json:"area,omitempty"`
		TargetDateTime *time.Time `json:"target_date_time,omitempty"`
		UpdatedAt      time.Time  `json:"updated_at"`
	}

	var results []SearchResult
	for _, n := range notes {
		shortID := n.ID
		if len(shortID) > 7 {
			shortID = shortID[:7]
		}
		sr := SearchResult{
			ID:        n.ID,
			ShortID:   shortID,
			Note:      n.Note,
			Type:      string(n.Type),
			Status:    string(n.Status),
			Area:      string(n.Area),
			UpdatedAt: n.UpdatedAt,
		}
		if !n.TargetDateTime.IsZero() {
			sr.TargetDateTime = &n.TargetDateTime
		}
		results = append(results, sr)
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize search results: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func handleCreateNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	noteTitle := req.GetString("note", "")
	content := req.GetString("content", "")
	if noteTitle == "" && content == "" {
		return mcp.NewToolResultError("at least note or content must be provided"), nil
	}

	noteTypeStr := req.GetString("type", string(models.DefaultNote))
	statusStr := req.GetString("status", string(models.Active))
	areaStr := req.GetString("area", "")
	sourceStr := req.GetString("source", "AI Agent")
	dueStr := req.GetString("due", "")
	importance := req.GetInt("importance", 0)
	clarity := req.GetInt("clarity", 0)

	var targetDT time.Time
	if dueStr != "" {
		parsedDate, err := utils.ParseDate(dueStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid due date format %q: %v", dueStr, err)), nil
		}
		targetDT = parsedDate
	}

	id := strings.ReplaceAll(uuid.New().String(), "-", "")
	n := &models.Note{
		ID:             id,
		Note:           noteTitle,
		NoteFlesh:      content,
		Type:           models.NoteType(noteTypeStr),
		Status:         models.Status(statusStr),
		Area:           models.Area(areaStr),
		Importance:     importance,
		Clarity:        clarity,
		Source:         sourceStr,
		TargetDateTime: targetDT,
	}

	if err := db.CreateNote(n); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to save note: %v", err)), nil
	}

	res := map[string]interface{}{
		"success":  true,
		"id":       n.ID,
		"short_id": n.ID[:7],
		"message":  fmt.Sprintf("Successfully created note [%s]", n.ID[:7]),
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleUpdateNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	n, err := db.GetNote(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("note not found: %v", err)), nil
	}

	args := req.GetArguments()
	if _, ok := args["note"]; ok {
		n.Note = req.GetString("note", "")
	}
	if _, ok := args["content"]; ok {
		n.NoteFlesh = req.GetString("content", "")
	}
	if _, ok := args["type"]; ok {
		n.Type = models.NoteType(req.GetString("type", string(models.DefaultNote)))
	}
	if _, ok := args["status"]; ok {
		n.Status = models.Status(req.GetString("status", string(models.Active)))
	}
	if _, ok := args["area"]; ok {
		n.Area = models.Area(req.GetString("area", ""))
	}
	if _, ok := args["source"]; ok {
		n.Source = req.GetString("source", "")
	}
	if _, ok := args["due"]; ok {
		dueStr := req.GetString("due", "")
		if strings.ToLower(strings.TrimSpace(dueStr)) == "clear" {
			n.TargetDateTime = time.Time{}
		} else if dueStr != "" {
			parsedDate, err := utils.ParseDate(dueStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid due date format %q: %v", dueStr, err)), nil
			}
			n.TargetDateTime = parsedDate
		}
	}
	if _, ok := args["importance"]; ok {
		n.Importance = req.GetInt("importance", 0)
	}
	if _, ok := args["clarity"]; ok {
		n.Clarity = req.GetInt("clarity", 0)
	}

	if err := db.UpdateNote(n); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update note: %v", err)), nil
	}

	res := map[string]interface{}{
		"success":  true,
		"id":       n.ID,
		"short_id": n.ID[:7],
		"message":  fmt.Sprintf("Successfully updated note [%s]", n.ID[:7]),
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDeleteNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	reason := req.GetString("reason", "deleted by AI")

	if err := db.SoftDeleteNote(id, reason); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete note: %v", err)), nil
	}

	res := map[string]interface{}{
		"success": true,
		"id":      id,
		"reason":  reason,
		"message": fmt.Sprintf("Successfully soft-deleted note [%s] (%s)", id, reason),
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleAddTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	noteID, err := req.RequireString("note_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	tag, err := req.RequireString("tag")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := db.AddTag(noteID, tag); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add tag: %v", err)), nil
	}

	res := map[string]interface{}{
		"success": true,
		"note_id": noteID,
		"tag":     tag,
		"message": fmt.Sprintf("Successfully added tag '%s' to note [%s]", tag, noteID),
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleLinkNotes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	fromID, err := req.RequireString("from_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	toID, err := req.RequireString("to_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	linkTypeStr := req.GetString("type", string(models.RelatedTo))

	if err := db.AddLink(fromID, toID, models.LinkType(linkTypeStr)); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to link notes: %v", err)), nil
	}

	res := map[string]interface{}{
		"success":   true,
		"from_id":   fromID,
		"to_id":     toID,
		"link_type": linkTypeStr,
		"message":   fmt.Sprintf("Successfully linked [%s] --> [%s] as '%s'", fromID, toID, linkTypeStr),
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
