package db

import (
	"bufio"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tsusheel/kb-cli/models"
)

var (
	ErrDocNotFound = errors.New("documentation node not found")
)

// DocNode represents a folder or document record in doc_nodes.
type DocNode struct {
	ID          string     `json:"id"`
	ParentID    *string    `json:"parent_id,omitempty"`
	Type        string     `json:"type"` // "folder" or "doc"
	Name        string     `json:"name"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Path        string     `json:"path"`
	Content     string     `json:"content,omitempty"`
	ContentType string     `json:"content_type"`
	OrderIndex  int        `json:"order_index"`
	Tags        []string   `json:"tags,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	DeletedNote *string    `json:"deleted_note,omitempty"`
}

// DocTreeItem represents a node in the hierarchical documentation tree.
// Backwards compatible with the web UI and CLI expectations.
type DocTreeItem struct {
	ID       string        `json:"id"`
	ParentID *string       `json:"parent_id,omitempty"`
	Name     string        `json:"name"`
	Title    string        `json:"title"`
	Path     string        `json:"path"`
	Type     string        `json:"type"` // "directory" or "file"
	Tags     []string      `json:"tags,omitempty"`
	Children []DocTreeItem `json:"children,omitempty"`
}

// BatchDocImportItem represents a single item to import in batch operations.
type BatchDocImportItem struct {
	Path    string `json:"path"`
	Type    string `json:"type"` // "folder" or "doc"
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
}

func generateDocID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "doc-" + hex.EncodeToString(b)
}

// NormalizeDocPath cleans and standardizes a document or folder path with forward slashes.
func NormalizeDocPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(p, "/")
	p = strings.TrimSuffix(p, "/")
	p = path.Clean(p)
	if p == "." {
		return ""
	}
	return p
}

var nonAlphaNumRegex = regexp.MustCompile(`[^a-zA-Z0-9\-_]+`)

// Slugify converts text into a clean URL-friendly slug.
func Slugify(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	text = strings.ReplaceAll(text, ".md", "")
	text = nonAlphaNumRegex.ReplaceAllString(text, "-")
	text = strings.Trim(text, "-")
	if text == "" {
		return "doc"
	}
	return text
}

// CreateDocNode inserts a new folder or document node into the database.
func CreateDocNode(node *DocNode) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	if node.ID == "" {
		node.ID = generateDocID()
	}
	node.Path = NormalizeDocPath(node.Path)
	if node.Path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if node.Name == "" {
		node.Name = path.Base(node.Path)
	}
	if node.Slug == "" {
		node.Slug = Slugify(node.Name)
	}
	if node.Title == "" {
		node.Title = node.Name
	}
	if node.Type != "folder" && node.Type != "doc" {
		node.Type = "doc"
	}
	if node.ContentType == "" {
		node.ContentType = "text/markdown"
	}

	now := time.Now().UTC()
	if node.CreatedAt.IsZero() {
		node.CreatedAt = now
	}
	if node.UpdatedAt.IsZero() {
		node.UpdatedAt = now
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// If parent ID is not explicitly set, auto-resolve/create ancestor folders
	if node.ParentID == nil || *node.ParentID == "" {
		parentDir := path.Dir(node.Path)
		if parentDir != "." && parentDir != "/" && parentDir != "" {
			pid, err := ensureFolderTx(tx, parentDir, nil, now, nil)
			if err != nil {
				return err
			}
			if pid != "" {
				node.ParentID = &pid
			}
		}
	}

	query := `INSERT INTO doc_nodes (
		id, parent_id, type, name, title, slug, path, content, content_type, order_index, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.Exec(query,
		node.ID,
		node.ParentID,
		node.Type,
		node.Name,
		node.Title,
		node.Slug,
		node.Path,
		node.Content,
		node.ContentType,
		node.OrderIndex,
		node.CreatedAt,
		node.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed inserting doc node: %w", err)
	}

	if len(node.Tags) > 0 {
		if err := SetTagsForDocTx(tx, node.ID, node.Tags); err != nil {
			return fmt.Errorf("failed setting tags for doc: %w", err)
		}
	}

	if node.Type == "doc" {
		_, _ = tx.Exec(`INSERT INTO docs_fts (doc_id, path, title, content) VALUES (?, ?, ?, ?)`,
			node.ID, node.Path, node.Title, node.Content)
	}

	return tx.Commit()
}

// GetAllDocTagsMap fetches a map of Doc ID -> slice of Tag Names in a single SQL query.
func GetAllDocTagsMap() (map[string][]string, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT dt.doc_id, t.name
		FROM doc_tags dt
		JOIN tags t ON dt.tag_id = t.id
		ORDER BY dt.created_at ASC
	`
	rows, err := DB.Query(query)
	if err != nil {
		return make(map[string][]string), nil
	}
	defer rows.Close()

	tagsMap := make(map[string][]string)
	for rows.Next() {
		var docID, tagName string
		if err := rows.Scan(&docID, &tagName); err == nil {
			tagsMap[docID] = append(tagsMap[docID], tagName)
		}
	}
	return tagsMap, nil
}

