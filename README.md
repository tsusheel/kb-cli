# Knowledge Base CLI (`kb`)

A lightning-fast, local-first personal knowledge base and second brain CLI with embedded SQLite storage, remote **PostgreSQL** synchronization, and built-in **Model Context Protocol (MCP)** server connectivity for AI agents.

Designed around a **two-speed capture philosophy**: capture fleeting thoughts in milliseconds directly from your terminal, flesh them out into deep markdown notes in your favorite editor, sync securely across machines via PostgreSQL, and seamlessly collaborate with AI assistants over MCP.

---

## Features

- **⚡ Two-Speed Thought Capture**:
  - **Instant Jot**: Fast `< 50ms` capture directly from the terminal without opening an editor (`kb "thought"`, `kb jot "idea"`).
  - **Deep Fleshing**: Dedicated editor workflow (`$EDITOR`) to craft rich markdown bodies (`kb add`, `kb edit <id>`).
- **🔄 Local-First PostgreSQL Sync (`kb sync`)**:
  - Keep 100% offline capability and sub-millisecond local SQLite speed.
  - Synchronize notes, tags, links, and daily logs with any PostgreSQL database (local, self-hosted, or cloud PostgreSQL).
- **🔐 Hardware/OS Encrypted Secrets**:
  - Passwords and connection secrets are **never stored in plaintext config files**.
  - Encrypted directly in your OS Credential Manager (Windows Credential Manager / macOS Keychain / Linux Secret Service) with masked terminal entry.
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
- **CGO is NOT required** (uses pure-Go drivers `modernc.org/sqlite` and `pgx/v5`)

### 1. Build from Source

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

### 3. Enable Shell Tab Auto-Completion (Optional but Recommended)

`kb` supports intelligent tab auto-completion for commands, flags, note types, and dynamic note IDs:

* **PowerShell (Windows)**:
  ```powershell
  # For current session:
  kb completion powershell | Out-String | Invoke-Expression

  # Or permanently add to your PowerShell profile:
  Add-Content $PROFILE "`nInvoke-Expression (&kb completion powershell | Out-String)"
  ```

* **Bash (Linux/macOS)**:
  ```bash
  # In your ~/.bashrc:
  source <(kb completion bash)
  ```

* **Zsh (macOS)**:
  ```zsh
  # In your ~/.zshrc:
  source <(kb completion zsh)
  ```

---

## Configuration & Secrets Management

`kb` works out-of-the-box with **zero manual configuration required**. 

On first run, `kb` automatically initializes `~/.config/kb/config.yaml` and `~/.config/kb/kb.db`.

### CLI Configuration Commands (`kb config`)

Manage configurations directly from the terminal without editing files manually:

```bash
# Set a public configuration value
kb config set date_format "2006-01-02"
kb config set remote.postgres_url "postgres://username@localhost:5432/dbname?sslmode=disable"

# Securely save a secret / password in the OS Credential Manager (input is hidden)
kb config set-secret postgres_password

# View settings
kb config get date_format
kb config list
```

### 🔒 Security Guarantee: Zero Plaintext Secrets

- Passwords stored via `kb config set-secret` are encrypted natively by your Operating System:
  - **Windows**: Windows Credential Manager / DPAPI
  - **macOS**: Apple Keychain
  - **Linux**: Secret Service (GNOME Keyring / KWallet)
- `config.yaml` only contains non-sensitive settings (database URL without password).
- Environment variables (e.g., `KB_POSTGRES_PASSWORD`, `KB_POSTGRES_URL`, `DATABASE_URL`) are automatically supported as overrides for CI/CD or headless environments.

---

## Remote PostgreSQL Synchronization

`kb` uses a **Local-First + Remote Sync** architecture: your daily CLI operations are always lightning-fast on local SQLite, and you can sync with remote PostgreSQL whenever you want.

### 1. Configure in `kb-cli`

Run the interactive setup wizard:

```bash
kb config setup
```

The wizard will:
1. Prompt for your **PostgreSQL Connection URL** (e.g. `postgres://username@localhost:5432/dbname?sslmode=disable`).
2. Prompt for your **PostgreSQL Password** (hidden masked input, saved to OS Keyring).
3. Automatically verify connectivity and initialize the remote tables.

