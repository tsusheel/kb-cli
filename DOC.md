# Knowledge Base CLI (`kb-cli`)

A fast, local-first personal knowledge base and second brain CLI with embedded SQLite storage, remote **PostgreSQL** synchronization, and built-in **Model Context Protocol (MCP)** server connectivity for AI agents.

---

## Key Features

- **Two-Speed Thought Capture**:
  - **Instant Capture**: Fast `< 50ms` capture directly from the terminal without opening an editor (`kb "thought"`, `kb add "idea"`).
  - **Deep Fleshing**: Dedicated editor workflow (`$EDITOR`) to write and edit detailed markdown bodies (`kb add`, `kb add -e`, `kb edit <id>`).
- **Title Editing & Renaming**: Update note titles directly without opening an editor (`kb rename <id> "new title"`, `kb edit <id> "new title"`).
- **Temporal Stream & Daily Logs**: Micro-logging throughout the day (`kb log "entry"`, `kb log -a`) with one-command promotion to permanent notes (`kb promote <id>`).
- **Inbox & Interactive Triage**: Accumulate raw thoughts chaotically, then rapidly triage, refine, or tag them (`kb inbox`, `kb triage`).
- **Non-Destructive Soft Deletes**: Soft-delete notes and daily logs with audit trails and fuzzy-finder fallback (`kb delete [id]`, `kb rm [id]`).
- **Local-First PostgreSQL Sync**: Two-way encrypted synchronization between local SQLite and remote PostgreSQL (`kb sync`, `kb sync push`, `kb sync pull`, `kb sync status`).
- **Hardware/OS Keyring Secrets**: Sensitive passwords are never stored in plaintext `config.yaml`; encrypted in OS Credential Manager (`kb config set-secret`).
- **Full-Text Search & Browsing**: Instant search and browsing with `kb ls [query]` across note titles and content bodies (`kb ls postgres`, `kb ls -t todo`).
- **Clean Text & Relational Graph**: Prose stays pure and readable; tags and links live in the database metadata layer without markup pollution.
- **AI Agent Connectivity (MCP Server)**: Run `kb serve` / `kb mcp` to connect `kb-cli` as an MCP server with Claude Desktop, Antigravity, Cursor, Cline, or any MCP-compliant AI assistant.

---

## CLI Command Reference

### 1. Add & Quick Capture (`kb [thought]` / `kb add`)
Capture thoughts instantly or open your editor for detailed notes:
```bash
# Direct positional capture (no command needed)
kb "Explore SQLite WAL mode performance"

# With type, status, area, and due date
kb add "Implement Raft log compaction" --type todo --due "tomorrow" --area work
kb add "Graph neural networks for note recommendation" --type idea

# Open $EDITOR to write long-form note body:
kb add "System Architecture" -e
kb add
```

---

### 2. List & Search Notes (`kb ls [query]`)
List notes or perform full-text search directly:
```bash
# List all notes in a clean rounded table
kb ls

# Filter by type, status, or area
kb ls -t todo
kb ls -s raw
kb ls -a work

# Search keywords across note titles and bodies
kb ls postgres
kb ls "query optimization"
kb ls -t todo postgres
```

---

### 3. Edit & Rename Note (`kb edit` / `kb rename` / `kb flesh`)
Update titles, metadata flags, or open `$EDITOR` to write/edit the detailed body (`note_flesh`):
```bash
# Rename / update title directly:
kb rename a1b2c3d "Updated Note Title"
kb edit a1b2c3d "Updated Note Title"

# Update title and metadata flags:
kb edit a1b2c3d -n "New Title" --status completed --type project

# Open $EDITOR to edit detailed body (note flesh):
kb edit a1b2c3d
kb flesh a1b2c3d
```

---

### 4. Delete Notes & Logs (`kb delete` / `kb rm`)
Soft-delete notes or logs non-destructively with audit trail:
```bash
# Delete a note by ID:
kb delete a1b2c3d
kb rm a1b2c3d

# Delete with custom attribution reason:
kb delete a1b2c3d --reason "superseded by new spec"

# Fuzzy-find and select a note to delete (if ID omitted):
kb delete

# Delete a daily log:
kb delete 6a178a8
kb rm 6a178a8
```

---

### 5. Daily Stream & Micro-Logs (`kb log`)
Record timestamped micro-logs throughout your workday and view today's chronological stream:
```bash
# Append micro-logs
kb log "Finished reviewing PR for MCP protocol"
kb log "Benchmarked query latency: 1.2ms average"

# View today's timeline (unpromoted only)
kb log

# View all logs for today (including promoted ones)
kb log -a

# View logs for a specific date
kb log -d 2026-09-12
kb log -d yesterday -a

# View logs for an inclusive date range
kb log --from 2026-09-01 --to 2026-09-13
kb log -f -7d -a

# View all historical logs across all time
kb log -A
kb log --all-time -a
```

---

### 6. Promote Log to Note (`kb promote`)
Promote a daily log entry non-interactively into a permanent active `Note` in `< 50ms`:
```bash
kb promote 08db3db --type project
```

---

