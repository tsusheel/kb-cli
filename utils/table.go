package utils

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/tsusheel/kb-cli/models"
	"golang.org/x/term"
)


// ShortID truncates full UUIDs to 7 characters for clean terminal display.
func ShortID(id string) string {
	if len(id) > 7 {
		return id[:7]
	}
	return id
}

// GetTerminalWidth detects the current terminal width or defaults to 100 columns.
func GetTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 100
	}
	return width
}

// RenderNotesTable renders a list of notes in a beautiful, structured table with rounded borders.
func RenderNotesTable(notes []models.Note, out io.Writer) {
	if out == nil {
		out = os.Stdout
	}
	if len(notes) == 0 {
		fmt.Fprintln(out, "No notes found.")
		return
	}

	termWidth := GetTerminalWidth()
	maxColWidth := termWidth - 48
	if maxColWidth < 30 {
		maxColWidth = 30
	}
	if maxColWidth > 80 {
		maxColWidth = 80
	}

	table := tablewriter.NewTable(
		out,
		tablewriter.WithHeader([]string{"ID", "NOTE", "TYPE", "STATUS", "UPDATED"}),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleRounded),
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows:    tw.On,
					BetweenColumns: tw.On,
					ShowHeader:     tw.On,
				},
			},
		}),
		tablewriter.WithRowMaxWidth(maxColWidth),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)

	for _, n := range notes {
		displayNote := n.Note
		if displayNote == "" {
			displayNote = "<Untitled>"
		}
		table.Append([]string{
			ShortID(n.ID),
			displayNote,
			string(n.Type),
			string(n.Status),
			n.UpdatedAt.Format("2006-01-02 15:04"),
		})
	}

	table.Render()
}

// RenderInboxTable renders a list of raw inbox notes with rounded borders.
func RenderInboxTable(notes []models.Note, out io.Writer) {
	if out == nil {
		out = os.Stdout
	}
	if len(notes) == 0 {
		fmt.Fprintln(out, "No notes found.")
		return
	}

	termWidth := GetTerminalWidth()
	maxColWidth := termWidth - 36
	if maxColWidth < 30 {
		maxColWidth = 30
	}
	if maxColWidth > 80 {
		maxColWidth = 80
	}

	table := tablewriter.NewTable(
		out,
		tablewriter.WithHeader([]string{"ID", "NOTE", "TYPE", "UPDATED"}),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleRounded),
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows:    tw.On,
					BetweenColumns: tw.On,
					ShowHeader:     tw.On,
				},
			},
		}),
		tablewriter.WithRowMaxWidth(maxColWidth),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)

	for _, n := range notes {
		displayNote := n.Note
		if displayNote == "" {
			displayNote = "<Untitled>"
		}
		table.Append([]string{
			ShortID(n.ID),
			displayNote,
			string(n.Type),
			n.UpdatedAt.Format("2006-01-02 15:04"),
		})
	}

	table.Render()
}

// RenderDailyLogsTable renders daily logs in a beautiful rounded table.
func RenderDailyLogsTable(logs []models.DailyLog, out io.Writer) {
	if out == nil {
		out = os.Stdout
	}
	if len(logs) == 0 {
		fmt.Fprintln(out, "No logs recorded.")
		return
	}

	// Determine if logs span multiple days
	sameDay := true
	if len(logs) > 1 {
		firstDay := logs[0].CreatedAt.Format("2006-01-02")
		for _, l := range logs[1:] {
			if l.CreatedAt.Format("2006-01-02") != firstDay {
				sameDay = false
				break
			}
		}
	}

	termWidth := GetTerminalWidth()
	maxColWidth := termWidth - 46
	if maxColWidth < 30 {
		maxColWidth = 30
	}
	if maxColWidth > 80 {
		maxColWidth = 80
	}

	timeHeader := "TIME"
	if !sameDay {
		timeHeader = "DATE/TIME"
	}

	table := tablewriter.NewTable(
		out,
		tablewriter.WithHeader([]string{"ID", timeHeader, "CONTENT", "STATUS"}),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleRounded),
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows:    tw.On,
					BetweenColumns: tw.On,
					ShowHeader:     tw.On,
				},
			},
		}),
		tablewriter.WithRowMaxWidth(maxColWidth),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)

	for _, l := range logs {
		status := "-"
		if l.NoteID != "" {
			status = fmt.Sprintf("promoted (%s)", ShortID(l.NoteID))
		}
		timeStr := l.CreatedAt.Format("15:04")
		if !sameDay {
			timeStr = l.CreatedAt.Format("2006-01-02 15:04")
		}
		table.Append([]string{
			ShortID(l.ID),
			timeStr,
			l.Content,
			status,
		})
	}

	table.Render()
}

