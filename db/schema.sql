CREATE TABLE IF NOT EXISTS notes (
  id TEXT PRIMARY KEY,
  title TEXT,
  content TEXT,
  type TEXT,
  status TEXT,
  area TEXT,
  importance INTEGER,
  clarity INTEGER,
  source TEXT,
  created_at DATETIME,
  updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS tags (
  id TEXT PRIMARY KEY,
  name TEXT UNIQUE
);

CREATE TABLE IF NOT EXISTS note_tags (
  note_id TEXT,
  tag_id TEXT,
  PRIMARY KEY (note_id, tag_id)
);

CREATE TABLE IF NOT EXISTS links (
  id TEXT PRIMARY KEY,
  from_note TEXT,
  to_note TEXT,
  type TEXT
);

CREATE TABLE IF NOT EXISTS daily_notes (
  id TEXT PRIMARY KEY,
  date DATETIME,
  note_id TEXT
);

-- Full Text Search
CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
  title,
  content,
  note_id UNINDEXED
);

