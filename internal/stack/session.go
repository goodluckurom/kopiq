package stack

import (
	"fmt"
	"sync"

	"kopiq/internal/domain"
)

// Step is one paste in an armed stack.
type Step struct {
	ItemID int64  `json:"itemId,omitempty"`
	Text   string `json:"text"`
	Label  string `json:"label,omitempty"`
}

// Status is shown in the tray / Quick Paste while a stack is armed.
type Status struct {
	Armed   bool   `json:"armed"`
	Index   int    `json:"index"`   // 0-based next step
	Total   int    `json:"total"`
	Label   string `json:"label"`   // "Stack 2 of 4"
	NextPreview string `json:"nextPreview,omitempty"`
}

// Session holds the currently armed stack (one at a time).
type Session struct {
	mu    sync.Mutex
	steps []Step
	index int
	armed bool
}

// Arm replaces any armed stack with these steps.
func (s *Session) Arm(steps []Step) error {
	if len(steps) == 0 {
		return fmt.Errorf("stack is empty")
	}
	for i, st := range steps {
		if st.Text == "" {
			return fmt.Errorf("step %d is missing text", i+1)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.steps = append([]Step{}, steps...)
	s.index = 0
	s.armed = true
	return nil
}

// Disarm clears the armed stack without deleting a saved definition.
func (s *Session) Disarm() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.armed = false
	s.index = 0
	s.steps = nil
}

// Armed reports whether paste-back should consume the stack.
func (s *Session) Armed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.armed
}

// Status returns the UI status line.
func (s *Session) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.armed || len(s.steps) == 0 {
		return Status{}
	}
	st := Status{
		Armed: true,
		Index: s.index,
		Total: len(s.steps),
		Label: fmt.Sprintf("Stack %d of %d", s.index+1, len(s.steps)),
	}
	if s.index < len(s.steps) {
		prev := s.steps[s.index].Text
		if len(prev) > 48 {
			prev = prev[:48] + "…"
		}
		st.NextPreview = prev
	}
	return st
}

// NextText returns the next step text without advancing.
// Missing empty text blocks (does not skip).
func (s *Session) NextText() (text string, done bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.armed || len(s.steps) == 0 {
		return "", true, nil
	}
	if s.index >= len(s.steps) {
		s.armed = false
		return "", true, nil
	}
	st := s.steps[s.index]
	if st.Text == "" {
		return "", false, fmt.Errorf("stack step %d is missing — stopped", s.index+1)
	}
	return st.Text, false, nil
}

// Advance moves to the next step after a successful paste. Disarms after the last.
func (s *Session) Advance() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.armed {
		return Status{}
	}
	s.index++
	if s.index >= len(s.steps) {
		s.armed = false
		s.index = 0
		s.steps = nil
		return Status{}
	}
	return s.StatusUnlocked()
}

func (s *Session) StatusUnlocked() Status {
	if !s.armed || len(s.steps) == 0 {
		return Status{}
	}
	st := Status{
		Armed: true,
		Index: s.index,
		Total: len(s.steps),
		Label: fmt.Sprintf("Stack %d of %d", s.index+1, len(s.steps)),
	}
	if s.index < len(s.steps) {
		prev := s.steps[s.index].Text
		if len(prev) > 48 {
			prev = prev[:48] + "…"
		}
		st.NextPreview = prev
	}
	return st
}

// StepsFromItems builds stack steps from clipboard items (text only).
func StepsFromItems(items []domain.ClipboardItem) ([]Step, error) {
	var steps []Step
	for _, it := range items {
		if it.Kind == "image" || it.DetectedType == "image" || it.DetectedType == "screenshot" {
			continue
		}
		text := it.TextContent
		if text == "" {
			text = it.TextPreview
		}
		if text == "" {
			return nil, fmt.Errorf("item %d has no text — stack blocked", it.ID)
		}
		label := it.Title
		if label == "" {
			label = it.TextPreview
			if len(label) > 40 {
				label = label[:40]
			}
		}
		steps = append(steps, Step{ItemID: it.ID, Text: text, Label: label})
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("no text items to stack")
	}
	return steps, nil
}
