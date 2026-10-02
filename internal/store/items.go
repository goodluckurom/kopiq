package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"kopiq/internal/blob"
	"kopiq/internal/domain"
	"kopiq/internal/search"
)

// listSelect joins context in one shot and omits text_content (detail loads via Get).
// Never call getContext while these rows are open — MaxOpenConns(1) would deadlock.
const listSelect = `
SELECT i.id, i.content_hash, i.kind, i.detected_type, i.language, i.title,
  i.text_preview, i.mime_type, i.blob_path, i.thumbnail_path,
  i.byte_size, i.width, i.height, i.image_kept, i.is_pinned, i.sensitivity,
  i.copy_count, i.metadata_json, i.created_at, i.last_copied_at, i.expires_at,
  c.id, c.application_name, c.application_id, c.icon_cache_key, c.process_id,
  c.window_title, c.document_title, c.url, c.domain, c.file_name, c.file_path,
  c.place_key, c.deep, c.captured_at
FROM clipboard_items i
LEFT JOIN copy_contexts c ON c.id = i.last_context_id
`

// List returns clipboard items matching the filter, newest first.
func (s *Store) List(filter domain.HistoryFilter) ([]domain.ClipboardItem, error) {
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	var (
		args  []any
		where []string
	)
	where = append(where, `(i.expires_at IS NULL OR i.expires_at > ?)`)
	args = append(args, time.Now().UnixMilli())

	if filter.Kind != "" {
		where = append(where, `i.kind = ?`)
		args = append(args, filter.Kind)
	}
	if filter.DetectedType != "" {
		where = append(where, `i.detected_type = ?`)
		args = append(args, filter.DetectedType)
	}
	if filter.PinnedOnly {
		where = append(where, `i.is_pinned = 1`)
	}
	if filter.AppID != "" {
		where = append(where, `(c.application_id = ? OR lower(c.application_name) LIKE ?)`)
		args = append(args, filter.AppID, "%"+strings.ToLower(filter.AppID)+"%")
	}

	q := listSelect + `
WHERE ` + strings.Join(where, " AND ") + `
ORDER BY i.last_copied_at DESC
LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanListRows(rows)
}

// Get returns one item by id (includes full text_content).
func (s *Store) Get(id int64) (*domain.ClipboardItem, error) {
	if id < 0 {
		sn, err := s.GetSnippet(-id)
		if err != nil {
			return nil, fmt.Errorf("snippet %d not found", -id)
		}
		item := snippetAsItem(sn)
		return &item, nil
	}
	row := s.db.QueryRow(`
SELECT i.id, i.content_hash, i.kind, i.detected_type, i.language, i.title,
  i.text_content, i.text_preview, i.mime_type, i.blob_path, i.thumbnail_path,
  i.byte_size, i.width, i.height, i.image_kept, i.is_pinned, i.sensitivity,
  i.copy_count, i.metadata_json, i.created_at, i.last_copied_at, i.expires_at,
  c.id, c.application_name, c.application_id, c.icon_cache_key, c.process_id,
  c.window_title, c.document_title, c.url, c.domain, c.file_name, c.file_path,
  c.place_key, c.deep, c.captured_at
FROM clipboard_items i
LEFT JOIN copy_contexts c ON c.id = i.last_context_id
WHERE i.id = ?`, id)

	item, err := scanDetailRow(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("item %d not found", id)
	}
	if err != nil {
		return nil, err
	}
	maskSensitive(&item)
	return &item, nil
}

// GetRaw returns an item without masking (for CopyToClipboard).
func (s *Store) GetRaw(id int64) (*domain.ClipboardItem, error) {
	if id < 0 {
		sn, err := s.GetSnippet(-id)
		if err != nil {
			return nil, err
		}
		item := snippetAsItem(sn)
		return &item, nil
	}
	row := s.db.QueryRow(`
SELECT i.id, i.content_hash, i.kind, i.detected_type, i.language, i.title,
  i.text_content, i.text_preview, i.mime_type, i.blob_path, i.thumbnail_path,
  i.byte_size, i.width, i.height, i.image_kept, i.is_pinned, i.sensitivity,
  i.copy_count, i.metadata_json, i.created_at, i.last_copied_at, i.expires_at,
  c.id, c.application_name, c.application_id, c.icon_cache_key, c.process_id,
  c.window_title, c.document_title, c.url, c.domain, c.file_name, c.file_path,
  c.place_key, c.deep, c.captured_at
FROM clipboard_items i
LEFT JOIN copy_contexts c ON c.id = i.last_context_id
WHERE i.id = ?`, id)

	item, err := scanDetailRow(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Delete removes an item and unreferenced contexts.
func (s *Store) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var blobPath, thumbPath string
	_ = tx.QueryRow(`SELECT COALESCE(blob_path,''), COALESCE(thumbnail_path,'') FROM clipboard_items WHERE id = ?`, id).
		Scan(&blobPath, &thumbPath)

	var ctxIDs []int64
	rows, err := tx.Query(`SELECT context_id FROM copy_events WHERE item_id = ?`, id)
	if err != nil {
		return err
	}
	for rows.Next() {
		var cid int64
		if err := rows.Scan(&cid); err != nil {
			rows.Close()
			return err
		}
		ctxIDs = append(ctxIDs, cid)
	}
	rows.Close()

	if s.fts {
		_, _ = tx.Exec(`INSERT INTO items_fts(items_fts, rowid) VALUES('delete', ?)`, id)
	}
	if _, err := tx.Exec(`DELETE FROM clipboard_items WHERE id = ?`, id); err != nil {
		return err
	}
	for _, cid := range ctxIDs {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM copy_events WHERE context_id = ?`, cid).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			_, _ = tx.Exec(`DELETE FROM copy_contexts WHERE id = ?`, cid)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// Best-effort blob cleanup after commit.
	if blobPath != "" {
		blob.Remove(blobPath)
	}
	if thumbPath != "" {
		blob.Remove(thumbPath)
	}
	return nil
}

