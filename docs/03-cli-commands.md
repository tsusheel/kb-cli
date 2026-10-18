# 03. Comprehensive CLI Command Reference

## 1. Overview & CLI Principles

`kb-cli` features a rich command-line interface built using **Cobra**. The CLI is architected around two core speed principles:
1. **Speed 1 (Instant Fleeting Capture):** Entering a thought without command flags or editor interruptions (`kb "thought"` completes in < 25ms).
2. **Speed 2 (Substantive Deep Refinement):** Opening `$EDITOR` or launching interactive terminal user interfaces (TUIs) to write detailed markdown, triage raw notes, or audit revision timelines.

Below is the complete reference for every command, subcommand, flag, and workflow supported by `kb-cli`.

---

## 2. Fast Thought Capture (`kb "<thought>"` / `kb add`)

### 2.1 Positional Fast Capture (Zero-Command Mode)
Any arguments passed to `kb` that do not match a registered subcommand are treated as a fleeting thought and saved as a `raw` note:
```bash
kb "Investigate SQLite WAL mode checkpointer starvation"
kb "Call doctor regarding annual checkup tomorrow"
```
- **Exit Code:** `0` on success.
- **Latency:** `< 25ms` total execution time.
- **Default Attributes:** `type='note'`, `status='raw'`, `created_at=NOW()`.

### 2.2 Structured Add (`kb add`)
Adds a note with explicit metadata flags or opens the long-form editor:
```bash
# Add a structured todo with due date and area
kb add "Implement Raft log compaction" --type todo --due "tomorrow" --area work

# Add an architecture decision record
kb add "Adopt PostgreSQL for multi-device sync" --type decision --area tech

# Open $EDITOR to compose long-form markdown body
kb add "System Architecture Document" -e
kb add "Meeting Notes with Core Team" --editor
```

#### Supported Flags for `kb add`:
- `-t`, `--type <string>`: Note intent type (`note`, `todo`, `project`, `idea`, `concept`, `decision`, `til`, `question`, `resource`, `experiment`, `person`). Default: `'note'`.
- `-s`, `--status <string>`: Lifecycle status (`raw`, `refined`, `in-progress`, `completed`, `archived`). Default: `'active'`.
- `-a`, `--area <string>`: Functional life or work domain (`work`, `finance`, `health`, `personal`).
- `-d`, `--due <string>`: Milestone or deadline date (supports `'today'`, `'tomorrow'`, `'monday'`, `'+3d'`, `'next week'`, or `'YYYY-MM-DD'`).
- `-e`, `--editor`: Opens your configured `$EDITOR` (or `nano`/`vim`) to compose the substantive body (`note_flesh`).
- `--importance <int>`: Priority score from 1 to 5.
- `--clarity <int>`: Clarity score from 1 to 5.
- `--source <string>`: Attribution source (URL, book title, meeting name).

---

## 3. Editing & Refinement (`kb edit`, `kb rename`, `kb flesh`)

### 3.1 Interactive TUI Edit Menu (`kb edit`)
When invoked without flags, `kb edit` launches an interactive terminal fuzzy-finder. Once you select a note, an interactive menu allows you to modify:
1. Title / headline summary
2. Detailed body (`note_flesh`) via `$EDITOR`
3. Note type
4. Note status
5. Functional area
6. Due date
7. Associated tags
8. Outgoing graph links

```bash
# Launch interactive fuzzy selector and edit menu
kb edit

# Launch edit menu for a specific note ID
kb edit a1b2c3d
```

### 3.2 Non-Interactive Quick Rename (`kb rename` / `kb edit`)
Update a note headline summary directly from the command line without opening an editor:
```bash
kb rename a1b2c3d "Updated Architecture Decision: Use WAL Mode"
kb edit a1b2c3d "Updated Architecture Decision: Use WAL Mode"
```

### 3.3 Programmatic Metadata Updates
Update specific metadata flags non-interactively:
```bash
kb edit a1b2c3d --status completed --type project
kb edit a1b2c3d --due "next friday" --area work
kb edit a1b2c3d --clear-due
```

### 3.4 Direct Flesh Body Editing (`kb flesh` / `kb edit -e`)
Bypasses menus and immediately opens `$EDITOR` loaded with the note's existing `note_flesh`:
```bash
kb flesh a1b2c3d
kb edit a1b2c3d -e
```

---

## 4. Deletion, Recovery & Restoration (`kb delete`, `kb rm`, `kb restore`)

### 4.1 Non-Destructive Soft Deletes (`kb delete` / `kb rm`)
Deletions in `kb-cli` are non-destructive soft-deletes with audit attribution:
```bash
# Soft-delete by ID
kb delete a1b2c3d
kb rm a1b2c3d

# Soft-delete with custom attribution reason
kb delete a1b2c3d --reason "Task completed; archived by sprint review"

# Interactive fuzzy-finder multi-select deletion (Tab to select multiple, Enter to delete)
kb delete
kb rm
```

### 4.2 Restoring Soft-Deleted Notes (`kb restore`)
Restores a previously soft-deleted note or daily log by clearing its `deleted_at` tombstone:
```bash
kb restore a1b2c3d
```

---

## 5. Daily Stream & Micro-Logs (`kb log`, `kb promote`)

The daily stream captures transient micro-thoughts throughout your day without cluttering the primary knowledge graph.

### 5.1 Appending Micro-Logs
```bash
kb log "Finished reviewing PR for PostgreSQL sync engine"
kb log "Ran benchmark: SQLite FTS5 query latency is 0.4ms"
```

