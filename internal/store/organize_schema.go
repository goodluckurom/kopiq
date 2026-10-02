package store

const organizeSQL = `
CREATE TABLE IF NOT EXISTS collections (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS collection_items (
  collection_id INTEGER NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
  item_id INTEGER NOT NULL REFERENCES clipboard_items(id) ON DELETE CASCADE,
  added_at INTEGER NOT NULL,
  PRIMARY KEY (collection_id, item_id)
);

CREATE TABLE IF NOT EXISTS snippets (
  id INTEGER PRIMARY KEY,
  source_item_id INTEGER,
  title TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL,
  language TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS snippet_aliases (
  snippet_id INTEGER NOT NULL REFERENCES snippets(id) ON DELETE CASCADE,
  alias TEXT NOT NULL UNIQUE,
  PRIMARY KEY (snippet_id, alias)
);

CREATE INDEX IF NOT EXISTS idx_collection_items_item ON collection_items(item_id);
CREATE INDEX IF NOT EXISTS idx_snippets_updated ON snippets(updated_at DESC);
`

const stacksSQL = `
CREATE TABLE IF NOT EXISTS stacks (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS stack_items (
  stack_id INTEGER NOT NULL REFERENCES stacks(id) ON DELETE CASCADE,
  position INTEGER NOT NULL,
  item_id INTEGER,
  inline_text TEXT NOT NULL DEFAULT '',
  label TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (stack_id, position)
);

CREATE INDEX IF NOT EXISTS idx_stack_items_item ON stack_items(item_id);
`