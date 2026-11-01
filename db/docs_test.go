package db

import (
	"testing"
	"testing/fstest"
)

func TestDocNodeCRUD(t *testing.T) {
	setupTestDB(t)

	// 1. Create a root document
	rootDoc := &DocNode{
		Type:    "doc",
		Path:    "README.md",
		Title:   "Overview",
		Content: "# Architecture Overview\nWelcome to KB-CLI.",
	}
	if err := CreateDocNode(rootDoc); err != nil {
		t.Fatalf("CreateDocNode root doc failed: %v", err)
	}
	if rootDoc.ID == "" {
		t.Fatalf("expected generated ID for root doc")
	}

	// 2. Fetch by path
	fetched, err := GetDocNodeByPath("README.md")
	if err != nil {
		t.Fatalf("GetDocNodeByPath failed: %v", err)
	}
	if fetched.Title != "Overview" || fetched.Content != rootDoc.Content {
		t.Fatalf("unexpected fetched content: %+v", fetched)
	}

	// 3. Update doc
	fetched.Title = "Updated Overview"
	fetched.Content = "# Updated Architecture Overview\nNew content here."
	if err := UpdateDocNode(fetched); err != nil {
		t.Fatalf("UpdateDocNode failed: %v", err)
	}

	updated, err := GetDocNodeByID(fetched.ID)
	if err != nil {
		t.Fatalf("GetDocNodeByID failed: %v", err)
	}
	if updated.Title != "Updated Overview" {
		t.Fatalf("expected title 'Updated Overview', got %q", updated.Title)
	}

	// 4. Search via FTS
	results, err := SearchDocs("Updated")
	if err != nil {
		t.Fatalf("SearchDocs failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected search results for 'Updated', got 0")
	}
	if results[0].ID != fetched.ID {
		t.Fatalf("expected search to return node %s, got %s", fetched.ID, results[0].ID)
	}

	// 5. Delete doc
	if err := DeleteDocNode(fetched.ID, "test deletion"); err != nil {
		t.Fatalf("DeleteDocNode failed: %v", err)
	}

	_, err = GetDocNodeByID(fetched.ID)
	if err != ErrDocNotFound {
		t.Fatalf("expected ErrDocNotFound after deletion, got: %v", err)
	}
}

func TestNestedFoldersAndArbitraryDepth(t *testing.T) {
	setupTestDB(t)

	// Simulate batch import of multiple nested folders and files
	batch := []BatchDocImportItem{
		{Path: "engineering", Type: "folder"},
		{Path: "engineering/backend", Type: "folder"},
		{Path: "engineering/backend/database", Type: "folder"},
		{
			Path:    "engineering/backend/database/schema.md",
			Type:    "doc",
			Title:   "Database Schema",
			Content: "# Database Schema\nSQLite and Postgres schemas.",
		},
		{
			Path:    "engineering/backend/database/migrations.md",
			Type:    "doc",
			Title:   "Migrations Guide",
			Content: "# Migrations Guide\nHow to write migrations.",
		},
		{
			Path:    "engineering/frontend/ui.md",
			Type:    "doc",
			Title:   "UI Components",
			Content: "# UI Components\nVanilla JS and CSS.",
		},
		{
			Path:    "README.md",
			Type:    "doc",
			Title:   "Project Root",
			Content: "# Root Readme",
		},
	}

	created, updated, err := BatchImportDocNodes(batch, true)
	if err != nil {
		t.Fatalf("BatchImportDocNodes failed: %v", err)
	}
	if created < 7 {
		t.Fatalf("expected at least 7 created items, got %d (updated: %d)", created, updated)
	}

	// Verify deep path lookup
	deepDoc, err := GetDocNodeByPath("engineering/backend/database/schema.md")
	if err != nil {
		t.Fatalf("failed retrieving deep nested doc: %v", err)
	}
	if deepDoc.Title != "Database Schema" {
		t.Fatalf("expected title 'Database Schema', got %q", deepDoc.Title)
	}
	if deepDoc.ParentID == nil {
		t.Fatalf("expected deep doc to have a parent ID")
	}

	// Verify hierarchical tree assembly
	tree, err := GetDocsTree()
	if err != nil {
		t.Fatalf("GetDocsTree failed: %v", err)
	}

	if len(tree) == 0 {
		t.Fatalf("expected non-empty docs tree")
	}

	// Find engineering folder in root
	var engFolder *DocTreeItem
	for i := range tree {
		if tree[i].Name == "engineering" {
			engFolder = &tree[i]
			break
		}
	}
	if engFolder == nil {
		t.Fatalf("engineering folder not found in tree root")
	}
	if engFolder.Type != "directory" {
		t.Fatalf("expected engineering to be directory, got %q", engFolder.Type)
	}

	// Find backend in engineering
	var backendFolder *DocTreeItem
	for i := range engFolder.Children {
		if engFolder.Children[i].Name == "backend" {
			backendFolder = &engFolder.Children[i]
			break
		}
	}
	if backendFolder == nil {
		t.Fatalf("backend folder not found inside engineering")
	}

	// Find database in backend
	var dbFolder *DocTreeItem
	for i := range backendFolder.Children {
		if backendFolder.Children[i].Name == "database" {
			dbFolder = &backendFolder.Children[i]
			break
		}
	}
	if dbFolder == nil {
		t.Fatalf("database folder not found inside backend")
	}

	// Check children of database
	if len(dbFolder.Children) != 2 {
		t.Fatalf("expected 2 files in database folder, got %d", len(dbFolder.Children))
	}

	// Soft-delete parent folder "engineering/backend" and verify cascading soft-delete
	if err := DeleteDocNode(backendFolder.ID, "deleting backend"); err != nil {
		t.Fatalf("failed deleting backend folder: %v", err)
	}

	// Schema.md should no longer be accessible
	_, err = GetDocNodeByPath("engineering/backend/database/schema.md")
	if err != ErrDocNotFound {
		t.Fatalf("expected child doc to be deleted with parent folder, got %v", err)
	}
}