// SetThumbnailPath stores a relative thumbnail path after async generation.
func (s *Store) SetThumbnailPath(id int64, rel string) error {
	_, err := s.db.Exec(`UPDATE clipboard_items SET thumbnail_path = ? WHERE id = ?`, rel, id)
	return err
}

// SetPinned sets the pin flag.
func (s *Store) SetPinned(id int64, pinned bool) error {
	_, err := s.db.Exec(`UPDATE clipboard_items SET is_pinned = ? WHERE id = ?`, boolInt(pinned), id)
	return err
}

// SetTitle renames an item (search includes title via FTS upsert on next bump;
// we also patch FTS here when available).
func (s *Store) SetTitle(id int64, title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	title = strings.TrimSpace(title)
	_, err := s.db.Exec(`UPDATE clipboard_items SET title = ? WHERE id = ?`, title, id)
	if err != nil {
		return err
	}
	if s.fts {
		var preview, content, searchBlob, detected, language string
		_ = s.db.QueryRow(`
SELECT COALESCE(text_preview,''), COALESCE(text_content,''), COALESCE(search_blob,''),
       COALESCE(detected_type,''), COALESCE(language,'')
FROM clipboard_items WHERE id = ?`, id).Scan(&preview, &content, &searchBlob, &detected, &language)
		_, _ = s.db.Exec(`INSERT INTO items_fts(items_fts, rowid) VALUES('delete', ?)`, id)
		_, _ = s.db.Exec(`
INSERT INTO items_fts(rowid, title, text_preview, text_content, search_blob, detected_type, language)
VALUES (?,?,?,?,?,?,?)`, id, title, preview, content, searchBlob, detected, language)
	}
	return nil
}

// ClearExpiresAt keeps a suspected item (KQ-008 Keep).
func (s *Store) ClearExpiresAt(id int64) error {
	_, err := s.db.Exec(`UPDATE clipboard_items SET expires_at = NULL, sensitivity = 'none' WHERE id = ?`, id)
	return err
}

