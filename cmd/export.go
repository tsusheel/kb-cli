package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/utils"
	"gopkg.in/yaml.v3"
)

var (
	exportDir            string
	exportFormat         string
	exportIncludeDeleted bool
)

var sanitizeFilenameRegex = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = sanitizeFilenameRegex.ReplaceAllString(name, "")
	name = strings.ToLower(name)
	if len(name) > 40 {
		name = name[:40]
	}
	return name
}

type NoteFrontmatter struct {
	ID        string   `yaml:"id"`
	Title     string   `yaml:"title"`
	Type      string   `yaml:"type"`
	Status    string   `yaml:"status"`
	Area      string   `yaml:"area,omitempty"`
	Due       string   `yaml:"due,omitempty"`
	Tags      []string `yaml:"tags,omitempty"`
	CreatedAt string   `yaml:"created_at"`
	UpdatedAt string   `yaml:"updated_at"`
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export knowledge base notes and logs to Markdown files (Obsidian-compatible) or JSON",
	Long: `Export notes, tags, and logs to a structured directory of Markdown files with YAML frontmatter or a unified JSON file.

Examples:
  # Export as Markdown files to default directory (./kb-export):
  kb export

  # Export as Markdown files to custom folder:
  kb export --dir ./my-obsidian-vault

  # Export as unified JSON dump:
  kb export --format json --dir ./backups`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if exportDir == "" {
			exportDir = "./kb-export"
		}

		if err := os.MkdirAll(exportDir, 0755); err != nil {
			return fmt.Errorf("failed creating export directory %s: %w", exportDir, err)
		}

		notes, err := db.ListNotesExtended("", "", "", exportIncludeDeleted)
		if err != nil {
			return fmt.Errorf("failed fetching notes: %w", err)
		}

		format := strings.ToLower(strings.TrimSpace(exportFormat))

		if format == "json" {
			type FullExport struct {
				Notes     []models.Note     `json:"notes"`
				DailyLogs []models.DailyLog `json:"daily_logs"`
			}
			logs, err := db.GetDailyLogsFilter(nil, nil, true, exportIncludeDeleted)
			if err != nil {
				return err
			}

			payload := FullExport{
				Notes:     notes,
				DailyLogs: logs,
			}

			data, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return err
			}

			outputPath := filepath.Join(exportDir, "kb-backup.json")
			if err := os.WriteFile(outputPath, data, 0644); err != nil {
				return fmt.Errorf("failed writing export file %s: %w", outputPath, err)
			}

			fmt.Printf("✔ Successfully exported %d note(s) and %d daily log(s) to JSON: %s\n", len(notes), len(logs), outputPath)
			return nil
		}

		// Markdown Export
		notesDir := filepath.Join(exportDir, "notes")
		if err := os.MkdirAll(notesDir, 0755); err != nil {
			return err
		}

		exportedCount := 0
		for _, n := range notes {
			tags, _ := db.GetTagsForNote(n.ID)
			var tagNames []string
			for _, t := range tags {
				tagNames = append(tagNames, t.Name)
			}

			dueStr := ""
			if !n.TargetDateTime.IsZero() {
				dueStr = n.TargetDateTime.Format("2006-01-02 15:04")
			}

			fm := NoteFrontmatter{
				ID:        n.ID,
				Title:     n.Note,
				Type:      string(n.Type),
				Status:    string(n.Status),
				Area:      string(n.Area),
				Due:       dueStr,
				Tags:      tagNames,
				CreatedAt: n.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt: n.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}

			fmBytes, err := yaml.Marshal(fm)
			if err != nil {
				continue
			}

			var b strings.Builder
			b.WriteString("---\n")
			b.Write(fmBytes)
			b.WriteString("---\n\n")

			title := n.Note
			if title == "" {
				title = "Untitled"
			}
			b.WriteString(fmt.Sprintf("# %s\n\n", title))

			if strings.TrimSpace(n.NoteFlesh) != "" {
				b.WriteString(n.NoteFlesh)
				b.WriteString("\n")
			}

			baseSlug := sanitizeFilename(title)
			if baseSlug == "" {
				baseSlug = "note"
			}
			filename := fmt.Sprintf("%s-%s.md", baseSlug, utils.ShortID(n.ID))
			noteFilePath := filepath.Join(notesDir, filename)

			if err := os.WriteFile(noteFilePath, []byte(b.String()), 0644); err != nil {
				return fmt.Errorf("failed writing note %s: %w", noteFilePath, err)
			}
			exportedCount++
		}

		// Export Daily Logs to daily_logs.md
		logs, _ := db.GetDailyLogsFilter(nil, nil, true, exportIncludeDeleted)
		if len(logs) > 0 {
			var logDoc strings.Builder
			logDoc.WriteString("# Knowledge Base Daily Logs\n\n")
			for _, l := range logs {
				status := ""
				if l.NoteID != "" {
					status = fmt.Sprintf(" (promoted: %s)", utils.ShortID(l.NoteID))
				}
				logDoc.WriteString(fmt.Sprintf("- **[%s]** %s%s\n", l.CreatedAt.Format("2006-01-02 15:04"), l.Content, status))
			}
			logFilePath := filepath.Join(exportDir, "daily_logs.md")
			_ = os.WriteFile(logFilePath, []byte(logDoc.String()), 0644)
		}

		fmt.Printf("✔ Successfully exported %d Markdown note(s) and %d daily log(s) to: %s\n", exportedCount, len(logs), exportDir)
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportDir, "dir", "d", "./kb-export", "Target directory for export")
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "markdown", "Export format: 'markdown' (Obsidian frontmatter) or 'json'")
	exportCmd.Flags().BoolVarP(&exportIncludeDeleted, "include-deleted", "a", false, "Include soft-deleted items in export")

	rootCmd.AddCommand(exportCmd)
}