// RenderNoteDetail renders a single note's complete metadata and body in a beautiful rounded card.
func RenderNoteDetail(n *models.Note, tags []models.Tag, links []models.Link, out io.Writer) {
	if n == nil {
		return
	}
	if out == nil {
		out = os.Stdout
	}

	termWidth := GetTerminalWidth()
	maxValWidth := termWidth - 24
	if maxValWidth < 30 {
		maxValWidth = 30
	}
	if maxValWidth > 100 {
		maxValWidth = 100
	}

	table := tablewriter.NewTable(
		out,
		tablewriter.WithHeader([]string{"PROPERTY", "VALUE"}),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleRounded),
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows:    tw.On,
					BetweenColumns: tw.On,
					ShowHeader:     tw.On,
				},
			},
		}),
		tablewriter.WithRowMaxWidth(maxValWidth),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)

	table.Append([]string{"ID", ShortID(n.ID)})

	title := n.Note
	if title == "" {
		title = "<Untitled>"
	}
	table.Append([]string{"Title", title})
	table.Append([]string{"Type", string(n.Type)})
	table.Append([]string{"Status", string(n.Status)})


	if n.Area != "" {
		table.Append([]string{"Area", string(n.Area)})
	}
	if !n.TargetDateTime.IsZero() {
		table.Append([]string{"Due", n.TargetDateTime.Format("2006-01-02 15:04")})
	}
	if n.Importance > 0 || n.Clarity > 0 {
		rating := fmt.Sprintf("Importance: %d/5  |  Clarity: %d/5", n.Importance, n.Clarity)
		table.Append([]string{"Ratings", rating})
	}
	if n.Source != "" {
		table.Append([]string{"Source", n.Source})
	}

	if len(tags) > 0 {
		var tagNames []string
		for _, t := range tags {
			tagNames = append(tagNames, "#"+t.Name)
		}
		table.Append([]string{"Tags", strings.Join(tagNames, ", ")})
	}

	if len(links) > 0 {
		var linkStrs []string
		for _, l := range links {
			otherID := l.ToNote
			arrow := "➔"
			if l.ToNote == n.ID {
				otherID = l.FromNote
				arrow = "⬅"
			}
			linkStrs = append(linkStrs, fmt.Sprintf("%s [%s] (%s)", arrow, ShortID(otherID), l.Type))
		}
		table.Append([]string{"Links", strings.Join(linkStrs, "\n")})
	}

	table.Append([]string{"Created", n.CreatedAt.Format("2006-01-02 15:04")})
	table.Append([]string{"Updated", n.UpdatedAt.Format("2006-01-02 15:04")})

	table.Render()

	// If body (flesh) exists, render it cleanly in a styled box
	if strings.TrimSpace(n.NoteFlesh) != "" {
		fmt.Println()
		contentWidth := termWidth - 10
		if contentWidth < 40 {
			contentWidth = 40
		}
		if contentWidth > 110 {
			contentWidth = 110
		}

		bodyTable := tablewriter.NewTable(
			out,
			tablewriter.WithHeader([]string{"BODY CONTENT"}),
			tablewriter.WithRendition(tw.Rendition{
				Symbols: tw.NewSymbols(tw.StyleRounded),
				Settings: tw.Settings{
					Separators: tw.Separators{
						BetweenRows:    tw.On,
						BetweenColumns: tw.On,
						ShowHeader:     tw.On,
					},
				},
			}),
			tablewriter.WithRowMaxWidth(contentWidth),
			tablewriter.WithHeaderAlignment(tw.AlignLeft),
			tablewriter.WithRowAlignment(tw.AlignLeft),
		)
		bodyTable.Append([]string{n.NoteFlesh})
		bodyTable.Render()
	}
}