// GetTagsForDoc retrieves all tag names assigned to a document.
func GetTagsForDoc(docID string) ([]string, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT t.name
		FROM doc_tags dt
		JOIN tags t ON dt.tag_id = t.id
		WHERE dt.doc_id = ?
		ORDER BY dt.created_at ASC
	`
	rows, err := DB.Query(query, docID)
	if err != nil {
		return []string{}, nil
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tName string
		if err := rows.Scan(&tName); err == nil {
			tags = append(tags, tName)
		}
	}
	return tags, nil
}

// SetTagsForDocTx synchronizes tags for a document inside an active transaction.
func SetTagsForDocTx(tx *sql.Tx, docID string, tags []string) error {
	_, err := tx.Exec(`DELETE FROM doc_tags WHERE doc_id = ?`, docID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, rawTag := range tags {
		tag := strings.TrimSpace(rawTag)
		tag = strings.TrimPrefix(tag, "#")
		if tag == "" {
			continue
		}

		var tagID string
		err = tx.QueryRow(`SELECT id FROM tags WHERE name = ?`, tag).Scan(&tagID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				tagID = generateDocID()
				_, err = tx.Exec(`INSERT INTO tags (id, name, created_at) VALUES (?, ?, ?)`, tagID, tag, now)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		_, err = tx.Exec(`INSERT OR IGNORE INTO doc_tags (doc_id, tag_id, created_at) VALUES (?, ?, ?)`, docID, tagID, now)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetTagsForDoc updates the tags for a document.
func SetTagsForDoc(docID string, tags []string) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := SetTagsForDocTx(tx, docID, tags); err != nil {
		return err
	}
	return tx.Commit()
}

// GetDocsTagsSummary returns all tags present on docs with their document counts.
func GetDocsTagsSummary() ([]models.TagCount, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT t.name, COUNT(dt.doc_id) as count
		FROM tags t
		JOIN doc_tags dt ON t.id = dt.tag_id
		JOIN doc_nodes dn ON dt.doc_id = dn.id AND dn.deleted_at IS NULL
		GROUP BY t.id, t.name
		ORDER BY count DESC
	`
	rows, err := DB.Query(query)
	if err != nil {
		return []models.TagCount{}, nil
	}
	defer rows.Close()

	var tags []models.TagCount
	for rows.Next() {
		var tc models.TagCount
		if err := rows.Scan(&tc.Name, &tc.Count); err == nil {
			tags = append(tags, tc)
		}
	}
	return tags, nil
}

// GetActiveDocsList returns all active document nodes (type = 'doc' and not deleted).
func GetActiveDocsList() ([]*DocNode, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	docTagsMap, _ := GetAllDocTagsMap()

	query := `SELECT id, parent_id, type, name, title, slug, path, content, content_type, order_index, created_at, updated_at, deleted_at, deleted_note
		FROM doc_nodes
		WHERE type = 'doc' AND deleted_at IS NULL
		ORDER BY updated_at DESC`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*DocNode
	for rows.Next() {
		var node DocNode
		var parentID, content, deletedNote sql.NullString
		var deletedAt sql.NullTime

		err := rows.Scan(
			&node.ID,
			&parentID,
			&node.Type,
			&node.Name,
			&node.Title,
			&node.Slug,
			&node.Path,
			&content,
			&node.ContentType,
			&node.OrderIndex,
			&node.CreatedAt,
			&node.UpdatedAt,
			&deletedAt,
			&deletedNote,
		)
		if err != nil {
			return nil, err
		}
		if parentID.Valid {
			node.ParentID = &parentID.String
		}
		if content.Valid {
			node.Content = content.String
		}
		if deletedAt.Valid {
			node.DeletedAt = &deletedAt.Time
		}
		if deletedNote.Valid {
			node.DeletedNote = &deletedNote.String
		}
		node.Tags = docTagsMap[node.ID]
		docs = append(docs, &node)
	}
	return docs, nil
}

