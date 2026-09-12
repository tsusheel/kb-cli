# Knowledge Base CLI (`kb`)

A lightning-fast, local-first personal knowledge base and second brain CLI with embedded SQLite storage and built-in **Model Context Protocol (MCP)** server connectivity for AI agents.

Designed around a **two-speed capture philosophy**: capture fleeting thoughts in milliseconds directly from your terminal, flesh them out into deep markdown notes in your favorite editor, and seamlessly collaborate with AI assistants over MCP.

---

## Features

- **⚡ Two-Speed Thought Capture**:
  - **Instant Jot**: Fast `< 50ms` capture directly from the terminal without opening an editor (`kb "thought"`, `kb jot "idea"`).
  - **Deep Fleshing**: Dedicated editor workflow (`$EDITOR`) to craft rich markdown bodies (`kb add`, `kb edit <id>`).
- **📅 Daily Temporal Stream**: Micro-log throughout the day (`kb log "..."`, `kb today`) and promote impactful logs into permanent notes (`kb promote <id>`).
- **📥 Inbox & Triage Wizard**: Capture chaotically in a `raw` state, then rapidly triage, refine, tag, or link notes (`kb inbox`, `kb triage`).
- **🔍 Full-Text Search (FTS5)**: Instant search across note headlines and detailed markdown content bodies (`kb search "query"`).
- **🕸️ Relational Graph & Pure Markdown**: Prose stays pure and uncluttered (no `#tag` or `[[bracket]]` markup pollution); tags and semantic links live in the relational database layer.
- **🛡️ Audit Trail & Soft Deletion**: Non-destructive soft deletes with timestamps and attribution (e.g., `deleted by AI`).
- **🤖 Built-in MCP Server**: Native stdio MCP server (`kb serve`) allowing AI assistants (Claude Desktop, Antigravity, Cursor, Cline) to read, search, create, and manage your notes.

---

## Installation & Building

### Prerequisites

- **Go 1.21+** installed ([Download Go](https://go.dev/dl/))
- **CGO is NOT required** (uses the pure-Go SQLite driver `modernc.org/sqlite`)

### 1. Build from Source

Clone the repository and build the binary:

```bash
# Clone the repository
git clone https://github.com/tsusheel/kb-cli.git
cd kb-cli

# Build binary
go build -o kb .

# (Optional) Install globally to $GOPATH/bin
go install .
```

On Windows:
```powershell
go build -o kb.exe .
```

### 2. Verify Installation

```bash
./kb --help
```

---

## Configuration

`kb` works out-of-the-box with **zero manual configuration required**. 

On first run, `kb` automatically initializes:
- **Config directory**: `~/.config/kb/`
- **Config file**: `~/.config/kb/config.yaml`
- **SQLite database**: `~/.config/kb/kb.db`

### Configuration Options (`config.yaml`)

```yaml
# Application identifier
app_name: "kb-app"

# Base directory where database and assets are stored
base_path: "~/.config/kb"

# Default custom date format (in Go layout format)
date_format: "2006-01-02"
```

---

## Quickstart & CLI Usage

### 1. Capturing Thoughts

```bash
# Rapid 1-line positional jot (defaults to note type, raw status)
kb "Explore SQLite WAL mode performance"

# Fast jot with flags
kb jot "Implement Raft log compaction" --type todo --due "tomorrow" --area work
kb jot "Graph neural networks for note recommendation" --type idea

# Add note with editor ($EDITOR) for detailed body/flesh
kb add -n "API Gateway Architecture" --type project --area work --tags "backend,arch"
```

### 2. Viewing & Editing Notes

```bash
# List all notes
kb list
kb list --todos      # Only todos
kb list --projects   # Only projects
kb list -t idea      # Filter by type

# Open / view note (fuzzy-finds if ID is omitted)
kb open
kb open a1b2c3d

# Open $EDITOR to write or update note flesh (body)
kb edit a1b2c3d
kb flesh a1b2c3d
```

### 3. Daily Logging & Stream

```bash
# Append a micro-log to today's stream
kb log "Finished reviewing PR for MCP protocol"
kb log "Benchmarked query latency: 1.2ms average"

# View today's chronological stream
kb today
kb log

# Promote a micro-log entry to a permanent Note
kb promote 08db3db --type project
```

### 4. Inbox & Triage

```bash
# View raw unrefined notes and unpromoted logs
kb inbox

# Launch interactive terminal triage wizard
kb triage
```

### 5. Full-Text Search

```bash
kb search "distributed caching"
kb search "sqlite WAL"
```

### 6. Linking Notes

```bash
# Create semantic relationships between notes
kb link <from_id> <to_id> --type related_to
kb link <from_id> <to_id> --type depends_on
```

---

## Connecting AI Agents (MCP Server)

`kb` includes a native Model Context Protocol (MCP) server. Run `kb serve` (or `kb mcp`) over standard input/output (stdio) to connect AI agents.

### Claude Desktop Configuration

Add the following to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "kb": {
      "command": "/path/to/kb",
      "args": ["serve"]
    }
  }
}
```

On Windows:
```json
{
  "mcpServers": {
    "kb": {
      "command": "E:\\kb-cli\\kb-cli\\kb.exe",
      "args": ["serve"]
    }
  }
}
```

### Available MCP Tools

AI assistants will have access to:
- `list_notes`: Filter notes by type, status, or area.
- `get_note`: Fetch full note body, metadata, tags, and links.
- `search_notes`: Run full-text searches across all notes.
- `create_note`: Create new notes, tasks, or projects.
- `update_note`: Update titles, bodies, types, statuses, and due dates.
- `delete_note`: Soft-delete notes with attribution reasons.
- `add_tag`: Associate tags with notes.
- `link_notes`: Create directed semantic relationships between notes.
- `get_daily_logs`: Retrieve daily logs for any date.
- `create_daily_log`: Append micro-logs to the temporal stream.
- `promote_daily_log`: Convert logs to permanent notes.
- `get_inbox`: Fetch unrefined notes and unpromoted logs for triage.

---

## Running Tests

Run the full test suite across database, MCP server, and utility packages:

```bash
go test -v ./...
```

---

## License

[MIT](LICENSE)
