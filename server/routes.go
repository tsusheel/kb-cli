package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/docs"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/sync"
	"github.com/tsusheel/kb-cli/utils"
)

//go:generate npm run build
//go:embed web/*
var embeddedFiles embed.FS

// RegisterRoutes creates the HTTP request multiplexer and registers all API & static routes.
func RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	// Auth Routes
	mux.HandleFunc("/api/auth/login", handleAuthLogin)
	mux.HandleFunc("/api/auth/status", handleAuthStatus)
	mux.HandleFunc("/api/auth/logout", handleAuthLogout)

	// Protected API Routes
	mux.HandleFunc("/api/items", requireAuth(handleGetItems))
	mux.HandleFunc("/api/notes", requireAuth(handleNotesCollection))
	mux.HandleFunc("/api/notes/", requireAuth(handleNoteResource))
	mux.HandleFunc("/api/logs", requireAuth(handleLogsCollection))
	mux.HandleFunc("/api/stats", requireAuth(handleGetStats))
	mux.HandleFunc("/api/tags", requireAuth(handleGetTags))
	mux.HandleFunc("/api/sync", requireAuth(handleSync))

	// Plugin API Routes
	mux.HandleFunc("/api/plugins", requireAuth(handlePluginList))
	mux.HandleFunc("/api/plugins/", requireAuth(handlePluginDispatch))

	// Documentation Routes
	mux.HandleFunc("/api/docs", handleDocsDispatch)
	mux.HandleFunc("/api/docs/content", handleDocContent)
	mux.HandleFunc("/api/docs/search", handleSearchDocs)
	mux.HandleFunc("/api/docs/folders", requireAuth(handleCreateDocFolder))
	mux.HandleFunc("/api/docs/import-tree", requireAuth(handleImportDocsTree))

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
		if path == "manifest.webmanifest" || path == "site.webmanifest" {
			r.URL.Path = "/assets/site.webmanifest"
			path = "assets/site.webmanifest"
		} else if path == "favicon.ico" {
			r.URL.Path = "/assets/favicon.ico"
			path = "assets/favicon.ico"
		}
		if _, err := fs.Stat(webFS, path); err != nil {
			// Fallback to index.html for SPA routing
			r.URL.Path = "/"
		}
		if strings.HasSuffix(path, "manifest.json") || strings.HasSuffix(path, ".webmanifest") {
			w.Header().Set("Content-Type", "application/manifest+json")
		} else if strings.HasSuffix(path, ".js") {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			if path == "sw.js" {
				w.Header().Set("Service-Worker-Allowed", "/")
			}
		} else if strings.HasSuffix(path, ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		} else if strings.HasSuffix(path, ".wasm") {
			w.Header().Set("Content-Type", "application/wasm")
		}
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.Header().Set("Cloudflare-CDN-Cache-Control", "no-store")
		w.Header().Set("CDN-Cache-Control", "no-store")
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

