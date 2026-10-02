package service

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"kopiq/internal/blob"
	"kopiq/internal/capture"
	"kopiq/internal/domain"
	"kopiq/internal/paste"
	"kopiq/internal/settings"
	"kopiq/internal/stack"
	"kopiq/internal/store"
	"kopiq/internal/transform"
)

// PasteBackResult is returned to the UI after Enter in Quick Paste.
type PasteBackResult struct {
	Pasted     bool   `json:"pasted"`
	CopiedOnly bool   `json:"copiedOnly"`
	Message    string `json:"message,omitempty"`
}

// QuickPasteService owns the Quick Paste panel session (KQ-012 / S5).
type QuickPasteService struct {
	Store    *store.Store
	Pipeline *capture.Pipeline
	Settings *settings.Settings
	Stack    *stack.Session

	// Show / Hide are set by main once windows exist.
	Show func()
	Hide func()

	mu        sync.Mutex
	target    paste.Target
	hasTarget bool
	selfPID   int
	selfBundle string
}

// Prepare records the frontmost app, then shows the panel.
func (q *QuickPasteService) Prepare() error {
	t, err := paste.CaptureTarget()
	q.mu.Lock()
	if err != nil || q.isSelf(t) {
		q.hasTarget = false
		q.target = paste.Target{}
	} else {
		q.target = t
		q.hasTarget = true
	}
	q.mu.Unlock()

	if q.Show != nil {
		q.Show()
	}
	return nil
}

// Dismiss hides the panel and restores focus to the paste target without pasting.
func (q *QuickPasteService) Dismiss() error {
	if q.Hide != nil {
		q.Hide()
	}
	q.mu.Lock()
	t := q.target
	ok := q.hasTarget
	q.mu.Unlock()
	if ok {
		_ = paste.Activate(t)
	}
	return nil
}

// PasteBack writes the item, activates the remembered target, and synthesizes paste.
// If a stack is armed, consumes the next stack step instead.
func (q *QuickPasteService) PasteBack(id int64) (PasteBackResult, error) {
	if q.Stack != nil && q.Stack.Armed() {
		text, done, err := q.Stack.NextText()
		if err != nil {
			return PasteBackResult{}, err
		}
		if done {
			// fall through to normal paste of selected id
		} else {
			res, err := pasteTextBack(q, text)
			if err != nil {
				return res, err
			}
			if res.Pasted || res.CopiedOnly {
				q.Stack.Advance()
			}
			st := q.Stack.Status()
			if st.Armed && res.Message == "" {
				res.Message = st.Label
			} else if !st.Armed && res.Pasted {
				res.Message = "Stack finished"
			}
			return res, nil
		}
	}

	item, err := q.Store.GetRaw(id)
	if err != nil {
		return PasteBackResult{}, err
	}
	if q.Pipeline == nil {
		return PasteBackResult{}, fmt.Errorf("capture pipeline not ready")
	}

	// 1. Hide first so the panel cannot receive the paste.
	if q.Hide != nil {
		q.Hide()
	}
	time.Sleep(20 * time.Millisecond)

	q.mu.Lock()
	t := q.target
	ok := q.hasTarget
	q.mu.Unlock()

	manualHint := pasteManualHint()

	if !ok || !t.Valid() {
		if err := q.writeItem(item); err != nil {
			return PasteBackResult{}, err
		}
		return PasteBackResult{
			CopiedOnly: true,
			Message:    "Copied. " + manualHint,
		}, nil
	}

	if !paste.CanSynthesizePaste() {
		if err := q.writeItem(item); err != nil {
			return PasteBackResult{}, err
		}
		_ = paste.Activate(t)
		return PasteBackResult{
			CopiedOnly: true,
			Message:    pastePermissionHint(manualHint),
		}, nil
	}

	// 3. Activate target
	if err := paste.Activate(t); err != nil {
		if werr := q.writeItem(item); werr != nil {
			return PasteBackResult{}, werr
		}
		return PasteBackResult{
			CopiedOnly: true,
			Message:    "Copied. Could not focus " + t.String() + ". " + manualHint,
		}, nil
	}
	paste.Settle()

	// 4. Write clipboard (self-owned)
	if err := q.writeItem(item); err != nil {
		return PasteBackResult{}, err
	}
	paste.Settle()

	// 5. Send paste shortcut
	if err := paste.SendPaste(); err != nil {
		return PasteBackResult{
			CopiedOnly: true,
			Message:    "Copied. " + manualHint,
		}, nil
	}

	return PasteBackResult{Pasted: true}, nil
}