// GetDocNodeByID retrieves a document node by its primary ID.
func GetDocNodeByID(id string) (*DocNode, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `SELECT id, parent_id, type, name, title, slug, path, content, content_type, order_index, created_at, updated_at, deleted_at, deleted_note
		FROM doc_nodes WHERE id = ? AND deleted_at IS NULL`

	var node DocNode
	var parentID, content, deletedNote sql.NullString
	var deletedAt sql.NullTime

	err := DB.QueryRow(query, id).Scan(
		&node.ID,
		&parentID,
		&node.Type,
		&node.Name,
		&node.Title,
		&node.Slug,
		&node.Path,
		&content,
		&node.ContentType,
		&node.OrderIndex,
		&node.CreatedAt,
		&node.UpdatedAt,
		&deletedAt,
		&deletedNote,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDocNotFound
		}
		return nil, err
	}

	if parentID.Valid {
		node.ParentID = &parentID.String
	}
	if content.Valid {
		node.Content = content.String
	}
	if deletedAt.Valid {
		node.DeletedAt = &deletedAt.Time
	}
	if deletedNote.Valid {
		node.DeletedNote = &deletedNote.String
	}

	node.Tags, _ = GetTagsForDoc(node.ID)
	return &node, nil
}

// GetDocNodeByPath retrieves an active document node by normalized path.
func GetDocNodeByPath(docPath string) (*DocNode, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	normPath := NormalizeDocPath(docPath)
	query := `SELECT id, parent_id, type, name, title, slug, path, content, content_type, order_index, created_at, updated_at, deleted_at, deleted_note
		FROM doc_nodes WHERE path = ? AND deleted_at IS NULL`

	var node DocNode
	var parentID, content, deletedNote sql.NullString
	var deletedAt sql.NullTime

	err := DB.QueryRow(query, normPath).Scan(
		&node.ID,
		&parentID,
		&node.Type,
		&node.Name,
		&node.Title,
		&node.Slug,
		&node.Path,
		&content,
		&node.ContentType,
		&node.OrderIndex,
		&node.CreatedAt,
		&node.UpdatedAt,
		&deletedAt,
		&deletedNote,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if !strings.HasPrefix(normPath, "docs/") && !strings.HasPrefix(normPath, "proposals/") {
				if fallbackNode, fallbackErr := GetDocNodeByPath("docs/" + normPath); fallbackErr == nil {
					return fallbackNode, nil
				}
			}
			return nil, ErrDocNotFound
		}
		return nil, err
	}

	if parentID.Valid {
		node.ParentID = &parentID.String
	}
	if content.Valid {
		node.Content = content.String
	}
	if deletedAt.Valid {
		node.DeletedAt = &deletedAt.Time
	}
	if deletedNote.Valid {
		node.DeletedNote = &deletedNote.String
	}

	node.Tags, _ = GetTagsForDoc(node.ID)
	return &node, nil
}

