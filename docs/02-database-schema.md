# 02. Active Database Schema & Models

## 1. Storage Overview

The storage architecture of `kb-cli` is implemented in two tiers:
1. **Local Embedded SQLite Tier ([db/schema.sql](file:///e:/kb-cli/kb-cli/db/schema.sql)):** Serves as the primary operational database, running locally at `~/.config/kb/kb.db` with Write-Ahead Logging (WAL) and Full-Text Search (FTS5).
2. **Remote PostgreSQL Tier ([sync/postgres.go](file:///e:/kb-cli/kb-cli/sync/postgres.go)):** Serves as the centralized synchronization hub, mirroring the local relational model using cloud-native PostgreSQL data types (`TIMESTAMPTZ`, `JSONB`).

The schema is built around an **atomic thought model** with distinct headline summaries (`note`), detailed Markdown bodies (`note_flesh`), relational tags, semantic graph edges, daily logs, and revision audit logs.

```mermaid
erDiagram
    NOTES ||--o{ NOTE_TAGS : "tagged with"
    TAGS ||--o{ NOTE_TAGS : "belongs to"
    NOTES ||--o{ LINKS : "outgoing link (from_note)"
    NOTES ||--o{ LINKS : "incoming link (to_note)"
    NOTES ||--o{ DAILY_LOGS : "promoted from"
    NOTES ||--o{ AUDIT_LOGS : "revision history"

    NOTES {
        string id PK "UUID v4 or random hex"
        string note "Headline summary (< 120 chars)"
        string note_flesh "Long-form Markdown body"
        string type "Note intent type"
        string status "Lifecycle status"
        string area "Functional life/work area"
        int importance "1-5 priority score"
        int clarity "1-5 clarity score"
        string source "Origin (URL, book, agent)"
        datetime target_date_time "Due/milestone date"
        datetime created_at "Creation timestamp"
        datetime updated_at "Update watermark"
        datetime deleted_at "Soft delete timestamp"
        string deleted_note "Deletion attribution"
    }

    TAGS {
        string id PK "UUID v4"
        string name UK "Unique tag string"
        datetime created_at "Creation timestamp"
    }

    NOTE_TAGS {
        string note_id PK,FK "References notes(id)"
        string tag_id PK,FK "References tags(id)"
        datetime created_at "Association timestamp"
    }

    LINKS {
        string id PK "UUID v4"
        string from_note FK "Source note ID"
        string to_note FK "Target note ID"
        string type "Semantic relationship"
        datetime created_at "Creation timestamp"
        datetime deleted_at "Soft delete timestamp"
        string deleted_note "Deletion attribution"
    }

    DAILY_LOGS {
        string id PK "UUID v4"
        string content "Micro-log text"
        string note_id FK "Promoted note ID (nullable)"
        datetime created_at "Creation timestamp"
        datetime deleted_at "Soft delete timestamp"
        string deleted_note "Deletion attribution"
    }

    SYNC_STATE {
        string provider PK "e.g. 'postgres'"
        datetime last_synced_at "High-water mark"
        datetime created_at "Initial sync time"
    }

    AUDIT_LOGS {
        string id PK "UUID v4"
        string entity_type "Entity category"
        string entity_id "Entity UUID"
        string action "Operation performed"
        string changes_summary "Human readable diff"
        string snapshot_json "Full JSON state snapshot"
        datetime created_at "Timestamp of change"
    }
```

---

## 2. Table-by-Table Schema Specification

### 2.1 Table: `notes`
The core entity table representing thoughts, tasks, projects, ideas, and decisions.

```sql
CREATE TABLE IF NOT EXISTS notes (
  id TEXT PRIMARY KEY,
  note TEXT,
  note_flesh TEXT,
  type TEXT,
  status TEXT,
  area TEXT,
  importance INTEGER,
  clarity INTEGER,
  source TEXT,
  target_date_time DATETIME,
  created_at DATETIME,
  updated_at DATETIME,
  deleted_at DATETIME,
  deleted_note TEXT
);
```

#### Column Details
- **`id` (TEXT, PK):** Unique record identifier (UUID v4 or truncated 7-character prefix for CLI display).
- **`note` (TEXT):** The concise headline or summary thesis statement. Typically under 120 characters to allow compact presentation in terminal tables, search lists, and LLM context windows.
- **`note_flesh` (TEXT):** The long-form substantive Markdown body. Contains detailed explanations, code blocks, checklists, references, and mathematical reasoning.
- **`type` (TEXT):** Encodes the intent of the note:
  - `'note'`: General atomic thought or synthesis.
  - `'todo'`: Actionable task requiring completion.
  - `'project'`: Multi-step initiative grouping related notes.
  - `'idea'`: Nascent, unvalidated concept or hypothesis.
  - `'decision'`: Architecture Decision Record (ADR) capturing evaluated options and trade-offs.
  - `'concept'`: Enduring mental model, definition, or systemic abstraction.
  - `'til'`: "Today I Learned" — discrete discovery or technique.
  - `'resource'`: Reference, bookmark, documentation link, or tool review.
  - `'question'`: Open research inquiry.
  - `'experiment'`: Empirical test, benchmark, or experiment log.
  - `'person'`: Colleague, collaborator, author, or subject expert.
- **`status` (TEXT):** Represents progressive refinement lifecycle:
  - `'raw'`: Chaotic, unedited initial capture.
  - `'refined'`: Structured, reviewed, and categorized.
  - `'in-progress'`: Actively being executed.
  - `'completed'`: Task finished or project concluded.
  - `'archived'`: Retired knowledge preserved for reference.
- **`area` (TEXT):** Functional domain (e.g., `'work'`, `'finance'`, `'health'`, `'personal'`).
- **`importance` (INTEGER, 1-5):** Priority rating from 1 (low) to 5 (critical).
- **`clarity` (INTEGER, 1-5):** Articulation score from 1 (vague idea) to 5 (completely refined).
- **`source` (TEXT):** Attribution (e.g., `'AI Agent'`, `'Web'`, `'Book: DDIA'`).
- **`target_date_time` (DATETIME, Nullable):** Target due date, deadline, or milestone timestamp.
- **`created_at` (DATETIME):** Timestamp of initial creation.
- **`updated_at` (DATETIME):** Monotonically updated timestamp for sync watermarking.
- **`deleted_at` (DATETIME, Nullable):** Non-destructive soft-delete tombstone timestamp.
- **`deleted_note` (TEXT, Nullable):** Context or attribution explaining why the note was deleted.

---

### 2.2 Tables: `tags` & `note_tags`
Implements a normalized many-to-many relationship without embedding inline `#tags` into prose.

```sql
CREATE TABLE IF NOT EXISTS tags (
  id TEXT PRIMARY KEY,
  name TEXT UNIQUE,
  created_at DATETIME
);

CREATE TABLE IF NOT EXISTS note_tags (
  note_id TEXT,
  tag_id TEXT,
  created_at DATETIME,
  PRIMARY KEY (note_id, tag_id)
);
```

#### Foreign Keys & Cascading
In PostgreSQL:
```sql
CREATE TABLE IF NOT EXISTS note_tags (
  note_id VARCHAR(64) REFERENCES notes(id) ON DELETE CASCADE,
  tag_id VARCHAR(64) REFERENCES tags(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (note_id, tag_id)
);
```
- Tag names are unique (`UNIQUE` constraint on `name`).
- Associating or removing tags requires inserting or deleting rows from `note_tags`.
- Tag names support hierarchical namespace notation (e.g., `'tech/golang'`, `'project/kb'`).

---

### 2.3 Table: `links`
Stores directional semantic relationships connecting notes in an interconnected knowledge graph.

```sql
CREATE TABLE IF NOT EXISTS links (
  id TEXT PRIMARY KEY,
  from_note TEXT,
  to_note TEXT,
  type TEXT,
  created_at DATETIME,
  deleted_at DATETIME,
  deleted_note TEXT
);
```

#### Semantic Relationship Types (`type`)
- `'related_to'`: General associative connection (bidirectional semantics).
- `'part_of'`: Component-to-whole relationship (e.g., task is part of project).
- `'inspired_by'`: Lineage of thought (e.g., idea inspired by article).
- `'depends_on'`: Prerequisite dependency (e.g., task B depends on task A).
- `'supports'`: Evidence, benchmark, or supporting argument.
- `'contradicts'`: Counterpoint or competing hypothesis.
- `'about'`: Topical connection (e.g., note is about a concept).
- `'created_by'`: Attribution (e.g., project created by a person).

---

### 2.4 Table: `daily_logs`
Chronological micro-stream for timestamped thoughts recorded throughout the day.

```sql
CREATE TABLE IF NOT EXISTS daily_logs (
  id TEXT PRIMARY KEY,
  content TEXT,
  note_id TEXT,
  created_at DATETIME,
  deleted_at DATETIME,
  deleted_note TEXT
);
```

- **`content` (TEXT):** Raw log text.
- **`note_id` (TEXT, Nullable):** If the log is promoted to a permanent note via `kb promote <id>`, this column records the promoted note's UUID, linking the micro-log to its evolved form.
- **`deleted_at` (DATETIME, Nullable):** Soft-delete timestamp.

---

### 2.5 Table: `sync_state`
Tracks replication progress with external synchronization providers.

```sql
CREATE TABLE IF NOT EXISTS sync_state (
  provider TEXT PRIMARY KEY,
  last_synced_at DATETIME,
  created_at DATETIME
);
```

- When syncing with PostgreSQL, `provider` is `'postgres'`.
- `last_synced_at` stores the high-water mark timestamp used in subsequent sync runs.

---

### 2.6 Table: `audit_logs`
Centralized immutable change journal recording point-in-time snapshots for version history and reversion.

```sql
CREATE TABLE IF NOT EXISTS audit_logs (
  id TEXT PRIMARY KEY,
  entity_type TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  action TEXT NOT NULL,
  changes_summary TEXT,
  snapshot_json TEXT,
  created_at DATETIME NOT NULL
);
```

- **`entity_type`:** `'note'`, `'daily_log'`, or `'link'`.
- **`action`:** `'create'`, `'update'`, `'delete'`, or `'promote'`.
- **`changes_summary`:** Human-readable diff summary (e.g., `"Updated title from X to Y"`).
- **`snapshot_json`:** Full serialized JSON dump of the entity before/after mutation, enabling `kb history revert <id> <rev>`.

---

## 3. SQLite Full-Text Search Virtual Table (`notes_fts`)

Full-text search is implemented natively in SQLite using **FTS5**:

```sql
CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
  note,
  note_flesh,
  note_id UNINDEXED
);
```

### Search Operation
Queries execute using SQLite FTS5 MATCH syntax:
```sql
SELECT n.id, n.note, n.type, n.status, n.created_at
FROM notes n
JOIN notes_fts fts ON n.id = fts.note_id
WHERE notes_fts MATCH ? AND n.deleted_at IS NULL;
```
FTS5 tokenizes words automatically, supporting prefix queries (e.g., `postgre*`), boolean operators (`postgres AND sync`), and exact phrase matches (`"WAL mode"`).

---

## 4. Performance Indexes

The database maintains 7 dedicated B-Tree indexes to ensure sub-millisecond query performance:

```sql
-- Fast filtering of active vs soft-deleted notes
CREATE INDEX IF NOT EXISTS idx_notes_deleted_at ON notes(deleted_at);

-- Filtering notes by intent type (todo, note, project, idea)
CREATE INDEX IF NOT EXISTS idx_notes_type ON notes(type);

-- Filtering notes by lifecycle status (active, raw, completed)
CREATE INDEX IF NOT EXISTS idx_notes_status ON notes(status);

-- Bidirectional graph traversal (finding incoming and outgoing links)
CREATE INDEX IF NOT EXISTS idx_links_from_to ON links(from_note, to_note);

-- Chronological daily stream queries (today, date ranges)
CREATE INDEX IF NOT EXISTS idx_daily_logs_created_at ON daily_logs(created_at);

-- Entity-specific version history queries
CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_logs(entity_type, entity_id, created_at DESC);

-- Chronological global audit stream
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_logs(created_at DESC);
```

---

## 5. Go Models & Struct Definitions (`models/`)

The database maps directly to Go structs defined in `models/`:

### 5.1 `models.Note` ([models/note.go](file:///e:/kb-cli/kb-cli/models/note.go))
```go
type Note struct {
    ID             string
    Note           string
    NoteFlesh      string
    Type           NoteType
    Status         Status
    Area           Area
    CreatedAt      time.Time
    UpdatedAt      time.Time
    DeletedAt      time.Time
    DeletedNote    string
    Importance     int
    Clarity        int
    Source         string
    TargetDateTime time.Time
}
```

### 5.2 `models.Tag` & `models.NoteTag` ([models/tag.go](file:///e:/kb-cli/kb-cli/models/tag.go) & [models/notetag.go](file:///e:/kb-cli/kb-cli/models/notetag.go))
```go
type Tag struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}

type NoteTag struct {
    NoteID    string    `json:"note_id"`
    TagID     string    `json:"tag_id"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 5.3 `models.Link` ([models/link.go](file:///e:/kb-cli/kb-cli/models/link.go))
```go
type Link struct {
    ID          string    `json:"id"`
    FromNote    string    `json:"from_note"`
    ToNote      string    `json:"to_note"`
    Type        LinkType  `json:"type"`
    CreatedAt   time.Time `json:"created_at"`
    DeletedAt   time.Time `json:"deleted_at,omitempty"`
    DeletedNote string    `json:"deleted_note,omitempty"`
}
```

### 5.4 `models.DailyLog` ([models/dailylog.go](file:///e:/kb-cli/kb-cli/models/dailylog.go))
```go
type DailyLog struct {
    ID          string    `json:"id"`
    Content     string    `json:"content"`
    NoteID      string    `json:"note_id,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    DeletedAt   time.Time `json:"deleted_at,omitempty"`
    DeletedNote string    `json:"deleted_note,omitempty"`
}
```

### 5.5 `models.AuditEntry` ([models/audit.go](file:///e:/kb-cli/kb-cli/models/audit.go))
```go
type AuditAction string

const (
    AuditCreate  AuditAction = "create"
    AuditUpdate  AuditAction = "update"
    AuditDelete  AuditAction = "delete"
    AuditPromote AuditAction = "promote"
)

type AuditEntry struct {
    ID             string      `json:"id"`
    EntityType     string      `json:"entity_type"`
    EntityID       string      `json:"entity_id"`
    Action         AuditAction `json:"action"`
    ChangesSummary string      `json:"changes_summary"`
    SnapshotJSON   string      `json:"snapshot_json"`
    CreatedAt      time.Time   `json:"created_at"`
}
```

---

## 6. Versioned Schema Migrations (`db/migrations.go`)

`kb-cli` uses an automated, versioned migration runner with checksum validation to manage database evolution without data loss:

### 6.1 Core Schema Migrations (`schema_migrations`)
```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     INTEGER PRIMARY KEY,
    name        TEXT NOT NULL,
    applied_at  DATETIME NOT NULL
);
```

### 6.2 Plugin DDL Migrations (`plugin_migrations`)
```sql
CREATE TABLE IF NOT EXISTS plugin_migrations (
    plugin_id   TEXT NOT NULL,
    version     INTEGER NOT NULL,
    checksum    TEXT NOT NULL,             -- SHA-256 hash of the migration script
    applied_at  DATETIME NOT NULL,
    PRIMARY KEY (plugin_id, version)
);
```

---

## 7. Plugin Registry & Capabilities

### 7.1 Plugin Registry (`plugins`)
```sql
CREATE TABLE IF NOT EXISTS plugins (
    plugin_id       TEXT PRIMARY KEY,
    version         TEXT NOT NULL,
    api_version     TEXT NOT NULL,
    schema_version  INTEGER NOT NULL DEFAULT 1,
    enabled         INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
    source          TEXT NOT NULL,         -- e.g. 'external:/path/to/binary' or 'builtin'
    manifest_json   TEXT NOT NULL,
    installed_at    DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL
);
```

### 7.2 Plugin Capabilities (`plugin_capabilities`)
```sql
CREATE TABLE IF NOT EXISTS plugin_capabilities (
    plugin_id       TEXT NOT NULL,
    capability      TEXT NOT NULL,         -- e.g. 'storage.entries', 'audit.write'
    granted         INTEGER NOT NULL DEFAULT 0 CHECK (granted IN (0, 1)),
    PRIMARY KEY (plugin_id, capability)
);
```

---

## 8. Universal Storage Envelope (`plugin_entries`)

Polymorphic JSON document store enabling external plugins to persist arbitrary domain records while retaining indexed query performance and full-text search:

```sql
CREATE TABLE IF NOT EXISTS plugin_entries (
    id              TEXT PRIMARY KEY,
    plugin_id       TEXT NOT NULL,
    entity_type     TEXT NOT NULL,
    schema_version  INTEGER NOT NULL DEFAULT 1,
    parent_id       TEXT,
    status          TEXT NOT NULL DEFAULT 'active',
    occurred_at     DATETIME,
    entry_date      TEXT,                           -- 'YYYY-MM-DD'
    num_val         REAL DEFAULT 0,
    search_text     TEXT NOT NULL DEFAULT '',
    data            TEXT NOT NULL DEFAULT '{}',     -- Domain payload JSON
    created_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL,
    deleted_at      DATETIME,
    deleted_note    TEXT,
    revision        INTEGER NOT NULL DEFAULT 1,
    UNIQUE (plugin_id, id),
    FOREIGN KEY (plugin_id) REFERENCES plugins(plugin_id)
);

CREATE INDEX IF NOT EXISTS idx_plugin_entries_date ON plugin_entries(plugin_id, entity_type, entry_date);
CREATE INDEX IF NOT EXISTS idx_plugin_entries_parent ON plugin_entries(plugin_id, parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_plugin_entries_updated ON plugin_entries(updated_at, id);
CREATE INDEX IF NOT EXISTS idx_plugin_entries_status ON plugin_entries(plugin_id, status, entry_date);
CREATE INDEX IF NOT EXISTS idx_plugin_entries_numeric ON plugin_entries(plugin_id, entity_type, num_val);
```

### 8.1 SQLite FTS5 Full-Text Index (`plugin_entries_fts`)
```sql
CREATE VIRTUAL TABLE IF NOT EXISTS plugin_entries_fts USING fts5(
    entry_id UNINDEXED,
    plugin_id UNINDEXED,
    entity_type UNINDEXED,
    search_text,
    tokenize = 'unicode61'
);
```

---

## 9. Entity Revisions, Tags, & Links

### 9.1 Immutable Entity Revisions (`entity_revisions`)
```sql
CREATE TABLE IF NOT EXISTS entity_revisions (
    revision_id     TEXT PRIMARY KEY,
    plugin_id       TEXT NOT NULL,
    entity_type     TEXT NOT NULL,
    entity_id       TEXT NOT NULL,
    revision        INTEGER NOT NULL,
    schema_version  INTEGER NOT NULL,
    snapshot_json   TEXT NOT NULL,
    created_at      DATETIME NOT NULL,
    UNIQUE (plugin_id, entity_type, entity_id, revision)
);

CREATE INDEX IF NOT EXISTS idx_revisions_entity ON entity_revisions(plugin_id, entity_type, entity_id, revision DESC);
```

### 9.2 Cross-Entity Tags & Links
```sql
CREATE TABLE IF NOT EXISTS entity_tags (
    plugin_id       TEXT NOT NULL,
    entity_type     TEXT NOT NULL,
    entity_id       TEXT NOT NULL,
    tag_id          TEXT NOT NULL,
    PRIMARY KEY (plugin_id, entity_type, entity_id, tag_id),
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS entity_links (
    link_id         TEXT PRIMARY KEY,
    source_plugin   TEXT NOT NULL,
    source_type     TEXT NOT NULL,
    source_id       TEXT NOT NULL,
    target_plugin   TEXT NOT NULL,
    target_type     TEXT NOT NULL,
    target_id       TEXT NOT NULL,
    relation_type   TEXT NOT NULL DEFAULT 'related_to',
    created_at      DATETIME NOT NULL,
    UNIQUE (source_plugin, source_type, source_id, target_plugin, target_type, target_id, relation_type)
);
```

---

## 10. Transactional Outbox & Sync Cursors

```sql
CREATE TABLE IF NOT EXISTS sync_outbox (
    sequence        INTEGER PRIMARY KEY AUTOINCREMENT,
    change_id       TEXT NOT NULL UNIQUE,
    plugin_id       TEXT NOT NULL,
    entity_type     TEXT NOT NULL,
    entity_id       TEXT NOT NULL,
    operation       TEXT NOT NULL CHECK (operation IN ('upsert', 'tombstone')),
    base_version    INTEGER NOT NULL DEFAULT 0,
    entity_version  INTEGER NOT NULL,
    schema_version  INTEGER NOT NULL,
    payload_json    TEXT NOT NULL,
    created_at      DATETIME NOT NULL,
    attempts        INTEGER NOT NULL DEFAULT 0,
    last_error      TEXT,
    acknowledged_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON sync_outbox(sequence) WHERE acknowledged_at IS NULL;

CREATE TABLE IF NOT EXISTS sync_cursors (
    remote_id       TEXT NOT NULL,
    stream_id       TEXT NOT NULL,
    cursor_value    TEXT,
    updated_at      DATETIME NOT NULL,
    PRIMARY KEY (remote_id, stream_id)
);
```

---

## 11. Extended Audit Logs Columns

Migration v2 enhances `audit_logs` with backward-compatible column extensions:
- `plugin_id TEXT DEFAULT 'core'`
- `revision INTEGER DEFAULT 1`
- `actor_type TEXT DEFAULT 'user'` (`user`, `plugin`, `ai`, `system`)
- `actor_id TEXT`
- `request_id TEXT`
- `before_revision TEXT`
- `after_revision TEXT`
- `diff_json TEXT DEFAULT '{}'`

---

## 12. Go Domain Models ([models/plugin.go](file:///e:/kb-cli/kb-cli/models/plugin.go))

```go
type PluginEntry struct {
    ID            string     `json:"id"`
    PluginID      string     `json:"plugin_id"`
    EntityType    string     `json:"entity_type"`
    SchemaVersion int        `json:"schema_version"`
    ParentID      *string    `json:"parent_id,omitempty"`
    Status        string     `json:"status"`
    OccurredAt    *time.Time `json:"occurred_at,omitempty"`
    EntryDate     *string    `json:"entry_date,omitempty"` // YYYY-MM-DD
    NumVal        float64    `json:"num_val"`
    SearchText    string     `json:"search_text"`
    Data          string     `json:"data"` // Validated JSON payload
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
    DeletedAt     *time.Time `json:"deleted_at,omitempty"`
    DeletedNote   *string    `json:"deleted_note,omitempty"`
    Revision      int64      `json:"revision"`
}

type EntityRevision struct {
    RevisionID    string    `json:"revision_id"`
    PluginID      string    `json:"plugin_id"`
    EntityType    string    `json:"entity_type"`
    EntityID      string    `json:"entity_id"`
    Revision      int64     `json:"revision"`
    SchemaVersion int       `json:"schema_version"`
    SnapshotJSON  string    `json:"snapshot_json"`
    CreatedAt     time.Time `json:"created_at"`
}

type OutboxChange struct {
    Sequence       int64      `json:"sequence"`
    ChangeID       string     `json:"change_id"`
    PluginID       string     `json:"plugin_id"`
    EntityType     string     `json:"entity_type"`
    EntityID       string     `json:"entity_id"`
    Operation      string     `json:"operation"` // "upsert" or "tombstone"
    BaseVersion    int64      `json:"base_version"`
    EntityVersion  int64      `json:"entity_version"`
    SchemaVersion  int        `json:"schema_version"`
    PayloadJSON    string     `json:"payload_json"`
    CreatedAt      time.Time  `json:"created_at"`
    Attempts       int        `json:"attempts"`
    LastError      *string    `json:"last_error,omitempty"`
    AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
}
```

