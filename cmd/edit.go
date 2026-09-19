package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/utils"
)

var (
	editTitle      string
	editType       string
	editStatus     string
	editArea       string
	editDue        string
	openEditorFlag bool
)

func interactiveEditNote(n *models.Note) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		freshNote, err := db.GetNote(n.ID)
		if err == nil {
			n = freshNote
		}
		tags, _ := db.GetTagsForNote(n.ID)
		links, _ := db.GetLinksForNote(n.ID)

		var tagNames []string
		for _, t := range tags {
			tagNames = append(tagNames, "#"+t.Name)
		}
		tagStr := strings.Join(tagNames, ", ")
		if tagStr == "" {
			tagStr = "(none)"
		}

		dueStr := "none"
		if !n.TargetDateTime.IsZero() {
			dueStr = n.TargetDateTime.Format("2006-01-02 15:04")
		}

		fleshPreview := "(empty)"
		if strings.TrimSpace(n.NoteFlesh) != "" {
			lines := strings.Split(strings.TrimSpace(n.NoteFlesh), "\n")
			firstLine := lines[0]
			if len(firstLine) > 50 {
				firstLine = firstLine[:47] + "..."
			}
			fleshPreview = fmt.Sprintf("%q (%d chars)", firstLine, len(n.NoteFlesh))
		}

		fmt.Println()
		fmt.Printf("=== Editing Note [%s] ===\n", utils.ShortID(n.ID))
		fmt.Printf("1. Title:   %s\n", n.Note)
		fmt.Printf("2. Body:    %s\n", fleshPreview)
		fmt.Printf("3. Type:    %s\n", n.Type)
		fmt.Printf("4. Status:  %s\n", n.Status)
		fmt.Printf("5. Area:    %s\n", n.Area)
		fmt.Printf("6. Due:     %s\n", dueStr)
		fmt.Printf("7. Tags:    %s\n", tagStr)
		fmt.Printf("8. Links:   %d linked note(s)\n", len(links))
		fmt.Println("--------------------------------------------------")
		fmt.Print("Select option to edit [1-8] or [q/Enter] to finish: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" || input == "q" || input == "quit" || input == "exit" {
			fmt.Printf("✔ Finished editing note [%s].\n", utils.ShortID(n.ID))
			return nil
		}

		switch input {
		case "1", "title", "name":
			fmt.Printf("Current title: %s\n", n.Note)
			fmt.Print("Enter new title (press Enter to keep): ")
			newTitle, _ := reader.ReadString('\n')
			newTitle = strings.TrimSpace(newTitle)
			if newTitle != "" {
				n.Note = newTitle
				if err := db.UpdateNote(n); err != nil {
					fmt.Printf("❌ Failed to update title: %v\n", err)
				} else {
					fmt.Println("✔ Title updated.")
				}
			}

		case "2", "body", "flesh", "editor":
			newFlesh, err := captureEditorContent(n.NoteFlesh)
			if err != nil {
				fmt.Printf("❌ Editor error: %v\n", err)
			} else {
				n.NoteFlesh = strings.TrimSpace(newFlesh)
				if err := db.UpdateNote(n); err != nil {
					fmt.Printf("❌ Failed to update body: %v\n", err)
				} else {
					fmt.Println("✔ Body content updated.")
				}
			}

		case "3", "type":
			fmt.Println("Available types: [1] note  [2] todo  [3] project  [4] idea  [5] concept  [6] decision  [7] til  [8] question  [9] resource  [10] experiment")
			fmt.Print("Enter type name or number (press Enter to keep): ")
			tInput, _ := reader.ReadString('\n')
			tInput = strings.TrimSpace(strings.ToLower(tInput))
			typeMap := map[string]models.NoteType{
				"1": models.DefaultNote, "note": models.DefaultNote,
				"2": models.Todo, "todo": models.Todo,
				"3": models.Project, "project": models.Project,
				"4": models.Idea, "idea": models.Idea,
				"5": models.Concept, "concept": models.Concept,
				"6": models.Decision, "decision": models.Decision,
				"7": models.TIL, "til": models.TIL,
				"8": models.Question, "question": models.Question,
				"9": models.Resource, "resource": models.Resource,
				"10": models.Experiment, "experiment": models.Experiment,
			}
			if newT, ok := typeMap[tInput]; ok {
				n.Type = newT
				if err := db.UpdateNote(n); err != nil {
					fmt.Printf("❌ Failed to update type: %v\n", err)
				} else {
					fmt.Printf("✔ Type updated to %s.\n", newT)
				}
			} else if tInput != "" {
				n.Type = models.NoteType(tInput)
				if err := db.UpdateNote(n); err != nil {
					fmt.Printf("❌ Failed to update type: %v\n", err)
				} else {
					fmt.Printf("✔ Type updated to %s.\n", tInput)
				}
			}

		case "4", "status":
			fmt.Println("Available statuses: [1] active  [2] in-progress  [3] refined  [4] completed  [5] raw  [6] archived")
			fmt.Print("Enter status name or number (press Enter to keep): ")
			sInput, _ := reader.ReadString('\n')
			sInput = strings.TrimSpace(strings.ToLower(sInput))
			statusMap := map[string]models.Status{
				"1": models.Active, "active": models.Active,
				"2": models.InProgress, "in-progress": models.InProgress,
				"3": models.Refined, "refined": models.Refined,
				"4": models.Completed, "completed": models.Completed,
				"5": models.Raw, "raw": models.Raw,
				"6": models.Archived, "archived": models.Archived,
			}
			if newS, ok := statusMap[sInput]; ok {
				n.Status = newS
				if err := db.UpdateNote(n); err != nil {
					fmt.Printf("❌ Failed to update status: %v\n", err)
				} else {
					fmt.Printf("✔ Status updated to %s.\n", newS)
				}
			} else if sInput != "" {
				n.Status = models.Status(sInput)
				if err := db.UpdateNote(n); err != nil {
					fmt.Printf("❌ Failed to update status: %v\n", err)
				} else {
					fmt.Printf("✔ Status updated to %s.\n", sInput)
				}
			}

		case "5", "area":
			fmt.Println("Common areas: [1] work  [2] personal  [3] finance  [c] clear")
			fmt.Print("Enter area name or number (press Enter to keep): ")
			aInput, _ := reader.ReadString('\n')
			aInput = strings.TrimSpace(aInput)
			switch strings.ToLower(aInput) {
			case "1", "work":
				n.Area = models.Work
			case "2", "personal":
				n.Area = models.Personal
			case "3", "finance":
				n.Area = models.Finance
			case "c", "clear", "none":
				n.Area = ""
			default:
				if aInput != "" {
					n.Area = models.Area(aInput)
				}
			}
			if aInput != "" {
				if err := db.UpdateNote(n); err != nil {
					fmt.Printf("❌ Failed to update area: %v\n", err)
				} else {
					fmt.Printf("✔ Area updated to %q.\n", n.Area)
				}
			}

		case "6", "due":
			fmt.Print("Enter due date (e.g. today, tomorrow, +3d, 2026-09-20, or 'clear'): ")
			dInput, _ := reader.ReadString('\n')
			dInput = strings.TrimSpace(dInput)
			if dInput != "" {
				if strings.EqualFold(dInput, "clear") || strings.EqualFold(dInput, "none") {
					n.TargetDateTime = time.Time{}
					if err := db.UpdateNote(n); err != nil {
						fmt.Printf("❌ Failed to clear due date: %v\n", err)
					} else {
						fmt.Println("✔ Due date cleared.")
					}
				} else {
					targetDT, err := utils.ParseDate(dInput)
					if err != nil {
						fmt.Printf("❌ Invalid date: %v\n", err)
					} else {
						n.TargetDateTime = targetDT
						if err := db.UpdateNote(n); err != nil {
							fmt.Printf("❌ Failed to set due date: %v\n", err)
						} else {
							fmt.Printf("✔ Due date set to %s.\n", targetDT.Format("2006-01-02 15:04"))
						}
					}
				}
			}

		case "7", "tags", "tag":
			fmt.Printf("Current tags: %s\n", tagStr)
			fmt.Print("Action: [a]dd tag, [r]emove tag, or [Enter] back: ")
			tAction, _ := reader.ReadString('\n')
			tAction = strings.TrimSpace(strings.ToLower(tAction))
			if tAction == "a" || tAction == "add" {
				fmt.Print("Enter tag name to add: ")
				tName, _ := reader.ReadString('\n')
				tName = strings.TrimSpace(tName)
				if tName != "" {
					if err := db.AddTag(n.ID, tName); err != nil {
						fmt.Printf("❌ Failed to add tag: %v\n", err)
					} else {
						fmt.Printf("✔ Added tag #%s.\n", tName)
					}
				}
			} else if tAction == "r" || tAction == "remove" || tAction == "del" {
				fmt.Print("Enter tag name to remove: ")
				tName, _ := reader.ReadString('\n')
				tName = strings.TrimSpace(tName)
				if tName != "" {
					if err := db.RemoveTag(n.ID, tName); err != nil {
						fmt.Printf("❌ Failed to remove tag: %v\n", err)
					} else {
						fmt.Printf("✔ Removed tag #%s.\n", tName)
					}
				}
			}

		case "8", "link", "links":
			fmt.Print("Enter target note ID to link to: ")
			targetID, _ := reader.ReadString('\n')
			targetID = strings.TrimSpace(targetID)
			if targetID != "" {
				fmt.Print("Enter link type (related_to, depends_on, part_of, supports, contradicts) [default: related_to]: ")
				lType, _ := reader.ReadString('\n')
				lType = strings.TrimSpace(lType)
				if lType == "" {
					lType = string(models.RelatedTo)
				}
				if err := db.AddLink(n.ID, targetID, models.LinkType(lType)); err != nil {
					fmt.Printf("❌ Failed to create link: %v\n", err)
				} else {
					fmt.Printf("✔ Created link: [%s] --(%s)--> [%s]\n", utils.ShortID(n.ID), lType, targetID)
				}
			}

		default:
			fmt.Println("Unrecognized option. Please choose [1-8] or [q] to finish.")
		}
	}
}

