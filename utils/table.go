package utils

import (
	"fmt"
	"io"
	"os"
	"strings"

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
	if len(notes) == 0 {
		fmt.Fprintln(out, "No notes found.")
		return
	}
	if out == nil {
		out = os.Stdout
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
	if len(notes) == 0 {
		fmt.Fprintln(out, "No notes found.")
		return
	}
	if out == nil {
		out = os.Stdout
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

	if len(logs) == 0 {
		fmt.Fprintln(out, "No logs recorded.")
		return
	}
	if out == nil {
		out = os.Stdout
	}

	termWidth := GetTerminalWidth()
	maxColWidth := termWidth - 42
	if maxColWidth < 30 {
		maxColWidth = 30
	}
	if maxColWidth > 80 {
		maxColWidth = 80
	}

	table := tablewriter.NewTable(
		out,
		tablewriter.WithHeader([]string{"ID", "TIME", "CONTENT", "STATUS"}),
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
		table.Append([]string{
			ShortID(l.ID),
			l.CreatedAt.Format("15:04"),
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




