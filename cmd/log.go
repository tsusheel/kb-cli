package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
)

var showAllLogs bool

func displayTodayLogs(date time.Time) error {
	logs, err := db.GetDailyLogsForDate(date, true)
	if err != nil {
		return err
	}

	if len(logs) == 0 {
		fmt.Printf("No logs recorded for %s.\n", date.Format("2006-01-02"))
		return nil
	}

	fmt.Printf("=== [%s] Daily Stream ===\n", date.Format("2006-01-02"))
	for _, l := range logs {
		status := ""
		if l.NoteID != "" {
			noteShortID := l.NoteID
			if len(noteShortID) > 7 {
				noteShortID = noteShortID[:7]
			}
			status = fmt.Sprintf(" (promoted -> [%s])", noteShortID)
		}
		fmt.Printf("%s  [%s]  %s%s\n", l.CreatedAt.Format("15:04"), l.ID[:7], l.Content, status)
	}
	fmt.Println("=================================")
	return nil
}

var logCmd = &cobra.Command{
	Use:   "log [entry]",
	Short: "Append a micro-log to today's stream or view today's timeline",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return displayTodayLogs(time.Now())
		}

		entry := strings.Join(args, " ")
		l, err := db.CreateDailyLog(entry)
		if err != nil {
			return err
		}

		fmt.Printf("Logged [%s] at %s: %s\n", l.ID[:7], l.CreatedAt.Format("15:04"), l.Content)
		return nil
	},
}

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "View today's chronological stream of logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		return displayTodayLogs(time.Now())
	},
}

func init() {
	logCmd.Flags().BoolVarP(&showAllLogs, "all", "a", false, "Show all logs including promoted ones")
	rootCmd.AddCommand(logCmd)
	rootCmd.AddCommand(todayCmd)
}
