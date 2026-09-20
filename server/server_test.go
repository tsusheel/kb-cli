package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
)

func setupTestDB(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "server_test.db")
	db.InitDB(dbPath)
	t.Cleanup(func() {
		db.CloseDB()
	})
	if err := db.InitSchema(); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}
}

func TestServerStaticRoutes(t *testing.T) {
	setupTestDB(t)
	handler := RegisterRoutes()

	// 1. Test index.html
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "KB Finder") {
		t.Errorf("index.html missing expected title text: %s", rec.Body.String())
	}

	// 2. Test style.css
	reqCSS := httptest.NewRequest(http.MethodGet, "/style.css", nil)
	recCSS := httptest.NewRecorder()
	handler.ServeHTTP(recCSS, reqCSS)

	if recCSS.Code != http.StatusOK {
		t.Errorf("expected 200 for /style.css, got %d", recCSS.Code)
	}

	// 3. Test app.js
	reqJS := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	recJS := httptest.NewRecorder()
	handler.ServeHTTP(recJS, reqJS)

	if recJS.Code != http.StatusOK {
		t.Errorf("expected 200 for /app.js, got %d", recJS.Code)
	}
}

func TestServerAPIRoutes(t *testing.T) {
	setupTestDB(t)
	handler := RegisterRoutes()

	// 1. Test POST /api/notes
	createPayload := map[string]interface{}{
		"note":    "Distributed Architecture",
		"content": "Designing resilient microservices in Go.",
		"type":    "project",
		"status":  "active",
		"area":    "work",
		"tags":    []string{"golang", "system-design"},
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/notes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for POST /api/notes, got %d: %s", rec.Code, rec.Body.String())
	}

	var createRes map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createRes)
	noteID := createRes["id"].(string)

	// 2. Test POST /api/logs
	logPayload := map[string]interface{}{
		"content": "Finished system design diagram",
	}
	logBody, _ := json.Marshal(logPayload)
	reqLog := httptest.NewRequest(http.MethodPost, "/api/logs", bytes.NewReader(logBody))
	reqLog.Header.Set("Content-Type", "application/json")
	recLog := httptest.NewRecorder()
	handler.ServeHTTP(recLog, reqLog)

	if recLog.Code != http.StatusCreated {
		t.Fatalf("expected 201 for POST /api/logs, got %d", recLog.Code)
	}

	// 3. Test GET /api/items
	reqItems := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	recItems := httptest.NewRecorder()
	handler.ServeHTTP(recItems, reqItems)

	if recItems.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET /api/items, got %d", recItems.Code)
	}

	var items []ItemPayload
	json.Unmarshal(recItems.Body.Bytes(), &items)
	if len(items) != 2 {
		t.Fatalf("expected 2 items (1 note + 1 log), got %d", len(items))
	}

	// 4. Test GET /api/notes/{id}
	reqGet := httptest.NewRequest(http.MethodGet, "/api/notes/"+noteID[:7], nil)
	recGet := httptest.NewRecorder()
	handler.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET /api/notes/{id}, got %d: %s", recGet.Code, recGet.Body.String())
	}
	if !strings.Contains(recGet.Body.String(), "Distributed Architecture") {
		t.Errorf("expected note title in response: %s", recGet.Body.String())
	}

	// 5. Test PUT /api/notes/{id}
	updatedTitle := "Distributed Architecture v2"
	completedStatus := "completed"
	updatePayload := map[string]interface{}{
		"note":   updatedTitle,
		"status": completedStatus,
		"tags":   []string{"golang", "architecture"},
	}
	updateBody, _ := json.Marshal(updatePayload)
	reqPut := httptest.NewRequest(http.MethodPut, "/api/notes/"+noteID, bytes.NewReader(updateBody))
	reqPut.Header.Set("Content-Type", "application/json")
	recPut := httptest.NewRecorder()
	handler.ServeHTTP(recPut, reqPut)

	if recPut.Code != http.StatusOK {
		t.Fatalf("expected 200 for PUT /api/notes/{id}, got %d", recPut.Code)
	}

	// Verify update
	recGet2 := httptest.NewRecorder()
	handler.ServeHTTP(recGet2, reqGet)
	if !strings.Contains(recGet2.Body.String(), "Distributed Architecture v2") || !strings.Contains(recGet2.Body.String(), "completed") {
		t.Errorf("update was not persisted: %s", recGet2.Body.String())
	}

	// 6. Test GET /api/tags
	reqTags := httptest.NewRequest(http.MethodGet, "/api/tags", nil)
	recTags := httptest.NewRecorder()
	handler.ServeHTTP(recTags, reqTags)

	if recTags.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET /api/tags, got %d", recTags.Code)
	}
	var tagList []models.TagCount
	json.Unmarshal(recTags.Body.Bytes(), &tagList)
	if len(tagList) == 0 || tagList[0].Name != "architecture" && tagList[0].Name != "golang" {
		t.Errorf("unexpected tags response: %v", tagList)
	}

	// 7. Test GET /api/stats
	reqStats := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	recStats := httptest.NewRecorder()
	handler.ServeHTTP(recStats, reqStats)

	if recStats.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET /api/stats, got %d", recStats.Code)
	}
	if !strings.Contains(recStats.Body.String(), "total_active_notes") {
		t.Errorf("expected stats fields in response: %s", recStats.Body.String())
	}

	// 8. Test DELETE /api/notes/{id} (soft-delete)
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/notes/"+noteID, bytes.NewReader([]byte(`{"reason":"test delete"}`)))
	recDel := httptest.NewRecorder()
	handler.ServeHTTP(recDel, reqDel)

	if recDel.Code != http.StatusOK {
		t.Fatalf("expected 200 for DELETE /api/notes/{id}, got %d", recDel.Code)
	}

	// 9. Test POST /api/notes/{id}/restore
	reqRestore := httptest.NewRequest(http.MethodPost, "/api/notes/"+noteID+"/restore", nil)
	recRestore := httptest.NewRecorder()
	handler.ServeHTTP(recRestore, reqRestore)

	if recRestore.Code != http.StatusOK {
		t.Fatalf("expected 200 for POST /api/notes/{id}/restore, got %d", recRestore.Code)
	}
}
