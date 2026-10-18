package docs

import (
	"testing"
)

func TestGetDocsTree(t *testing.T) {
	tree, err := GetDocsTree()
	if err != nil {
		t.Fatalf("unexpected error getting docs tree: %v", err)
	}

	if len(tree) != 2 {
		t.Fatalf("expected 2 root folders (docs and proposals), got %d", len(tree))
	}

	rootDocs := tree[0]
	if rootDocs.Name != "docs" || len(rootDocs.Children) == 0 {
		t.Errorf("expected root docs folder with children, got %v", rootDocs)
	}

	proposalsDir := tree[1]
	if proposalsDir.Name != "proposals" || len(proposalsDir.Children) == 0 {
		t.Errorf("expected proposals folder with children, got %v", proposalsDir)
	}

	// Verify reading a file
	firstDoc := rootDocs.Children[0]
	content, err := ReadDocFile(firstDoc.Path)
	if err != nil {
		t.Fatalf("failed to read doc file %s: %v", firstDoc.Path, err)
	}
	if len(content) == 0 {
		t.Errorf("doc content for %s is unexpectedly empty", firstDoc.Path)
	}
}