### 7. Inbox & Triage (`kb inbox` / `kb triage`)
View today's raw unrefined thoughts, or launch the interactive terminal triage wizard:
```bash
# View today's raw notes awaiting triage
kb inbox

# View all raw notes across all dates
kb inbox -a

# Interactively triage today's raw notes
kb triage

# Interactively triage all raw notes across all dates
kb triage -a
```

---

### 8. List Notes (`kb list` / `kb ls`)
List notes with optional filters for type, status, or area:
```bash
kb list
kb list --notes              # Only notes
kb list --projects           # Only projects
kb list --todos              # Only todos
kb list --type idea          # Filter by note type
kb list --status completed   # Filter by status
kb list --area work          # Filter by area
```

---

### 9. Open Note (`kb open` / `kb view`)
View full note contents, metadata, tags, and links. Uses interactive fuzzy-finder if no ID is passed:
```bash
kb open
kb open a1b2c3d
```

---

### 10. Search Notes (`kb search` / `kb find`)
Fast full-text search across titles and note bodies using SQLite FTS5:
```bash
kb search "sqlite migrations"
kb find "postgres sync"
```

---

### 11. Link Notes (`kb link`)
Create a directional semantic link between two notes:
```bash
kb link <from_id> <to_id> --type depends_on
```
Supported link types: `related_to`, `part_of`, `inspired_by`, `depends_on`, `supports`, `contradicts`, `about`, `created_by`.

---

### 12. Remote PostgreSQL Synchronization (`kb sync`)
Two-way synchronization between local SQLite and remote PostgreSQL:
```bash
# Two-way sync (push local + pull remote)
kb sync

# Push local modifications only
kb sync push

# Pull remote modifications only
kb sync pull

# View sync status and pending change counts
kb sync status

# Test connection to remote PostgreSQL
kb sync test
```

---

### 13. Configuration & Secrets (`kb config`)
Manage configuration settings and hardware-encrypted OS Keyring secrets:
```bash
# Interactive setup wizard for remote sync
kb config setup

# Set a public configuration value
kb config set date_format "2006-01-02"

# Securely store a database password in OS Keyring (masked input)
kb config set-secret postgres_password

# Inspect current configuration
kb config list
kb config get remote.postgres_url

# Delete configuration options or OS Keyring secrets
kb config delete custom_key
kb config rm remote.enabled
kb config delete --secret postgres_password
```

---

### 14. Audit & Version History (`kb history` / `kb audit`)
Inspect modification timelines, view point-in-time diffs, and revert notes to earlier revisions:
```bash
# View recent global audit stream across all notes and logs
kb history
kb audit

# View complete version timeline for a specific note or log
kb history a1b2c3d

# Inspect snapshot diff for a specific revision
kb history diff a1b2c3d 9f8e7d6

# Revert note to a previous revision
kb history revert a1b2c3d 9f8e7d6
```

---

### 15. Start MCP Server (`kb serve` / `kb mcp`)
Starts the Model Context Protocol (MCP) server over standard I/O (`stdio`):
```bash
kb serve
# or
kb mcp
```


---

## AI Agent & MCP Configuration

Configure `kb-cli` in your client's MCP configuration.

### Antigravity / Gemini IDE
Add to `mcp_config.json` (or `.agents/mcp_config.json`):
```json
{
  "mcpServers": {
    "kb": {
      "command": "kb",
      "args": ["serve"]
    }
  }
}
```

### Claude Desktop
Add to `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "kb": {
      "command": "kb",
      "args": ["serve"]
    }
  }
}
```

### Cursor / Cline
Configure command as `kb` with arguments `["serve"]`.

---

## MCP Tool Reference

When running as an MCP server, `kb-cli` provides the following tools:

| Tool | Description | Parameters |
| :--- | :--- | :--- |
| `list_notes` | List notes with optional filters | `type`, `status`, `area`, `include_deleted` |
| `get_note` | Retrieve full note, flesh body, tags, and links | `id` (required) |
| `search_notes` | Full-text search across notes | `query` (required) |
| `create_note` | Create a new note/todo/project | `note` (required), `content` (required), `type`, `status`, `area`, `due`, `importance`, `clarity`, `source` |
| `update_note` | Update fields of an existing note | `id` (required), `note`, `content`, `type`, `status`, `area`, `due`, `importance`, `clarity`, `source` |
| `delete_note` | Soft-delete a note with audit reason | `id` (required), `reason` (optional, default: `deleted by AI`) |
| `add_tag` | Attach a tag to a note | `note_id` (required), `tag` (required) |
| `link_notes` | Create relationship between two notes | `from_id` (required), `to_id` (required), `type` |
| `create_log` | Append micro-log to today's daily stream | `content` (required) |
| `list_daily_logs` | List daily stream micro-logs for a date | `date`, `include_promoted` |
| `promote_log` | Non-interactively promote a log into a Note | `log_id` (required), `type`, `status` |
| `get_inbox` | Get raw notes and unpromoted logs for triage | |

---

## Data Storage & Configuration

- **Config File**: `~/.config/kb/config.yaml`
- **Database File**: `~/.config/kb/kb.db` (or custom `base_path` specified in `config.yaml`)
- **Keyring Service**: `kb-cli` in Windows Credential Manager / macOS Keychain / Linux Secret Service