// UpdateDocNode updates title, content, path, or parent of a document node.
func UpdateDocNode(node *DocNode) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	node.Path = NormalizeDocPath(node.Path)
	if node.Path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if node.Name == "" {
		node.Name = path.Base(node.Path)
	}
	if node.Slug == "" {
		node.Slug = Slugify(node.Name)
	}
	node.UpdatedAt = time.Now().UTC()

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE doc_nodes SET 
		parent_id = ?, name = ?, title = ?, slug = ?, path = ?, content = ?, order_index = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`

	res, err := tx.Exec(query,
		node.ParentID,
		node.Name,
		node.Title,
		node.Slug,
		node.Path,
		node.Content,
		node.OrderIndex,
		node.UpdatedAt,
		node.ID,
	)
	if err != nil {
		return fmt.Errorf("failed updating doc node: %w", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrDocNotFound
	}

	if node.Tags != nil {
		if err := SetTagsForDocTx(tx, node.ID, node.Tags); err != nil {
			return fmt.Errorf("failed updating tags for doc: %w", err)
		}
	}

	// Update full-text search
	_, _ = tx.Exec(`DELETE FROM docs_fts WHERE doc_id = ?`, node.ID)
	if node.Type == "doc" {
		_, _ = tx.Exec(`INSERT INTO docs_fts (doc_id, path, title, content) VALUES (?, ?, ?, ?)`,
			node.ID, node.Path, node.Title, node.Content)
	}

	return tx.Commit()
}

// DeleteDocNode soft-deletes a document or folder (recursively soft-deleting any children).
func DeleteDocNode(id string, reason string) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	if reason == "" {
		reason = "deleted by user"
	}

	// Recursive CTE to collect target node and all sub-children
	cte := `
	WITH RECURSIVE subtree AS (
		SELECT id FROM doc_nodes WHERE id = ?
		UNION ALL
		SELECT d.id FROM doc_nodes d
		JOIN subtree s ON d.parent_id = s.id
	)
	UPDATE doc_nodes SET deleted_at = ?, deleted_note = ?
	WHERE id IN (SELECT id FROM subtree) AND deleted_at IS NULL;
	`

	res, err := tx.Exec(cte, id, now, reason)
	if err != nil {
		return fmt.Errorf("failed deleting doc node: %w", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrDocNotFound
	}

	// Clean FTS for the deleted nodes
	_, _ = tx.Exec(`DELETE FROM docs_fts WHERE doc_id IN (
		WITH RECURSIVE subtree AS (
			SELECT id FROM doc_nodes WHERE id = ?
			UNION ALL
			SELECT d.id FROM doc_nodes d JOIN subtree s ON d.parent_id = s.id
		) SELECT id FROM subtree
	)`, id)

	return tx.Commit()
}

// GetDocsTree returns the complete hierarchical documentation tree.
func GetDocsTree() ([]DocTreeItem, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	docTagsMap, _ := GetAllDocTagsMap()

	query := `SELECT id, parent_id, type, name, title, path, order_index 
		FROM doc_nodes 
		WHERE deleted_at IS NULL 
		ORDER BY order_index ASC, name ASC`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type rawNode struct {
		id       string
		parentID *string
		item     DocTreeItem
	}

	var allNodes []rawNode
	childrenMap := make(map[string][]DocTreeItem)

	for rows.Next() {
		var id, nodeType, name, title, docPath string
		var parentID sql.NullString
		var orderIndex int

		if err := rows.Scan(&id, &parentID, &nodeType, &name, &title, &docPath, &orderIndex); err != nil {
			return nil, err
		}

		itemType := "file"
		if nodeType == "folder" {
			itemType = "directory"
		}

		var pid *string
		if parentID.Valid {
			s := parentID.String
			pid = &s
		}

		item := DocTreeItem{
			ID:       id,
			ParentID: pid,
			Name:     name,
			Title:    title,
			Path:     docPath,
			Type:     itemType,
			Tags:     docTagsMap[id],
			Children: []DocTreeItem{},
		}

		allNodes = append(allNodes, rawNode{id: id, parentID: pid, item: item})
	}

	// Index folder paths to IDs for fallback hierarchy matching
	folderPathToID := make(map[string]string)
	for _, n := range allNodes {
		if n.item.Type == "directory" {
			folderPathToID[n.item.Path] = n.id
		}
	}

	// Group items by parent ID (with fallback to folder path if parentID is missing)
	for i := range allNodes {
		if allNodes[i].parentID == nil && strings.Contains(allNodes[i].item.Path, "/") {
			parentDir := path.Dir(allNodes[i].item.Path)
			if pID, found := folderPathToID[parentDir]; found {
				allNodes[i].parentID = &pID
				allNodes[i].item.ParentID = &pID
			}
		}
		pKey := ""
		if allNodes[i].parentID != nil {
			pKey = *allNodes[i].parentID
		}
		childrenMap[pKey] = append(childrenMap[pKey], allNodes[i].item)
	}

	// Recursive assembly function
	var buildTree func(parentKey string) []DocTreeItem
	buildTree = func(parentKey string) []DocTreeItem {
		children, exists := childrenMap[parentKey]
		if !exists {
			return []DocTreeItem{}
		}

		// Sort: directories first (or README first), then alphabetical
		sort.SliceStable(children, func(i, j int) bool {
			if children[i].Name == "README.md" {
				return true
			}
			if children[j].Name == "README.md" {
				return false
			}
			if children[i].Type != children[j].Type {
				return children[i].Type == "directory"
			}
			return children[i].Name < children[j].Name
		})

		for i := range children {
			if children[i].Type == "directory" {
				children[i].Children = buildTree(children[i].ID)
			}
		}
		return children
	}

	return buildTree(""), nil
}

// SearchDocs performs full-text search across documentation paths, titles, and contents.
func SearchDocs(query string) ([]DocNode, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return nil, nil
	}

	// Escape FTS special chars
	ftsTerm := strings.ReplaceAll(cleanQuery, `"`, `""`)
	ftsQuery := `SELECT d.id, d.parent_id, d.type, d.name, d.title, d.slug, d.path, d.content, d.content_type, d.order_index, d.created_at, d.updated_at
		FROM doc_nodes d
		JOIN docs_fts f ON d.id = f.doc_id
		WHERE docs_fts MATCH ? AND d.deleted_at IS NULL
		ORDER BY rank ASC
		LIMIT 50`

	rows, err := DB.Query(ftsQuery, `"`+ftsTerm+`"*`)
	if err != nil {
		// Fallback to LIKE if FTS expression failed
		fallbackQuery := `SELECT id, parent_id, type, name, title, slug, path, content, content_type, order_index, created_at, updated_at
			FROM doc_nodes
			WHERE deleted_at IS NULL AND (title LIKE ? OR content LIKE ? OR path LIKE ?)
			LIMIT 50`
		pat := "%" + cleanQuery + "%"
		rows, err = DB.Query(fallbackQuery, pat, pat, pat)
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()

	var results []DocNode
	for rows.Next() {
		var node DocNode
		var parentID, content sql.NullString
		if err := rows.Scan(
			&node.ID,
			&parentID,
			&node.Type,
			&node.Name,
			&node.Title,
			&node.Slug,
			&node.Path,
			&content,
			&node.ContentType,
			&node.OrderIndex,
			&node.CreatedAt,
			&node.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if parentID.Valid {
			node.ParentID = &parentID.String
		}
		if content.Valid {
			node.Content = content.String
		}
		results = append(results, node)
	}

	return results, nil
}

// ExtractTitleFromMarkdown extracts the first # H1 header or falls back to filename formatting.
func ExtractTitleFromMarkdown(filePath, content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			t := strings.TrimPrefix(line, "# ")
			return strings.TrimSpace(t)
		}
	}

	base := path.Base(filePath)
	base = strings.TrimSuffix(base, ".md")
	base = strings.TrimSuffix(base, ".markdown")
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.ReplaceAll(base, "_", " ")
	return strings.Title(base)
}

