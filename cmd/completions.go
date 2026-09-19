package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/utils"
)

// CLIItem represents a selectable note or daily log in fuzzy finders
type CLIItem struct {
	ID            string
	Type          string
	Status        string
	Area          string
	Display       string
	Flesh         string
	Tags          []string
	Timestamp     string
	IsLog         bool
	DeletedReason string
}

// FormatFuzzy returns standardized fuzzy-finder string: [id] [date] (type:status) [area] [#tags]  title — flesh
func (item CLIItem) FormatFuzzy() string {
	shortID := utils.ShortID(item.ID)
	typeStr := item.Type
	if item.Status != "" && !item.IsLog {
		typeStr = fmt.Sprintf("%s:%s", item.Type, item.Status)
	}

	var meta []string
	if item.Area != "" {
		meta = append(meta, string(item.Area))
	}
	if len(item.Tags) > 0 {
		meta = append(meta, "#"+strings.Join(item.Tags, " #"))
	}
	metaStr := ""
	if len(meta) > 0 {
		metaStr = " [" + strings.Join(meta, " ") + "]"
	}

	fleshSnippet := ""
	if item.Flesh != "" {
		cleanedFlesh := strings.ReplaceAll(item.Flesh, "\r\n", " ")
		cleanedFlesh = strings.ReplaceAll(cleanedFlesh, "\n", " ")
		cleanedFlesh = strings.ReplaceAll(cleanedFlesh, "\t", " ")
		cleanedFlesh = strings.TrimSpace(cleanedFlesh)
		if len(cleanedFlesh) > 0 {
			fleshSnippet = " — " + cleanedFlesh
		}
	}

	return fmt.Sprintf("[%s] [%s] (%-10s)%s  %s%s", shortID, item.Timestamp, typeStr, metaStr, item.Display, fleshSnippet)
}

// wrapText wraps text to a maximum column width at word boundaries.
func wrapText(text string, width int) string {
	if width <= 0 {
		width = 80
	}
	if width < 10 {
		width = 10
	}

	lines := strings.Split(text, "\n")
	var wrappedLines []string

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if len(line) <= width {
			wrappedLines = append(wrappedLines, line)
			continue
		}

		words := strings.Fields(line)
		if len(words) == 0 {
			wrappedLines = append(wrappedLines, "")
			continue
		}

		var currentLine strings.Builder
		currentLen := 0

		for _, word := range words {
			wordLen := len(word)
			if currentLen == 0 {
				if wordLen > width {
					for len(word) > width {
						wrappedLines = append(wrappedLines, word[:width])
						word = word[width:]
					}
					if len(word) > 0 {
						currentLine.WriteString(word)
						currentLen = len(word)
					}
				} else {
					currentLine.WriteString(word)
					currentLen = wordLen
				}
			} else {
				if currentLen+1+wordLen <= width {
					currentLine.WriteString(" ")
					currentLine.WriteString(word)
					currentLen += 1 + wordLen
				} else {
					wrappedLines = append(wrappedLines, currentLine.String())
					currentLine.Reset()
					currentLen = 0

					if wordLen > width {
						for len(word) > width {
							wrappedLines = append(wrappedLines, word[:width])
							word = word[width:]
						}
						if len(word) > 0 {
							currentLine.WriteString(word)
							currentLen = len(word)
						}
					} else {
						currentLine.WriteString(word)
						currentLen = wordLen
					}
				}
			}
		}
		if currentLine.Len() > 0 {
			wrappedLines = append(wrappedLines, currentLine.String())
		}
	}

	return strings.Join(wrappedLines, "\n")
}

