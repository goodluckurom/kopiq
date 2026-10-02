package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"kopiq/internal/domain"
)

// ListCollections returns all collections with item counts.
func (s *Store) ListCollections() ([]domain.Collection, error) {
	rows, err := s.db.Query(`
SELECT c.id, c.name, c.created_at, c.sort_order,
       (SELECT COUNT(*) FROM collection_items ci WHERE ci.collection_id = c.id) AS n
FROM collections c
ORDER BY c.sort_order ASC, c.name COLLATE NOCASE ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Collection
	for rows.Next() {
		var c domain.Collection
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt, &c.SortOrder, &c.ItemCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateCollection inserts a named collection.
func (s *Store) CreateCollection(name string) (domain.Collection, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Collection{}, fmt.Errorf("name required")
	}
	now := time.Now().UnixMilli()
	var maxOrder int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) FROM collections`).Scan(&maxOrder)
	res, err := s.db.Exec(
		`INSERT INTO collections(name, created_at, sort_order) VALUES (?,?,?)`,
		name, now, maxOrder+1,
	)
	if err != nil {
		return domain.Collection{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Collection{}, err
	}
	return domain.Collection{ID: id, Name: name, CreatedAt: now, SortOrder: maxOrder + 1}, nil
}

// RenameCollection updates a collection name.
func (s *Store) RenameCollection(id int64, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name required")
	}
	res, err := s.db.Exec(`UPDATE collections SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("collection not found")
	}
	return nil
}

// DeleteCollection removes a collection but keeps clipboard items.
func (s *Store) DeleteCollection(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM collection_items WHERE collection_id = ?`, id); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM collections WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("collection not found")
	}
	return tx.Commit()
}

// AddToCollection links an item into a collection.
func (s *Store) AddToCollection(collectionID, itemID int64) error {
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO collection_items(collection_id, item_id, added_at) VALUES (?,?,?)`,
		collectionID, itemID, now,
	)
	return err
}

// RemoveFromCollection unlinks an item.
func (s *Store) RemoveFromCollection(collectionID, itemID int64) error {
	_, err := s.db.Exec(
		`DELETE FROM collection_items WHERE collection_id = ? AND item_id = ?`,
		collectionID, itemID,
	)
	return err
}

// ListCollectionItems returns clipboard items in a collection (newest first).
func (s *Store) ListCollectionItems(collectionID int64, limit int) ([]domain.ClipboardItem, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(`
SELECT i.id FROM collection_items ci
JOIN clipboard_items i ON i.id = ci.item_id
WHERE ci.collection_id = ?
  AND (i.expires_at IS NULL OR i.expires_at > ?)
ORDER BY ci.added_at DESC
LIMIT ?`, collectionID, time.Now().UnixMilli(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]domain.ClipboardItem, 0, len(ids))
	for _, id := range ids {
		item, err := s.Get(id)
		if err != nil {
			continue
		}
		out = append(out, *item)
	}
	return out, nil
}

// ListSnippets returns snippets newest-updated first.
func (s *Store) ListSnippets(limit int) ([]domain.Snippet, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(`
SELECT id, source_item_id, title, body, language, created_at, updated_at
FROM snippets
ORDER BY updated_at DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Snippet
	for rows.Next() {
		var sn domain.Snippet
		var src sql.NullInt64
		if err := rows.Scan(&sn.ID, &src, &sn.Title, &sn.Body, &sn.Language, &sn.CreatedAt, &sn.UpdatedAt); err != nil {
			return nil, err
		}
		if src.Valid {
			v := src.Int64
			sn.SourceItemID = &v
		}
		aliases, _ := s.snippetAliases(sn.ID)
		sn.Aliases = aliases
		out = append(out, sn)
	}
	return out, rows.Err()
}