// RenderDailyLogDetail renders a single daily log's complete metadata and content in a styled card.
func RenderDailyLogDetail(l *models.DailyLog, promotedNote *models.Note, out io.Writer) {
	if l == nil {
		return
	}
	if out == nil {
		out = os.Stdout
	}

	termWidth := GetTerminalWidth()
	maxValWidth := termWidth - 24
	if maxValWidth < 30 {
		maxValWidth = 30
	}
	if maxValWidth > 100 {
		maxValWidth = 100
	}

	table := tablewriter.NewTable(
		out,
		tablewriter.WithHeader([]string{"PROPERTY", "VALUE"}),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleRounded),
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows:    tw.On,
					BetweenColumns: tw.On,
					ShowHeader:     tw.On,
				},
			},
		}),
		tablewriter.WithRowMaxWidth(maxValWidth),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)

	table.Append([]string{"ID", ShortID(l.ID)})
	table.Append([]string{"Type", "daily_log"})

	status := "unpromoted"
	if l.NoteID != "" {
		if promotedNote != nil && promotedNote.Note != "" {
			status = fmt.Sprintf("promoted ➔ note [%s] %q", ShortID(l.NoteID), promotedNote.Note)
		} else {
			status = fmt.Sprintf("promoted ➔ note [%s]", ShortID(l.NoteID))
		}
	}
	if !l.DeletedAt.IsZero() {
		status = "deleted"
		if l.DeletedNote != "" {
			status = fmt.Sprintf("deleted (%s)", l.DeletedNote)
		}
	}
	table.Append([]string{"Status", status})

	table.Append([]string{"Created", l.CreatedAt.Format("2006-01-02 15:04:05")})
	if !l.DeletedAt.IsZero() {
		table.Append([]string{"Deleted At", l.DeletedAt.Format("2006-01-02 15:04:05")})
	}

	table.Render()

	// Content Card
	fmt.Println()
	contentWidth := termWidth - 10
	if contentWidth < 40 {
		contentWidth = 40
	}
	if contentWidth > 110 {
		contentWidth = 110
	}

	contentTable := tablewriter.NewTable(
		out,
		tablewriter.WithHeader([]string{"LOG CONTENT"}),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleRounded),
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows:    tw.On,
					BetweenColumns: tw.On,
					ShowHeader:     tw.On,
				},
			},
		}),
		tablewriter.WithRowMaxWidth(contentWidth),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	contentTable.Append([]string{l.Content})
	contentTable.Render()
}

// ConfigItem represents a configuration or secret key-value-source entry.
type ConfigItem struct {
	Key    string
	Value  string
	Source string
}

// RenderConfigTable renders configuration settings and secret statuses in a beautiful rounded table.
func RenderConfigTable(items []ConfigItem, out io.Writer) {
	if out == nil {
		out = os.Stdout
	}
	if len(items) == 0 {
		fmt.Fprintln(out, "No configuration found.")
		return
	}

	termWidth := GetTerminalWidth()
	maxValWidth := termWidth - 45
	if maxValWidth < 30 {
		maxValWidth = 30
	}
	if maxValWidth > 80 {
		maxValWidth = 80
	}

	table := tablewriter.NewTable(
		out,
		tablewriter.WithHeader([]string{"KEY", "VALUE", "SOURCE"}),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleRounded),
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows:    tw.On,
					BetweenColumns: tw.On,
					ShowHeader:     tw.On,
				},
			},
		}),
		tablewriter.WithRowMaxWidth(maxValWidth),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)

	for _, item := range items {
		table.Append([]string{
			item.Key,
			item.Value,
			item.Source,
		})
	}

	table.Render()
}

// SyncStatusInfo holds synchronization status metrics and connection properties.
type SyncStatusInfo struct {
	Enabled       bool
	Provider      string
	DatabaseURL   string
	HasPassword   bool
	LastSyncedAt  time.Time
	UnsyncedNotes int
	UnsyncedLogs  int
}

