# Knowledge Base CLI (`kb-cli`)

A fast, local-first personal knowledge base and second brain CLI with embedded SQLite storage, remote **PostgreSQL** synchronization, and built-in **Model Context Protocol (MCP)** server connectivity for AI agents.

---

## Key Features

- **Two-Speed Thought Capture**:
  - **Instant Jot**: Fast `< 50ms` capture directly from the terminal without opening an editor (`kb "thought"`, `kb jot "idea"`).
  - **Deep Fleshing**: Dedicated editor workflow (`$EDITOR`) to write and edit detailed markdown bodies (`kb add`, `kb edit <id>`, `kb flesh <id>`).
- **Title Editing & Renaming**: Update note titles directly without opening an editor (`kb rename <id> "new title"`, `kb edit <id> "new title"`).
- **Temporal Stream & Daily Logs**: Micro-logging throughout the day (`kb log "entry"`, `kb today`) with one-command promotion to permanent notes (`kb promote <id>`).
- **Inbox & Interactive Triage**: Accumulate raw thoughts chaotically, then rapidly triage, refine, or tag them (`kb inbox`, `kb triage`).
- **Non-Destructive Soft Deletes**: Soft-delete notes and daily logs with audit trails and fuzzy-finder fallback (`kb delete [id]`, `kb rm [id]`).
- **Local-First PostgreSQL Sync**: Two-way encrypted synchronization between local SQLite and remote PostgreSQL (`kb sync`, `kb sync push`, `kb sync pull`, `kb sync status`).
- **Hardware/OS Keyring Secrets**: Sensitive passwords are never stored in plaintext `config.yaml`; encrypted in OS Credential Manager (`kb config set-secret`).
- **Full-Text Search (FTS5)**: Instant search across note titles and content bodies with relevance ranking (`kb search "query"`, `kb find "query"`).
- **Clean Text & Relational Graph**: Prose stays pure and readable; tags and links live in the database metadata layer without markup pollution.
- **AI Agent Connectivity (MCP Server)**: Run `kb serve` / `kb mcp` to connect `kb-cli` as an MCP server with Claude Desktop, Antigravity, Cursor, Cline, or any MCP-compliant AI assistant.

---

## CLI Command Reference

### 1. Instant Jot (`kb <thought>` / `kb jot`)
Capture fleeting thoughts instantly without opening an editor:
```bash
# Direct positional jot
kb "Explore SQLite WAL mode performance"

# With type, status, area, and due date
kb jot "Implement Raft log compaction" --type todo --due "tomorrow" --area work
kb jot "Graph neural networks for note recommendation" --type idea
```

---

### 2. Add Note (`kb add`)
Create a new note and open your configured editor (`$EDITOR` or default `notepad`/`vi`) to write note flesh:
```bash
kb add -n "Fix database migrations" --type todo --due "tomorrow" --area work
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

### 5. Daily Stream & Micro-Logs (`kb log` / `kb today`)
Record timestamped micro-logs throughout your workday and view today's chronological stream:
```bash
# Append micro-logs
kb log "Finished reviewing PR for MCP protocol"
kb log "Benchmarked query latency: 1.2ms average"

# View today's timeline (unpromoted only)
kb log

# View all logs including promoted ones
kb log -a
kb today
```

---

### 6. Promote Log to Note (`kb promote`)
Promote a daily log entry non-interactively into a permanent, typed `Note` in `< 50ms`:
```bash
kb promote 08db3db --type project --status active
```

---

### 7. Inbox & Triage (`kb inbox` / `kb triage`)
View all unrefined raw thoughts and unpromoted logs, or launch the interactive terminal triage wizard:
```bash
# View pending raw items
kb inbox

# Launch interactive triage wizard
kb triage
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
```

---

### 14. Start MCP Server (`kb serve` / `kb mcp`)
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