// PasteTransformed applies a transform, writes the result, and paste-backs like PasteBack.
func (q *QuickPasteService) PasteTransformed(itemID int64, transformID string) (PasteBackResult, error) {
	item, err := q.Store.GetRaw(itemID)
	if err != nil {
		return PasteBackResult{}, err
	}
	if item.Sensitivity == "suspected" || item.Masked {
		return PasteBackResult{}, fmt.Errorf("suspected items cannot be transformed")
	}
	if q.Pipeline == nil {
		return PasteBackResult{}, fmt.Errorf("capture pipeline not ready")
	}
	reg := transform.Default()
	res := reg.Apply(item, transformID)
	if (res.Kind != "payload" && res.Kind != "view") || res.Text == "" {
		if res.Error != "" {
			return PasteBackResult{}, fmt.Errorf("%s", res.Error)
		}
		return PasteBackResult{}, fmt.Errorf("transform produced no text")
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
	write := func() error { return q.Pipeline.WriteText(res.Text) }

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

// HasPasteTarget reports whether Prepare captured a foreign app.
func (q *QuickPasteService) HasPasteTarget() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.hasTarget
}

// PasteTargetName returns the remembered app name for UI hints.
func (q *QuickPasteService) PasteTargetName() string {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.hasTarget {
		return ""
	}
	return q.target.String()
}

func (q *QuickPasteService) writeItem(item *domain.ClipboardItem) error {
	if item.Kind == "image" {
		pngBytes, err := imageItemPNG(item)
		if err != nil {
			// Fall back: try blob as-is via thumb path for jpeg-only oversized
			if item.BlobPath != "" {
				data, rerr := blob.Read(item.BlobPath)
				if rerr != nil {
					return err
				}
				return q.Pipeline.WriteImage(data)
			}
			return err
		}
		return q.Pipeline.WriteImage(pngBytes)
	}
	text := item.TextContent
	if text == "" {
		text = item.TextPreview
	}
	if text == "" {
		return fmt.Errorf("item has no pasteable content")
	}
	return q.Pipeline.WriteText(text)
}

func (q *QuickPasteService) isSelf(t paste.Target) bool {
	if q.selfPID == 0 {
		q.selfPID = os.Getpid()
	}
	if t.ProcessID > 0 && t.ProcessID == q.selfPID {
		return true
	}
	bid := strings.ToLower(t.ApplicationID)
	if bid == "" {
		return false
	}
	if q.selfBundle != "" && strings.EqualFold(bid, q.selfBundle) {
		return true
	}
	// Heuristic: our process name / known ids
	if strings.Contains(bid, "kopiq") {
		return true
	}
	if runtime.GOOS == "darwin" && (bid == "com.wails.kopiq" || strings.HasSuffix(bid, ".kopiq")) {
		return true
	}
	return false
}

func pasteManualHint() string {
	if runtime.GOOS == "darwin" {
		return "Paste with ⌘V"
	}
	return "Paste with Ctrl+V"
}

func pastePermissionHint(manualHint string) string {
	if runtime.GOOS == "darwin" {
		return "Copied. Grant Accessibility to auto-paste, or " + manualHint
	}
	return "Copied. Could not auto-paste. " + manualHint
}