// RenderSyncStatusTable renders synchronization status information in a clean rounded table.
func RenderSyncStatusTable(info SyncStatusInfo, out io.Writer) {
	if out == nil {
		out = os.Stdout
	}

	termWidth := GetTerminalWidth()
	maxValWidth := termWidth - 28
	if maxValWidth < 30 {
		maxValWidth = 30
	}
	if maxValWidth > 100 {
		maxValWidth = 100
	}

	table := tablewriter.NewTable(
		out,
		tablewriter.WithHeader([]string{"PROPERTY", "VALUE"}),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleRounded),
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows:    tw.On,
					BetweenColumns: tw.On,
					ShowHeader:     tw.On,
				},
			},
		}),
		tablewriter.WithRowMaxWidth(maxValWidth),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)

	statusStr := "ENABLED"
	if !info.Enabled {
		statusStr = "DISABLED (remote.enabled = false)"
	}
	table.Append([]string{"Sync Status", statusStr})

	if info.Provider != "" {
		table.Append([]string{"Provider", info.Provider})
	}

	urlStr := info.DatabaseURL
	if urlStr == "" {
		urlStr = "[Not Configured]"
	}
	table.Append([]string{"Database URL", urlStr})

	passStr := "[Configured in OS Keyring]"
	if !info.HasPassword {
		passStr = "[Not Configured]"
	}
	table.Append([]string{"Password", passStr})

	lastSyncStr := "Never"
	if !info.LastSyncedAt.IsZero() {
		lastSyncStr = info.LastSyncedAt.Format("2006-01-02 15:04:05")
	}
	table.Append([]string{"Last Synced", lastSyncStr})

	pendingStr := fmt.Sprintf("%d un-synced notes, %d un-synced logs", info.UnsyncedNotes, info.UnsyncedLogs)
	table.Append([]string{"Pending Changes", pendingStr})

	table.Render()
}

// RenderAuditTable renders a list of audit revision entries in a rounded table.
func RenderAuditTable(entries []models.AuditEntry, title string, out io.Writer) {
	if out == nil {
		out = os.Stdout
	}
	if len(entries) == 0 {
		fmt.Fprintln(out, "No audit history found.")
		return
	}

	if title != "" {
		fmt.Fprintln(out, title)
	}

	termWidth := GetTerminalWidth()
	maxDiffWidth := termWidth - 55
	if maxDiffWidth < 30 {
		maxDiffWidth = 30
	}
	if maxDiffWidth > 80 {
		maxDiffWidth = 80
	}

	table := tablewriter.NewTable(
		out,
		tablewriter.WithHeader([]string{"REV", "TIME", "ENTITY", "ACTION", "CHANGES"}),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleRounded),
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows:    tw.On,
					BetweenColumns: tw.On,
					ShowHeader:     tw.On,
				},
			},
		}),
		tablewriter.WithRowMaxWidth(maxDiffWidth),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)

	for _, e := range entries {
		entityDisplay := fmt.Sprintf("%s [%s]", e.EntityType, ShortID(e.EntityID))
		summary := e.ChangesSummary
		if summary == "" {
			summary = "-"
		}
		table.Append([]string{
			ShortID(e.ID),
			e.CreatedAt.Format("2006-01-02 15:04"),
			entityDisplay,
			string(e.Action),
			summary,
		})
	}

	table.Render()
}

// RenderAuditDiffCard renders complete snapshot inspection for an audit revision.
func RenderAuditDiffCard(entry *models.AuditEntry, out io.Writer) {
	if entry == nil {
		return
	}
	if out == nil {
		out = os.Stdout
	}

	termWidth := GetTerminalWidth()
	maxValWidth := termWidth - 28
	if maxValWidth < 30 {
		maxValWidth = 30
	}
	if maxValWidth > 100 {
		maxValWidth = 100
	}

	table := tablewriter.NewTable(
		out,
		tablewriter.WithHeader([]string{"PROPERTY", "VALUE"}),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleRounded),
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows:    tw.On,
					BetweenColumns: tw.On,
					ShowHeader:     tw.On,
				},
			},
		}),
		tablewriter.WithRowMaxWidth(maxValWidth),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)

	table.Append([]string{"Revision ID", ShortID(entry.ID)})
	table.Append([]string{"Entity", fmt.Sprintf("%s [%s]", entry.EntityType, ShortID(entry.EntityID))})
	table.Append([]string{"Action", string(entry.Action)})
	table.Append([]string{"Timestamp", entry.CreatedAt.Format("2006-01-02 15:04:05")})
	table.Append([]string{"Summary", entry.ChangesSummary})

	table.Render()

	if strings.TrimSpace(entry.SnapshotJSON) != "" {
		fmt.Println()
		contentWidth := termWidth - 10
		if contentWidth < 40 {
			contentWidth = 40
		}
		if contentWidth > 110 {
			contentWidth = 110
		}

		snapTable := tablewriter.NewTable(
			out,
			tablewriter.WithHeader([]string{"POINT-IN-TIME SNAPSHOT"}),
			tablewriter.WithRendition(tw.Rendition{
				Symbols: tw.NewSymbols(tw.StyleRounded),
				Settings: tw.Settings{
					Separators: tw.Separators{
						BetweenRows:    tw.On,
						BetweenColumns: tw.On,
						ShowHeader:     tw.On,
					},
				},
			}),
			tablewriter.WithRowMaxWidth(contentWidth),
			tablewriter.WithHeaderAlignment(tw.AlignLeft),
			tablewriter.WithRowAlignment(tw.AlignLeft),
		)
		snapTable.Append([]string{entry.SnapshotJSON})
		snapTable.Render()
	}
}

