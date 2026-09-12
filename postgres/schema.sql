-- ==============================================================================
-- PostgreSQL Schema for Knowledge Base CLI (kb-cli)
-- ==============================================================================

-- 1. Notes Table
CREATE TABLE IF NOT EXISTS notes (
  id VARCHAR(64) PRIMARY KEY,
  note TEXT NOT NULL,
  note_flesh TEXT DEFAULT '',
  type VARCHAR(64) DEFAULT 'note',
  status VARCHAR(64) DEFAULT 'active',
  area VARCHAR(64) DEFAULT '',
  importance INTEGER DEFAULT 0,
  clarity INTEGER DEFAULT 0,
  source TEXT DEFAULT '',
  target_date_time TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  deleted_note TEXT
);

-- 2. Tags Table
CREATE TABLE IF NOT EXISTS tags (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(255) UNIQUE NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Note Tags (Many-to-Many junction)
CREATE TABLE IF NOT EXISTS note_tags (
  note_id VARCHAR(64) REFERENCES notes(id) ON DELETE CASCADE,
  tag_id VARCHAR(64) REFERENCES tags(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (note_id, tag_id)
);

-- 4. Links Table (Graph relationships)
CREATE TABLE IF NOT EXISTS links (
  id VARCHAR(64) PRIMARY KEY,
  from_note VARCHAR(64) REFERENCES notes(id) ON DELETE CASCADE,
  to_note VARCHAR(64) REFERENCES notes(id) ON DELETE CASCADE,
  type VARCHAR(64) DEFAULT 'related_to',
  created_at TIMESTAMPTZ DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  deleted_note TEXT
);

-- 5. Daily Logs Table (Stream & micro-logging)
CREATE TABLE IF NOT EXISTS daily_logs (
  id VARCHAR(64) PRIMARY KEY,
  content TEXT NOT NULL,
  note_id VARCHAR(64),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  deleted_note TEXT
);

-- 6. Indexes for Performance
CREATE INDEX IF NOT EXISTS idx_pg_notes_deleted_at ON notes(deleted_at);
CREATE INDEX IF NOT EXISTS idx_pg_notes_updated_at ON notes(updated_at);
CREATE INDEX IF NOT EXISTS idx_pg_notes_type ON notes(type);
CREATE INDEX IF NOT EXISTS idx_pg_notes_status ON notes(status);
CREATE INDEX IF NOT EXISTS idx_pg_daily_logs_created_at ON daily_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_pg_links_from_to ON links(from_note, to_note);
