package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/utils"
)

var (
	inboxShowAll  bool
	triageShowAll bool
)

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "View raw notes awaiting triage (today's notes by default, or all with -a/--all)",
	RunE: func(cmd *cobra.Command, args []string) error {
		rawNotes, err := db.GetRawNotes(!inboxShowAll)
		if err != nil {
			return err
		}

		if len(rawNotes) == 0 {
			if inboxShowAll {
				fmt.Println("Inbox is clear! No raw notes.")
			} else {
				fmt.Println("Today's inbox is clear! No raw notes for today.")
				fmt.Println("Tip: Run 'kb inbox -a' to view all raw notes across all dates.")
			}
			return nil
		}

		if inboxShowAll {
			fmt.Printf("=== All Raw Notes Awaiting Triage (%d) ===\n", len(rawNotes))
		} else {
			fmt.Printf("=== Today's Raw Notes Awaiting Triage (%d) ===\n", len(rawNotes))
		}

		utils.RenderInboxTable(rawNotes, os.Stdout)

		fmt.Println("\nTip: Run 'kb triage' to interactively process today's notes ('kb triage -a' for all).")
		return nil
	},
}

var triageCmd = &cobra.Command{
	Use:   "triage",
	Short: "Interactive triage wizard to process raw notes (today's notes by default, or all with -a/--all)",
	RunE: func(cmd *cobra.Command, args []string) error {
		rawNotes, err := db.GetRawNotes(!triageShowAll)
		if err != nil {
			return err
		}

		if len(rawNotes) == 0 {
			if triageShowAll {
				fmt.Println("No raw notes to triage!")
			} else {
				fmt.Println("No raw notes for today to triage! Run 'kb triage -a' to triage all raw notes.")
			}
			return nil
		}

		reader := bufio.NewReader(os.Stdin)
		headerScope := "today's"
		if triageShowAll {
			headerScope = "all"
		}
		fmt.Printf("Starting triage of %d %s raw notes...\n\n", len(rawNotes), headerScope)

		for i, n := range rawNotes {
			fmt.Printf("--------------------------------------------------\n")
			fmt.Printf("[%d/%d] Note: [%s] %s (%s)\n", i+1, len(rawNotes), utils.ShortID(n.ID), n.Note, n.Type)
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
					fmt.Printf("✔ Note marked as refined.\n")
				}
			case "t", "type":
				fmt.Print("Enter new type (todo, idea, project, concept, decision, note): ")
				newType, _ := reader.ReadString('\n')
				newType = strings.TrimSpace(newType)
				if newType != "" {
					n.Type = models.NoteType(newType)
					if n.Status == models.Raw {
						n.Status = models.Active
					}
					db.UpdateNote(&n)
					fmt.Printf("✔ Type updated to %s (status marked active).\n", newType)
				}
			case "s", "status":
				fmt.Print("Enter status (active, refined, in-progress, completed, archived): ")
				newStatus, _ := reader.ReadString('\n')
				newStatus = strings.TrimSpace(newStatus)
				if newStatus != "" {
					n.Status = models.Status(newStatus)
					db.UpdateNote(&n)
					fmt.Printf("✔ Status updated to %s.\n", newStatus)
				}
			case "d", "due":
				fmt.Print("Enter due date (e.g. today, tomorrow, monday, +3d, 2026-09-10): ")
				dueInput, _ := reader.ReadString('\n')
				dueInput = strings.TrimSpace(dueInput)
				if dueInput != "" {
					targetDT, err := utils.ParseDate(dueInput)
					if err == nil {
						n.TargetDateTime = targetDT
						if n.Status == models.Raw {
							n.Status = models.Active
						}
						db.UpdateNote(&n)
						fmt.Printf("✔ Target date set to %s (status marked active).\n", targetDT.Format("2006-01-02"))
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
					if n.Status == models.Raw {
						n.Status = models.Active
						db.UpdateNote(&n)
					}
					fmt.Printf("✔ Added tag %s (status marked active).\n", tagInput)
				}
			case "c", "complete":
				n.Status = models.Completed
				db.UpdateNote(&n)
				fmt.Println("✔ Note marked as completed.")
			case "D", "delete":
				db.SoftDeleteNote(n.ID, "deleted during triage")
				fmt.Println("✔ Note deleted.")
			default:
				// Skip to next
			}
		}

		fmt.Println("\nTriage session finished!")
		return nil
	},
}

func init() {
	inboxCmd.Flags().BoolVarP(&inboxShowAll, "all", "a", false, "Show all raw notes across all dates")
	triageCmd.Flags().BoolVarP(&triageShowAll, "all", "a", false, "Triage all raw notes across all dates")

	rootCmd.AddCommand(inboxCmd)
	rootCmd.AddCommand(triageCmd)
}