func interactiveEditDailyLog(l *models.DailyLog) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		freshLog, err := db.GetDailyLog(l.ID)
		if err == nil {
			l = freshLog
		}

		status := "unpromoted"
		if l.NoteID != "" {
			status = fmt.Sprintf("promoted ➔ note [%s]", utils.ShortID(l.NoteID))
		}

		fmt.Println()
		fmt.Printf("=== Editing Daily Log [%s] ===\n", utils.ShortID(l.ID))
		fmt.Printf("1. Content: %s\n", l.Content)
		fmt.Printf("2. Status:  %s\n", status)
		fmt.Printf("3. Created: %s\n", l.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println("--------------------------------------------------")
		fmt.Print("Actions: [1] edit content | [2] promote to Note | [3] soft-delete | [q/Enter] finish: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" || input == "q" || input == "quit" || input == "exit" {
			fmt.Printf("✔ Finished editing daily log [%s].\n", utils.ShortID(l.ID))
			return nil
		}

		switch input {
		case "1", "edit", "content":
			fmt.Printf("Current content: %s\n", l.Content)
			fmt.Print("Enter new content (or press Enter to keep): ")
			newContent, _ := reader.ReadString('\n')
			newContent = strings.TrimSpace(newContent)
			if newContent != "" {
				l.Content = newContent
				if err := db.UpdateDailyLog(l); err != nil {
					fmt.Printf("❌ Failed to update daily log: %v\n", err)
				} else {
					fmt.Println("✔ Daily log content updated.")
				}
			}

		case "2", "promote":
			if l.NoteID != "" {
				fmt.Printf("⚠️  Log is already promoted to note [%s].\n", utils.ShortID(l.NoteID))
				continue
			}
			fmt.Print("Enter note type (todo, project, idea, note) [default: note]: ")
			nType, _ := reader.ReadString('\n')
			nType = strings.TrimSpace(nType)
			if nType == "" {
				nType = "note"
			}
			n, err := db.PromoteDailyLog(l.ID, models.NoteType(nType), models.Raw)
			if err != nil {
				fmt.Printf("❌ Failed to promote log: %v\n", err)
			} else {
				fmt.Printf("✔ Successfully promoted log to Note [%s] (%s)!\n", utils.ShortID(n.ID), n.Type)
			}

		case "3", "delete":
			if err := db.SoftDeleteDailyLog(l.ID, "deleted during edit"); err != nil {
				fmt.Printf("❌ Failed to delete log: %v\n", err)
			} else {
				fmt.Printf("✔ Soft-deleted daily log [%s].\n", utils.ShortID(l.ID))
				return nil
			}

		default:
			fmt.Println("Unrecognized option. Please choose [1-3] or [q] to finish.")
		}
	}
}

