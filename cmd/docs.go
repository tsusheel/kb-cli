package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/docs"
	"github.com/tsusheel/kb-cli/server"
)

var (
	docsOpenWeb      bool
	docsShowTree     bool
	docsShowCommands bool
	docTagsFlag      []string
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
├── plugin                             Manage external and built-in plugins (plugins)
│   ├── list                           List all discovered and installed plugins
│   ├── install <path>                 Install an external plugin binary
│   ├── enable <id>                    Enable a plugin
│   ├── disable <id>                   Disable a plugin
│   ├── grant <id> <cap>               Grant a capability to a plugin (storage.entries, audit.write, etc.)
│   ├── revoke <id> <cap>              Revoke a capability from a plugin
│   └── doctor <id>                    Run diagnostic health check on a plugin
│
└── completion <shell>                 Generate shell completion scripts (bash, zsh, fish, powershell)
`

type docOption struct {
	ID       string
	Title    string
	Path     string
	Category string
}

func getFlattenedDocOptions() ([]docOption, error) {
	tree, err := db.GetDocsTree()
	if err != nil || len(tree) == 0 {
		_ = db.SeedDocsFromFS(docs.FS)
		tree, err = db.GetDocsTree()
	}
	if err != nil {
		return nil, err
	}

	var options []docOption
	var walk func(items []db.DocTreeItem, category string)
	walk = func(items []db.DocTreeItem, category string) {
		for _, it := range items {
			if it.Type == "directory" {
				subCat := it.Title
				if category != "" {
					subCat = category + " / " + it.Title
				}
				walk(it.Children, subCat)
			} else {
				options = append(options, docOption{
					ID:       it.ID,
					Title:    it.Title,
					Path:     it.Path,
					Category: category,
				})
			}
		}
	}
	walk(tree, "")
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
	node, err := db.GetDocNodeByPath(docPath)
	if err == nil {
		fmt.Println(node.Content)
		return nil
	}

	content, err := docs.ReadDocFile(docPath)
	if err != nil {
		return fmt.Errorf("error reading doc %q: %w", docPath, err)
	}

	fmt.Println(content)
	return nil
}

func formatAsciiDocTree(items []db.DocTreeItem, prefix string) string {
	var sb strings.Builder
	for i, item := range items {
		isLast := i == len(items)-1
		connector := "├── "
		childPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		}

		if item.Type == "directory" {
			sb.WriteString(fmt.Sprintf("%s%s📁 %s/\n", prefix, connector, item.Title))
			if len(item.Children) > 0 {
				sb.WriteString(formatAsciiDocTree(item.Children, childPrefix))
			}
		} else {
			sb.WriteString(fmt.Sprintf("%s%s📄 %s (%s)\n", prefix, connector, item.Title, item.Path))
		}
	}
	return sb.String()
}

var docsTreeCmd = &cobra.Command{
	Use:     "tree",
	Aliases: []string{"dir"},
	Short:   "Display hierarchical documentation tree (or CLI command tree with --commands)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if docsShowCommands {
			fmt.Print(commandHierarchyTree)
			return nil
		}

		tree, err := db.GetDocsTree()
		if err != nil || len(tree) == 0 {
			_ = db.SeedDocsFromFS(docs.FS)
			tree, err = db.GetDocsTree()
		}
		if err != nil {
			return fmt.Errorf("failed retrieving docs tree: %w", err)
		}

		fmt.Println("Knowledge Base Documentation Tree")
		fmt.Println("=================================")
		fmt.Print(formatAsciiDocTree(tree, ""))
		return nil
	},
}

var (
	docTargetFolder string
	docOverwrite    bool
	docTitleFlag    string
	docFileFlag     string
)

var docsImportCmd = &cobra.Command{
	Use:     "import <directory>",
	Aliases: []string{"load"},
	Short:   "Recursively import a folder of Markdown files into the database tree",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceDir := args[0]
		info, err := os.Stat(sourceDir)
		if err != nil {
			return fmt.Errorf("invalid directory path: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%q is not a directory", sourceDir)
		}

		var items []db.BatchDocImportItem
		err = filepath.WalkDir(sourceDir, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(sourceDir, p)
			if err != nil || rel == "." {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if docTargetFolder != "" {
				rel = path.Join(docTargetFolder, rel)
			}

			if d.IsDir() {
				items = append(items, db.BatchDocImportItem{
					Path: rel,
					Type: "folder",
				})
				return nil
			}

			if strings.HasSuffix(strings.ToLower(d.Name()), ".md") || strings.HasSuffix(strings.ToLower(d.Name()), ".markdown") {
				contentBytes, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				content := string(contentBytes)
				title := db.ExtractTitleFromMarkdown(rel, content)
				items = append(items, db.BatchDocImportItem{
					Path:    rel,
					Type:    "doc",
					Title:   title,
					Content: content,
				})
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed walking directory %q: %w", sourceDir, err)
		}

		if len(items) == 0 {
			fmt.Println("No Markdown (.md) files found in directory.")
			return nil
		}

		fmt.Printf("Importing %d items into database...\n", len(items))
		created, updated, err := db.BatchImportDocNodes(items, docOverwrite)
		if err != nil {
			return fmt.Errorf("import failed: %w", err)
		}

		fmt.Printf("Done! Created %d new items, updated %d existing items.\n", created, updated)
		return nil
	},
}

var docsExportCmd = &cobra.Command{
	Use:     "export <destination_dir>",
	Aliases: []string{"dump"},
	Short:   "Export all documentation from the database to a directory of Markdown files",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := args[0]
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed creating target directory: %w", err)
		}

		options, err := getFlattenedDocOptions()
		if err != nil {
			return err
		}

		exported := 0
		for _, opt := range options {
			node, err := db.GetDocNodeByPath(opt.Path)
			if err != nil {
				continue
			}
			fullPath := filepath.Join(targetDir, filepath.FromSlash(node.Path))
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(fullPath, []byte(node.Content), 0644); err != nil {
				return fmt.Errorf("failed writing %s: %w", fullPath, err)
			}
			exported++
		}

		fmt.Printf("Successfully exported %d documents to %s\n", exported, targetDir)
		return nil
	},
}

var docsAddCmd = &cobra.Command{
	Use:   "add <path>",
	Short: "Create a new document in the database tree",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		docPath := db.NormalizeDocPath(args[0])
		if docTargetFolder != "" {
			docPath = path.Join(db.NormalizeDocPath(docTargetFolder), docPath)
		}

		var content string
		if docFileFlag != "" {
			bytes, err := os.ReadFile(docFileFlag)
			if err != nil {
				return fmt.Errorf("failed reading file %s: %w", docFileFlag, err)
			}
			content = string(bytes)
		} else {
			initial := fmt.Sprintf("# %s\n\n", path.Base(docPath))
			edited, err := captureEditorContent(initial)
			if err != nil {
				return fmt.Errorf("editor error: %w", err)
			}
			content = edited
		}

		title := docTitleFlag
		if title == "" {
			title = db.ExtractTitleFromMarkdown(docPath, content)
		}

		node := &db.DocNode{
			Type:    "doc",
			Path:    docPath,
			Title:   title,
			Content: content,
			Tags:    docTagsFlag,
		}
		if err := db.CreateDocNode(node); err != nil {
			return fmt.Errorf("failed creating doc: %w", err)
		}

		fmt.Printf("Created doc %q (%s)\n", node.Title, node.Path)
		return nil
	},
}

var docsEditCmd = &cobra.Command{
	Use:   "edit <path_or_id>",
	Short: "Edit a document in your configured $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		node, err := db.GetDocNodeByPath(target)
		if err != nil {
			node, err = db.GetDocNodeByID(target)
		}
		if err != nil {
			return fmt.Errorf("doc %q not found: %w", target, err)
		}

		updatedContent, err := captureEditorContent(node.Content)
		if err != nil {
			return fmt.Errorf("editor error: %w", err)
		}

		node.Content = updatedContent
		node.Title = db.ExtractTitleFromMarkdown(node.Path, updatedContent)
		if err := db.UpdateDocNode(node); err != nil {
			return fmt.Errorf("failed saving doc update: %w", err)
		}

		fmt.Printf("Updated doc %q (%s)\n", node.Title, node.Path)
		return nil
	},
}

var docsDeleteCmd = &cobra.Command{
	Use:     "delete <path_or_id>",
	Aliases: []string{"rm"},
	Short:   "Soft-delete a document or folder from the tree",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		node, err := db.GetDocNodeByPath(target)
		if err != nil {
			node, err = db.GetDocNodeByID(target)
		}
		if err != nil {
			return fmt.Errorf("item %q not found: %w", target, err)
		}

		if err := db.DeleteDocNode(node.ID, "deleted via CLI"); err != nil {
			return fmt.Errorf("failed deleting item: %w", err)
		}

		fmt.Printf("Deleted %s %q (%s)\n", node.Type, node.Title, node.Path)
		return nil
	},
}

var docsSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search documentation titles and contents using full-text search",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		results, err := db.SearchDocs(query)
		if err != nil {
			return fmt.Errorf("search error: %w", err)
		}

		if len(results) == 0 {
			fmt.Printf("No documentation matching %q\n", query)
			return nil
		}

		fmt.Printf("Found %d matching documentation files:\n\n", len(results))
		for _, r := range results {
			fmt.Printf("• %s\n  Path: %s\n", r.Title, r.Path)
		}
		return nil
	},
}

var docsFolderCmd = &cobra.Command{
	Use:     "folder",
	Aliases: []string{"dir"},
	Short:   "Manage documentation folders in the tree",
}

var docsFolderCreateCmd = &cobra.Command{
	Use:   "create <path>",
	Short: "Create a documentation folder (or nested path)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		folderPath := db.NormalizeDocPath(args[0])
		title := docTitleFlag
		if title == "" {
			title = path.Base(folderPath)
		}

		node := &db.DocNode{
			Type:  "folder",
			Path:  folderPath,
			Title: title,
		}
		if err := db.CreateDocNode(node); err != nil {
			return fmt.Errorf("failed creating folder: %w", err)
		}

		fmt.Printf("Created folder %q (%s)\n", node.Title, node.Path)
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
Use the --tree flag to view the directory tree at a glance.
Use the --web flag to open the documentation in your web browser.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Command hierarchy tree requested via flag
		if docsShowCommands {
			fmt.Print(commandHierarchyTree)
			return nil
		}

		// 2. Tree mode requested
		if docsShowTree {
			return docsTreeCmd.RunE(cmd, args)
		}

		options, err := getFlattenedDocOptions()
		if err != nil {
			return err
		}

		// 3. Direct topic argument provided
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

		// 4. Web mode requested without topic
		if docsOpenWeb {
			port := 8080
			if viper.IsSet("server.port") {
				port = viper.GetInt("server.port")
			}
			url := fmt.Sprintf("http://127.0.0.1:%d/docs", port)
			fmt.Printf("Opening documentation at %s...\n", url)
			return server.OpenInBrowser(url)
		}

		// 5. Interactive fuzzy finder
		idx, err := fuzzyfinder.Find(
			options,
			func(i int) string {
				if options[i].Category != "" {
					return fmt.Sprintf("[%s] %s (%s)", options[i].Category, options[i].Title, options[i].Path)
				}
				return fmt.Sprintf("%s (%s)", options[i].Title, options[i].Path)
			},
			fuzzyfinder.WithPreviewWindow(func(i int, width, height int) string {
				if i < 0 || i >= len(options) {
					return ""
				}
				node, err := db.GetDocNodeByPath(options[i].Path)
				if err == nil {
					return node.Content
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
	docsCmd.Flags().BoolVarP(&docsShowTree, "tree", "t", false, "Display documentation directory tree")
	docsCmd.Flags().BoolVar(&docsShowCommands, "commands", false, "Display CLI command hierarchy and option tree")

	docsTreeCmd.Flags().BoolVar(&docsShowCommands, "commands", false, "Display CLI command hierarchy and option tree")

	docsImportCmd.Flags().StringVar(&docTargetFolder, "target", "", "Target folder in the documentation tree")
	docsImportCmd.Flags().BoolVarP(&docOverwrite, "overwrite", "o", false, "Overwrite existing docs if path matches")

	docsAddCmd.Flags().StringVarP(&docTitleFlag, "title", "t", "", "Document title")
	docsAddCmd.Flags().StringVarP(&docFileFlag, "file", "f", "", "Source file path to import content from")
	docsAddCmd.Flags().StringVar(&docTargetFolder, "folder", "", "Target folder path")
	docsAddCmd.Flags().StringSliceVar(&docTagsFlag, "tags", []string{}, "Comma-separated tags to attach to the document")

	docsFolderCreateCmd.Flags().StringVarP(&docTitleFlag, "title", "t", "", "Folder display title")

	docsFolderCmd.AddCommand(docsFolderCreateCmd)

	docsCmd.AddCommand(docsTreeCmd)
	docsCmd.AddCommand(docsImportCmd)
	docsCmd.AddCommand(docsExportCmd)
	docsCmd.AddCommand(docsAddCmd)
	docsCmd.AddCommand(docsEditCmd)
	docsCmd.AddCommand(docsDeleteCmd)
	docsCmd.AddCommand(docsSearchCmd)
	docsCmd.AddCommand(docsFolderCmd)

	rootCmd.AddCommand(docsCmd)
}
