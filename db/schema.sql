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

CREATE TABLE IF NOT EXISTS links (
  id TEXT PRIMARY KEY,
  from_note TEXT,
  to_note TEXT,
  type TEXT,
  created_at DATETIME,
  deleted_at DATETIME,
  deleted_note TEXT
);

CREATE TABLE IF NOT EXISTS daily_notes (
  id TEXT PRIMARY KEY,
  date DATETIME,
  note_id TEXT,
  created_at DATETIME
);

-- Full Text Search
CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
  note,
  note_flesh,
  note_id UNINDEXED
);

