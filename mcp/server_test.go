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

	// 7. Test restore_note
	restoreReq := makeToolRequest("restore_note", map[string]interface{}{
		"id": noteID,
	})
	restoreRes, err := handleRestoreNote(ctx, restoreReq)
	if err != nil {
		t.Fatalf("handleRestoreNote failed: %v", err)
	}
	if !strings.Contains(restoreRes.Content[0].(mcp.TextContent).Text, "Successfully restored note") {
		t.Errorf("unexpected restore note response: %s", restoreRes.Content[0].(mcp.TextContent).Text)
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

	// Remove Tag
	remTagRes, err := handleRemoveTag(ctx, makeToolRequest("remove_tag", map[string]interface{}{
		"note_id": id1,
		"tag":     "ai/mcp",
	}))
	if err != nil {
		t.Fatalf("handleRemoveTag failed: %v", err)
	}
	if !strings.Contains(remTagRes.Content[0].(mcp.TextContent).Text, "Successfully removed tag") {
		t.Errorf("remove_tag response mismatch: %s", remTagRes.Content[0].(mcp.TextContent).Text)
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

	// Unlink notes
	unlinkRes, err := handleUnlinkNotes(ctx, makeToolRequest("unlink_notes", map[string]interface{}{
		"from_id": id1,
		"to_id":   id2,
	}))
	if err != nil {
		t.Fatalf("handleUnlinkNotes failed: %v", err)
	}
	if !strings.Contains(unlinkRes.Content[0].(mcp.TextContent).Text, "Successfully unlinked") {
		t.Errorf("unlink response mismatch: %s", unlinkRes.Content[0].(mcp.TextContent).Text)
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

	// 5. Test get_inbox after promotion
	inboxRes2, _ := handleGetInbox(ctx, makeToolRequest("get_inbox", map[string]interface{}{}))
	inboxText2 := inboxRes2.Content[0].(mcp.TextContent).Text
	if !strings.Contains(inboxText2, `"logs_count": 0`) {
		t.Errorf("expected 0 unpromoted logs after promotion, got: %s", inboxText2)
	}

	// 6. Test restore_log
	_ = db.SoftDeleteDailyLog(logID, "test delete")
	restoreLogRes, err := handleRestoreLog(ctx, makeToolRequest("restore_log", map[string]interface{}{
		"id": logID,
	}))
	if err != nil {
		t.Fatalf("handleRestoreLog failed: %v", err)
	}
	if !strings.Contains(restoreLogRes.Content[0].(mcp.TextContent).Text, "Successfully restored daily log") {
		t.Errorf("unexpected restore log response: %s", restoreLogRes.Content[0].(mcp.TextContent).Text)
	}
}

func TestMCPHistoryAndRevert(t *testing.T) {
	setupMCPTestDB(t)
	ctx := context.Background()

	// 1. Create a note
	createRes, _ := handleCreateNote(ctx, makeToolRequest("create_note", map[string]interface{}{
		"note":    "Versioned Note",
		"content": "Initial Body",
	}))
	var noteMap map[string]interface{}
	json.Unmarshal([]byte(createRes.Content[0].(mcp.TextContent).Text), &noteMap)
	noteID := noteMap["id"].(string)

	// 2. Fetch history
	histRes, err := handleGetHistory(ctx, makeToolRequest("get_history", map[string]interface{}{
		"id": noteID,
	}))
	if err != nil {
		t.Fatalf("handleGetHistory failed: %v", err)
	}
	var histList []map[string]interface{}
	json.Unmarshal([]byte(histRes.Content[0].(mcp.TextContent).Text), &histList)
	if len(histList) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(histList))
	}
	firstRevID := histList[0]["id"].(string)

	// 3. Update note
	handleUpdateNote(ctx, makeToolRequest("update_note", map[string]interface{}{
		"id":      noteID,
		"note":    "Updated Version",
		"content": "Updated Body",
	}))

	// 4. Revert note to first revision
	revertRes, err := handleRevertNote(ctx, makeToolRequest("revert_note", map[string]interface{}{
		"note_id":     noteID,
		"revision_id": firstRevID,
	}))
	if err != nil {
		t.Fatalf("handleRevertNote failed: %v", err)
	}
	if !strings.Contains(revertRes.Content[0].(mcp.TextContent).Text, "Successfully reverted note") {
		t.Errorf("unexpected revert response: %s", revertRes.Content[0].(mcp.TextContent).Text)
	}
}

