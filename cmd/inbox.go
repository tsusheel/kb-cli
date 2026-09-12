package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/utils"
)

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "View unrefined raw notes and unpromoted daily logs awaiting triage",
	RunE: func(cmd *cobra.Command, args []string) error {
		rawNotes, err := db.ListNotesExtended("", string(models.Raw), "", false)
		if err != nil {
			return err
		}

		unpromotedLogs, err := db.GetUnpromotedDailyLogs()
		if err != nil {
			return err
		}

		if len(rawNotes) == 0 && len(unpromotedLogs) == 0 {
			fmt.Println("Inbox is clear! No raw notes or unpromoted logs.")
			return nil
		}

		if len(rawNotes) > 0 {
			fmt.Printf("=== Raw Notes Awaiting Triage (%d) ===\n", len(rawNotes))
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, n := range rawNotes {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", n.ID[:7], n.Note, n.Type, n.UpdatedAt.Format("2006-01-02 15:04"))
			}
			w.Flush()
			fmt.Println()
		}

		if len(unpromotedLogs) > 0 {
			fmt.Printf("=== Unpromoted Daily Logs (%d) ===\n", len(unpromotedLogs))
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, l := range unpromotedLogs {
				fmt.Fprintf(w, "%s\t%s\t%s\n", l.ID[:7], l.Content, l.CreatedAt.Format("2006-01-02 15:04"))
			}
			w.Flush()
			fmt.Println()
		}

		fmt.Println("Tip: Run 'kb triage' to interactively process your inbox.")
		return nil
	},
}

var triageCmd = &cobra.Command{
	Use:   "triage",
	Short: "Interactive triage wizard to process raw notes and fleeting thoughts",
	RunE: func(cmd *cobra.Command, args []string) error {
		rawNotes, err := db.ListNotesExtended("", string(models.Raw), "", false)
		if err != nil {
			return err
		}

		if len(rawNotes) == 0 {
			fmt.Println("No raw notes in inbox to triage!")
			return nil
		}

		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Starting triage of %d raw notes...\n\n", len(rawNotes))

		for i, n := range rawNotes {
			fmt.Printf("--------------------------------------------------\n")
			fmt.Printf("[%d/%d] Note: [%s] %s (%s)\n", i+1, len(rawNotes), n.ID[:7], n.Note, n.Type)
			if n.NoteFlesh != "" {
				fmt.Printf("Flesh: %s\n", n.NoteFlesh)
			}
			fmt.Printf("Actions: [r]efine flesh | [t]ype | [s]tatus | [d]ue | [a]dd tag | [c]omplete | [D]elete | [Enter] next | [q]uit\n")
			fmt.Print("> ")

			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)

			if choice == "q" || choice == "quit" {
				fmt.Println("Triage exited.")
				break
			}

			switch choice {
			case "r", "refine":
				flesh, err := captureEditorContent(n.NoteFlesh)
				if err == nil {
					n.NoteFlesh = strings.TrimSpace(flesh)
					n.Status = models.Refined
					db.UpdateNote(&n)
					fmt.Printf("Note marked as refined.\n")
				}
			case "t", "type":
				fmt.Print("Enter new type (todo, idea, project, concept, decision, note): ")
				newType, _ := reader.ReadString('\n')
				newType = strings.TrimSpace(newType)
				if newType != "" {
					n.Type = models.NoteType(newType)
					db.UpdateNote(&n)
					fmt.Printf("Type updated to %s.\n", newType)
				}
			case "s", "status":
				fmt.Print("Enter status (active, refined, in-progress, completed, archived): ")
				newStatus, _ := reader.ReadString('\n')
				newStatus = strings.TrimSpace(newStatus)
				if newStatus != "" {
					n.Status = models.Status(newStatus)
					db.UpdateNote(&n)
					fmt.Printf("Status updated to %s.\n", newStatus)
				}
			case "d", "due":
				fmt.Print("Enter due date (e.g. today, tomorrow, monday, +3d, 2026-09-10): ")
				dueInput, _ := reader.ReadString('\n')
				dueInput = strings.TrimSpace(dueInput)
				if dueInput != "" {
					targetDT, err := utils.ParseDate(dueInput)
					if err == nil {
						n.TargetDateTime = targetDT
						db.UpdateNote(&n)
						fmt.Printf("Target date set to %s.\n", targetDT.Format("2006-01-02"))
					} else {
						fmt.Printf("Invalid date: %v\n", err)
					}
				}
			case "a", "tag":
				fmt.Print("Enter tag name: ")
				tagInput, _ := reader.ReadString('\n')
				tagInput = strings.TrimSpace(tagInput)
				if tagInput != "" {
					db.AddTag(n.ID, tagInput)
					fmt.Printf("Added tag %s.\n", tagInput)
				}
			case "c", "complete":
				n.Status = models.Completed
				db.UpdateNote(&n)
				fmt.Println("Note marked as completed.")
			case "D", "delete":
				db.SoftDeleteNote(n.ID, "deleted during triage")
				fmt.Println("Note deleted.")
			default:
				// Skip to next
			}
		}

		fmt.Println("\nTriage session finished!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(inboxCmd)
	rootCmd.AddCommand(triageCmd)
}