var editCmd = &cobra.Command{
	Use:     "edit [id] [new_title]",
	Aliases: []string{"flesh", "rename", "update"},
	Short:   "Edit note or daily log (fuzzy-select and interactive menu if args omitted)",
	Long: `Edit an existing note or daily log.

When run without arguments (or with only an ID), launches an interactive menu to edit title, body ($EDITOR), type, status, area, due date, tags, and links.

Examples:
  # Interactive fuzzy-select and edit:
  kb edit

  # Interactive edit for a specific note:
  kb edit f59a66e

  # Update title directly:
  kb edit f59a66e "today's note"
  kb rename f59a66e "today's note"

  # Update title and metadata via flags:
  kb edit f59a66e -n "today's note" --status completed --type project

  # Open $EDITOR directly to edit body (note flesh):
  kb edit f59a66e -e
  kb flesh f59a66e`,
	Args: cobra.RangeArgs(0, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		hasPositionalTitle := len(args) == 2
		hasFlagUpdates := editTitle != "" || editType != "" || editStatus != "" || editArea != "" || editDue != ""

		// 1. Fuzzy finder mode if no args provided
		if len(args) == 0 {
			items, err := GetActiveCLIItems()
			if err != nil {
				return err
			}

			if len(items) == 0 {
				fmt.Println("No active notes or daily logs found to edit.")
				return nil
			}

			selected, err := SelectCLIItem(items)
			if err != nil {
				return err
			}
			if selected == nil {
				return nil
			}

			if selected.IsLog {
				l, err := db.GetDailyLog(selected.ID)
				if err != nil {
					return err
				}
				return interactiveEditDailyLog(l)
			}

			n, err := db.GetNote(selected.ID)
			if err != nil {
				return err
			}
			return interactiveEditNote(n)
		}

		// 2. ID passed as arg
		id := args[0]
		if hasPositionalTitle {
			editTitle = args[1]
		}

		// Try resolving as Note first
		if n, err := db.GetNote(id); err == nil {
			if openEditorFlag {
				updatedContent, err := captureEditorContent(n.NoteFlesh)
				if err != nil {
					return fmt.Errorf("failed to open editor: %w", err)
				}
				n.NoteFlesh = strings.TrimSpace(updatedContent)
				if err := db.UpdateNote(n); err != nil {
					return fmt.Errorf("failed to update note: %w", err)
				}
				fmt.Printf("Successfully updated note [%s] %s\n", utils.ShortID(n.ID), n.Note)
				return nil
			}

			if hasPositionalTitle || hasFlagUpdates {
				if editTitle != "" {
					n.Note = strings.TrimSpace(editTitle)
				}
				if editType != "" {
					n.Type = models.NoteType(editType)
				}
				if editStatus != "" {
					n.Status = models.Status(editStatus)
				}
				if editArea != "" {
					n.Area = models.Area(editArea)
				}
				if editDue != "" {
					if editDue == "clear" || editDue == "none" {
						n.TargetDateTime = time.Time{}
					} else {
						targetDT, err := utils.ParseDate(editDue)
						if err != nil {
							return fmt.Errorf("invalid due date %q: %w", editDue, err)
						}
						n.TargetDateTime = targetDT
					}
				}

				if err := db.UpdateNote(n); err != nil {
					return fmt.Errorf("failed to update note: %w", err)
				}

				fmt.Printf("Successfully updated note [%s] %s\n", utils.ShortID(n.ID), n.Note)
				return nil
			}

			// If only ID was passed without flags or new title, enter interactive mode for this note!
			return interactiveEditNote(n)
		}

		// Try resolving as Daily Log
		if l, err := db.GetDailyLog(id); err == nil {
			if hasPositionalTitle || editTitle != "" {
				newContent := editTitle
				if newContent == "" && hasPositionalTitle {
					newContent = args[1]
				}
				l.Content = strings.TrimSpace(newContent)
				if err := db.UpdateDailyLog(l); err != nil {
					return fmt.Errorf("failed to update daily log: %w", err)
				}
				fmt.Printf("Successfully updated daily log [%s]\n", utils.ShortID(l.ID))
				return nil
			}
			return interactiveEditDailyLog(l)
		}

		return fmt.Errorf("no note or daily log found matching ID %q", id)
	},
}

func init() {
	editCmd.Flags().StringVarP(&editTitle, "note", "n", "", "New title or summary of the note")
	editCmd.Flags().StringVarP(&editTitle, "title", "", "", "Alias for --note")
	editCmd.Flags().StringVarP(&editType, "type", "t", "", "New note type (e.g. todo, project, note, idea)")
	editCmd.Flags().StringVarP(&editStatus, "status", "s", "", "New status (active, raw, refined, in-progress, completed, archived)")
	editCmd.Flags().StringVarP(&editArea, "area", "a", "", "New area (work, personal, finance)")
	editCmd.Flags().StringVarP(&editDue, "due", "d", "", "New due date or target date ('clear' to remove)")
	editCmd.Flags().BoolVarP(&openEditorFlag, "editor", "e", false, "Open editor directly to edit body")

	editCmd.ValidArgsFunction = completeDeletableIDs
	editCmd.RegisterFlagCompletionFunc("type", completeNoteTypes)
	editCmd.RegisterFlagCompletionFunc("status", completeNoteStatuses)
	editCmd.RegisterFlagCompletionFunc("area", completeAreas)

	rootCmd.AddCommand(editCmd)
}
