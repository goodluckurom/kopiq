package store

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"kopiq/internal/domain"
)

// Store is the SQLite-backed repository.
type Store struct {
	db  *sql.DB
	mu  sync.Mutex
	fts bool
}

// Open opens (or creates) the database at path and runs migrations.
func Open(path string) (*Store, error) {
	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(0)

	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("wal: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		_ = db.Close()
		return nil, err
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	if _, err := s.db.Exec(ftsSQL); err != nil {
		// FTS5 may be unavailable; keep going with LIKE search.
		s.fts = false
	} else {
		s.fts = true
	}
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 1`).Scan(&n)
	if err != nil {
		return err
	}
	if n == 0 {
		if _, err := s.db.Exec(`INSERT INTO schema_migrations(version, applied_at) VALUES (1, ?)`, time.Now().UnixMilli()); err != nil {
			return err
		}
		if err := s.seedBuiltinExclusions(); err != nil {
			return err
		}
	}
	// v2: collections + snippets (KQ-017)
	if err := s.ensureMigration(2, organizeSQL); err != nil {
		return err
	}
	// v3: clipboard stacks (KQ-018)
	if err := s.ensureMigration(3, stacksSQL); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureMigration(version int, sqlText string) error {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version).Scan(&n)
	if err != nil {
		return err
	}
	if n > 0 {
		// Still run CREATE IF NOT EXISTS for safety on partial applies.
		if _, err := s.db.Exec(sqlText); err != nil {
			return fmt.Errorf("schema v%d: %w", version, err)
		}
		return nil
	}
	if _, err := s.db.Exec(sqlText); err != nil {
		return fmt.Errorf("schema v%d: %w", version, err)
	}
	_, err = s.db.Exec(`INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)`, version, time.Now().UnixMilli())
	return err
}

func (s *Store) seedBuiltinExclusions() error {
	builtins := []struct{ id, name string }{
		{"com.1password.1password", "1Password"},
		{"com.bitwarden.desktop", "Bitwarden"},
		{"org.keepassxc.keepassxc", "KeePassXC"},
		{"com.apple.Passwords", "Passwords"},
		{"com.apple.keychainaccess", "Keychain Access"},
	}
	for _, b := range builtins {
		_, err := s.db.Exec(
			`INSERT INTO exclusions(application_id, application_name, source, enabled) VALUES (?,?, 'builtin', 1)`,
			b.id, b.name,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB exposes the underlying *sql.DB for settings helpers.
func (s *Store) DB() *sql.DB { return s.db }

// HasFTS reports whether FTS5 is available.
func (s *Store) HasFTS() bool { return s.fts }

// CountItems returns the number of clipboard items.
func (s *Store) CountItems() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM clipboard_items`).Scan(&n)
	return n, err
}

// IsExcluded returns true if applicationID (or name fallback) is excluded.
func (s *Store) IsExcluded(applicationID, applicationName string) (bool, error) {
	var n int
	if applicationID != "" {
		err := s.db.QueryRow(
			`SELECT COUNT(*) FROM exclusions WHERE enabled = 1 AND application_id = ?`,
			applicationID,
		).Scan(&n)
		if err != nil {
			return false, err
		}
		if n > 0 {
			return true, nil
		}
	}
	if applicationName != "" {
		err := s.db.QueryRow(
			`SELECT COUNT(*) FROM exclusions WHERE enabled = 1 AND application_id = '' AND lower(application_name) = lower(?)`,
			applicationName,
		).Scan(&n)
		return n > 0, err
	}
	return false, nil
}

