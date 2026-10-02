package store

const schemaSQL = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS clipboard_items (
  id INTEGER PRIMARY KEY,
  content_hash TEXT NOT NULL UNIQUE,
  kind TEXT NOT NULL,
  detected_type TEXT NOT NULL DEFAULT 'text',
  language TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  text_content TEXT,
  text_preview TEXT NOT NULL DEFAULT '',
  mime_type TEXT NOT NULL DEFAULT '',
  blob_path TEXT NOT NULL DEFAULT '',
  thumbnail_path TEXT NOT NULL DEFAULT '',
  byte_size INTEGER NOT NULL DEFAULT 0,
  width INTEGER NOT NULL DEFAULT 0,
  height INTEGER NOT NULL DEFAULT 0,
  image_kept INTEGER NOT NULL DEFAULT 1,
  is_pinned INTEGER NOT NULL DEFAULT 0,
  sensitivity TEXT NOT NULL DEFAULT 'none',
  copy_count INTEGER NOT NULL DEFAULT 1,
  search_blob TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  last_copied_at INTEGER NOT NULL,
  last_context_id INTEGER,
  expires_at INTEGER
);

CREATE TABLE IF NOT EXISTS copy_contexts (
  id INTEGER PRIMARY KEY,
  application_name TEXT NOT NULL,
  application_id TEXT NOT NULL DEFAULT '',
  icon_cache_key TEXT NOT NULL DEFAULT '',
  process_id INTEGER NOT NULL DEFAULT 0,
  window_title TEXT NOT NULL DEFAULT '',
  document_title TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT '',
  domain TEXT NOT NULL DEFAULT '',
  file_name TEXT NOT NULL DEFAULT '',
  file_path TEXT NOT NULL DEFAULT '',
  place_key TEXT NOT NULL DEFAULT '',
  deep INTEGER NOT NULL DEFAULT 0,
  captured_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS copy_events (
  id INTEGER PRIMARY KEY,
  item_id INTEGER NOT NULL REFERENCES clipboard_items(id) ON DELETE CASCADE,
  context_id INTEGER NOT NULL REFERENCES copy_contexts(id),
  copied_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS exclusions (
  id INTEGER PRIMARY KEY,
  application_id TEXT NOT NULL DEFAULT '',
  application_name TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT 'user',
  enabled INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_items_last_copied ON clipboard_items(last_copied_at DESC);
CREATE INDEX IF NOT EXISTS idx_items_detected ON clipboard_items(detected_type, last_copied_at DESC);
CREATE INDEX IF NOT EXISTS idx_items_pinned ON clipboard_items(is_pinned, last_copied_at DESC);
CREATE INDEX IF NOT EXISTS idx_items_expires ON clipboard_items(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_events_item ON copy_events(item_id, copied_at DESC);
CREATE INDEX IF NOT EXISTS idx_contexts_app ON copy_contexts(application_id);
CREATE INDEX IF NOT EXISTS idx_contexts_domain ON copy_contexts(domain);
CREATE INDEX IF NOT EXISTS idx_contexts_place ON copy_contexts(place_key);
`

// ftsSQL is attempted separately; if FTS5 is unavailable we continue without it.
const ftsSQL = `
CREATE VIRTUAL TABLE IF NOT EXISTS items_fts USING fts5(
  title,
  text_preview,
  text_content,
  search_blob,
  detected_type,
  language,
  content='',
  tokenize='unicode61 remove_diacritics 2'
);
`
