package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"kopiq/internal/appdata"
	"kopiq/internal/blob"
)

// RetentionConfig drives ApplyRetention.
type RetentionConfig struct {
	TextCap          int
	ImageDays        int
	ImageBudgetBytes int64
	Now              int64
}

// RetentionResult summarizes a cleanup pass.
type RetentionResult struct {
	Removed    int   `json:"removed"`
	BytesFreed int64 `json:"bytesFreed"`
}

// StorageBreakdown is the Settings → Storage screen.
type StorageBreakdown struct {
	TextBytes     int64 `json:"textBytes"`
	ImageBytes    int64 `json:"imageBytes"`
	FileBytes     int64 `json:"fileBytes"`
	DatabaseBytes int64 `json:"databaseBytes"`
	TotalBytes    int64 `json:"totalBytes"`
	ItemCount     int   `json:"itemCount"`
	PinnedCount   int   `json:"pinnedCount"`
}

// ApplyRetention removes aged/capped unpinned items and enforces the image budget.
// Pinned items are never chosen. Suspected expiry is handled by PurgeExpired.
func (s *Store) ApplyRetention(cfg RetentionConfig) (RetentionResult, error) {
	if cfg.Now == 0 {
		cfg.Now = time.Now().UnixMilli()
	}
	var res RetentionResult

	// 1) Image age: unpinned images older than ImageDays.
	if cfg.ImageDays > 0 {
		cutoff := cfg.Now - int64(cfg.ImageDays)*24*60*60*1000
		n, freed, err := s.deleteMatching(`
SELECT id, COALESCE(blob_path,''), COALESCE(thumbnail_path,''), COALESCE(byte_size,0)
FROM clipboard_items
WHERE is_pinned = 0
  AND (kind = 'image' OR detected_type IN ('image','screenshot'))
  AND last_copied_at < ?
LIMIT 400`, cutoff)
		if err != nil {
			return res, err
		}
		res.Removed += n
		res.BytesFreed += freed
	}

	// 2) Text entry cap: oldest unpinned non-image over TextCap.
	if cfg.TextCap > 0 {
		var textCount int
		_ = s.db.QueryRow(`
SELECT COUNT(*) FROM clipboard_items
WHERE is_pinned = 0
  AND kind != 'image'
  AND detected_type NOT IN ('image','screenshot')`).Scan(&textCount)
		overflow := textCount - cfg.TextCap
		for overflow > 0 {
			batch := overflow
			if batch > 200 {
				batch = 200
			}
			n, freed, err := s.deleteMatching(`
SELECT id, COALESCE(blob_path,''), COALESCE(thumbnail_path,''), COALESCE(byte_size,0)
FROM clipboard_items
WHERE is_pinned = 0
  AND kind != 'image'
  AND detected_type NOT IN ('image','screenshot')
ORDER BY last_copied_at ASC
LIMIT ?`, batch)
			if err != nil {
				return res, err
			}
			if n == 0 {
				break
			}
			res.Removed += n
			res.BytesFreed += freed
			overflow -= n
		}
	}

	// 3) Image disk budget: delete oldest unpinned image blobs until under budget.
	if cfg.ImageBudgetBytes > 0 {
		for {
			var used int64
			_ = s.db.QueryRow(`
SELECT COALESCE(SUM(byte_size),0) FROM clipboard_items
WHERE kind = 'image' OR detected_type IN ('image','screenshot')`).Scan(&used)
			if used <= cfg.ImageBudgetBytes {
				break
			}
			n, freed, err := s.deleteMatching(`
SELECT id, COALESCE(blob_path,''), COALESCE(thumbnail_path,''), COALESCE(byte_size,0)
FROM clipboard_items
WHERE is_pinned = 0
  AND (kind = 'image' OR detected_type IN ('image','screenshot'))
ORDER BY last_copied_at ASC
LIMIT 50`)
			if err != nil {
				return res, err
			}
			if n == 0 {
				break
			}
			res.Removed += n
			res.BytesFreed += freed
		}
	}

	// 4) Orphan blob files (best effort).
	orphans, err := s.removeOrphanBlobs()
	if err == nil {
		res.BytesFreed += orphans
	}

	return res, nil
}