func TestSeedDocsFromFS(t *testing.T) {
	setupTestDB(t)

	mockFS := fstest.MapFS{
		"README.md": &fstest.MapFile{
			Data: []byte("# Architecture Overview\nMain documentation."),
		},
		"guides/quickstart.md": &fstest.MapFile{
			Data: []byte("# Quickstart Guide\nGetting started."),
		},
		"guides/advanced/deep.md": &fstest.MapFile{
			Data: []byte("# Deep Guide\nDeep level."),
		},
	}

	if err := SeedDocsFromFS(mockFS); err != nil {
		t.Fatalf("SeedDocsFromFS failed: %v", err)
	}

	tree, err := GetDocsTree()
	if err != nil {
		t.Fatalf("GetDocsTree failed: %v", err)
	}
	if len(tree) < 2 {
		t.Fatalf("expected at least 2 root items in tree, got %d", len(tree))
	}
}

func TestDocTagsCRUD(t *testing.T) {
	setupTestDB(t)

	doc := &DocNode{
		Type:    "doc",
		Path:    "guides/tagged-doc.md",
		Title:   "Tagged Doc",
		Content: "# Tagged Doc\nContent with tags.",
		Tags:    []string{"architecture", "api", "guides"},
	}
	if err := CreateDocNode(doc); err != nil {
		t.Fatalf("CreateDocNode with tags failed: %v", err)
	}

	// 1. Fetch tags for doc
	tags, err := GetTagsForDoc(doc.ID)
	if err != nil {
		t.Fatalf("GetTagsForDoc failed: %v", err)
	}
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d: %v", len(tags), tags)
	}

	// 2. Fetch doc by path and verify Tags field populated
	fetched, err := GetDocNodeByPath("guides/tagged-doc.md")
	if err != nil {
		t.Fatalf("GetDocNodeByPath failed: %v", err)
	}
	if len(fetched.Tags) != 3 {
		t.Fatalf("expected 3 tags on fetched doc, got %d", len(fetched.Tags))
	}

	// 3. Update tags on doc
	fetched.Tags = []string{"architecture", "security"}
	if err := UpdateDocNode(fetched); err != nil {
		t.Fatalf("UpdateDocNode with updated tags failed: %v", err)
	}

	updatedTags, err := GetTagsForDoc(doc.ID)
	if err != nil {
		t.Fatalf("GetTagsForDoc after update failed: %v", err)
	}
	if len(updatedTags) != 2 {
		t.Fatalf("expected 2 tags after update, got %d: %v", len(updatedTags), updatedTags)
	}

	// 4. Verify summary
	summary, err := GetDocsTagsSummary()
	if err != nil {
		t.Fatalf("GetDocsTagsSummary failed: %v", err)
	}
	if len(summary) < 2 {
		t.Fatalf("expected at least 2 tag summary entries, got %d", len(summary))
	}

	// 5. Verify ActiveDocsList returns doc with tags
	activeDocs, err := GetActiveDocsList()
	if err != nil {
		t.Fatalf("GetActiveDocsList failed: %v", err)
	}
	if len(activeDocs) == 0 {
		t.Fatalf("expected at least 1 active doc")
	}
	if len(activeDocs[0].Tags) != 2 {
		t.Fatalf("expected active doc to have 2 tags, got %d", len(activeDocs[0].Tags))
	}
}
