package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"kopiq/internal/domain"
	"kopiq/internal/stack"
)

// CreateStack inserts a named stack with ordered steps.
func (s *Store) CreateStack(name string, steps []stack.Step) (domain.Stack, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Stack{}, fmt.Errorf("name required")
	}
	if len(steps) == 0 {
		return domain.Stack{}, fmt.Errorf("stack is empty")
	}

	now := time.Now().UnixMilli()
	tx, err := s.db.Begin()
	if err != nil {
		return domain.Stack{}, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(`INSERT INTO stacks(name, created_at) VALUES (?, ?)`, name, now)
	if err != nil {
		return domain.Stack{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Stack{}, err
	}
	for i, st := range steps {
		var itemID any
		if st.ItemID > 0 {
			itemID = st.ItemID
		}
		if _, err := tx.Exec(
			`INSERT INTO stack_items(stack_id, position, item_id, inline_text, label) VALUES (?, ?, ?, ?, ?)`,
			id, i, itemID, st.Text, st.Label,
		); err != nil {
			return domain.Stack{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.Stack{}, err
	}
	return domain.Stack{ID: id, Name: name, CreatedAt: now, StepCount: len(steps)}, nil
}

// ListStacks returns saved stacks newest first.
func (s *Store) ListStacks() ([]domain.Stack, error) {
	rows, err := s.db.Query(`
SELECT s.id, s.name, s.created_at,
  (SELECT COUNT(*) FROM stack_items si WHERE si.stack_id = s.id)
FROM stacks s
ORDER BY s.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Stack
	for rows.Next() {
		var st domain.Stack
		if err := rows.Scan(&st.ID, &st.Name, &st.CreatedAt, &st.StepCount); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// DeleteStack removes a stack definition.
func (s *Store) DeleteStack(id int64) error {
	_, err := s.db.Exec(`DELETE FROM stacks WHERE id = ?`, id)
	return err
}

// LoadStackSteps returns persisted steps for a stack.
func (s *Store) LoadStackSteps(id int64) ([]domain.StackStepRow, error) {
	rows, err := s.db.Query(`
SELECT position, item_id, inline_text, label
FROM stack_items WHERE stack_id = ? ORDER BY position ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.StackStepRow
	for rows.Next() {
		var row domain.StackStepRow
		var itemID sql.NullInt64
		if err := rows.Scan(&row.Position, &itemID, &row.InlineText, &row.Label); err != nil {
			return nil, err
		}
		if itemID.Valid {
			v := itemID.Int64
			row.ItemID = &v
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ResolveStackSteps builds live stack.Step values; missing history items block.
func (s *Store) ResolveStackSteps(id int64) ([]stack.Step, error) {
	rows, err := s.LoadStackSteps(id)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("stack is empty")
	}
	steps := make([]stack.Step, 0, len(rows))
	for _, row := range rows {
		text := row.InlineText
		label := row.Label
		var itemID int64
		if row.ItemID != nil {
			itemID = *row.ItemID
			it, err := s.GetRaw(itemID)
			if err != nil || it == nil {
				return nil, fmt.Errorf("stack step %d is missing — stopped", row.Position+1)
			}
			text = it.TextContent
			if text == "" {
				text = it.TextPreview
			}
			if text == "" {
				return nil, fmt.Errorf("stack step %d is missing text — stopped", row.Position+1)
			}
			if label == "" {
				label = it.Title
			}
		} else if text == "" {
			return nil, fmt.Errorf("stack step %d is missing — stopped", row.Position+1)
		}
		steps = append(steps, stack.Step{ItemID: itemID, Text: text, Label: label})
	}
	return steps, nil
}

func (s *Store) countStacks() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM stacks`).Scan(&n)
	return n, err
}