func TestMCPResourcesAndPrompts(t *testing.T) {
	setupMCPTestDB(t)
	ctx := context.Background()

	// 1. Test Resource: kb://today
	db.CreateDailyLog("Test today log entry")
	todayContents, err := handleResourceToday(ctx, mcp.ReadResourceRequest{})
	if err != nil {
		t.Fatalf("handleResourceToday failed: %v", err)
	}
	if len(todayContents) == 0 || !strings.Contains(todayContents[0].(mcp.TextResourceContents).Text, "Test today log entry") {
		t.Errorf("unexpected today resource content: %v", todayContents)
	}

	// 2. Test Resource: kb://inbox
	inboxContents, err := handleResourceInbox(ctx, mcp.ReadResourceRequest{})
	if err != nil {
		t.Fatalf("handleResourceInbox failed: %v", err)
	}
	if len(inboxContents) == 0 || !strings.Contains(inboxContents[0].(mcp.TextResourceContents).Text, "raw_notes") {
		t.Errorf("unexpected inbox resource content: %v", inboxContents)
	}

	// 3. Test Prompt: triage-inbox
	triagePrompt, err := handlePromptTriageInbox(ctx, mcp.GetPromptRequest{})
	if err != nil {
		t.Fatalf("handlePromptTriageInbox failed: %v", err)
	}
	if len(triagePrompt.Messages) == 0 || !strings.Contains(triagePrompt.Messages[0].Content.(mcp.TextContent).Text, "Please review") {
		t.Errorf("unexpected triage prompt: %v", triagePrompt)
	}

	// 4. Test Prompt: daily-summary
	summaryPrompt, err := handlePromptDailySummary(ctx, mcp.GetPromptRequest{})
	if err != nil {
		t.Fatalf("handlePromptDailySummary failed: %v", err)
	}
	if len(summaryPrompt.Messages) == 0 || !strings.Contains(summaryPrompt.Messages[0].Content.(mcp.TextContent).Text, "Test today log entry") {
		t.Errorf("unexpected daily summary prompt: %v", summaryPrompt)
	}
}

