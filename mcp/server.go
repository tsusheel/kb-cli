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
		server.WithPromptCapabilities(true),
	)

	registerTools(s)
	registerResources(s)
	registerPrompts(s)
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

	// 8. remove_tag
	removeTagTool := mcp.NewTool("remove_tag",
		mcp.WithDescription("Remove a tag association from a note."),
		mcp.WithString("note_id", mcp.Required(), mcp.Description("Full UUID or short ID of note")),
		mcp.WithString("tag", mcp.Required(), mcp.Description("Tag name to remove")),
	)
	s.AddTool(removeTagTool, handleRemoveTag)

	// 9. link_notes
	linkNotesTool := mcp.NewTool("link_notes",
		mcp.WithDescription("Create a directional relation link between two notes."),
		mcp.WithString("from_id", mcp.Required(), mcp.Description("Source note ID")),
		mcp.WithString("to_id", mcp.Required(), mcp.Description("Target note ID")),
		mcp.WithString("type", mcp.Description("Link type: 'related_to', 'part_of', 'inspired_by', 'depends_on', 'supports', 'contradicts', 'about', 'created_by' (default 'related_to')")),
	)
	s.AddTool(linkNotesTool, handleLinkNotes)

	// 10. unlink_notes
	unlinkNotesTool := mcp.NewTool("unlink_notes",
		mcp.WithDescription("Remove or soft-delete a directional link between two notes."),
		mcp.WithString("from_id", mcp.Required(), mcp.Description("Source note ID")),
		mcp.WithString("to_id", mcp.Required(), mcp.Description("Target note ID")),
		mcp.WithString("reason", mcp.Description("Optional deletion reason (default 'unlinked by AI')")),
	)
	s.AddTool(unlinkNotesTool, handleUnlinkNotes)

	// 11. create_log
	createLogTool := mcp.NewTool("create_log",
		mcp.WithDescription("Append a timestamped micro-log to today's daily stream."),
		mcp.WithString("content", mcp.Required(), mcp.Description("Log entry text")),
	)
	s.AddTool(createLogTool, handleCreateLog)

	// 12. list_daily_logs
	listDailyLogsTool := mcp.NewTool("list_daily_logs",
		mcp.WithDescription("List daily stream micro-logs for today, a specific date, or an inclusive date range."),
		mcp.WithString("date", mcp.Description("Optional specific date (e.g., 'today', 'yesterday', '2026-09-06')")),
		mcp.WithString("from", mcp.Description("Optional inclusive start date (e.g., '2026-09-01', 'yesterday', '-7d')")),
		mcp.WithString("to", mcp.Description("Optional inclusive end date (e.g., '2026-09-13', 'today')")),
		mcp.WithBoolean("include_promoted", mcp.Description("Whether to include logs that were already promoted to notes (default true)")),
	)
	s.AddTool(listDailyLogsTool, handleListDailyLogs)

	// 13. promote_log
	promoteLogTool := mcp.NewTool("promote_log",
		mcp.WithDescription("Promote a daily log entry into a standalone permanent Note non-interactively."),
		mcp.WithString("log_id", mcp.Required(), mcp.Description("ID of the daily log to promote")),
		mcp.WithString("type", mcp.Description("Note type for the promoted note (default 'note')")),
		mcp.WithString("status", mcp.Description("Status for the promoted note (default 'raw')")),
	)
	s.AddTool(promoteLogTool, handlePromoteLog)

	// 14. get_inbox
	getInboxTool := mcp.NewTool("get_inbox",
		mcp.WithDescription("Get all raw unrefined notes and unpromoted daily logs awaiting triage."),
	)
	s.AddTool(getInboxTool, handleGetInbox)

	// 15. get_history
	getHistoryTool := mcp.NewTool("get_history",
		mcp.WithDescription("Retrieve audit revision history and timeline for a note, daily log, or global stream."),
		mcp.WithString("id", mcp.Description("Optional note or log ID (if omitted, returns recent global activity)")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of revisions to return (default 20)")),
	)
	s.AddTool(getHistoryTool, handleGetHistory)

	// 16. revert_note
	revertNoteTool := mcp.NewTool("revert_note",
		mcp.WithDescription("Revert a note to a previous point-in-time snapshot revision."),
		mcp.WithString("note_id", mcp.Required(), mcp.Description("ID of the note to revert")),
		mcp.WithString("revision_id", mcp.Required(), mcp.Description("ID of the audit revision to revert to")),
	)
	s.AddTool(revertNoteTool, handleRevertNote)

	// 17. restore_note
	restoreNoteTool := mcp.NewTool("restore_note",
		mcp.WithDescription("Restore a soft-deleted note back to active state."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Full UUID or short ID of note to restore")),
	)
	s.AddTool(restoreNoteTool, handleRestoreNote)

	// 18. restore_log
	restoreLogTool := mcp.NewTool("restore_log",
		mcp.WithDescription("Restore a soft-deleted daily log back to active state."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Full UUID or short ID of daily log to restore")),
	)
	s.AddTool(restoreLogTool, handleRestoreLog)
}

func registerResources(s *server.MCPServer) {
	// Resource 1: Today's stream
	todayResource := mcp.NewResource("kb://today", "Today's Daily Stream", mcp.WithResourceDescription("Stream of daily logs recorded today"), mcp.WithMIMEType("application/json"))
	s.AddResource(todayResource, handleResourceToday)

	// Resource 2: Inbox
	inboxResource := mcp.NewResource("kb://inbox", "Triage Inbox", mcp.WithResourceDescription("Raw notes and unpromoted logs awaiting triage"), mcp.WithMIMEType("application/json"))
	s.AddResource(inboxResource, handleResourceInbox)
}

func registerPrompts(s *server.MCPServer) {
	// Prompt 1: Triage Inbox
	triagePrompt := mcp.NewPrompt("triage-inbox",
		mcp.WithPromptDescription("Analyze raw inbox notes and unpromoted logs, proposing tags, types, and refinements."),
	)
	s.AddPrompt(triagePrompt, handlePromptTriageInbox)

	// Prompt 2: Daily Summary
	summaryPrompt := mcp.NewPrompt("daily-summary",
		mcp.WithPromptDescription("Synthesize today's daily logs and completed tasks into an executive summary."),
	)
	s.AddPrompt(summaryPrompt, handlePromptDailySummary)
}

func handleResourceToday(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	now := time.Now()
	logs, err := db.GetDailyLogsForDate(now, true)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		return nil, err
	}
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "kb://today",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

func handleResourceInbox(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	rawNotes, err := db.ListNotesExtended("", string(models.Raw), "", false)
	if err != nil {
		return nil, err
	}
	unpromotedLogs, err := db.GetUnpromotedDailyLogs()
	if err != nil {
		return nil, err
	}
	res := map[string]interface{}{
		"raw_notes":       rawNotes,
		"unpromoted_logs": unpromotedLogs,
	}
	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return nil, err
	}
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "kb://inbox",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

func handlePromptTriageInbox(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	rawNotes, _ := db.ListNotesExtended("", string(models.Raw), "", false)
	unpromotedLogs, _ := db.GetUnpromotedDailyLogs()

	promptText := fmt.Sprintf("Please review the following %d raw note(s) and %d unpromoted daily log(s):\n\n", len(rawNotes), len(unpromotedLogs))
	for _, n := range rawNotes {
		promptText += fmt.Sprintf("- Note [%s]: %s (flesh: %s)\n", utils.ShortID(n.ID), n.Note, n.NoteFlesh)
	}
	for _, l := range unpromotedLogs {
		promptText += fmt.Sprintf("- Daily Log [%s]: %s\n", utils.ShortID(l.ID), l.Content)
	}
	promptText += "\nPropose appropriate note types (todo, project, idea, concept, decision), tags, and suggest which logs should be promoted to notes."

	return &mcp.GetPromptResult{
		Description: "Triage inbox prompt",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: promptText,
				},
			},
		},
	}, nil
}