// ClearHistory deletes unpinned items (and their blobs).
func (s *Store) ClearHistory() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`SELECT id, COALESCE(blob_path,''), COALESCE(thumbnail_path,'') FROM clipboard_items WHERE is_pinned = 0`)
	if err != nil {
		return 0, err
	}
	type row struct {
		id               int64
		blobPath, thumb  string
	}
	var list []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.blobPath, &r.thumb); err != nil {
			rows.Close()
			return 0, err
		}
		list = append(list, r)
	}
	rows.Close()

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	for _, r := range list {
		if s.fts {
			_, _ = tx.Exec(`INSERT INTO items_fts(items_fts, rowid) VALUES('delete', ?)`, r.id)
		}
		if _, err := tx.Exec(`DELETE FROM clipboard_items WHERE id = ?`, r.id); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	for _, r := range list {
		blob.Remove(r.blobPath)
		blob.Remove(r.thumb)
	}
	return len(list), nil
}

// PurgeExpired deletes rows past expires_at.
func (s *Store) PurgeExpired(now int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
SELECT id, COALESCE(blob_path,''), COALESCE(thumbnail_path,'')
FROM clipboard_items WHERE expires_at IS NOT NULL AND expires_at <= ?`, now)
	if err != nil {
		return 0, err
	}
	type row struct {
		id              int64
		blobPath, thumb string
	}
	var list []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.blobPath, &r.thumb); err != nil {
			rows.Close()
			return 0, err
		}
		list = append(list, r)
	}
	rows.Close()
	if len(list) == 0 {
		return 0, nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	for _, r := range list {
		if s.fts {
			_, _ = tx.Exec(`INSERT INTO items_fts(items_fts, rowid) VALUES('delete', ?)`, r.id)
		}
		if _, err := tx.Exec(`DELETE FROM clipboard_items WHERE id = ?`, r.id); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	for _, r := range list {
		blob.Remove(r.blobPath)
		blob.Remove(r.thumb)
	}
	return len(list), nil
}

// Exclusion is a capture exclusion row.
type Exclusion struct {
	ID              int64  `json:"id"`
	ApplicationID   string `json:"applicationId"`
	ApplicationName string `json:"applicationName"`
	Source          string `json:"source"`
	Enabled         bool   `json:"enabled"`
	IconCacheKey    string `json:"iconCacheKey,omitempty"`
}

// ListExclusions returns all exclusion rows.
func (s *Store) ListExclusions() ([]Exclusion, error) {
	rows, err := s.db.Query(`
SELECT id, COALESCE(application_id,''), COALESCE(application_name,''), COALESCE(source,''), enabled
FROM exclusions ORDER BY application_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Exclusion
	for rows.Next() {
		var e Exclusion
		var en int
		if err := rows.Scan(&e.ID, &e.ApplicationID, &e.ApplicationName, &e.Source, &en); err != nil {
			return nil, err
		}
		e.Enabled = en == 1
		out = append(out, e)
	}
	return out, nil
}

// SetExclusionEnabled toggles an exclusion by id.
func (s *Store) SetExclusionEnabled(id int64, enabled bool) error {
	_, err := s.db.Exec(`UPDATE exclusions SET enabled = ? WHERE id = ?`, boolInt(enabled), id)
	return err
}

// AddExclusion inserts a user exclusion.
func (s *Store) AddExclusion(appID, appName string) error {
	_, err := s.db.Exec(`
INSERT INTO exclusions(application_id, application_name, source, enabled)
VALUES (?,?, 'user', 1)`, appID, appName)
	return err
}

// DeleteExclusion removes a user-added exclusion permanently. Builtin rows are left alone.
func (s *Store) DeleteExclusion(id int64) error {
	res, err := s.db.Exec(`DELETE FROM exclusions WHERE id = ? AND source = 'user'`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("exclusion not found or not removable")
	}
	return nil
}