// RenderPreview generates formatted terminal text for the fuzzy-finder preview pane with word wrapping.
func (item CLIItem) RenderPreview(totalWidth int) string {
	if totalWidth <= 0 {
		totalWidth = 80
	}
	// In go-fuzzyfinder, the preview pane occupies the right half of the terminal: [width/2 .. width-1]
	// The maximum usable text width inside the preview pane is (totalWidth / 2) - 5
	previewWidth := (totalWidth / 2) - 5
	if previewWidth < 15 {
		previewWidth = 15
	}

	var b strings.Builder
	if item.IsLog {
		b.WriteString(fmt.Sprintf("=== DAILY LOG [%s] ===\n\n", utils.ShortID(item.ID)))
		b.WriteString(fmt.Sprintf("Created  : %s\n", item.Timestamp))
		if item.Type == "log:promoted" {
			b.WriteString("Status   : Promoted to Note\n")
		}
		if item.DeletedReason != "" {
			b.WriteString(fmt.Sprintf("Deleted  : %s\n", wrapText(item.DeletedReason, previewWidth)))
		}
		b.WriteString("\nContent  :\n")
		b.WriteString(wrapText(item.Display, previewWidth))
		b.WriteString("\n")
	} else {
		b.WriteString(fmt.Sprintf("=== NOTE [%s] ===\n\n", utils.ShortID(item.ID)))
		b.WriteString(fmt.Sprintf("Title    : %s\n", wrapText(item.Display, previewWidth)))
		b.WriteString(fmt.Sprintf("Type     : %s\n", item.Type))
		if item.Status != "" {
			b.WriteString(fmt.Sprintf("Status   : %s\n", item.Status))
		}
		if item.Area != "" {
			b.WriteString(fmt.Sprintf("Area     : %s\n", item.Area))
		}
		b.WriteString(fmt.Sprintf("Updated  : %s\n", item.Timestamp))
		if len(item.Tags) > 0 {
			b.WriteString(fmt.Sprintf("Tags     : #%s\n", strings.Join(item.Tags, " #")))
		}
		if item.DeletedReason != "" {
			b.WriteString(fmt.Sprintf("Deleted  : %s\n", wrapText(item.DeletedReason, previewWidth)))
		}
		b.WriteString("\n--- BODY / FLESH ---\n")
		trimmedFlesh := strings.TrimSpace(item.Flesh)
		if trimmedFlesh != "" {
			b.WriteString(wrapText(trimmedFlesh, previewWidth))
		} else {
			b.WriteString("(no flesh body)")
		}
		b.WriteString("\n")
	}
	return b.String()
}

// GetActiveCLIItems fetches all active notes and daily logs formatted as CLIItems.
func GetActiveCLIItems() ([]CLIItem, error) {
	var items []CLIItem

	// 1. Active Notes
	notes, err := db.ListNotes("")
	if err != nil {
		return nil, err
	}
	for _, n := range notes {
		var tagNames []string
		if tags, err := db.GetTagsForNote(n.ID); err == nil {
			for _, t := range tags {
				tagNames = append(tagNames, t.Name)
			}
		}

		items = append(items, CLIItem{
			ID:        n.ID,
			Type:      string(n.Type),
			Status:    string(n.Status),
			Area:      string(n.Area),
			Display:   n.Note,
			Flesh:     n.NoteFlesh,
			Tags:      tagNames,
			Timestamp: n.UpdatedAt.Format("2006-01-02 15:04"),
			IsLog:     false,
		})
	}

	// 2. Active Daily Logs
	logs, err := db.GetDailyLogsFilter(nil, nil, true, false)
	if err != nil {
		return nil, err
	}
	for _, l := range logs {
		logType := "log"
		if l.NoteID != "" {
			logType = "log:promoted"
		}
		items = append(items, CLIItem{
			ID:        l.ID,
			Type:      logType,
			Display:   l.Content,
			Timestamp: l.CreatedAt.Format("2006-01-02 15:04"),
			IsLog:     true,
		})
	}

	return items, nil
}

// GetDeletedCLIItems fetches all soft-deleted notes and daily logs formatted as CLIItems.
func GetDeletedCLIItems() ([]CLIItem, error) {
	var items []CLIItem

	// 1. Deleted Notes
	allNotes, err := db.ListNotesExtended("", "", "", true)
	if err != nil {
		return nil, err
	}
	for _, n := range allNotes {
		if !n.DeletedAt.IsZero() {
			var tagNames []string
			if tags, err := db.GetTagsForNote(n.ID); err == nil {
				for _, t := range tags {
					tagNames = append(tagNames, t.Name)
				}
			}

			items = append(items, CLIItem{
				ID:            n.ID,
				Type:          string(n.Type),
				Status:        string(n.Status),
				Area:          string(n.Area),
				Display:       n.Note,
				Flesh:         n.NoteFlesh,
				Tags:          tagNames,
				Timestamp:     n.DeletedAt.Format("2006-01-02 15:04"),
				IsLog:         false,
				DeletedReason: n.DeletedNote,
			})
		}
	}

	// 2. Deleted Daily Logs
	allLogs, err := db.GetDailyLogsFilter(nil, nil, true, true)
	if err != nil {
		return nil, err
	}
	for _, l := range allLogs {
		if !l.DeletedAt.IsZero() {
			items = append(items, CLIItem{
				ID:            l.ID,
				Type:          "log",
				Display:       l.Content,
				Timestamp:     l.DeletedAt.Format("2006-01-02 15:04"),
				IsLog:         true,
				DeletedReason: l.DeletedNote,
			})
		}
	}

	return items, nil
}