// InsertOrBump inserts a new item or bumps an existing one by content hash.
// Returns the item id and whether it was an insert.
func (s *Store) InsertOrBump(item *domain.CanonicalItem, ctx domain.CopyContext, now int64) (id int64, inserted bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = tx.Rollback() }()

	var existingID int64
	err = tx.QueryRow(`SELECT id FROM clipboard_items WHERE content_hash = ?`, item.ContentHash).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return 0, false, err
	}
	isNew := err == sql.ErrNoRows

	ctxID, err := insertContext(tx, ctx)
	if err != nil {
		return 0, false, err
	}

	searchBlob := buildSearchBlob(ctx)

	if isNew {
		res, err := tx.Exec(`
INSERT INTO clipboard_items(
  content_hash, kind, detected_type, language, title, text_content, text_preview,
  mime_type, blob_path, thumbnail_path, byte_size, width, height, image_kept,
  is_pinned, sensitivity, copy_count, search_blob, metadata_json,
  created_at, last_copied_at, last_context_id, expires_at
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,0,?,1,?,?,?,?,?,?)`,
			item.ContentHash, item.Kind, item.DetectedType, item.Language, item.Title,
			nullString(item.TextContent), item.TextPreview,
			item.MimeType, item.BlobPath, item.ThumbnailPath, item.ByteSize, item.Width, item.Height, boolInt(item.ImageKept),
			item.Sensitivity, searchBlob, defaultJSON(item.MetadataJSON),
			now, now, ctxID, nullInt64(item.ExpiresAt),
		)
		if err != nil {
			return 0, false, err
		}
		existingID, err = res.LastInsertId()
		if err != nil {
			return 0, false, err
		}
		if _, err := tx.Exec(`INSERT INTO copy_events(item_id, context_id, copied_at) VALUES (?,?,?)`, existingID, ctxID, now); err != nil {
			return 0, false, err
		}
		if err := s.upsertFTS(tx, existingID, item, searchBlob); err != nil {
			return 0, false, err
		}
		if err := tx.Commit(); err != nil {
			return 0, false, err
		}
		return existingID, true, nil
	}

	// Bump existing
	expires := item.ExpiresAt
	if item.Sensitivity == "suspected" && expires != nil {
		// refresh handled by caller setting ExpiresAt
	}
	_, err = tx.Exec(`
UPDATE clipboard_items SET
  copy_count = copy_count + 1,
  last_copied_at = ?,
  last_context_id = ?,
  search_blob = ?,
  detected_type = CASE WHEN detected_type = '' OR detected_type = 'text' THEN ? ELSE detected_type END,
  language = CASE WHEN language = '' THEN ? ELSE language END,
  sensitivity = ?,
  expires_at = ?,
  text_preview = CASE WHEN text_preview = '' THEN ? ELSE text_preview END
WHERE id = ?`,
		now, ctxID, searchBlob, item.DetectedType, item.Language,
		item.Sensitivity, nullInt64(expires), item.TextPreview, existingID,
	)
	if err != nil {
		return 0, false, err
	}
	if _, err := tx.Exec(`INSERT INTO copy_events(item_id, context_id, copied_at) VALUES (?,?,?)`, existingID, ctxID, now); err != nil {
		return 0, false, err
	}
	if err := pruneEvents(tx, existingID, 200); err != nil {
		return 0, false, err
	}
	if err := s.upsertFTS(tx, existingID, item, searchBlob); err != nil {
		return 0, false, err
	}
	if err := tx.Commit(); err != nil {
		return 0, false, err
	}
	return existingID, false, nil
}

