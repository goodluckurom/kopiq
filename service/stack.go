package service

import (
	"fmt"
	"time"

	"kopiq/internal/domain"
	"kopiq/internal/paste"
	"kopiq/internal/stack"
	"kopiq/internal/store"
)

// StackService arms sequential paste-back (KQ-018).
type StackService struct {
	Store   *store.Store
	Session *stack.Session
}

func (s *StackService) session() *stack.Session {
	if s.Session == nil {
		s.Session = &stack.Session{}
	}
	return s.Session
}

// StartFromItems arms a stack from history item IDs (order preserved) and saves it.
func (s *StackService) StartFromItems(ids []int64) (stack.Status, error) {
	items := make([]domain.ClipboardItem, 0, len(ids))
	for _, id := range ids {
		it, err := s.Store.GetRaw(id)
		if err != nil || it == nil {
			return stack.Status{}, fmt.Errorf("item %d missing — stack blocked", id)
		}
		items = append(items, *it)
	}
	steps, err := stack.StepsFromItems(items)
	if err != nil {
		return stack.Status{}, err
	}
	name := fmt.Sprintf("Stack · %d items", len(steps))
	if len(steps) == 1 && steps[0].Label != "" {
		name = steps[0].Label
	}
	if _, err := s.Store.CreateStack(name, steps); err != nil {
		return stack.Status{}, err
	}
	if err := s.session().Arm(steps); err != nil {
		return stack.Status{}, err
	}
	return s.session().Status(), nil
}

// List returns saved stacks.
func (s *StackService) List() ([]domain.Stack, error) {
	return s.Store.ListStacks()
}

// Start arms a previously saved stack. Missing steps block.
func (s *StackService) Start(id int64) (stack.Status, error) {
	steps, err := s.Store.ResolveStackSteps(id)
	if err != nil {
		return stack.Status{}, err
	}
	if err := s.session().Arm(steps); err != nil {
		return stack.Status{}, err
	}
	return s.session().Status(), nil
}

// Delete removes a saved stack definition.
func (s *StackService) Delete(id int64) error {
	return s.Store.DeleteStack(id)
}

// Stop disarms the stack.
func (s *StackService) Stop() {
	s.session().Disarm()
}

// Status returns the armed stack status.
func (s *StackService) Status() stack.Status {
	return s.session().Status()
}

// pasteText is used by QuickPaste when a stack is armed.
func pasteTextBack(q *QuickPasteService, text string) (PasteBackResult, error) {
	if q.Pipeline == nil {
		return PasteBackResult{}, fmt.Errorf("capture pipeline not ready")
	}
	if q.Hide != nil {
		q.Hide()
	}
	time.Sleep(20 * time.Millisecond)

	q.mu.Lock()
	t := q.target
	ok := q.hasTarget
	q.mu.Unlock()

	manualHint := pasteManualHint()
	write := func() error { return q.Pipeline.WriteText(text) }

	if !ok || !t.Valid() {
		if err := write(); err != nil {
			return PasteBackResult{}, err
		}
		return PasteBackResult{CopiedOnly: true, Message: "Copied. " + manualHint}, nil
	}
	if !paste.CanSynthesizePaste() {
		if err := write(); err != nil {
			return PasteBackResult{}, err
		}
		_ = paste.Activate(t)
		return PasteBackResult{
			CopiedOnly: true,
			Message:    pastePermissionHint(manualHint),
		}, nil
	}
	if err := paste.Activate(t); err != nil {
		if werr := write(); werr != nil {
			return PasteBackResult{}, werr
		}
		return PasteBackResult{
			CopiedOnly: true,
			Message:    "Copied. Could not focus " + t.String() + ". " + manualHint,
		}, nil
	}
	paste.Settle()
	if err := write(); err != nil {
		return PasteBackResult{}, err
	}
	paste.Settle()
	if err := paste.SendPaste(); err != nil {
		return PasteBackResult{CopiedOnly: true, Message: "Copied. " + manualHint}, nil
	}
	return PasteBackResult{Pasted: true}, nil
}
