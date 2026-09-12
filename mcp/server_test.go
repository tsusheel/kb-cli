package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tsusheel/kb-cli/db"
)

func setupMCPTestDB(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "mcp_test.db")
	db.InitDB(dbPath)
	t.Cleanup(func() {
		db.CloseDB()
	})
	if err := db.InitSchema(); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}
}

func makeToolRequest(name string, args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}
}

func TestMCPCreateAndGetNote(t *testing.T) {
	setupMCPTestDB(t)
	ctx := context.Background()

	// 1. Test create_note
	createReq := makeToolRequest("create_note", map[string]interface{}{
		"note":       "MCP Design Document",
		"content":    "# MCP Server\nDetailed markdown body.",
		"type":       "project",
		"status":     "in-progress",
		"area":       "work",
		"due":        "tomorrow",
		"importance": float64(4),
		"clarity":    float64(5),
	})

	createRes, err := handleCreateNote(ctx, createReq)
	if err != nil {
		t.Fatalf("handleCreateNote failed: %v", err)
	}
	if len(createRes.Content) == 0 {
		t.Fatal("expected response content")
	}

	textContent, ok := createRes.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", createRes.Content[0])
	}

	var createdMap map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &createdMap); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	noteID, ok := createdMap["id"].(string)
	if !ok || noteID == "" {
		t.Fatalf("expected note ID in response: %v", createdMap)
	}

	// 2. Test get_note
	getReq := makeToolRequest("get_note", map[string]interface{}{
		"id": noteID[:7],
	})

	getRes, err := handleGetNote(ctx, getReq)
	if err != nil {
		t.Fatalf("handleGetNote failed: %v", err)
	}
	getText := getRes.Content[0].(mcp.TextContent).Text
	if !strings.Contains(getText, "MCP Design Document") {
		t.Errorf("get_note did not contain title: %s", getText)
	}

	// 3. Test list_notes
	listReq := makeToolRequest("list_notes", map[string]interface{}{
		"type": "project",
	})
	listRes, err := handleListNotes(ctx, listReq)
	if err != nil {
		t.Fatalf("handleListNotes failed: %v", err)
	}
	listText := listRes.Content[0].(mcp.TextContent).Text
	if !strings.Contains(listText, "MCP Design Document") {
		t.Errorf("list_notes did not contain project note: %s", listText)
	}

	// 4. Test search_notes
	searchReq := makeToolRequest("search_notes", map[string]interface{}{
		"query": "markdown",
	})
	searchRes, err := handleSearchNotes(ctx, searchReq)
	if err != nil {
		t.Fatalf("handleSearchNotes failed: %v", err)
	}
	searchText := searchRes.Content[0].(mcp.TextContent).Text
	if !strings.Contains(searchText, "MCP Design Document") {
		t.Errorf("search_notes did not find note matching 'markdown': %s", searchText)
	}

	// 5. Test update_note
	updateReq := makeToolRequest("update_note", map[string]interface{}{
		"id":     noteID,
		"status": "completed",
	})
	_, err = handleUpdateNote(ctx, updateReq)
	if err != nil {
		t.Fatalf("handleUpdateNote failed: %v", err)
	}

	// Verify update in get_note
	getRes2, _ := handleGetNote(ctx, getReq)
	if !strings.Contains(getRes2.Content[0].(mcp.TextContent).Text, "completed") {
		t.Errorf("expected status 'completed', got: %s", getRes2.Content[0].(mcp.TextContent).Text)
	}

	// 6. Test delete_note (soft delete)
	deleteReq := makeToolRequest("delete_note", map[string]interface{}{
		"id":     noteID,
		"reason": "deleted by AI: project completed",
	})
	_, err = handleDeleteNote(ctx, deleteReq)
	if err != nil {
		t.Fatalf("handleDeleteNote failed: %v", err)
	}

	// Verify excluded from normal list
	listRes2, _ := handleListNotes(ctx, listReq)
	if strings.Contains(listRes2.Content[0].(mcp.TextContent).Text, "MCP Design Document") {
		t.Errorf("deleted note should not appear in default list: %s", listRes2.Content[0].(mcp.TextContent).Text)
	}

	// Verify included with include_deleted: true
	listDeletedReq := makeToolRequest("list_notes", map[string]interface{}{
		"include_deleted": true,
	})
	listRes3, _ := handleListNotes(ctx, listDeletedReq)
	if !strings.Contains(listRes3.Content[0].(mcp.TextContent).Text, "deleted by AI: project completed") {
		t.Errorf("expected deleted note reason in list: %s", listRes3.Content[0].(mcp.TextContent).Text)
	}
}