*(Optional)* If you prefer to manually run the PostgreSQL schema, you can find it in [`postgres/schema.sql`](postgres/schema.sql).

### 2. Synchronizing Notes

```bash
# Two-way sync (push local modifications + pull remote modifications)
kb sync

# Push only local changes
kb sync push

# Pull only remote changes
kb sync pull

# Check sync status and pending changes
kb sync status

# Test connection to PostgreSQL
kb sync test
```

---

## Quickstart & CLI Usage

### 1. Capturing Thoughts (`kb [thought]` / `kb add`)

```bash
# Rapid 1-line positional capture (no command needed)
kb "Explore SQLite WAL mode performance"

# Fast add with flags
kb add "Implement Raft log compaction" --type todo --due "tomorrow" --area work
kb add "Graph neural networks for note recommendation" --type idea

# Add note with editor ($EDITOR) for detailed body/flesh
kb add "API Gateway Architecture" -e --type project --area work --tags "backend,arch"
kb add
```

### 2. Viewing, Browsing & Searching Notes (`kb ls [query]`)

```bash
# List all notes in a rounded table
kb ls
kb ls --todos      # Only todos
kb ls --projects   # Only projects
kb ls -t idea      # Filter by type

# Search keywords across note titles and bodies
kb ls "distributed caching"
kb ls postgres
kb ls -t todo "migration"

# Open / view note details (fuzzy-finds if ID is omitted)
kb open
kb open a1b2c3d

# Rename / edit note title directly:
kb edit a1b2c3d "Updated Note Title"
kb rename a1b2c3d "Updated Note Title"
kb edit a1b2c3d -n "Updated Note Title" --status in-progress --type project

# Open $EDITOR to write or update note flesh (body)
kb edit a1b2c3d
kb flesh a1b2c3d
```

### 3. Deleting Notes & Logs (`kb delete` / `kb rm`)

All deletions are safe, non-destructive soft deletes with timestamps and attribution:

```bash
# Delete a note by ID
kb delete a1b2c3d
kb rm a1b2c3d

# Delete with custom attribution reason
kb rm a1b2c3d --reason "superseded by new architecture"

# Interactive fuzzy selection (if ID is omitted)
kb delete

# Delete a daily log
kb delete 6a178a8
kb rm 6a178a8
```

### 4. Daily Logging & Stream (`kb log`)

```bash
# Append a micro-log to today's stream
kb log "Finished reviewing PR for MCP protocol"
kb log "Benchmarked query latency: 1.2ms average"

# View today's unpromoted logs
kb log

# View all of today's logs (including promoted ones)
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

# Promote a micro-log entry to a permanent Note
kb promote 08db3db --type project
```

### 5. Inbox & Triage (`kb inbox` / `kb triage`)

```bash
# View today's raw unrefined notes awaiting triage
kb inbox

# View all raw notes across all dates
kb inbox -a

# Launch interactive terminal triage wizard for today's raw notes
kb triage

# Triage all raw notes across all dates
kb triage -a
```

### 6. Audit & Version History (`kb history` / `kb audit`)

```bash
# View recent global activity stream
kb history
kb audit

# View version timeline for a note or log
kb history a1b2c3d

# Inspect snapshot diff for a specific revision
kb history diff a1b2c3d 9f8e7d6

# Revert a note to a previous revision
kb history revert a1b2c3d 9f8e7d6
```

### 7. Linking Notes (`kb link`)

```bash
# Create semantic relationships between notes
kb link <from_id> <to_id> --type related_to
kb link <from_id> <to_id> --type depends_on
```

### 8. Configuration & Secrets (`kb config`)

```bash
# Set a public configuration value
kb config set date_format "2006-01-02"

# View all configuration settings and OS Keyring secret statuses
kb config list
kb config get remote.postgres_url

# Delete configuration settings or OS Keyring secrets
kb config delete test_key
kb config rm remote.enabled
kb config delete --secret postgres_password
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

Run the full test suite across database, sync engine, OS secrets, MCP server, and utility packages:

```bash
go test -v ./...
```

---

## License

[MIT](LICENSE)