// RenderBacklinksTable renders incoming backlinks pointing to a note in a clean rounded table.
func RenderBacklinksTable(links []models.Link, noteMap map[string]*models.Note, out io.Writer) {
	if out == nil {
		out = os.Stdout
	}
	if len(links) == 0 {
		fmt.Fprintln(out, "No incoming backlinks found.")
		return
	}

	termWidth := GetTerminalWidth()
	maxValWidth := termWidth - 45
	if maxValWidth < 30 {
		maxValWidth = 30
	}
	if maxValWidth > 80 {
		maxValWidth = 80
	}

	table := tablewriter.NewTable(
		out,
		tablewriter.WithHeader([]string{"FROM ID", "SOURCE NOTE TITLE", "RELATION TYPE", "LINKED AT"}),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleRounded),
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows:    tw.On,
					BetweenColumns: tw.On,
					ShowHeader:     tw.On,
				},
			},
		}),
		tablewriter.WithRowMaxWidth(maxValWidth),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)

	for _, l := range links {
		title := "<Untitled>"
		if sourceNote, ok := noteMap[l.FromNote]; ok && sourceNote != nil && sourceNote.Note != "" {
			title = sourceNote.Note
		}
		table.Append([]string{
			ShortID(l.FromNote),
			title,
			string(l.Type),
			l.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	table.Render()
}

// RenderOrphansTable renders unlinked, untagged notes in a rounded table.
func RenderOrphansTable(notes []models.Note, out io.Writer) {
	if out == nil {
		out = os.Stdout
	}
	if len(notes) == 0 {
		fmt.Fprintln(out, "🎉 No orphan notes! All active notes have tags or relationships.")
		return
	}

	termWidth := GetTerminalWidth()
	maxValWidth := termWidth - 38
	if maxValWidth < 30 {
		maxValWidth = 30
	}
	if maxValWidth > 80 {
		maxValWidth = 80
	}

	table := tablewriter.NewTable(
		out,
		tablewriter.WithHeader([]string{"ID", "ORPHAN NOTE TITLE", "TYPE", "UPDATED"}),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleRounded),
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows:    tw.On,
					BetweenColumns: tw.On,
					ShowHeader:     tw.On,
				},
			},
		}),
		tablewriter.WithRowMaxWidth(maxValWidth),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)

	for _, n := range notes {
		displayNote := n.Note
		if displayNote == "" {
			displayNote = "<Untitled>"
		}
		table.Append([]string{
			ShortID(n.ID),
			displayNote,
			string(n.Type),
			n.UpdatedAt.Format("2006-01-02 15:04"),
		})
	}

	table.Render()
}

