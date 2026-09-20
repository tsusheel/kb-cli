package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/sync"
	"github.com/tsusheel/kb-cli/utils"
)

//go:embed web/*
var embeddedFiles embed.FS

// RegisterRoutes creates the HTTP request multiplexer and registers all API & static routes.
func RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("/api/items", handleGetItems)
	mux.HandleFunc("/api/notes", handleNotesCollection)
	mux.HandleFunc("/api/notes/", handleNoteResource)
	mux.HandleFunc("/api/logs", handleLogsCollection)
	mux.HandleFunc("/api/stats", handleGetStats)
	mux.HandleFunc("/api/tags", handleGetTags)
	mux.HandleFunc("/api/sync", handleSync)

	// Static Web Assets
	webFS, err := fs.Sub(embeddedFiles, "web")
	if err != nil {
		panic(fmt.Sprintf("failed to load embedded web assets: %v", err))
	}
	fileServer := http.FileServer(http.FS(webFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		// Serve static files
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(webFS, path); err != nil {
			// Fallback to index.html for SPA routing
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})

	return enableCORS(mux)
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func jsonResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func jsonError(w http.ResponseWriter, statusCode int, message string) {
	jsonResponse(w, statusCode, map[string]string{"error": message})
}

// ItemPayload represents a unified note or daily log item for web search
type ItemPayload struct {
	ID        string    `json:"id"`
	ShortID   string    `json:"short_id"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Area      string    `json:"area,omitempty"`
	Display   string    `json:"display"`
	Flesh     string    `json:"flesh,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	Timestamp string    `json:"timestamp"`
	IsLog     bool      `json:"is_log"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func handleGetItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	includeDeleted := strings.EqualFold(r.URL.Query().Get("include_deleted"), "true")

	// 1. Fetch tags map
	tagsMap, _ := db.GetAllNoteTagsMap()

	// 2. Fetch Notes
	notes, err := db.ListNotesExtended("", "", "", includeDeleted)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var items []ItemPayload
	for _, n := range notes {
		shortID := n.ID
		if len(shortID) > 7 {
			shortID = shortID[:7]
		}
		item := ItemPayload{
			ID:        n.ID,
			ShortID:   shortID,
			Type:      string(n.Type),
			Status:    string(n.Status),
			Area:      string(n.Area),
			Display:   n.Note,
			Flesh:     n.NoteFlesh,
			Tags:      tagsMap[n.ID],
			Timestamp: n.UpdatedAt.Format("2006-01-02 15:04"),
			IsLog:     false,
		}
		if !n.DeletedAt.IsZero() {
			item.DeletedAt = &n.DeletedAt
		}
		items = append(items, item)
	}

	// 3. Fetch Daily Logs
	logs, err := db.GetDailyLogsFilter(nil, nil, true, includeDeleted)
	if err == nil {
		for _, l := range logs {
			shortID := l.ID
			if len(shortID) > 7 {
				shortID = shortID[:7]
			}
			logType := "log"
			if l.NoteID != "" {
				logType = "log:promoted"
			}
			item := ItemPayload{
				ID:        l.ID,
				ShortID:   shortID,
				Type:      logType,
				Display:   l.Content,
				Timestamp: l.CreatedAt.Format("2006-01-02 15:04"),
				IsLog:     true,
			}
			if !l.DeletedAt.IsZero() {
				item.DeletedAt = &l.DeletedAt
			}
			items = append(items, item)
		}
	}

	if items == nil {
		items = []ItemPayload{}
	}

	jsonResponse(w, http.StatusOK, items)
}

func handleNotesCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Note       string   `json:"note"`
			Content    string   `json:"content"`
			Type       string   `json:"type"`
			Status     string   `json:"status"`
			Area       string   `json:"area"`
			Due        string   `json:"due"`
			Importance int      `json:"importance"`
			Clarity    int      `json:"clarity"`
			Tags       []string `json:"tags"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid request payload")
			return
		}

		if strings.TrimSpace(req.Note) == "" && strings.TrimSpace(req.Content) == "" {
			jsonError(w, http.StatusBadRequest, "title or content is required")
			return
		}

		if req.Type == "" {
			req.Type = string(models.DefaultNote)
		}
		if req.Status == "" {
			req.Status = string(models.Active)
		}

		var targetDT time.Time
		if req.Due != "" {
			if parsed, err := utils.ParseDate(req.Due); err == nil {
				targetDT = parsed
			}
		}

		id := strings.ReplaceAll(uuid.New().String(), "-", "")
		n := &models.Note{
			ID:             id,
			Note:           req.Note,
			NoteFlesh:      req.Content,
			Type:           models.NoteType(req.Type),
			Status:         models.Status(req.Status),
			Area:           models.Area(req.Area),
			Importance:     req.Importance,
			Clarity:        req.Clarity,
			Source:         "Web Finder",
			TargetDateTime: targetDT,
		}

		if err := db.CreateNote(n); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}

		for _, tag := range req.Tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				_ = db.AddTag(n.ID, tag)
			}
		}

		jsonResponse(w, http.StatusCreated, map[string]interface{}{
			"success":  true,
			"id":       n.ID,
			"short_id": n.ID[:7],
		})

	default:
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleNoteResource(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/notes/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		jsonError(w, http.StatusBadRequest, "note ID is required")
		return
	}

	noteID := parts[0]

	// Sub-actions: /api/notes/{id}/restore
	if len(parts) > 1 && parts[1] == "restore" {
		if r.Method != http.MethodPost {
			jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		n, err := db.RestoreNote(noteID)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"note":    n,
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		n, err := db.GetNote(noteID)
		if err != nil {
			jsonError(w, http.StatusNotFound, "note not found")
			return
		}

		tags, _ := db.GetTagsForNote(n.ID)
		links, _ := db.GetLinksForNote(n.ID)
		history, _ := db.GetAuditHistory("note", n.ID, 10)

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
			"id":          n.ID,
			"short_id":    n.ID[:7],
			"note":        n.Note,
			"note_flesh":  n.NoteFlesh,
			"type":        n.Type,
			"status":      n.Status,
			"area":        n.Area,
			"importance":  n.Importance,
			"clarity":     n.Clarity,
			"source":      n.Source,
			"created_at":  n.CreatedAt,
			"updated_at":  n.UpdatedAt,
			"tags":        tagNames,
			"links":       linkDetails,
			"history":     history,
		}
		if !n.TargetDateTime.IsZero() {
			res["target_date_time"] = n.TargetDateTime
		}
		if !n.DeletedAt.IsZero() {
			res["deleted_at"] = n.DeletedAt
			res["deleted_note"] = n.DeletedNote
		}

		jsonResponse(w, http.StatusOK, res)

	case http.MethodPut:
		n, err := db.GetNote(noteID)
		if err != nil {
			jsonError(w, http.StatusNotFound, "note not found")
			return
		}

		var req struct {
			Note       *string  `json:"note"`
			Content    *string  `json:"content"`
			Type       *string  `json:"type"`
			Status     *string  `json:"status"`
			Area       *string  `json:"area"`
			Due        *string  `json:"due"`
			Importance *int     `json:"importance"`
			Clarity    *int     `json:"clarity"`
			Tags       []string `json:"tags,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid update payload")
			return
		}

		if req.Note != nil {
			n.Note = *req.Note
		}
		if req.Content != nil {
			n.NoteFlesh = *req.Content
		}
		if req.Type != nil {
			n.Type = models.NoteType(*req.Type)
		}
		if req.Status != nil {
			n.Status = models.Status(*req.Status)
		}
		if req.Area != nil {
			n.Area = models.Area(*req.Area)
		}
		if req.Importance != nil {
			n.Importance = *req.Importance
		}
		if req.Clarity != nil {
			n.Clarity = *req.Clarity
		}
		if req.Due != nil {
			dueStr := strings.TrimSpace(*req.Due)
			if strings.EqualFold(dueStr, "clear") || dueStr == "" {
				n.TargetDateTime = time.Time{}
			} else {
				if parsed, err := utils.ParseDate(dueStr); err == nil {
					n.TargetDateTime = parsed
				}
			}
		}

		if err := db.UpdateNote(n); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Update tags if provided
		if req.Tags != nil {
			existingTags, _ := db.GetTagsForNote(n.ID)
			existingMap := make(map[string]bool)
			for _, t := range existingTags {
				existingMap[t.Name] = true
			}

			newMap := make(map[string]bool)
			for _, t := range req.Tags {
				t = strings.TrimSpace(t)
				if t != "" {
					newMap[t] = true
					if !existingMap[t] {
						_ = db.AddTag(n.ID, t)
					}
				}
			}

			for _, t := range existingTags {
				if !newMap[t.Name] {
					_ = db.RemoveTag(n.ID, t.Name)
				}
			}
		}

		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"id":      n.ID,
		})

	case http.MethodDelete:
		var req struct {
			Reason string `json:"reason"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		reason := req.Reason
		if reason == "" {
			reason = "deleted from web UI"
		}

		if err := db.SoftDeleteNote(noteID, reason); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}

		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"id":      noteID,
		})

	default:
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleLogsCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Content) == "" {
		jsonError(w, http.StatusBadRequest, "content is required")
		return
	}

	log, err := db.CreateDailyLog(req.Content)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"success":  true,
		"id":       log.ID,
		"short_id": log.ID[:7],
	})
}

func handleGetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	kmap, err := db.GetKnowledgeMap()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, kmap)
}

func handleGetTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := `
		SELECT t.name, COUNT(nt.note_id) as count
		FROM tags t
		JOIN note_tags nt ON t.id = nt.tag_id
		JOIN notes n ON nt.note_id = n.id AND n.deleted_at IS NULL
		GROUP BY t.id, t.name
		ORDER BY count DESC
	`
	rows, err := db.DB.Query(query)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var tags []models.TagCount
	for rows.Next() {
		var tc models.TagCount
		if err := rows.Scan(&tc.Name, &tc.Count); err == nil {
			tags = append(tags, tc)
		}
	}
	if tags == nil {
		tags = []models.TagCount{}
	}

	jsonResponse(w, http.StatusOK, tags)
}

func handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !utils.IsRemoteEnabled() {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"remote":  false,
			"message": "Local only (remote sync is not enabled in config)",
		})
		return
	}

	rawURL := utils.GetPostgresURL()
	if rawURL == "" {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"remote":  false,
			"message": "Local only (no remote database URL configured)",
		})
		return
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		jsonError(w, http.StatusBadRequest, fmt.Sprintf("invalid postgres URL: %v", err))
		return
	}

	if parsed.User != nil {
		if _, hasPass := parsed.User.Password(); !hasPass {
			if secretPass, err := utils.GetSecret("postgres_password"); err == nil && secretPass != "" {
				parsed.User = url.UserPassword(parsed.User.Username(), secretPass)
			}
		}
	}

	client, err := sync.NewPostgresClient(parsed.String())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("failed to connect to remote database: %v", err))
		return
	}
	defer client.Close()

	stats, err := client.TwoWaySync()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("sync failed: %v", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"remote":  true,
		"stats":   stats,
		"message": fmt.Sprintf("Synced: %d pushed, %d pulled (%s)", stats.NotesPushed+stats.LogsPushed, stats.NotesPulled+stats.LogsPulled, stats.Duration.Round(time.Millisecond)),
	})
}