// SeedItem inserts a fully-formed demo item (used by seed).
func (s *Store) SeedItem(item domain.ClipboardItem, ctx domain.CopyContext) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	ctx.PlaceKey = placeKey(ctx)
	ctxID, err := insertContext(tx, ctx)
	if err != nil {
		return err
	}
	searchBlob := buildSearchBlob(ctx)
	res, err := tx.Exec(`
INSERT INTO clipboard_items(
  content_hash, kind, detected_type, language, title, text_content, text_preview,
  mime_type, blob_path, thumbnail_path, byte_size, width, height, image_kept,
  is_pinned, sensitivity, copy_count, search_blob, metadata_json,
  created_at, last_copied_at, last_context_id, expires_at
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		item.ContentHash, item.Kind, item.DetectedType, item.Language, item.Title,
		nullString(item.TextContent), item.TextPreview,
		item.MimeType, item.BlobPath, item.ThumbnailPath, item.ByteSize, item.Width, item.Height, boolInt(item.ImageKept),
		boolInt(item.IsPinned), item.Sensitivity, item.CopyCount, searchBlob, defaultJSON(item.MetadataJSON),
		item.CreatedAt, item.LastCopiedAt, ctxID, nullInt64(item.ExpiresAt),
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO copy_events(item_id, context_id, copied_at) VALUES (?,?,?)`, id, ctxID, item.LastCopiedAt); err != nil {
		return err
	}
	canon := &domain.CanonicalItem{
		DetectedType: item.DetectedType,
		Language:     item.Language,
		TextContent:  item.TextContent,
		TextPreview:  item.TextPreview,
	}
	if err := s.upsertFTS(tx, id, canon, searchBlob); err != nil {
		return err
	}
	return tx.Commit()
}

func insertContext(tx *sql.Tx, ctx domain.CopyContext) (int64, error) {
	if ctx.PlaceKey == "" {
		ctx.PlaceKey = placeKey(ctx)
	}
	res, err := tx.Exec(`
INSERT INTO copy_contexts(
  application_name, application_id, icon_cache_key, process_id,
  window_title, document_title, url, domain, file_name, file_path,
  place_key, deep, captured_at
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ctx.ApplicationName, ctx.ApplicationID, ctx.IconCacheKey, ctx.ProcessID,
		ctx.WindowTitle, ctx.DocumentTitle, ctx.URL, ctx.Domain, ctx.FileName, ctx.FilePath,
		ctx.PlaceKey, boolInt(ctx.Deep), ctx.CapturedAt,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func pruneEvents(tx *sql.Tx, itemID int64, keep int) error {
	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM copy_events WHERE item_id = ?`, itemID).Scan(&n); err != nil {
		return err
	}
	if n <= keep {
		return nil
	}
	_, err := tx.Exec(`
DELETE FROM copy_events WHERE id IN (
  SELECT id FROM copy_events WHERE item_id = ? ORDER BY copied_at ASC LIMIT ?
)`, itemID, n-keep)
	return err
}

func (s *Store) upsertFTS(tx *sql.Tx, id int64, item *domain.CanonicalItem, searchBlob string) error {
	if !s.fts {
		return nil
	}
	// contentless FTS5 rejects DELETE; use the special 'delete' command.
	_, _ = tx.Exec(`INSERT INTO items_fts(items_fts, rowid) VALUES('delete', ?)`, id)
	if item.Sensitivity == "suspected" {
		return nil
	}
	text := item.TextContent
	if len(text) > 256*1024 {
		text = text[:256*1024]
	}
	_, err := tx.Exec(`
INSERT INTO items_fts(rowid, title, text_preview, text_content, search_blob, detected_type, language)
VALUES (?, '', ?, ?, ?, ?, ?)`,
		id, item.TextPreview, text, searchBlob, item.DetectedType, item.Language,
	)
	return err
}

func buildSearchBlob(ctx domain.CopyContext) string {
	parts := []string{
		ctx.ApplicationName,
		ctx.ApplicationID,
		ctx.Domain,
		ctx.WindowTitle,
		ctx.DocumentTitle,
		ctx.FileName,
		ctx.URL,
	}
	var out []string
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	blob := strings.Join(out, "\n")
	if len(blob) > 8*1024 {
		blob = blob[:8*1024]
	}
	return blob
}

func placeKey(ctx domain.CopyContext) string {
	if ctx.PlaceKey != "" {
		return ctx.PlaceKey
	}
	if ctx.Domain != "" {
		return ctx.Domain
	}
	if ctx.FilePath != "" {
		return ctx.FilePath
	}
	return ctx.WindowTitle
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullInt64(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

func defaultJSON(s string) string {
	if s == "" {
		return "{}"
	}
	return s
}
