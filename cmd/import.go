package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/utils"
	"gopkg.in/yaml.v3"
)

var importDir string

func parseMarkdownNote(content string) (NoteFrontmatter, string) {
	var fm NoteFrontmatter
	content = strings.ReplaceAll(content, "\r\n", "\n")
	trimmed := strings.TrimSpace(content)

	if strings.HasPrefix(trimmed, "---") {
		parts := strings.SplitN(trimmed[3:], "---", 2)
		if len(parts) == 2 {
			yamlHeader := parts[0]
			body := strings.TrimSpace(parts[1])
			_ = yaml.Unmarshal([]byte(yamlHeader), &fm)

			// Clean up leading '# Title' from body if it matches frontmatter title
			if strings.HasPrefix(body, "# ") {
				lines := strings.SplitN(body, "\n", 2)
				if len(lines) == 2 {
					body = strings.TrimSpace(lines[1])
				} else {
					body = ""
				}
			}
			return fm, body
		}
	}

	// If no frontmatter, extract first header as title
	lines := strings.Split(trimmed, "\n")
	var title string
	var bodyLines []string

	for i, l := range lines {
		trimmedLine := strings.TrimSpace(l)
		if title == "" && strings.HasPrefix(trimmedLine, "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "# "))
		} else if title == "" && trimmedLine != "" {
			title = trimmedLine
		} else {
			bodyLines = append(bodyLines, lines[i])
		}
	}

	fm.Title = title
	return fm, strings.TrimSpace(strings.Join(bodyLines, "\n"))
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import notes from a directory of Markdown files (Obsidian/Logseq compatible)",
	Long: `Batch import Markdown files (.md) from a directory into the knowledge base.
Automatically extracts YAML frontmatter metadata (title, tags, type, status, due date) or uses markdown headers.

Examples:
  # Import markdown files from a folder:
  kb import --dir ./my-obsidian-vault/notes
  kb import -d ./docs`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if importDir == "" {
			return fmt.Errorf("please specify the directory to import using --dir <path>")
		}

		info, err := os.Stat(importDir)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("directory %s does not exist or is not a valid directory", importDir)
		}

		entries, err := os.ReadDir(importDir)
		if err != nil {
			return fmt.Errorf("failed reading directory %s: %w", importDir, err)
		}

		importedCount := 0
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
				continue
			}
			if strings.EqualFold(entry.Name(), "daily_logs.md") {
				continue
			}

			filePath := filepath.Join(importDir, entry.Name())
			contentBytes, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			fm, body := parseMarkdownNote(string(contentBytes))

			title := fm.Title
			if title == "" {
				baseName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
				title = strings.ReplaceAll(baseName, "-", " ")
				title = strings.ReplaceAll(title, "_", " ")
			}

			nType := models.NoteType(fm.Type)
			if nType == "" {
				nType = models.DefaultNote
			}
			nStatus := models.Status(fm.Status)
			if nStatus == "" {
				nStatus = models.Active
			}

			var targetDT time.Time
			if fm.Due != "" {
				if t, err := utils.ParseDate(fm.Due); err == nil {
					targetDT = t
				}
			}

			id := strings.ReplaceAll(uuid.New().String(), "-", "")
			if fm.ID != "" && len(fm.ID) == 32 {
				id = strings.ReplaceAll(fm.ID, "-", "")
			}

			n := &models.Note{
				ID:             id,
				Note:           title,
				NoteFlesh:      body,
				Type:           nType,
				Status:         nStatus,
				Area:           models.Area(fm.Area),
				TargetDateTime: targetDT,
				Source:         "Markdown Import",
			}

			if err := db.CreateNote(n); err != nil {
				// If ID already exists, generate a new one
				n.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
				if err := db.CreateNote(n); err != nil {
					continue
				}
			}

			for _, tag := range fm.Tags {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					_ = db.AddTag(n.ID, tag)
				}
			}

			importedCount++
		}

		fmt.Printf("✔ Successfully imported %d Markdown note(s) from: %s\n", importedCount, importDir)
		return nil
	},
}

func init() {
	importCmd.Flags().StringVarP(&importDir, "dir", "d", "", "Directory containing Markdown files to import")
	_ = importCmd.MarkFlagRequired("dir")

	rootCmd.AddCommand(importCmd)
}