func handlePromptDailySummary(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	now := time.Now()
	logs, _ := db.GetDailyLogsForDate(now, true)

	promptText := fmt.Sprintf("Here are today's (%s) stream logs:\n\n", now.Format("2006-01-02"))
	for _, l := range logs {
		promptText += fmt.Sprintf("[%s] %s\n", l.CreatedAt.Format("15:04"), l.Content)
	}
	promptText += "\nPlease synthesize these logs into a structured executive daily summary with key accomplishments, ongoing items, and next steps."

	return &mcp.GetPromptResult{
		Description: "Daily summary prompt",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: promptText,
				},
			},
		},
	}, nil
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

func handleRemoveTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	noteID, err := req.RequireString("note_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	tag, err := req.RequireString("tag")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := db.RemoveTag(noteID, tag); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to remove tag: %v", err)), nil
	}

	res := map[string]interface{}{
		"success": true,
		"note_id": noteID,
		"tag":     tag,
		"message": fmt.Sprintf("Successfully removed tag '%s' from note [%s]", tag, noteID),
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

func handleUnlinkNotes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	fromID, err := req.RequireString("from_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	toID, err := req.RequireString("to_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	reason := req.GetString("reason", "unlinked by AI")

	if err := db.RemoveLink(fromID, toID, reason); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to unlink notes: %v", err)), nil
	}

	res := map[string]interface{}{
		"success": true,
		"from_id": fromID,
		"to_id":   toID,
		"message": fmt.Sprintf("Successfully unlinked [%s] --> [%s]", fromID, toID),
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleCreateLog(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	l, err := db.CreateDailyLog(content)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create log: %v", err)), nil
	}

	res := map[string]interface{}{
		"success":  true,
		"id":       l.ID,
		"short_id": l.ID[:7],
		"time":     l.CreatedAt.Format("15:04"),
		"message":  fmt.Sprintf("Logged [%s] at %s", l.ID[:7], l.CreatedAt.Format("15:04")),
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListDailyLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dateStr := req.GetString("date", "")
	fromStr := req.GetString("from", "")
	toStr := req.GetString("to", "")
	includePromoted := req.GetBool("include_promoted", true)

	var startDate *time.Time
	var endDate *time.Time

	if dateStr != "" {
		parsedDate, err := utils.ParseDate(dateStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid date format %q: %v", dateStr, err)), nil
		}
		startDate = &parsedDate
		endDate = &parsedDate
	} else if fromStr != "" || toStr != "" {
		if fromStr != "" {
			t, err := utils.ParseDate(fromStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid from date format %q: %v", fromStr, err)), nil
			}
			startDate = &t
		}
		if toStr != "" {
			t, err := utils.ParseDate(toStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid to date format %q: %v", toStr, err)), nil
			}
			endDate = &t
		}
		if startDate != nil && endDate == nil {
			now := time.Now()
			endDate = &now
		}
	} else {
		now := time.Now()
		startDate = &now
		endDate = &now
	}

	logs, err := db.GetDailyLogsFilter(startDate, endDate, includePromoted, false)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list daily logs: %v", err)), nil
	}

	type LogSummary struct {
		ID        string    `json:"id"`
		ShortID   string    `json:"short_id"`
		Content   string    `json:"content"`
		Time      string    `json:"time"`
		CreatedAt time.Time `json:"created_at"`
		NoteID    string    `json:"note_id,omitempty"`
	}

	var summaries []LogSummary
	for _, l := range logs {
		summaries = append(summaries, LogSummary{
			ID:        l.ID,
			ShortID:   l.ID[:7],
			Content:   l.Content,
			Time:      l.CreatedAt.Format("15:04"),
			CreatedAt: l.CreatedAt,
			NoteID:    l.NoteID,
		})
	}

	data, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize logs: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func handlePromoteLog(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logID, err := req.RequireString("log_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	noteType := req.GetString("type", string(models.DefaultNote))
	noteStatus := req.GetString("status", string(models.Raw))

	n, err := db.PromoteDailyLog(logID, models.NoteType(noteType), models.Status(noteStatus))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to promote log: %v", err)), nil
	}

	res := map[string]interface{}{
		"success":       true,
		"note_id":       n.ID,
		"note_short_id": n.ID[:7],
		"type":          n.Type,
		"status":        n.Status,
		"message":       fmt.Sprintf("Promoted log [%s] to Note [%s]", logID, n.ID[:7]),
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetInbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawNotes, err := db.ListNotesExtended("", string(models.Raw), "", false)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get raw notes: %v", err)), nil
	}

	unpromotedLogs, err := db.GetUnpromotedDailyLogs()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get unpromoted logs: %v", err)), nil
	}

	type NoteItem struct {
		ID        string    `json:"id"`
		ShortID   string    `json:"short_id"`
		Note      string    `json:"note"`
		Type      string    `json:"type"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	type LogItem struct {
		ID        string    `json:"id"`
		ShortID   string    `json:"short_id"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
	}

	var notesList []NoteItem
	for _, n := range rawNotes {
		notesList = append(notesList, NoteItem{
			ID:        n.ID,
			ShortID:   n.ID[:7],
			Note:      n.Note,
			Type:      string(n.Type),
			UpdatedAt: n.UpdatedAt,
		})
	}

	var logsList []LogItem
	for _, l := range unpromotedLogs {
		logsList = append(logsList, LogItem{
			ID:        l.ID,
			ShortID:   l.ID[:7],
			Content:   l.Content,
			CreatedAt: l.CreatedAt,
		})
	}

	res := map[string]interface{}{
		"raw_notes_count": len(notesList),
		"raw_notes":       notesList,
		"logs_count":      len(logsList),
		"unpromoted_logs": logsList,
	}

	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	limit := req.GetInt("limit", 20)
	if limit <= 0 {
		limit = 20
	}

	var entries []models.AuditEntry
	var err error

	if id == "" {
		entries, err = db.GetRecentAuditHistory(limit)
	} else if n, errNote := db.GetNote(id); errNote == nil {
		entries, err = db.GetAuditHistory("note", n.ID, limit)
	} else if l, errLog := db.GetDailyLog(id); errLog == nil {
		entries, err = db.GetAuditHistory("daily_log", l.ID, limit)
	} else {
		return mcp.NewToolResultError(fmt.Sprintf("no note or daily log found matching ID %q", id)), nil
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get history: %v", err)), nil
	}

	type HistorySummary struct {
		ID             string    `json:"id"`
		ShortID        string    `json:"short_id"`
		EntityType     string    `json:"entity_type"`
		EntityID       string    `json:"entity_id"`
		Action         string    `json:"action"`
		ChangesSummary string    `json:"changes_summary"`
		CreatedAt      time.Time `json:"created_at"`
		SnapshotJSON   string    `json:"snapshot_json,omitempty"`
	}

	var list []HistorySummary
	for _, e := range entries {
		shortID := e.ID
		if len(shortID) > 7 {
			shortID = shortID[:7]
		}
		list = append(list, HistorySummary{
			ID:             e.ID,
			ShortID:        shortID,
			EntityType:     e.EntityType,
			EntityID:       e.EntityID,
			Action:         string(e.Action),
			ChangesSummary: e.ChangesSummary,
			CreatedAt:      e.CreatedAt,
			SnapshotJSON:   e.SnapshotJSON,
		})
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize history: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func handleRevertNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	noteID, err := req.RequireString("note_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	revID, err := req.RequireString("revision_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	n, err := db.GetNote(noteID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("note not found: %v", err)), nil
	}

	entry, err := db.GetAuditEntry(revID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("audit revision not found: %v", err)), nil
	}

	if entry.EntityType != "note" || !strings.HasPrefix(entry.EntityID, n.ID) {
		return mcp.NewToolResultError(fmt.Sprintf("revision [%s] belongs to %s [%s], not note [%s]", revID, entry.EntityType, entry.EntityID, n.ID)), nil
	}

	if strings.TrimSpace(entry.SnapshotJSON) == "" {
		return mcp.NewToolResultError(fmt.Sprintf("revision [%s] does not contain a recoverable snapshot", revID)), nil
	}

	reverted, err := db.RevertNoteToSnapshot(n.ID, entry.SnapshotJSON)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to revert note: %v", err)), nil
	}

	res := map[string]interface{}{
		"success":     true,
		"note_id":     reverted.ID,
		"revision_id": entry.ID,
		"title":       reverted.Note,
		"status":      reverted.Status,
		"type":        reverted.Type,
		"message":     fmt.Sprintf("Successfully reverted note [%s] to revision [%s]", reverted.ID[:7], entry.ID[:7]),
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleRestoreNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	n, err := db.RestoreNote(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to restore note: %v", err)), nil
	}

	res := map[string]interface{}{
		"success":  true,
		"id":       n.ID,
		"short_id": n.ID[:7],
		"title":    n.Note,
		"message":  fmt.Sprintf("Successfully restored note [%s]", n.ID[:7]),
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleRestoreLog(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	l, err := db.RestoreDailyLog(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to restore daily log: %v", err)), nil
	}

	res := map[string]interface{}{
		"success":  true,
		"id":       l.ID,
		"short_id": l.ID[:7],
		"content":  l.Content,
		"message":  fmt.Sprintf("Successfully restored daily log [%s]", l.ID[:7]),
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
