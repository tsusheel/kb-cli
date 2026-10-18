package docs

import (
	"bufio"
	"embed"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// FS embeds all Markdown documentation files and proposals.
//
//go:embed *.md proposals/*.md
var FS embed.FS

// DocItem represents a node in the documentation directory tree.
type DocItem struct {
	Name     string    `json:"name"`
	Title    string    `json:"title"`
	Path     string    `json:"path"`
	Type     string    `json:"type"` // "directory" or "file"
	Children []DocItem `json:"children,omitempty"`
}

// GetDocsTree returns a structured tree representing the documentation hierarchy.
func GetDocsTree() ([]DocItem, error) {
	rootDocs := DocItem{
		Name:     "docs",
		Title:    "Architecture & Guides",
		Path:     "",
		Type:     "directory",
		Children: []DocItem{},
	}

	proposalsDir := DocItem{
		Name:     "proposals",
		Title:    "Extension Proposals",
		Path:     "proposals",
		Type:     "directory",
		Children: []DocItem{},
	}

	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".md") {
			filePath := entry.Name()
			title := extractTitle(filePath)
			rootDocs.Children = append(rootDocs.Children, DocItem{
				Name:  entry.Name(),
				Title: title,
				Path:  filePath,
				Type:  "file",
			})
		}
	}

	propEntries, err := fs.ReadDir(FS, "proposals")
	if err == nil {
		for _, entry := range propEntries {
			if entry.IsDir() {
				continue
			}
			if strings.HasSuffix(entry.Name(), ".md") {
				filePath := path.Join("proposals", entry.Name())
				title := extractTitle(filePath)
				proposalsDir.Children = append(proposalsDir.Children, DocItem{
					Name:  entry.Name(),
					Title: title,
					Path:  filePath,
					Type:  "file",
				})
			}
		}
	}

	sortChildren := func(items []DocItem) {
		sort.Slice(items, func(i, j int) bool {
			if items[i].Name == "README.md" {
				return true
			}
			if items[j].Name == "README.md" {
				return false
			}
			return items[i].Name < items[j].Name
		})
	}
	sortChildren(rootDocs.Children)
	sortChildren(proposalsDir.Children)

	return []DocItem{rootDocs, proposalsDir}, nil
}

// ReadDocFile returns the raw Markdown content of a doc file.
func ReadDocFile(filePath string) (string, error) {
	// Clean and normalize path
	cleanPath := path.Clean(strings.TrimPrefix(filePath, "/"))
	data, err := fs.ReadFile(FS, cleanPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// extractTitle reads the first H1 header line from a markdown file, or derives it from the filename.
func extractTitle(filePath string) string {
	if filePath == "README.md" {
		return "00. Architecture Overview"
	}
	if filePath == "proposals/README.md" {
		return "00. Proposals Overview"
	}

	f, err := FS.Open(filePath)
	if err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "# ") {
				t := strings.TrimPrefix(line, "# ")
				t = strings.TrimSpace(t)
				// Remove trailing "(Proposal)" if present for cleaner tree display
				t = strings.TrimSuffix(t, " (Proposal)")
				return t
			}
		}
	}

	// Fallback to filename formatting
	base := path.Base(filePath)
	base = strings.TrimSuffix(base, ".md")
	base = strings.ReplaceAll(base, "-", " ")
	return strings.Title(base)
}