func (s *Store) deleteMatching(query string, args ...any) (int, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return 0, 0, err
	}
	type victim struct {
		id, bytes          int64
		blobPath, thumbPath string
	}
	var list []victim
	for rows.Next() {
		var v victim
		if err := rows.Scan(&v.id, &v.blobPath, &v.thumbPath, &v.bytes); err != nil {
			rows.Close()
			return 0, 0, err
		}
		list = append(list, v)
	}
	rows.Close()
	if len(list) == 0 {
		return 0, 0, nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var freed int64
	for _, v := range list {
		if s.fts {
			_, _ = tx.Exec(`INSERT INTO items_fts(items_fts, rowid) VALUES('delete', ?)`, v.id)
		}
		if _, err := tx.Exec(`DELETE FROM clipboard_items WHERE id = ? AND is_pinned = 0`, v.id); err != nil {
			return 0, 0, err
		}
		freed += v.bytes
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	for _, v := range list {
		blob.Remove(v.blobPath)
		blob.Remove(v.thumbPath)
	}
	return len(list), freed, nil
}

func (s *Store) removeOrphanBlobs() (int64, error) {
	root, err := appdata.BlobRoot()
	if err != nil {
		return 0, err
	}
	known := map[string]struct{}{}
	rows, err := s.db.Query(`SELECT COALESCE(blob_path,''), COALESCE(thumbnail_path,'') FROM clipboard_items`)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var a, b string
		if err := rows.Scan(&a, &b); err != nil {
			rows.Close()
			return 0, err
		}
		if a != "" {
			known[a] = struct{}{}
		}
		if b != "" {
			known[b] = struct{}{}
		}
	}
	rows.Close()

	var freed int64
	for _, sub := range []string{blob.OriginalPrefix, blob.ThumbPrefix} {
		dir := filepath.Join(root, sub)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			rel := filepath.ToSlash(filepath.Join(sub, e.Name()))
			if _, ok := known[rel]; ok {
				continue
			}
			info, err := e.Info()
			if err == nil {
				freed += info.Size()
			}
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
	return freed, nil
}

// StorageStats returns disk usage for the Settings storage screen.
func (s *Store) StorageStats() (StorageBreakdown, error) {
	var out StorageBreakdown
	now := time.Now().UnixMilli()
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM clipboard_items WHERE expires_at IS NULL OR expires_at > ?`, now).Scan(&out.ItemCount)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM clipboard_items WHERE is_pinned = 1 AND (expires_at IS NULL OR expires_at > ?)`, now).Scan(&out.PinnedCount)

	_ = s.db.QueryRow(`
SELECT COALESCE(SUM(byte_size),0) FROM clipboard_items
WHERE kind != 'image' AND detected_type NOT IN ('image','screenshot')`).Scan(&out.TextBytes)

	_ = s.db.QueryRow(`
SELECT COALESCE(SUM(byte_size),0) FROM clipboard_items
WHERE kind = 'image' OR detected_type IN ('image','screenshot')`).Scan(&out.ImageBytes)

	_ = s.db.QueryRow(`
SELECT COALESCE(SUM(byte_size),0) FROM clipboard_items
WHERE kind = 'files' OR detected_type IN ('file','files')`).Scan(&out.FileBytes)

	if p, err := appdata.DBPath(); err == nil {
		if st, err := os.Stat(p); err == nil {
			out.DatabaseBytes = st.Size()
		}
		for _, suf := range []string{"-wal", "-shm"} {
			if st, err := os.Stat(p + suf); err == nil {
				out.DatabaseBytes += st.Size()
			}
		}
	}
	out.TotalBytes = out.TextBytes + out.ImageBytes + out.FileBytes + out.DatabaseBytes
	return out, nil
}

// ClearDeepFields strips window titles / URLs / file paths from all contexts
// and rebuilds search blobs. Application identity remains.
func (s *Store) ClearDeepFields() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(`
UPDATE copy_contexts SET
  window_title = '',
  document_title = '',
  url = '',
  domain = '',
  file_name = '',
  file_path = '',
  place_key = '',
  deep = 0
WHERE deep = 1
   OR COALESCE(window_title,'') != ''
   OR COALESCE(document_title,'') != ''
   OR COALESCE(url,'') != ''
   OR COALESCE(file_path,'') != ''`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()

	// Rebuild search_blob for items from remaining context fields.
	rows, err := s.db.Query(`
SELECT i.id, i.title, i.text_preview, i.detected_type, i.language, i.text_content,
  c.application_name, c.application_id
FROM clipboard_items i
LEFT JOIN copy_contexts c ON c.id = i.last_context_id`)
	if err != nil {
		return int(n), nil
	}
	defer rows.Close()
	type row struct {
		id                                 int64
		title, preview, content, dtype, lang, blob string
	}
	var updates []row
	for rows.Next() {
		var r row
		var app, appID sql.NullString
		var content sql.NullString
		if err := rows.Scan(&r.id, &r.title, &r.preview, &r.dtype, &r.lang, &content, &app, &appID); err != nil {
			continue
		}
		r.content = content.String
		r.blob = joinNonEmpty([]string{
			r.title, r.preview, r.dtype, r.lang, app.String, appID.String,
		})
		updates = append(updates, r)
	}
	for _, u := range updates {
		_, _ = s.db.Exec(`UPDATE clipboard_items SET search_blob = ? WHERE id = ?`, u.blob, u.id)
		if s.fts {
			_, _ = s.db.Exec(`INSERT INTO items_fts(items_fts, rowid) VALUES('delete', ?)`, u.id)
			_, _ = s.db.Exec(`INSERT INTO items_fts(rowid, title, text_preview, text_content, search_blob, detected_type, language) VALUES (?,?,?,?,?,?,?)`,
				u.id, u.title, u.preview, u.content, u.blob, u.dtype, u.lang)
		}
	}
	return int(n), nil
}

func joinNonEmpty(parts []string) string {
	out := ""
	for _, p := range parts {
		p = trimSpace(p)
		if p == "" {
			continue
		}
		if out != "" {
			out += " "
		}
		out += p
	}
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\n') {
		s = s[:len(s)-1]
	}
	return s
}