// Search runs FTS/LIKE with M2/M3 operators (type:, app:, pinned:, collection:, :alias).
func (s *Store) Search(raw string) ([]domain.ClipboardItem, error) {
	q := search.Parse(raw)
	if q.Empty() {
		return s.List(domain.HistoryFilter{Limit: 100})
	}

	var head []domain.ClipboardItem
	aliasLookup := q.Alias
	if aliasLookup == "" {
		aliasLookup = q.BareAlias()
	}
	if aliasLookup != "" {
		if sn, err := s.FindSnippetByAlias(aliasLookup); err == nil && sn != nil {
			head = append(head, snippetAsItem(sn))
			// Exact :alias alone → just the snippet.
			if q.Alias != "" && len(q.Terms) == 0 && q.Collection == "" &&
				q.Type == "" && q.App == "" && q.Pinned == nil &&
				q.Lang == "" && q.Domain == "" && q.Before == nil && q.After == nil {
				return head, nil
			}
		}
	}

	now := time.Now().UnixMilli()

	where := []string{`(i.expires_at IS NULL OR i.expires_at > ?)`}
	args := []any{now}

	if q.Type != "" {
		switch q.Type {
		case "image":
			where = append(where, `(i.detected_type IN ('image','screenshot') OR i.kind = 'image')`)
		case "screenshot":
			where = append(where, `i.detected_type = 'screenshot'`)
		case "file", "files":
			where = append(where, `(i.detected_type IN ('file','files') OR i.kind = 'files')`)
		default:
			where = append(where, `lower(i.detected_type) = ?`)
			args = append(args, q.Type)
		}
	}
	if q.Pinned != nil {
		where = append(where, `i.is_pinned = ?`)
		args = append(args, boolInt(*q.Pinned))
	}
	if q.App != "" {
		where = append(where, `EXISTS (
  SELECT 1 FROM copy_events e
  JOIN copy_contexts cx ON cx.id = e.context_id
  WHERE e.item_id = i.id AND (
    lower(cx.application_name) LIKE ? OR lower(cx.application_id) LIKE ?
  )
)`)
		likeApp := "%" + q.App + "%"
		args = append(args, likeApp, likeApp)
	}
	if q.Lang != "" {
		where = append(where, `lower(COALESCE(i.language,'')) = ?`)
		args = append(args, q.Lang)
	}
	if q.Domain != "" {
		where = append(where, `EXISTS (
  SELECT 1 FROM copy_events e
  JOIN copy_contexts cx ON cx.id = e.context_id
  WHERE e.item_id = i.id AND (
    lower(cx.domain) = ? OR lower(cx.domain) LIKE ? OR lower(cx.url) LIKE ?
  )
)`)
		args = append(args, q.Domain, "%."+q.Domain, "%"+q.Domain+"%")
	}
	if q.Collection != "" {
		where = append(where, `EXISTS (
  SELECT 1 FROM collection_items ci
  JOIN collections col ON col.id = ci.collection_id
  WHERE ci.item_id = i.id AND lower(col.name) LIKE ?
)`)
		args = append(args, "%"+q.Collection+"%")
	}
	if q.Before != nil {
		where = append(where, `i.last_copied_at < ?`)
		args = append(args, *q.Before)
	}
	if q.After != nil {
		where = append(where, `i.last_copied_at >= ?`)
		args = append(args, *q.After)
	}

	ftsExpr := q.FTSMatch()
	if ftsExpr != "" && s.fts {
		where = append(where, `i.id IN (SELECT rowid FROM items_fts WHERE items_fts MATCH ?)`)
		args = append(args, ftsExpr)
		sqlText := listSelect + " WHERE " + strings.Join(where, " AND ") +
			" ORDER BY i.last_copied_at DESC LIMIT 100"
		rows, err := s.db.Query(sqlText, args...)
		if err == nil {
			defer rows.Close()
			items, err := scanListRows(rows)
			if err != nil {
				return nil, err
			}
			return mergeSearchHead(head, items), nil
		}
		// fall through without FTS clause
		where = where[:len(where)-1]
		args = args[:len(args)-1]
	}

	if len(q.Terms) > 0 {
		for _, term := range q.Terms {
			term = strings.TrimSpace(term)
			if term == "" {
				continue
			}
			like := "%" + term + "%"
			where = append(where, `(
  i.text_preview LIKE ? OR i.title LIKE ? OR i.search_blob LIKE ?
  OR i.detected_type LIKE ? OR c.application_name LIKE ?
  OR c.domain LIKE ? OR c.window_title LIKE ?
)`)
			args = append(args, like, like, like, like, like, like, like)
		}
	}

	sqlText := listSelect + " WHERE " + strings.Join(where, " AND ") +
		" ORDER BY i.last_copied_at DESC LIMIT 100"
	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanListRows(rows)
	if err != nil {
		return nil, err
	}
	return mergeSearchHead(head, items), nil
}