func TestMCPContextEngineeringTools(t *testing.T) {
	setupMCPTestDB(t)
	ctx := context.Background()

	// 1. Seed notes with tags, areas, and links
	res1, _ := handleCreateNote(ctx, makeToolRequest("create_note", map[string]interface{}{
		"note":    "Distributed Tracing Architecture",
		"content": "Comprehensive OpenTelemetry implementation guide across microservices.",
		"type":    "project",
		"area":    "work",
		"status":  "active",
	}))
	var n1 map[string]interface{}
	json.Unmarshal([]byte(res1.Content[0].(mcp.TextContent).Text), &n1)
	id1 := n1["id"].(string)

	res2, _ := handleCreateNote(ctx, makeToolRequest("create_note", map[string]interface{}{
		"note":    "Implement OTel Collector",
		"content": "Deploy OTel collector with Jaeger exporter.",
		"type":    "todo",
		"area":    "work",
		"status":  "raw",
	}))
	var n2 map[string]interface{}
	json.Unmarshal([]byte(res2.Content[0].(mcp.TextContent).Text), &n2)
	id2 := n2["id"].(string)

	res3, _ := handleCreateNote(ctx, makeToolRequest("create_note", map[string]interface{}{
		"note":    "Solitary Untagged Note",
		"content": "This note has no tags and no links whatsoever.",
		"type":    "idea",
		"status":  "active",
	}))
	var n3 map[string]interface{}
	json.Unmarshal([]byte(res3.Content[0].(mcp.TextContent).Text), &n3)
	id3 := n3["id"].(string)

	// Add tags
	handleAddTag(ctx, makeToolRequest("add_tag", map[string]interface{}{"note_id": id1, "tag": "observability"}))
	handleAddTag(ctx, makeToolRequest("add_tag", map[string]interface{}{"note_id": id1, "tag": "golang"}))
	handleAddTag(ctx, makeToolRequest("add_tag", map[string]interface{}{"note_id": id2, "tag": "observability"}))

	// Add link
	handleLinkNotes(ctx, makeToolRequest("link_notes", map[string]interface{}{
		"from_id": id2,
		"to_id":   id1,
		"type":    "part_of",
	}))

	// 2. Test get_knowledge_map
	kmapRes, err := handleGetKnowledgeMap(ctx, makeToolRequest("get_knowledge_map", map[string]interface{}{}))
	if err != nil {
		t.Fatalf("handleGetKnowledgeMap failed: %v", err)
	}
	kmapText := kmapRes.Content[0].(mcp.TextContent).Text
	if !strings.Contains(kmapText, "total_active_notes") || !strings.Contains(kmapText, "observability") {
		t.Errorf("unexpected knowledge map output: %s", kmapText)
	}

	// 3. Test get_graph_neighborhood
	gnRes, err := handleGetGraphNeighborhood(ctx, makeToolRequest("get_graph_neighborhood", map[string]interface{}{
		"id": id1,
	}))
	if err != nil {
		t.Fatalf("handleGetGraphNeighborhood failed: %v", err)
	}
	gnText := gnRes.Content[0].(mcp.TextContent).Text
	if !strings.Contains(gnText, "focal_note") || !strings.Contains(gnText, "Implement OTel Collector") || !strings.Contains(gnText, "part_of") {
		t.Errorf("unexpected graph neighborhood output: %s", gnText)
	}

	// 4. Test suggest_links
	suggestRes, err := handleSuggestLinks(ctx, makeToolRequest("suggest_links", map[string]interface{}{
		"note_id": id3,
		"text":    "Need tracing and observability collector setup",
		"limit":   float64(5),
	}))
	if err != nil {
		t.Fatalf("handleSuggestLinks failed: %v", err)
	}
	suggestText := suggestRes.Content[0].(mcp.TextContent).Text
	if !strings.Contains(suggestText, "candidates") || !strings.Contains(suggestText, "confidence_score") {
		t.Errorf("unexpected suggest links output: %s", suggestText)
	}

	// 5. Test get_orphans
	orphansRes, err := handleGetOrphans(ctx, makeToolRequest("get_orphans", map[string]interface{}{
		"compact": true,
	}))
	if err != nil {
		t.Fatalf("handleGetOrphans failed: %v", err)
	}
	orphansText := orphansRes.Content[0].(mcp.TextContent).Text
	if !strings.Contains(orphansText, "Solitary Untagged Note") {
		t.Errorf("expected orphan note in get_orphans: %s", orphansText)
	}

	// 6. Test Compact mode previews in list_notes and search_notes
	listRes, err := handleListNotes(ctx, makeToolRequest("list_notes", map[string]interface{}{
		"compact": true,
	}))
	if err != nil {
		t.Fatalf("handleListNotes failed: %v", err)
	}
	listText := listRes.Content[0].(mcp.TextContent).Text
	if !strings.Contains(listText, "flesh_preview") {
		t.Errorf("expected flesh_preview in compact list_notes: %s", listText)
	}

	searchRes, err := handleSearchNotes(ctx, makeToolRequest("search_notes", map[string]interface{}{
		"query":   "OpenTelemetry",
		"compact": true,
	}))
	if err != nil {
		t.Fatalf("handleSearchNotes failed: %v", err)
	}
	searchText := searchRes.Content[0].(mcp.TextContent).Text
	if !strings.Contains(searchText, "snippet") {
		t.Errorf("expected snippet in compact search_notes: %s", searchText)
	}

	// 7. Test get_inbox compact previews
	inboxRes, err := handleGetInbox(ctx, makeToolRequest("get_inbox", map[string]interface{}{
		"compact": true,
	}))
	if err != nil {
		t.Fatalf("handleGetInbox failed: %v", err)
	}
	inboxText := inboxRes.Content[0].(mcp.TextContent).Text
	if !strings.Contains(inboxText, "flesh_preview") {
		t.Errorf("expected flesh_preview in compact get_inbox: %s", inboxText)
	}
}