### 5.2 Viewing the Daily Stream
```bash
# View today's chronological stream (unpromoted logs only)
kb log

# View all logs for today (including logs already promoted to notes)
kb log -a
kb log --all

# View logs for a specific historical date
kb log -d 2026-10-15
kb log -d yesterday -a

# View logs across an inclusive date range
kb log --from 2026-10-01 --to 2026-10-18
kb log -f -7d -a

# View entire log archive across all time
kb log -A
kb log --all-time
```

### 5.3 Promoting a Log to a Permanent Note (`kb promote`)
When a micro-log develops substance, promote it into a permanent active `models.Note` in `< 50ms`:
```bash
kb promote 08db3db --type project --status in-progress
```
The original daily log remains in the timeline but records `note_id`, linking the chronological micro-thought to the permanent note.

---

## 6. Inbox & Interactive Triage (`kb inbox`, `kb triage`)

Thoughts captured chaotically in `raw` status collect in the **Inbox**.

### 6.1 Inspecting the Inbox (`kb inbox`)
```bash
# View today's raw unrefined notes and unpromoted logs
kb inbox

# View all raw notes across all time
kb inbox -a
```

### 6.2 Interactive Triage Wizard (`kb triage`)
Launches a dedicated terminal wizard that steps through each inbox note sequentially:
```bash
kb triage
kb triage -a
```
For each item, you can:
- `[r]` Rename headline summary
- `[f]` Flesh out detailed body in `$EDITOR`
- `[t]` Change type (`todo`, `note`, `project`, `idea`)
- `[s]` Change status (`active`, `refined`, `archived`)
- `[a]` Set functional area (`work`, `personal`)
- `[g]` Add tags
- `[l]` Link to an existing note
- `[d]` Soft-delete item
- `[n]` Skip to next item

---

## 7. Listing, Searching & Viewing (`kb ls`, `kb search`, `kb open`)

### 7.1 Formatted Tabular Listing (`kb ls` / `kb list`)
Renders clean ASCII/Unicode rounded tables in your terminal:
```bash
# List all active notes
kb ls
kb list

# Filter by type, status, or area
kb ls -t todo
kb ls -s in-progress
kb ls -a work

# Convenient preset flags
kb list --todos
kb list --notes
kb list --projects

# Search keywords directly in list view
kb ls postgres
kb ls -t todo "migration"
```

### 7.2 Full-Text Search (`kb search` / `kb find`)
Executes instant keyword searches across both headlines and long-form flesh bodies using SQLite FTS5:
```bash
kb search "wal mode checkpointer"
kb find "postgres sync"
```

### 7.3 Viewing Full Note Content (`kb open` / `kb view`)
Renders full note details, tags, semantic links, and formatted markdown bodies:
```bash
# View note by ID
kb open a1b2c3d
kb view a1b2c3d

# View daily log by ID
kb open 6a178a8

# Launch interactive fuzzy finder to pick a note or log to view
kb open
```

---

## 8. Semantic Graph Links (`kb link`, `kb graph`)

### 8.1 Creating Directional Links (`kb link`)
Creates a directional semantic connection from one note to another:
```bash
kb link <from_id> <to_id> --type depends_on
kb link <from_id> <to_id> --type inspired_by
```
Supported types: `related_to`, `part_of`, `inspired_by`, `depends_on`, `supports`, `contradicts`, `about`, `created_by`.

### 8.2 Inspecting the Graph (`kb graph`)
Visualizes knowledge connections or exports relationships in Graphviz DOT format:
```bash
kb graph
kb graph --dot > graph.dot
```

---

## 9. Version History & Auditing (`kb history` / `kb audit`)

`kb-cli` maintains an immutable snapshot log in `audit_logs`:
```bash
# View recent global revision stream
kb history
kb audit

# View complete revision timeline for a specific note
kb history a1b2c3d

# Inspect unified diff between two revisions
kb history diff a1b2c3d <revision_id>

# Revert a note to a previous point-in-time snapshot
kb history revert a1b2c3d <revision_id>
```

---

## 10. Remote Synchronization (`kb sync`)

Coordinates two-way synchronization with PostgreSQL:
```bash
# Full bidirectional sync (pull remote + push local)
kb sync

# Push local modifications only
kb sync push

# Pull remote modifications only
kb sync pull

# Check pending unpushed change counts and connection state
kb sync status

# Test connection, authentication, and ping latency
kb sync test
```

---

## 11. Configuration & Security (`kb config`)

Manages configuration keys and hardware-encrypted OS Keyring credentials:
```bash
# Interactive setup wizard
kb config setup

# Set a public configuration value
kb config set date_format "2006-01-02"
kb config set server.port 3000

# Store database password securely in OS Keyring
kb config set-secret postgres_password

# List all configuration settings
kb config list
kb config get remote.host

# Delete settings or OS Keyring secrets
kb config delete custom_key
kb config delete --secret postgres_password
```

---

## 12. Web & MCP Services (`kb serve`)

Starts background or foreground services:
```bash
# Start Model Context Protocol (MCP) server over stdio for AI assistants
kb serve
kb serve mcp

# Start Web Fuzzy Finder HTTP server
kb serve http
kb serve web --port 8080 --open
kb serve http --tls-cert cert.pem --tls-key key.pem
```

---

## 13. Data Export & Import (`kb export`, `kb import`)

```bash
# Export knowledge base to JSON or Markdown directory
kb export --format json -o backup.json
kb export --format markdown -o ./notes_markdown/

# Import from backup file
kb import backup.json
```