// ensureFolderTx ensures that a folder (and any missing ancestor folders) exists within an active transaction.
func ensureFolderTx(tx *sql.Tx, fPath string, folderCache map[string]string, now time.Time, createdCount *int) (string, error) {
	fPath = NormalizeDocPath(fPath)
	if fPath == "" || fPath == "." {
		return "", nil
	}
	if folderCache != nil {
		if id, ok := folderCache[fPath]; ok {
			return id, nil
		}
	}

	// Check if exists in DB
	var existingID string
	err := tx.QueryRow(`SELECT id FROM doc_nodes WHERE path = ? AND type = 'folder' AND deleted_at IS NULL`, fPath).Scan(&existingID)
	if err == nil {
		if folderCache != nil {
			folderCache[fPath] = existingID
		}
		return existingID, nil
	}

	// Need to create folder. Ensure parent folder exists first.
	parentDir := path.Dir(fPath)
	var parentIDPtr *string
	if parentDir != "." && parentDir != "/" && parentDir != "" {
		pid, err := ensureFolderTx(tx, parentDir, folderCache, now, createdCount)
		if err != nil {
			return "", err
		}
		if pid != "" {
			parentIDPtr = &pid
		}
	}

	newID := generateDocID()
	folderName := path.Base(fPath)
	folderTitle := strings.ReplaceAll(folderName, "-", " ")
	folderTitle = strings.ReplaceAll(folderTitle, "_", " ")
	folderTitle = strings.Title(folderTitle)

	_, err = tx.Exec(`INSERT INTO doc_nodes (
		id, parent_id, type, name, title, slug, path, content_type, created_at, updated_at
	) VALUES (?, ?, 'folder', ?, ?, ?, ?, 'text/plain', ?, ?)`,
		newID, parentIDPtr, folderName, folderTitle, Slugify(folderName), fPath, now, now)
	if err != nil {
		return "", fmt.Errorf("failed creating parent folder %q: %w", fPath, err)
	}

	if folderCache != nil {
		folderCache[fPath] = newID
	}
	if createdCount != nil {
		*createdCount++
	}
	return newID, nil
}