func snippetAsItem(sn *domain.Snippet) domain.ClipboardItem {
	preview := sn.Body
	if len(preview) > 200 {
		preview = preview[:200]
	}
	title := sn.Title
	if title == "" {
		title = "Snippet"
	}
	det := "snippet"
	if sn.Language != "" {
		det = sn.Language
	}
	return domain.ClipboardItem{
		ID:           -sn.ID, // negative: paste/copy resolve via snippets table
		Kind:         "text",
		DetectedType: det,
		Language:     sn.Language,
		Title:        title,
		TextContent:  sn.Body,
		TextPreview:  preview,
		ByteSize:     int64(len(sn.Body)),
		ImageKept:    true,
		Sensitivity:  "none",
		CopyCount:    1,
		CreatedAt:    sn.CreatedAt,
		LastCopiedAt: sn.UpdatedAt,
		MetadataJSON: `{"snippet":true}`,
	}
}

func mergeSearchHead(head, rest []domain.ClipboardItem) []domain.ClipboardItem {
	if len(head) == 0 {
		return rest
	}
	seen := map[int64]bool{}
	for _, h := range head {
		seen[h.ID] = true
	}
	out := append([]domain.ClipboardItem{}, head...)
	for _, it := range rest {
		if seen[it.ID] {
			continue
		}
		out = append(out, it)
	}
	return out
}

// Counts returns category / source tallies for the sidebar.
func (s *Store) Counts() (domain.HistoryCounts, error) {
	now := time.Now().UnixMilli()
	var out domain.HistoryCounts
	out.ByCategory = map[string]int{}
	out.ByApp = []domain.AppCount{}

	_ = s.db.QueryRow(`
SELECT COUNT(*) FROM clipboard_items
WHERE expires_at IS NULL OR expires_at > ?`, now).Scan(&out.Total)

	_ = s.db.QueryRow(`
SELECT COUNT(*) FROM clipboard_items
WHERE is_pinned = 1 AND (expires_at IS NULL OR expires_at > ?)`, now).Scan(&out.Pinned)

	out.Snippets, _ = s.countSnippets()
	out.Collections, _ = s.countCollections()
	out.Stacks, _ = s.countStacks()

	rows, err := s.db.Query(`
SELECT detected_type, kind, COUNT(*) FROM clipboard_items
WHERE expires_at IS NULL OR expires_at > ?
GROUP BY detected_type, kind`, now)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var detected, kind string
		var n int
		if err := rows.Scan(&detected, &kind, &n); err != nil {
			return out, err
		}
		out.ByCategory["all"] += n
		switch {
		case kind == "image" || detected == "image" || detected == "screenshot":
			out.ByCategory["images"] += n
		case detected == "url":
			out.ByCategory["links"] += n
		case detected == "code" || detected == "json" || detected == "sql" || detected == "shell" || detected == "jwt":
			out.ByCategory["code"] += n
		case kind == "files" || detected == "file" || detected == "files":
			out.ByCategory["files"] += n
		default:
			out.ByCategory["text"] += n
		}
	}
	out.ByCategory["pinned"] = out.Pinned

	appRows, err := s.db.Query(`
SELECT COALESCE(NULLIF(c.application_name, ''), 'Unknown') AS name,
       COALESCE(c.application_id, '') AS app_id,
       COUNT(*) AS n,
       COALESCE(MAX(NULLIF(c.icon_cache_key, '')), '') AS icon_key
FROM clipboard_items i
LEFT JOIN copy_contexts c ON c.id = i.last_context_id
WHERE i.expires_at IS NULL OR i.expires_at > ?
GROUP BY name, app_id
ORDER BY n DESC
LIMIT 12`, now)
	if err != nil {
		return out, err
	}
	defer appRows.Close()
	for appRows.Next() {
		var a domain.AppCount
		if err := appRows.Scan(&a.Name, &a.AppID, &a.Count, &a.IconCacheKey); err != nil {
			return out, err
		}
		out.ByApp = append(out.ByApp, a)
	}
	return out, nil
}