// completeNoteIDs returns active note IDs with titles as descriptions (e.g. "2d64a5a\tnew note")
func completeNoteIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	notes, err := db.ListNotes("")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, n := range notes {
		shortID := utils.ShortID(n.ID)
		if strings.HasPrefix(shortID, toComplete) || toComplete == "" {
			title := n.Note
			if len(title) > 35 {
				title = title[:32] + "..."
			}
			completions = append(completions, fmt.Sprintf("%s\t%s (%s)", shortID, title, n.Type))
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeUnpromotedLogIDs returns unpromoted daily log IDs
func completeUnpromotedLogIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	logs, err := db.GetUnpromotedDailyLogs()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, l := range logs {
		shortID := utils.ShortID(l.ID)
		if strings.HasPrefix(shortID, toComplete) || toComplete == "" {
			content := l.Content
			if len(content) > 35 {
				content = content[:32] + "..."
			}
			completions = append(completions, fmt.Sprintf("%s\t%s", shortID, content))
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeDeletableIDs returns active note and daily log IDs with descriptions
func completeDeletableIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var completions []string
	idSet := make(map[string]bool)

	// 1. Notes
	if notes, err := db.ListNotes(""); err == nil {
		for _, n := range notes {
			shortID := utils.ShortID(n.ID)
			if strings.HasPrefix(shortID, toComplete) || toComplete == "" {
				idSet[shortID] = true
				title := n.Note
				if len(title) > 35 {
					title = title[:32] + "..."
				}
				completions = append(completions, fmt.Sprintf("%s\t[note:%s] %s", shortID, n.Type, title))
			}
		}
	}

	// 2. Daily Logs
	if logs, err := db.GetDailyLogsFilter(nil, nil, true, false); err == nil {
		for _, l := range logs {
			shortID := utils.ShortID(l.ID)
			if (strings.HasPrefix(shortID, toComplete) || toComplete == "") && !idSet[shortID] {
				idSet[shortID] = true
				content := l.Content
				if len(content) > 35 {
					content = content[:32] + "..."
				}
				status := "log"
				if l.NoteID != "" {
					status = "log:promoted"
				}
				completions = append(completions, fmt.Sprintf("%s\t[%s] %s", shortID, status, content))
			}
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeDeletedIDs returns soft-deleted note and daily log IDs with descriptions
func completeDeletedIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var completions []string
	idSet := make(map[string]bool)

	// 1. Deleted Notes
	if allNotes, err := db.ListNotesExtended("", "", "", true); err == nil {
		for _, n := range allNotes {
			if !n.DeletedAt.IsZero() {
				shortID := utils.ShortID(n.ID)
				if strings.HasPrefix(shortID, toComplete) || toComplete == "" {
					idSet[shortID] = true
					title := n.Note
					if len(title) > 35 {
						title = title[:32] + "..."
					}
					completions = append(completions, fmt.Sprintf("%s\t[deleted note] %s", shortID, title))
				}
			}
		}
	}

	// 2. Deleted Daily Logs
	if allLogs, err := db.GetDailyLogsFilter(nil, nil, true, true); err == nil {
		for _, l := range allLogs {
			if !l.DeletedAt.IsZero() {
				shortID := utils.ShortID(l.ID)
				if (strings.HasPrefix(shortID, toComplete) || toComplete == "") && !idSet[shortID] {
					idSet[shortID] = true
					content := l.Content
					if len(content) > 35 {
						content = content[:32] + "..."
					}
					completions = append(completions, fmt.Sprintf("%s\t[deleted log] %s", shortID, content))
				}
			}
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeNoteTypes returns available note types with descriptions
func completeNoteTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	types := []string{
		"note\tGeneral atomic thought or note",
		"todo\tActionable task with optional due date",
		"project\tOngoing initiative or milestone",
		"idea\tEarly-stage spark or hypothesis",
		"concept\tMental model or definition",
		"decision\tArchitecture decision record (ADR)",
		"til\tToday I Learned discovery",
		"question\tOpen inquiry driving research",
		"resource\tCurated reference or tool",
		"experiment\tEmpirical test or log",
		"person\tCollaborator or expert reference",
	}
	return types, cobra.ShellCompDirectiveNoFileComp
}

// completeNoteStatuses returns lifecycle statuses
func completeNoteStatuses(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	statuses := []string{
		"active\tStandard active note",
		"raw\tFast fleeting unrefined capture",
		"refined\tStructured and tagged note",
		"in-progress\tCurrently active project or task",
		"completed\tAccomplished task or validated experiment",
		"archived\tHistorical archived knowledge",
	}
	return statuses, cobra.ShellCompDirectiveNoFileComp
}

// completeLinkTypes returns semantic link types
func completeLinkTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	linkTypes := []string{
		"related_to\tGeneral bidirectional relationship",
		"depends_on\tPrerequisite dependency",
		"part_of\tSub-component of a larger project",
		"inspired_by\tOrigin or inspiration source",
		"supports\tSupporting evidence or reasoning",
		"contradicts\tConflicting viewpoint or trade-off",
		"about\tSubject matter or conceptual focus",
		"created_by\tAttribution or author relationship",
	}
	return linkTypes, cobra.ShellCompDirectiveNoFileComp
}

// completeAreas returns standard areas
func completeAreas(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	areas := []string{
		"work\tWork, job, and career initiatives",
		"personal\tPersonal life and habits",
		"finance\tInvestments and financial planning",
	}
	return areas, cobra.ShellCompDirectiveNoFileComp
}

// completeConfigKeys returns available config keys for kb config set / get / delete
func completeConfigKeys(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	keySet := make(map[string]bool)
	var completions []string

	// 1. Keys currently in config.yaml
	for _, k := range viper.AllKeys() {
		if strings.HasPrefix(k, toComplete) || toComplete == "" {
			keySet[k] = true
			completions = append(completions, fmt.Sprintf("%s\tConfigured in config.yaml", k))
		}
	}

	// 2. Known secrets in OS Keyring
	knownSecrets := []string{"postgres_password", "db_password", "remote_db_password"}
	for _, s := range knownSecrets {
		if (strings.HasPrefix(s, toComplete) || toComplete == "") && utils.HasSecret(s) && !keySet[s] {
			keySet[s] = true
			completions = append(completions, fmt.Sprintf("%s\tStored in OS Keyring", s))
		}
	}

	// 3. Standard defaults
	standard := []struct{ key, desc string }{
		{"date_format", "Custom date formatting layout"},
		{"remote.postgres_url", "Remote PostgreSQL connection URL"},
		{"remote.enabled", "Enable/disable remote synchronization"},
		{"remote.provider", "Remote provider name (postgres)"},
		{"app_name", "Application identifier"},
		{"base_path", "Base directory for database and assets"},
	}
	for _, s := range standard {
		if (strings.HasPrefix(s.key, toComplete) || toComplete == "") && !keySet[s.key] {
			keySet[s.key] = true
			completions = append(completions, fmt.Sprintf("%s\t%s", s.key, s.desc))
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeSecretKeys returns available secret keys for kb config set-secret
func completeSecretKeys(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	secrets := []string{
		"postgres_password\tPostgreSQL database password",
		"db_password\tDatabase password alias",
	}
	return secrets, cobra.ShellCompDirectiveNoFileComp
}

// completeAuditIDs returns audit revision IDs for a specific note or log if first arg is given
func completeAuditIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return completeNoteIDs(cmd, args, toComplete)
	}

	id := args[0]
	var entityID string
	var entityType string
	if n, err := db.GetNote(id); err == nil {
		entityID = n.ID
		entityType = "note"
	} else if l, err := db.GetDailyLog(id); err == nil {
		entityID = l.ID
		entityType = "daily_log"
	} else {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	entries, err := db.GetAuditHistory(entityType, entityID, 30)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, e := range entries {
		shortRev := utils.ShortID(e.ID)
		if strings.HasPrefix(shortRev, toComplete) || toComplete == "" {
			desc := fmt.Sprintf("[%s] %s (%s)", e.Action, e.ChangesSummary, e.CreatedAt.Format("15:04"))
			completions = append(completions, fmt.Sprintf("%s\t%s", shortRev, desc))
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
