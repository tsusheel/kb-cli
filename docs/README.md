# Knowledge Base CLI (`kb-cli`) Architecture Documentation

This directory contains the authoritative architecture documentation and technical references for the **active, currently implemented components of `kb-cli`**.

---

## Active Architecture Documentation Map

| Document | Subsystem | Description |
| :--- | :--- | :--- |
| [**01. Current System Architecture**](./01-system-architecture.md) | **System Overview** | High-level topology across Go Cobra CLI, embedded Web server, SQLite engine, PostgreSQL sync, and MCP agent bridge. |
| [**02. Database Schema & Models**](./02-database-schema.md) | **Database Layer** | Active SQLite and PostgreSQL tables (`notes`, `tags`, `links`, `daily_logs`, `audit_logs`), FTS5 virtual tables, and indexes. |
| [**03. CLI Command Reference**](./03-cli-commands.md) | **CLI Subsystem** | Comprehensive reference of all 27 Cobra commands, options, and workflows (`add`, `edit`, `delete`, `log`, `sync`, etc.). |
| [**04. Embedded Web Fuzzy Finder**](./04-web-fuzzy-finder.md) | **Web SPA & PWA** | Dual-pane interface, real-time fuzzy search, Markdown and Mermaid rendering, keyboard navigation, and offline Service Worker. |
| [**05. PostgreSQL Sync Engine**](./05-postgres-sync-engine.md) | **Cloud Sync** | Incremental timestamp-watermark synchronization (`updated_at > last_synced_at`), soft-delete replication, and conflict handling. |
| [**06. Model Context Protocol Server**](./06-mcp-server.md) | **AI Agent Connectivity** | Native MCP server implementation over `stdio` providing 12 tools for autonomous agent interaction with audit logging. |
| [**07. Configuration & Security**](./07-configuration-and-security.md) | **Config & Secrets** | Viper configuration (`config.yaml`), path resolution, and hardware-encrypted OS Keyring password management. |
| [**08. Audit & Revision History**](./08-audit-and-revision-history.md) | **Audit & Time Travel** | Immutable snapshot logs, automated diff engine, atomic rollback/reversion, and AI attribution trail. |
| [**09. Multi-Plugin Platform Architecture**](./09-plugin-system.md) | **Plugin Extensibility** | Out-of-process JSON-RPC plugin runner, public Go SDK, capability security model, universal polymorphic storage, and dynamic MCP/Web mounting. |

---

## Future Architecture & Extension Proposals

For architectural plans and specifications regarding upcoming features and external ecosystem plugins (such as the ThingsPad mobile command console and the external Gym logging plugin repository), see the proposals directory:

👉 [**docs/proposals/ (Extension Proposals & Plugin Guides)**](./proposals/README.md)