// BatchImportDocNodes imports a batch of documents and folders into the database in an atomic transaction.
// Automatically ensures all parent folder nodes exist.
func BatchImportDocNodes(items []BatchDocImportItem, overwrite bool) (created int, updated int, err error) {
	if DB == nil {
		return 0, 0, fmt.Errorf("database not initialized")
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	folderCache := make(map[string]string) // normalized path -> folder id

	ensureFolder := func(fPath string) (string, error) {
		return ensureFolderTx(tx, fPath, folderCache, now, &created)
	}

	// 1. First process all explicit folder items
	for _, it := range items {
		if it.Type == "folder" {
			_, err := ensureFolder(it.Path)
			if err != nil {
				return 0, 0, err
			}
		}
	}

	// 2. Process all documents
	for _, it := range items {
		if it.Type == "folder" {
			continue
		}

		docPath := NormalizeDocPath(it.Path)
		if docPath == "" {
			continue
		}

		// Ensure parent folder
		parentDir := path.Dir(docPath)
		var parentIDPtr *string
		if parentDir != "." && parentDir != "/" && parentDir != "" {
			pid, err := ensureFolder(parentDir)
			if err != nil {
				return 0, 0, err
			}
			if pid != "" {
				parentIDPtr = &pid
			}
		}

		title := it.Title
		if title == "" {
			title = ExtractTitleFromMarkdown(docPath, it.Content)
		}
		docName := path.Base(docPath)

		// Check if exists
		var existingID string
		err := tx.QueryRow(`SELECT id FROM doc_nodes WHERE path = ? AND deleted_at IS NULL`, docPath).Scan(&existingID)
		if err == nil {
			if overwrite {
				// Update existing
				_, err = tx.Exec(`UPDATE doc_nodes SET parent_id = ?, name = ?, title = ?, slug = ?, content = ?, updated_at = ? WHERE id = ?`,
					parentIDPtr, docName, title, Slugify(docName), it.Content, now, existingID)
				if err != nil {
					return 0, 0, fmt.Errorf("failed updating doc %q: %w", docPath, err)
				}
				_, _ = tx.Exec(`DELETE FROM docs_fts WHERE doc_id = ?`, existingID)
				_, _ = tx.Exec(`INSERT INTO docs_fts (doc_id, path, title, content) VALUES (?, ?, ?, ?)`,
					existingID, docPath, title, it.Content)
				updated++
			}
			continue
		}

		// Insert new doc
		newDocID := generateDocID()
		_, err = tx.Exec(`INSERT INTO doc_nodes (
			id, parent_id, type, name, title, slug, path, content, content_type, created_at, updated_at
		) VALUES (?, ?, 'doc', ?, ?, ?, ?, ?, 'text/markdown', ?, ?)`,
			newDocID, parentIDPtr, docName, title, Slugify(docName), docPath, it.Content, now, now)
		if err != nil {
			return 0, 0, fmt.Errorf("failed inserting doc %q: %w", docPath, err)
		}

		_, _ = tx.Exec(`INSERT INTO docs_fts (doc_id, path, title, content) VALUES (?, ?, ?, ?)`,
			newDocID, docPath, title, it.Content)
		created++
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return created, updated, nil
}

// SeedDocsFromFS imports embedded or on-disk documentation into the database if the doc_nodes table is empty.
func SeedDocsFromFS(fsys fs.FS) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM doc_nodes WHERE deleted_at IS NULL`).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // already seeded or user created documents
	}

	var items []BatchDocImportItem
	items = append(items,
		BatchDocImportItem{Path: "docs", Type: "folder", Title: "Architecture & Guides"},
		BatchDocImportItem{Path: "proposals", Type: "folder", Title: "Extension Proposals"},
	)

	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if p != "." && p != "" && p != "docs" && p != "proposals" {
				items = append(items, BatchDocImportItem{
					Path: p,
					Type: "folder",
				})
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			content, err := fs.ReadFile(fsys, p)
			if err != nil {
				return err
			}
			targetPath := p
			if !strings.Contains(p, "/") {
				targetPath = "docs/" + p
			}
			items = append(items, BatchDocImportItem{
				Path:    targetPath,
				Type:    "doc",
				Title:   ExtractTitleFromMarkdown(targetPath, string(content)),
				Content: string(content),
			})
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed walking fs for docs seeding: %w", err)
	}

	if len(items) > 0 {
		_, _, err = BatchImportDocNodes(items, true)
		return err
	}
	return nil
}
