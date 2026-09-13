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

var (
	showAllLogs bool
	allTimeLogs bool
	logDate     string
	logFrom     string
	logTo       string
)

var logCmd = &cobra.Command{
	Use:   "log [entry]",
	Short: "Append a micro-log to today's stream or view today's timeline",
	Long: `Append a micro-log to today's stream, or view logs for today, a specific date, or a date range.

Examples:
  # Append a new log entry:
  kb log "Investigated query cache hit rates"

  # View today's unpromoted logs:
  kb log

  # View all of today's logs (including promoted ones):
  kb log -a

  # View logs for a specific date:
  kb log -d 2026-09-12
  kb log -d yesterday -a

  # View logs for an inclusive date range:
  kb log --from 2026-09-01 --to 2026-09-13
  kb log -f -7d -a

  # View all historical logs across all time:
  kb log -A
  kb log --all-time -a`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			entry := strings.Join(args, " ")
			l, err := db.CreateDailyLog(entry)
			if err != nil {
				return err
			}

			fmt.Printf("Logged [%s] at %s: %s\n", l.ID[:7], l.CreatedAt.Format("15:04"), l.Content)
			return nil
		}

		if logDate != "" && (logFrom != "" || logTo != "" || allTimeLogs) {
			return fmt.Errorf("cannot combine --date with --from, --to, or --all-time")
		}

		var startDate *time.Time
		var endDate *time.Time
		var title string

		if allTimeLogs {
			title = "All Daily Logs"
		} else if logDate != "" {
			parsedDate, err := utils.ParseDate(logDate)
			if err != nil {
				return fmt.Errorf("invalid --date format %q: %w", logDate, err)
			}
			startDate = &parsedDate
			endDate = &parsedDate
			title = fmt.Sprintf("[%s] Daily Stream", parsedDate.Format("2006-01-02"))
		} else if logFrom != "" || logTo != "" {
			if logFrom != "" {
				t, err := utils.ParseDate(logFrom)
				if err != nil {
					return fmt.Errorf("invalid --from date format %q: %w", logFrom, err)
				}
				startDate = &t
			}
			if logTo != "" {
				t, err := utils.ParseDate(logTo)
				if err != nil {
					return fmt.Errorf("invalid --to date format %q: %w", logTo, err)
				}
				endDate = &t
			}

			if startDate != nil && endDate == nil {
				now := time.Now()
				endDate = &now
			}

			if startDate != nil && endDate != nil && startDate.After(*endDate) {
				return fmt.Errorf("--from date (%s) cannot be after --to date (%s)", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
			}

			if startDate != nil && endDate != nil {
				title = fmt.Sprintf("[%s to %s] Daily Stream", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
			} else if startDate != nil {
				title = fmt.Sprintf("[from %s] Daily Stream", startDate.Format("2006-01-02"))
			} else if endDate != nil {
				title = fmt.Sprintf("[up to %s] Daily Stream", endDate.Format("2006-01-02"))
			}
		} else {
			// Default to today
			now := time.Now()
			startDate = &now
			endDate = &now
			title = fmt.Sprintf("[%s] Daily Stream", now.Format("2006-01-02"))
		}

		logs, err := db.GetDailyLogsFilter(startDate, endDate, showAllLogs, false)
		if err != nil {
			return err
		}

		if len(logs) == 0 {
			if title != "" {
				fmt.Printf("No logs recorded for %s.\n", title)
			} else {
				fmt.Println("No logs recorded.")
			}
			return nil
		}

		fmt.Printf("=== %s (%d) ===\n", title, len(logs))
		utils.RenderDailyLogsTable(logs, os.Stdout)
		return nil
	},
}

func init() {
	logCmd.Flags().BoolVarP(&showAllLogs, "all", "a", false, "Show all logs including promoted ones")
	logCmd.Flags().BoolVarP(&allTimeLogs, "all-time", "A", false, "Show all logs across all time")
	logCmd.Flags().StringVarP(&logDate, "date", "d", "", "Filter logs for a specific date (e.g. '2026-09-12', 'yesterday')")
	logCmd.Flags().StringVarP(&logFrom, "from", "f", "", "Filter logs starting from date (inclusive, e.g. '2026-09-01', '-7d')")
	logCmd.Flags().StringVarP(&logTo, "to", "t", "", "Filter logs up to date (inclusive, e.g. '2026-09-13', 'today')")
	rootCmd.AddCommand(logCmd)
}
