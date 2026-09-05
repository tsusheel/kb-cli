# Knowledge Base CLI (`kb-cli`)

A fast, SQLite-backed personal knowledge base and second brain CLI with built-in **Model Context Protocol (MCP)** server connectivity for AI agents.

---

## Key Features

- **Personal Knowledge Management**: Manage notes, todos, projects, ideas, and decisions from your terminal.
- **Full-Text Search (FTS5)**: Instant search across note titles and content bodies with relevance ranking.
- **Flexible Due / Target Dates**: Supports relative keywords (`today`, `tomorrow`, `monday`, `+3d`) and formatted dates.
- **Tags & Semantic Links**: Interlink notes (`related_to`, `depends_on`, `supports`, etc.) and organize with hierarchical tags.
- **Soft Deletes & Audit Trail**: Notes and links are soft-deleted with timestamps and deletion attribution (e.g. `deleted by AI`).
- **AI Agent Connectivity (MCP Server)**: Run `kb serve` / `kb mcp` to connect `kb-cli` as an MCP server with Claude Desktop, Antigravity, Cursor, Cline, or any MCP-compliant AI assistant.

---

## CLI Command Reference

### 1. Add Note (`kb add`)
Create a new note or todo. Opens your configured editor (`$EDITOR` or default `notepad`/`vi`) to write note content.
```bash
# Add a note with title and due date
kb add -n "Fix database migrations" --type todo --target "tomorrow" --area work

# Add an idea with tags
kb add -n "Agentic Knowledge Base" --type idea --tags "ai,golang"
```

Flags:
- `-n`, `--note string`: Title or summary of the note.
- `--target string`: Due or target date (e.g., `today`, `tomorrow`, `monday`, `+3d`, `2026-09-10`).
- `--type string`: Note type (`note`, `todo`, `project`, `idea`, `concept`, `decision`, etc. - default `note`).
- `--status string`: Status (`active`, `raw`, `refined`, `in-progress`, `completed`, `archived` - default `active`).
- `--area string`: Area (`work`, `finance`, `personal`).
- `--tags strings`: Comma-separated list of tags.

---

### 2. List Notes (`kb list` / `kb ls`)
List notes with optional filters.
```bash
# List all active notes
kb list

# Filter by type
kb list --notes      # Only notes
kb list --projects   # Only projects
kb list --todos      # Only todos
kb list --type idea  # Filter by any custom note type
```

---

### 3. Open Note (`kb open` / `kb view`)
View full note contents, metadata, tags, and links. Uses interactive fuzzy-finder if no ID is passed.
```bash
# Interactive fuzzy finder
kb open

# Open specific note by full UUID or short ID prefix
kb open a1b2c3d
```

---

### 4. Search Notes (`kb search` / `kb s` / `kb find`)
Fast full-text search across titles and note bodies using SQLite FTS5.
```bash
kb search "sqlite migrations"
```

---

### 5. Link Notes (`kb link`)
Create a directional link between two notes.
```bash
kb link <from_id> <to_id> --type depends_on
```
Supported link types: `related_to`, `part_of`, `inspired_by`, `depends_on`, `supports`, `contradicts`, `about`, `created_by`.

---

### 6. Start MCP Server (`kb serve` / `kb mcp`)
Starts the Model Context Protocol (MCP) server over standard I/O (`stdio`).
```bash
kb serve
# or
kb mcp
```

---

## AI Agent & MCP Configuration

To allow an AI assistant to read, search, create, filter, and update your notes, configure `kb-cli` in your client's MCP configuration.

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

---

## Data Storage & Configuration

- **Config File**: `~/.config/kb/config.yaml`
- **Database File**: `~/.config/kb/kb.db` (or custom `base_path` specified in `config.yaml`)
- **Default Config**:
  ```yaml
  app_name: kb-app
  base_path: ~/.config/kb
  date_format: 2006-01-02
  ```