func (s *Store) snippetAliases(snippetID int64) ([]string, error) {
	rows, err := s.db.Query(`SELECT alias FROM snippet_aliases WHERE snippet_id = ? ORDER BY alias`, snippetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CreateSnippetFromItem copies an item's body into a new snippet.
func (s *Store) CreateSnippetFromItem(itemID int64, title, alias string) (domain.Snippet, error) {
	item, err := s.GetRaw(itemID)
	if err != nil {
		return domain.Snippet{}, err
	}
	body := item.TextContent
	if body == "" {
		body = item.TextPreview
	}
	if strings.TrimSpace(body) == "" {
		return domain.Snippet{}, fmt.Errorf("item has no text body")
	}
	if title == "" {
		title = item.Title
	}
	if title == "" {
		line := strings.Split(body, "\n")[0]
		if len(line) > 60 {
			line = line[:60]
		}
		title = line
	}
	return s.CreateSnippet(&itemID, title, body, item.Language, alias)
}

// CreateSnippet inserts a snippet with an optional alias (without leading colon).
func (s *Store) CreateSnippet(sourceItemID *int64, title, body, language, alias string) (domain.Snippet, error) {
	body = strings.TrimRight(body, "\n")
	if strings.TrimSpace(body) == "" {
		return domain.Snippet{}, fmt.Errorf("body required")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Snippet"
	}
	now := time.Now().UnixMilli()
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return domain.Snippet{}, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(
		`INSERT INTO snippets(source_item_id, title, body, language, created_at, updated_at) VALUES (?,?,?,?,?,?)`,
		nullInt64(sourceItemID), title, body, language, now, now,
	)
	if err != nil {
		return domain.Snippet{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Snippet{}, err
	}
	alias = normalizeAlias(alias)
	if alias != "" {
		if _, err := tx.Exec(`INSERT INTO snippet_aliases(snippet_id, alias) VALUES (?,?)`, id, alias); err != nil {
			return domain.Snippet{}, fmt.Errorf("alias taken or invalid: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.Snippet{}, err
	}
	sn := domain.Snippet{
		ID: id, SourceItemID: sourceItemID, Title: title, Body: body,
		Language: language, CreatedAt: now, UpdatedAt: now,
	}
	if alias != "" {
		sn.Aliases = []string{alias}
	}
	return sn, nil
}

// DeleteSnippet removes a snippet and its aliases.
func (s *Store) DeleteSnippet(id int64) error {
	_, err := s.db.Exec(`DELETE FROM snippets WHERE id = ?`, id)
	return err
}

// SetSnippetAlias adds or replaces the primary alias for a snippet.
func (s *Store) SetSnippetAlias(snippetID int64, alias string) error {
	alias = normalizeAlias(alias)
	if alias == "" {
		_, err := s.db.Exec(`DELETE FROM snippet_aliases WHERE snippet_id = ?`, snippetID)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM snippet_aliases WHERE snippet_id = ?`, snippetID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO snippet_aliases(snippet_id, alias) VALUES (?,?)`, snippetID, alias); err != nil {
		return err
	}
	return tx.Commit()
}

// FindSnippetByAlias returns a snippet for an alias like "go-handler" or ":go-handler".
func (s *Store) FindSnippetByAlias(alias string) (*domain.Snippet, error) {
	alias = normalizeAlias(alias)
	if alias == "" {
		return nil, sql.ErrNoRows
	}
	var id int64
	err := s.db.QueryRow(`SELECT snippet_id FROM snippet_aliases WHERE alias = ?`, alias).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetSnippet(id)
}

// GetSnippet loads one snippet.
func (s *Store) GetSnippet(id int64) (*domain.Snippet, error) {
	var sn domain.Snippet
	var src sql.NullInt64
	err := s.db.QueryRow(`
SELECT id, source_item_id, title, body, language, created_at, updated_at
FROM snippets WHERE id = ?`, id).Scan(
		&sn.ID, &src, &sn.Title, &sn.Body, &sn.Language, &sn.CreatedAt, &sn.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if src.Valid {
		v := src.Int64
		sn.SourceItemID = &v
	}
	sn.Aliases, _ = s.snippetAliases(id)
	return &sn, nil
}

func normalizeAlias(alias string) string {
	alias = strings.TrimSpace(alias)
	alias = strings.TrimPrefix(alias, ":")
	alias = strings.ToLower(alias)
	alias = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, alias)
	return alias
}

func (s *Store) countSnippets() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM snippets`).Scan(&n)
	return n, err
}

func (s *Store) countCollections() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM collections`).Scan(&n)
	return n, err
}
