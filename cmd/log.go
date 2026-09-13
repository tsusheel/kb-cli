package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/utils"
)


var showAllLogs bool

func displayTodayLogs(date time.Time, includePromoted bool) error {
	logs, err := db.GetDailyLogsForDate(date, includePromoted)
	if err != nil {
		return err
	}

	if len(logs) == 0 {
		fmt.Printf("No logs recorded for %s.\n", date.Format("2006-01-02"))
		return nil
	}

	fmt.Printf("=== [%s] Daily Stream (%d) ===\n", date.Format("2006-01-02"), len(logs))
	utils.RenderDailyLogsTable(logs, os.Stdout)
	return nil
}


var logCmd = &cobra.Command{
	Use:   "log [entry]",
	Short: "Append a micro-log to today's stream or view today's timeline",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return displayTodayLogs(time.Now(), showAllLogs)
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

func init() {
	logCmd.Flags().BoolVarP(&showAllLogs, "all", "a", false, "Show all logs including promoted ones")
	rootCmd.AddCommand(logCmd)
}

