package cmd

import (
	"fmt"
	"path"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tsusheel/kb-cli/docs"
	"github.com/tsusheel/kb-cli/server"
)

var (
	docsOpenWeb  bool
	docsShowTree bool
)

const commandHierarchyTree = `kb-cli Command Hierarchy & Option Reference
===========================================

kb [thought] (positional capture: saves note with status='raw' in <25ms)
│
├── add [text]                         Create a new structured note
│   ├── -n, --note <string>            Title or summary of the note
│   ├── -t, --type <string>            Note type (note, todo, idea, project, decision, til, ...) [default: note]
│   ├── -s, --status <string>          Status (raw, active, refined, in-progress, completed, archived)
│   ├── -a, --area <string>            Domain area (work, personal, finance, health, tech, ...)
│   ├── -d, --due <date>               Target / deadline date (today, tomorrow, +3d, YYYY-MM-DD)
│   ├──     --tags <strings>           Comma-separated tags to attach to the note
│   └── -e, --edit                     Open $EDITOR to write long-form note body (flesh)
│
├── list [query]                       List and filter notes with fuzzy keyword matching (ls, search, find)
│   ├── -t, --type <string>            Filter by note type
│   ├── -s, --status <string>          Filter by lifecycle status
│   ├── -a, --area <string>            Filter by domain area
│   ├── -d, --todos                    Shorthand: list only todos
│   ├── -n, --notes                    Shorthand: list only notes
│   └── -p, --projects                 Shorthand: list only projects
│
├── edit [id] [new_title]              Interactive TUI edit menu or direct flag-based updates (flesh, rename)
│   ├── -n, --note, --title <string>   Update title or headline
│   ├── -t, --type <string>            Update note type
│   ├── -s, --status <string>          Update lifecycle status
│   ├── -a, --area <string>            Update domain area
│   ├── -d, --due <date>               Update due date ('clear' to remove)
│   └── -e, --editor                   Open configured $EDITOR directly to edit note flesh
│
├── open [id]                          Inspect full note or log details (launches fuzzy selector if ID omitted)
│
├── delete [id...]                     Non-destructive soft-delete (interactive multi-select if ID omitted) (rm, del)
│   └── -r, --reason <string>          Attribution reason for audit trail [default: 'deleted by user']
│
├── restore [id...]                    Restore soft-deleted note or log (interactive multi-select if ID omitted) (undelete)
│
├── inbox                              View raw notes awaiting triage (today's capture by default)
│   └── -a, --all                      Show all raw notes across all dates
│
├── triage                             Interactive step-by-step triage wizard for raw notes
│   └── -a, --all                      Process raw notes across all dates (not just today)
│
├── log [entry]                        Micro-logging stream and timeline
│   ├── -a, --all                      Display all logs including already-promoted entries
│   ├── -A, --all-time                 Display historical logs across all dates
│   ├── -d, --date <date>              Filter logs for a specific date (yesterday, YYYY-MM-DD)
│   ├── -f, --from <date>              Filter logs starting from date (inclusive, -7d, 2026-09-01)
│   └── -t, --to <date>                Filter logs up to date (inclusive, today, 2026-09-13)
│
├── promote <log_id>                   Promote a daily log entry into a permanent note
│   ├── -t, --type <string>            Target note type [default: note]
│   └── -s, --status <string>          Target note status [default: active]
│
├── link <from_id> <to_id>             Create a bidirectional knowledge graph connection between two notes
│   └── -t, --type <string>            Link type (related_to, part_of, depends_on, ...) [default: related_to]
│
├── backlinks <id>                     Display all incoming references pointing to a note (bl, incoming)
│
├── graph [id]                         Display interactive ASCII relationship tree for a note (fuzzy if ID omitted)
│
├── orphans                            List unlinked and untagged notes requiring connection or triage (isolated)
│
├── history [id]                       View audit logs, modification timelines, and snapshot revisions (audit)
│   ├── -l, --limit <int>              Maximum number of audit records to display [default: 30]
│   ├── diff <note_id> <rev_id>        Inspect before/after field diff for a specific revision
│   └── revert <note_id> <rev_id>      Roll back a note to a previous revision snapshot
│
├── stats                              Display knowledge base metrics, activity streaks, and graph distribution
│
├── config                             Manage configuration settings and OS Keyring encrypted secrets
│   ├── list                           List all configuration settings and credential statuses
│   ├── get <key>                      Retrieve a specific configuration value
│   ├── set <key> <val>                Write a configuration key/value pair to config.yaml
│   ├── set-secret <key>               Securely prompt for password/token and store in OS Credential Manager
│   ├── delete <key>                   Remove a configuration option or keyring secret
│   └── setup                          Interactive configuration wizard for remote PostgreSQL sync
│
├── sync                               Bidirectional PostgreSQL sync engine with conflict resolution
│   ├── pull                           Pull remote modifications into local SQLite database
│   ├── push                           Push local modifications to remote PostgreSQL database
│   ├── status                         Display remote sync timestamp watermarks and server status
│   └── test                           Verify connection and latency to remote PostgreSQL
│
├── serve                              Start background service daemons (mcp)
│   ├── http                           Start embedded Web Fuzzy Finder with live Markdown & Mermaid preview
│   └── mcp                            Start Model Context Protocol (MCP) server over stdio for LLMs / AI agents
│
├── export                             Export entire knowledge base to Markdown vault or unified JSON dump
│   ├── -d, --dir <path>               Target directory for export [default: ./kb-export]
│   ├── -f, --format <format>          Export format: 'markdown' (Obsidian frontmatter) or 'json' [default: markdown]
│   └── -a, --include-deleted          Include soft-deleted notes and logs in the export
│
├── import                             Batch import Markdown notes with frontmatter from a directory
│   └── -d, --dir <path>               Directory containing Markdown files to parse and import
│
├── docs [topic]                       Browse and search system architecture documentation (doc, documentation)
│   ├── -t, --tree                     Display complete CLI command hierarchy and option tree
│   └── -w, --web                      Open documentation in default web browser
│
└── completion <shell>                 Generate shell completion scripts (bash, zsh, fish, powershell)
`