func scanListRows(rows *sql.Rows) ([]domain.ClipboardItem, error) {
	var items []domain.ClipboardItem
	for rows.Next() {
		item, err := scanListRow(rows)
		if err != nil {
			return nil, err
		}
		maskSensitive(&item)
		items = append(items, item)
	}
	return items, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanListRow(row scannable) (domain.ClipboardItem, error) {
	var (
		item                                  domain.ClipboardItem
		exp                                   sql.NullInt64
		kept, pinned                          int
		ctxID                                 sql.NullInt64
		appName, appID, iconKey               sql.NullString
		pid                                   sql.NullInt64
		win, doc, u, domainName, file, path   sql.NullString
		place                                 sql.NullString
		deep                                  sql.NullInt64
		captured                              sql.NullInt64
	)
	err := row.Scan(
		&item.ID, &item.ContentHash, &item.Kind, &item.DetectedType, &item.Language, &item.Title,
		&item.TextPreview, &item.MimeType, &item.BlobPath, &item.ThumbnailPath,
		&item.ByteSize, &item.Width, &item.Height, &kept, &pinned, &item.Sensitivity,
		&item.CopyCount, &item.MetadataJSON, &item.CreatedAt, &item.LastCopiedAt, &exp,
		&ctxID, &appName, &appID, &iconKey, &pid,
		&win, &doc, &u, &domainName, &file, &path,
		&place, &deep, &captured,
	)
	if err != nil {
		return item, err
	}
	item.ImageKept = kept == 1
	item.IsPinned = pinned == 1
	if exp.Valid {
		v := exp.Int64
		item.ExpiresAt = &v
	}
	if ctxID.Valid {
		ctx := domain.CopyContext{
			ID:              ctxID.Int64,
			ApplicationName: appName.String,
			ApplicationID:   appID.String,
			IconCacheKey:    iconKey.String,
			ProcessID:       int(pid.Int64),
			WindowTitle:     win.String,
			DocumentTitle:   doc.String,
			URL:             u.String,
			Domain:          domainName.String,
			FileName:        file.String,
			FilePath:        path.String,
			PlaceKey:        place.String,
			Deep:            deep.Int64 == 1,
			CapturedAt:      captured.Int64,
		}
		item.LastContext = &ctx
	}
	return item, nil
}

func scanDetailRow(row scannable) (domain.ClipboardItem, error) {
	var (
		item                                  domain.ClipboardItem
		text                                  sql.NullString
		exp                                   sql.NullInt64
		kept, pinned                          int
		ctxID                                 sql.NullInt64
		appName, appID, iconKey               sql.NullString
		pid                                   sql.NullInt64
		win, doc, u, domainName, file, path   sql.NullString
		place                                 sql.NullString
		deep                                  sql.NullInt64
		captured                              sql.NullInt64
	)
	err := row.Scan(
		&item.ID, &item.ContentHash, &item.Kind, &item.DetectedType, &item.Language, &item.Title,
		&text, &item.TextPreview, &item.MimeType, &item.BlobPath, &item.ThumbnailPath,
		&item.ByteSize, &item.Width, &item.Height, &kept, &pinned, &item.Sensitivity,
		&item.CopyCount, &item.MetadataJSON, &item.CreatedAt, &item.LastCopiedAt, &exp,
		&ctxID, &appName, &appID, &iconKey, &pid,
		&win, &doc, &u, &domainName, &file, &path,
		&place, &deep, &captured,
	)
	if err != nil {
		return item, err
	}
	if text.Valid {
		item.TextContent = text.String
	}
	item.ImageKept = kept == 1
	item.IsPinned = pinned == 1
	if exp.Valid {
		v := exp.Int64
		item.ExpiresAt = &v
	}
	if ctxID.Valid {
		ctx := domain.CopyContext{
			ID:              ctxID.Int64,
			ApplicationName: appName.String,
			ApplicationID:   appID.String,
			IconCacheKey:    iconKey.String,
			ProcessID:       int(pid.Int64),
			WindowTitle:     win.String,
			DocumentTitle:   doc.String,
			URL:             u.String,
			Domain:          domainName.String,
			FileName:        file.String,
			FilePath:        path.String,
			PlaceKey:        place.String,
			Deep:            deep.Int64 == 1,
			CapturedAt:      captured.Int64,
		}
		item.LastContext = &ctx
	}
	return item, nil
}

func maskSensitive(item *domain.ClipboardItem) {
	if item.Sensitivity != "suspected" {
		return
	}
	item.Masked = true
	item.TextContent = ""
	item.TextPreview = "••••••••"
}