func TestMCPTagsAndLinks(t *testing.T) {
	setupMCPTestDB(t)
	ctx := context.Background()

	// Create 2 notes
	create1, _ := handleCreateNote(ctx, makeToolRequest("create_note", map[string]interface{}{
		"note":    "Source Note",
		"content": "Source content",
	}))
	var res1 map[string]interface{}
	json.Unmarshal([]byte(create1.Content[0].(mcp.TextContent).Text), &res1)
	id1 := res1["id"].(string)

	create2, _ := handleCreateNote(ctx, makeToolRequest("create_note", map[string]interface{}{
		"note":    "Target Note",
		"content": "Target content",
	}))
	var res2 map[string]interface{}
	json.Unmarshal([]byte(create2.Content[0].(mcp.TextContent).Text), &res2)
	id2 := res2["id"].(string)

	// Add Tag
	tagRes, err := handleAddTag(ctx, makeToolRequest("add_tag", map[string]interface{}{
		"note_id": id1,
		"tag":     "ai/mcp",
	}))
	if err != nil {
		t.Fatalf("handleAddTag failed: %v", err)
	}
	if !strings.Contains(tagRes.Content[0].(mcp.TextContent).Text, "ai/mcp") {
		t.Errorf("tag response mismatch: %s", tagRes.Content[0].(mcp.TextContent).Text)
	}

	// Link notes
	linkRes, err := handleLinkNotes(ctx, makeToolRequest("link_notes", map[string]interface{}{
		"from_id": id1,
		"to_id":   id2,
		"type":    "depends_on",
	}))
	if err != nil {
		t.Fatalf("handleLinkNotes failed: %v", err)
	}
	if !strings.Contains(linkRes.Content[0].(mcp.TextContent).Text, "depends_on") {
		t.Errorf("link response mismatch: %s", linkRes.Content[0].(mcp.TextContent).Text)
	}
}

func TestMCPDailyLogsAndInbox(t *testing.T) {
	setupMCPTestDB(t)
	ctx := context.Background()

	// 1. Test create_log
	logRes, err := handleCreateLog(ctx, makeToolRequest("create_log", map[string]interface{}{
		"content": "Worked on MCP protocol daily logs",
	}))
	if err != nil {
		t.Fatalf("handleCreateLog failed: %v", err)
	}
	var logData map[string]interface{}
	json.Unmarshal([]byte(logRes.Content[0].(mcp.TextContent).Text), &logData)
	logID := logData["id"].(string)

	// 2. Test list_daily_logs
	listLogsRes, err := handleListDailyLogs(ctx, makeToolRequest("list_daily_logs", map[string]interface{}{
		"date": "today",
	}))
	if err != nil {
		t.Fatalf("handleListDailyLogs failed: %v", err)
	}
	if !strings.Contains(listLogsRes.Content[0].(mcp.TextContent).Text, "Worked on MCP protocol daily logs") {
		t.Errorf("list_daily_logs did not return expected log: %s", listLogsRes.Content[0].(mcp.TextContent).Text)
	}

	// 3. Test get_inbox (should contain unpromoted log)
	inboxRes, err := handleGetInbox(ctx, makeToolRequest("get_inbox", map[string]interface{}{}))
	if err != nil {
		t.Fatalf("handleGetInbox failed: %v", err)
	}
	if !strings.Contains(inboxRes.Content[0].(mcp.TextContent).Text, "Worked on MCP protocol daily logs") {
		t.Errorf("get_inbox did not contain unpromoted log: %s", inboxRes.Content[0].(mcp.TextContent).Text)
	}

	// 4. Test promote_log non-interactively
	promoteRes, err := handlePromoteLog(ctx, makeToolRequest("promote_log", map[string]interface{}{
		"log_id": logID[:7],
		"type":   "concept",
		"status": "raw",
	}))
	if err != nil {
		t.Fatalf("handlePromoteLog failed: %v", err)
	}
	promoteText := promoteRes.Content[0].(mcp.TextContent).Text
	if !strings.Contains(promoteText, "Promoted log") {
		t.Errorf("promote_log response mismatch: %s", promoteText)
	}

	// 5. Test get_inbox after promotion (should show note in raw_notes and empty unpromoted_logs)
	inboxRes2, _ := handleGetInbox(ctx, makeToolRequest("get_inbox", map[string]interface{}{}))
	inboxText2 := inboxRes2.Content[0].(mcp.TextContent).Text
	if !strings.Contains(inboxText2, `"logs_count": 0`) {
		t.Errorf("expected 0 unpromoted logs after promotion, got: %s", inboxText2)
	}
	if !strings.Contains(inboxText2, "Worked on MCP protocol daily logs") {
		t.Errorf("expected promoted raw note in inbox, got: %s", inboxText2)
	}
}
