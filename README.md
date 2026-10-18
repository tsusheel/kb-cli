# Knowledge Base CLI (`kb`)

A fast, local-first personal knowledge base and second-brain application with embedded SQLite storage, remote PostgreSQL synchronization, an embedded Web Fuzzy Finder (PWA), and native Model Context Protocol (MCP) server connectivity for AI agents.

---

## Documentation

All architectural details, command references, database schemas, and guides are thoroughly documented in the [**docs/**](docs/README.md) directory:

- [**01. System Architecture**](docs/01-system-architecture.md): Runtime lifecycle, concurrency, and subsystem topology.
- [**02. Database Schema & Models**](docs/02-database-schema.md): Complete SQLite & PostgreSQL schemas, indexes, and Go models.
- [**03. CLI Command Reference**](docs/03-cli-commands.md): Complete usage guide for all 27 CLI commands and workflows.
- [**04. Embedded Web Fuzzy Finder**](docs/04-web-fuzzy-finder.md): Dual-pane interface, Markdown/Mermaid engine, and PWA offline setup.
- [**05. PostgreSQL Sync Engine**](docs/05-postgres-sync-engine.md): Incremental watermark sync, conflict resolution, and DDL.
- [**06. MCP Server**](docs/06-mcp-server.md): Stdio Model Context Protocol integration (all 22 tools, resources, and prompts).
- [**07. Configuration & Security**](docs/07-configuration-and-security.md): `config.yaml` reference, OS Keyring secrets, and TLS.
- [**Future Extension Proposals**](docs/proposals/README.md): Upcoming Plugin System, ThingsPad Console, and Gym Logging specs.

---

## Installation & Setup

### 1. Automated Install (Linux & macOS)

Run the automated installer script:
```bash
curl -fsSL https://raw.githubusercontent.com/tsusheel/kb-cli/main/install.sh | bash
```

---

### 2. Native Package Managers

#### Ubuntu / Debian / Linux Mint / Pop!_OS (`apt`)
Via Cloudflare R2 APT Repository:
```bash
# 1. Add GPG Key
curl -fsSL https://apt.tsusheel.com/KEY.gpg | gpg --dearmor | sudo tee /etc/apt/keyrings/kb.gpg > /dev/null

# 2. Add Repository
echo "deb [signed-by=/etc/apt/keyrings/kb.gpg] https://apt.tsusheel.com stable main" | sudo tee /etc/apt/sources.list.d/kb.list

# 3. Install & auto-update
sudo apt update && sudo apt install kb
```

#### Fedora / RHEL / CentOS / Rocky (`dnf`)
Via Fedora COPR:
```bash
sudo dnf copr enable tsusheel/kb-cli
sudo dnf install kb
```

#### Nix / NixOS (`nix`)
Run instantly without installing:
```bash
nix run github:tsusheel/kb-cli
```
Or install into your Nix profile:
```bash
nix profile install github:tsusheel/kb-cli
```

---

### 3. Build from Source

#### Prerequisites
- **Go 1.21+** installed ([Download Go](https://go.dev/dl/))
- **Node.js & npm** (optional, only required if modifying web frontend assets in `server/web/`)

#### Build Steps
```bash
# 1. Clone the repository
git clone https://github.com/tsusheel/kb-cli.git
cd kb-cli

# 2. (Optional) Rebuild web assets if edited
npm install
npm run build

# 3. Build the binary
go build -o kb .

# 4. (Optional) Install globally to $GOPATH/bin
go install .
```

On Windows (PowerShell):
```powershell
go build -o kb.exe .
```

---

### 4. Verification & Initial Run

`kb` works out of the box with zero manual configuration. On first run, it automatically initializes its local database directory at `~/.config/kb/`:

```bash
# Verify installation
kb --help

# Capture your first note
kb "My first captured note in kb"

# View note in terminal
kb ls
```

---

### 5. Shell Tab Auto-Completion (Recommended)

Enable tab auto-completion for commands, flags, note types, and dynamic note IDs:

#### PowerShell (Windows)
```powershell
# In your current session:
kb completion powershell | Out-String | Invoke-Expression

# Or add permanently to your profile:
Add-Content $PROFILE "`nInvoke-Expression (&kb completion powershell | Out-String)"
```

#### Bash (Linux/macOS)
```bash
# In your ~/.bashrc:
source <(kb completion bash)
```

#### Zsh (macOS/Linux)
```zsh
# In your ~/.zshrc:
source <(kb completion zsh)
```

#### Fish
```fish
kb completion fish | source
```

---

### 6. Running Tests

Run the full test suite across the database, PostgreSQL sync engine, OS secrets, MCP server, and CLI commands:

```bash
go test -v ./...
```

---

## License

[MIT](LICENSE)