// RenderStatsDashboard renders the comprehensive knowledge base analytics dashboard.
func RenderStatsDashboard(stats *models.KBStats, out io.Writer) {
	if stats == nil {
		return
	}
	if out == nil {
		out = os.Stdout
	}

	termWidth := GetTerminalWidth()
	maxValWidth := termWidth - 28
	if maxValWidth < 30 {
		maxValWidth = 30
	}
	if maxValWidth > 100 {
		maxValWidth = 100
	}

	table := tablewriter.NewTable(
		out,
		tablewriter.WithHeader([]string{"METRIC", "VALUE"}),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleRounded),
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows:    tw.On,
					BetweenColumns: tw.On,
					ShowHeader:     tw.On,
				},
			},
		}),
		tablewriter.WithRowMaxWidth(maxValWidth),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)

	table.Append([]string{"Total Active Notes", fmt.Sprintf("%d", stats.TotalActiveNotes)})
	table.Append([]string{"Soft-Deleted Notes", fmt.Sprintf("%d", stats.TotalDeletedNotes)})
	table.Append([]string{"Daily Stream Logs", fmt.Sprintf("%d total (%d today)", stats.TotalDailyLogs, stats.TodayDailyLogs)})

	streakStr := fmt.Sprintf("%d day(s)", stats.ConsecutiveStreak)
	if stats.ConsecutiveStreak > 1 {
		streakStr += " 🔥"
	}
	table.Append([]string{"Current Activity Streak", streakStr})

	// Types breakdown
	var typeList []string
	for k, v := range stats.NotesByType {
		typeList = append(typeList, fmt.Sprintf("%s: %d", k, v))
	}
	if len(typeList) > 0 {
		table.Append([]string{"Notes by Type", strings.Join(typeList, " | ")})
	}

	// Status breakdown
	var statusList []string
	for k, v := range stats.NotesByStatus {
		statusList = append(statusList, fmt.Sprintf("%s: %d", k, v))
	}
	if len(statusList) > 0 {
		table.Append([]string{"Notes by Status", strings.Join(statusList, " | ")})
	}

	// Area breakdown
	var areaList []string
	for k, v := range stats.NotesByArea {
		areaList = append(areaList, fmt.Sprintf("%s: %d", k, v))
	}
	if len(areaList) > 0 {
		table.Append([]string{"Notes by Area", strings.Join(areaList, " | ")})
	}

	// Top Tags
	table.Append([]string{"Total Unique Tags", fmt.Sprintf("%d", stats.TotalTags)})
	if len(stats.TopTags) > 0 {
		var tagStrs []string
		for _, tt := range stats.TopTags {
			tagStrs = append(tagStrs, fmt.Sprintf("#%s (%d)", tt.Name, tt.Count))
		}
		table.Append([]string{"Top Tags", strings.Join(tagStrs, ", ")})
	}

	// Top Hub Notes
	if len(stats.TopHubNotes) > 0 {
		var hubStrs []string
		for _, hn := range stats.TopHubNotes {
			hubStrs = append(hubStrs, fmt.Sprintf("[%s] %q (%d links)", ShortID(hn.ID), hn.Title, hn.LinkCount))
		}
		table.Append([]string{"Top Hub Notes", strings.Join(hubStrs, "\n")})
	}

	table.Render()
}

// RenderGraphTree renders ASCII tree representation of note connections.
func RenderGraphTree(rootNote *models.Note, outgoing []models.Link, incoming []models.Link, noteMap map[string]*models.Note, out io.Writer) {
	if out == nil {
		out = os.Stdout
	}
	if rootNote == nil {
		return
	}

	fmt.Fprintf(out, "📦 [%s] %s (%s)\n", ShortID(rootNote.ID), rootNote.Note, rootNote.Type)

	if len(outgoing) == 0 && len(incoming) == 0 {
		fmt.Fprintln(out, "  └── (no outgoing or incoming links)")
		return
	}

	if len(outgoing) > 0 {
		fmt.Fprintln(out, "  ├── ➔ Outgoing Connections:")
		for i, l := range outgoing {
			prefix := "  │   ├──"
			if i == len(outgoing)-1 && len(incoming) == 0 {
				prefix = "  │   └──"
			}
			targetTitle := "<Untitled>"
			if tn, ok := noteMap[l.ToNote]; ok && tn != nil && tn.Note != "" {
				targetTitle = tn.Note
			}
			fmt.Fprintf(out, "%s --(%s)--> [%s] %s\n", prefix, l.Type, ShortID(l.ToNote), targetTitle)
		}
	}

	if len(incoming) > 0 {
		fmt.Fprintln(out, "  └── ⬅ Incoming Backlinks:")
		for i, l := range incoming {
			prefix := "      ├──"
			if i == len(incoming)-1 {
				prefix = "      └──"
			}
			sourceTitle := "<Untitled>"
			if sn, ok := noteMap[l.FromNote]; ok && sn != nil && sn.Note != "" {
				sourceTitle = sn.Note
			}
			fmt.Fprintf(out, "%s <--( %s )-- [%s] %s\n", prefix, l.Type, ShortID(l.FromNote), sourceTitle)
		}
	}
}