type docOption struct {
	Title    string
	Path     string
	Category string
}

func getFlattenedDocOptions() ([]docOption, error) {
	tree, err := docs.GetDocsTree()
	if err != nil {
		return nil, err
	}

	var options []docOption
	for _, folder := range tree {
		for _, child := range folder.Children {
			options = append(options, docOption{
				Title:    child.Title,
				Path:     child.Path,
				Category: folder.Title,
			})
		}
	}
	return options, nil
}

func findMatchingDoc(query string, options []docOption) *docOption {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}

	// Exact or substring match on path/filename
	for _, opt := range options {
		base := strings.ToLower(path.Base(opt.Path))
		if strings.Contains(base, query) || strings.EqualFold(opt.Path, query) {
			return &opt
		}
	}

	// Substring match on title
	for _, opt := range options {
		if strings.Contains(strings.ToLower(opt.Title), query) {
			return &opt
		}
	}

	return nil
}

func displayDocContent(docPath string) error {
	content, err := docs.ReadDocFile(docPath)
	if err != nil {
		return fmt.Errorf("error reading doc %q: %w", docPath, err)
	}

	fmt.Println(content)
	return nil
}

var docsTreeCmd = &cobra.Command{
	Use:     "tree",
	Aliases: []string{"commands", "cmd"},
	Short:   "Display complete CLI command hierarchy and option tree diagram",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print(commandHierarchyTree)
		return nil
	},
}

var docsCmd = &cobra.Command{
	Use:     "docs [topic]",
	Aliases: []string{"doc", "documentation"},
	Short:   "Browse, search, or view documentation in terminal or browser",
	Long: `Browse and search the built-in kb-cli architecture documentation, guides, and extension proposals.

Run without arguments to launch an interactive terminal fuzzy-finder with live preview.
Pass a topic keyword to directly jump to a matching document.
Use the --tree flag to view the full command hierarchy and option tree at a glance.
Use the --web flag to open the documentation in your web browser.

Examples:
  kb docs                  # Interactive fuzzy selector with live preview
  kb docs --tree           # View full CLI command hierarchy & option tree at a glance
  kb docs tree             # View command tree via subcommand
  kb docs sync             # Directly view PostgreSQL sync documentation
  kb docs mcp              # Directly view Model Context Protocol server docs
  kb docs thingspad        # View ThingsPad touch console proposal
  kb docs --web            # Open documentation in default web browser
  kb docs sync -w          # Open sync docs in web browser`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Command hierarchy tree requested via flag or keyword
		if docsShowTree || (len(args) > 0 && (strings.EqualFold(args[0], "tree") || strings.EqualFold(args[0], "commands") || strings.EqualFold(args[0], "cmd") || strings.EqualFold(args[0], "hierarchy"))) {
			fmt.Print(commandHierarchyTree)
			return nil
		}

		options, err := getFlattenedDocOptions()
		if err != nil {
			return err
		}

		// 2. Direct topic argument provided
		if len(args) > 0 {
			target := findMatchingDoc(args[0], options)
			if target == nil {
				return fmt.Errorf("no documentation found matching query %q", args[0])
			}

			if docsOpenWeb {
				port := 8080
				if viper.IsSet("server.port") {
					port = viper.GetInt("server.port")
				}
				url := fmt.Sprintf("http://127.0.0.1:%d/docs/%s", port, target.Path)
				fmt.Printf("Opening %s in web browser...\n", url)
				return server.OpenInBrowser(url)
			}

			return displayDocContent(target.Path)
		}

		// 3. Web mode requested without topic
		if docsOpenWeb {
			port := 8080
			if viper.IsSet("server.port") {
				port = viper.GetInt("server.port")
			}
			url := fmt.Sprintf("http://127.0.0.1:%d/docs", port)
			fmt.Printf("Opening documentation at %s...\n", url)
			return server.OpenInBrowser(url)
		}

		// 4. Interactive fuzzy finder
		idx, err := fuzzyfinder.Find(
			options,
			func(i int) string {
				return fmt.Sprintf("[%s] %s (%s)", options[i].Category, options[i].Title, options[i].Path)
			},
			fuzzyfinder.WithPreviewWindow(func(i int, width, height int) string {
				if i < 0 || i >= len(options) {
					return ""
				}
				content, err := docs.ReadDocFile(options[i].Path)
				if err != nil {
					return fmt.Sprintf("Error loading preview: %v", err)
				}
				return content
			}),
		)

		if err != nil {
			if err == fuzzyfinder.ErrAbort {
				return nil
			}
			return err
		}

		selected := options[idx]
		return displayDocContent(selected.Path)
	},
}

func init() {
	docsCmd.Flags().BoolVarP(&docsOpenWeb, "web", "w", false, "Open documentation in default web browser")
	docsCmd.Flags().BoolVarP(&docsShowTree, "tree", "t", false, "Display complete CLI command hierarchy and option tree")
	docsCmd.AddCommand(docsTreeCmd)
	rootCmd.AddCommand(docsCmd)
}