// ItemPayload represents a unified note, daily log, or documentation item for web search
type ItemPayload struct {
	ID        string     `json:"id"`
	ShortID   string     `json:"short_id"`
	Type      string     `json:"type"`
	Status    string     `json:"status"`
	Area      string     `json:"area,omitempty"`
	Display   string     `json:"display"`
	Flesh     string     `json:"flesh,omitempty"`
	Tags      []string   `json:"tags,omitempty"`
	Timestamp string     `json:"timestamp"`
	IsLog     bool       `json:"is_log"`
	IsDoc     bool       `json:"is_doc"`
	DocPath   string     `json:"doc_path,omitempty"`
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
			IsDoc:     false,
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
				IsDoc:     false,
			}
			if !l.DeletedAt.IsZero() {
				item.DeletedAt = &l.DeletedAt
			}
			items = append(items, item)
		}
	}

	// 4. Fetch Active Docs
	docs, err := db.GetActiveDocsList()
	if err == nil {
		for _, d := range docs {
			shortID := d.ID
			if len(shortID) > 7 {
				shortID = shortID[:7]
			}
			folderArea := path.Dir(d.Path)
			if folderArea == "." || folderArea == "/" {
				folderArea = "docs"
			}
			item := ItemPayload{
				ID:        d.ID,
				ShortID:   shortID,
				Type:      "doc",
				Status:    "active",
				Area:      folderArea,
				Display:   d.Title,
				Flesh:     d.Content,
				Tags:      d.Tags,
				Timestamp: d.UpdatedAt.Format("2006-01-02 15:04"),
				IsLog:     false,
				IsDoc:     true,
				DocPath:   d.Path,
			}
			if d.DeletedAt != nil {
				item.DeletedAt = d.DeletedAt
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

	scope := strings.ToLower(r.URL.Query().Get("scope"))
	if scope == "docs" {
		tags, err := db.GetDocsTagsSummary()
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, tags)
		return
	}

	if scope == "notes" {
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
		return
	}

	// Default/all: combine notes and docs tags
	query := `
		SELECT t.name, (
			COALESCE((SELECT COUNT(*) FROM note_tags nt JOIN notes n ON nt.note_id = n.id AND n.deleted_at IS NULL WHERE nt.tag_id = t.id), 0) +
			COALESCE((SELECT COUNT(*) FROM doc_tags dt JOIN doc_nodes dn ON dt.doc_id = dn.id AND dn.deleted_at IS NULL WHERE dt.tag_id = t.id), 0)
		) as total_count
		FROM tags t
		WHERE (
			EXISTS (SELECT 1 FROM note_tags nt JOIN notes n ON nt.note_id = n.id AND n.deleted_at IS NULL WHERE nt.tag_id = t.id)
			OR EXISTS (SELECT 1 FROM doc_tags dt JOIN doc_nodes dn ON dt.doc_id = dn.id AND dn.deleted_at IS NULL WHERE dt.tag_id = t.id)
		)
		ORDER BY total_count DESC
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

func handleDocsDispatch(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetDocsTree(w, r)
	case http.MethodPost:
		requireAuth(handleCreateDoc)(w, r)
	case http.MethodDelete:
		requireAuth(handleDeleteDoc)(w, r)
	default:
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleDocContent(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetDocContent(w, r)
	case http.MethodPut:
		requireAuth(handleUpdateDocContent)(w, r)
	default:
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleGetDocsTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tree, err := db.GetDocsTree()
	if err != nil || len(tree) == 0 {
		_ = db.SeedDocsFromFS(docs.FS)
		tree, err = db.GetDocsTree()
	}
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get docs tree: %v", err))
		return
	}
	jsonResponse(w, http.StatusOK, tree)
}

func handleGetDocContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	docPath := r.URL.Query().Get("path")
	if docPath == "" {
		docPath = "README.md"
	}

	node, err := db.GetDocNodeByPath(docPath)
	if err == nil {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"id":         node.ID,
			"path":       node.Path,
			"title":      node.Title,
			"content":    node.Content,
			"tags":       node.Tags,
			"updated_at": node.UpdatedAt,
		})
		return
	}

	// Fallback to static embed
	content, err := docs.ReadDocFile(docPath)
	if err != nil {
		jsonError(w, http.StatusNotFound, fmt.Sprintf("doc not found: %v", err))
		return
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"path":    docPath,
		"content": content,
		"tags":    []string{},
	})
}

func handleCreateDoc(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Title    string   `json:"title"`
		Path     string   `json:"path"`
		ParentID *string  `json:"parent_id"`
		Content  string   `json:"content"`
		Tags     []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(payload.Path) == "" {
		jsonError(w, http.StatusBadRequest, "path is required")
		return
	}
	if strings.TrimSpace(payload.Title) == "" {
		payload.Title = path.Base(payload.Path)
	}

	node := &db.DocNode{
		ParentID: payload.ParentID,
		Type:     "doc",
		Path:     payload.Path,
		Title:    payload.Title,
		Content:  payload.Content,
		Tags:     payload.Tags,
	}
	if err := db.CreateDocNode(node); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, node)
}

func handleUpdateDocContent(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ID      string   `json:"id"`
		Path    string   `json:"path"`
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Tags    []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	var node *db.DocNode
	var err error
	if payload.ID != "" {
		node, err = db.GetDocNodeByID(payload.ID)
	} else if payload.Path != "" {
		node, err = db.GetDocNodeByPath(payload.Path)
	} else {
		jsonError(w, http.StatusBadRequest, "either id or path must be provided")
		return
	}
	if err != nil {
		jsonError(w, http.StatusNotFound, "doc not found")
		return
	}

	if payload.Title != "" {
		node.Title = payload.Title
	}
	node.Content = payload.Content
	if payload.Tags != nil {
		node.Tags = payload.Tags
	}
	if err := db.UpdateDocNode(node); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, node)
}

func handleDeleteDoc(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	docPath := r.URL.Query().Get("path")
	if id == "" && docPath != "" {
		node, err := db.GetDocNodeByPath(docPath)
		if err == nil {
			id = node.ID
		}
	}
	if id == "" {
		jsonError(w, http.StatusBadRequest, "id or path is required")
		return
	}
	reason := r.URL.Query().Get("reason")
	if reason == "" {
		reason = "deleted from web UI"
	}
	if err := db.DeleteDocNode(id, reason); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"success": true, "id": id})
}

func handleCreateDocFolder(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Path     string  `json:"path"`
		Title    string  `json:"title"`
		ParentID *string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(payload.Path) == "" {
		jsonError(w, http.StatusBadRequest, "folder path is required")
		return
	}
	folderName := path.Base(payload.Path)
	if strings.TrimSpace(payload.Title) == "" {
		payload.Title = folderName
	}
	node := &db.DocNode{
		ParentID: payload.ParentID,
		Type:     "folder",
		Path:     payload.Path,
		Name:     folderName,
		Title:    payload.Title,
	}
	if err := db.CreateDocNode(node); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, node)
}

func handleImportDocsTree(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Items     []db.BatchDocImportItem `json:"items"`
		Overwrite bool                    `json:"overwrite"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if len(payload.Items) == 0 {
		jsonError(w, http.StatusBadRequest, "items list is empty")
		return
	}

	created, updated, err := db.BatchImportDocNodes(payload.Items, payload.Overwrite)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("batch import failed: %v", err))
		return
	}

	newTree, _ := db.GetDocsTree()
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"created": created,
		"updated": updated,
		"tree":    newTree,
	})
}

func handleSearchDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	query := r.URL.Query().Get("q")
	if strings.TrimSpace(query) == "" {
		jsonResponse(w, http.StatusOK, []interface{}{})
		return
	}
	results, err := db.SearchDocs(query)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, results)
}

func handlePluginList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	records, err := db.ListPluginRecords()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, records)
}

func handlePluginDispatch(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/plugins/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	pluginID := parts[0]
	if len(parts) >= 2 && parts[1] == "entries" {
		if len(parts) == 2 {
			switch r.Method {
			case http.MethodGet:
				filter := models.EntryFilter{
					PluginID:   pluginID,
					EntityType: r.URL.Query().Get("type"),
					Status:     r.URL.Query().Get("status"),
					SearchText: r.URL.Query().Get("search"),
				}
				if date := r.URL.Query().Get("date"); date != "" {
					filter.EntryDate = &date
				}
				if parentID := r.URL.Query().Get("parent_id"); parentID != "" {
					filter.ParentID = &parentID
				}
				entries, err := db.QueryEntries(filter)
				if err != nil {
					jsonError(w, http.StatusInternalServerError, err.Error())
					return
				}
				jsonResponse(w, http.StatusOK, entries)
			case http.MethodPost:
				var entry models.PluginEntry
				if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
					jsonError(w, http.StatusBadRequest, "invalid json body")
					return
				}
				entry.PluginID = pluginID
				actor := models.DefaultUserActor()
				if err := db.CreateEntry(nil, &entry, actor); err != nil {
					jsonError(w, http.StatusInternalServerError, err.Error())
					return
				}
				jsonResponse(w, http.StatusCreated, entry)
			default:
				jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		} else if len(parts) == 3 {
			entryID := parts[2]
			switch r.Method {
			case http.MethodGet:
				entry, err := db.GetEntry(pluginID, entryID)
				if err != nil {
					if err == db.ErrEntryNotFound {
						http.NotFound(w, r)
					} else {
						jsonError(w, http.StatusInternalServerError, err.Error())
					}
					return
				}
				jsonResponse(w, http.StatusOK, entry)
			case http.MethodPut:
				var entry models.PluginEntry
				if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
					jsonError(w, http.StatusBadRequest, "invalid json body")
					return
				}
				entry.PluginID = pluginID
				entry.ID = entryID
				actor := models.DefaultUserActor()
				if err := db.UpdateEntry(nil, &entry, 0, actor); err != nil {
					jsonError(w, http.StatusInternalServerError, err.Error())
					return
				}
				jsonResponse(w, http.StatusOK, entry)
			case http.MethodDelete:
				reason := r.URL.Query().Get("reason")
				if reason == "" {
					reason = "deleted via web API"
				}
				actor := models.DefaultUserActor()
				if err := db.SoftDeleteEntry(nil, pluginID, entryID, reason, actor); err != nil {
					jsonError(w, http.StatusInternalServerError, err.Error())
					return
				}
				jsonResponse(w, http.StatusOK, map[string]bool{"deleted": true})
			default:
				jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		}
	}

	http.NotFound(w, r)
}


