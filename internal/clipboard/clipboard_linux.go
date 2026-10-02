//go:build linux || freebsd || openbsd || netbsd

package clipboard

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"kopiq/internal/domain"
)

// linuxSource watches CLIPBOARD via wl-clipboard or xclip/xsel (S7 interim).
// Native data-control / X11 bindings can replace this without changing Source.
type linuxSource struct {
	ch     chan domain.Generation
	cancel context.CancelFunc
	mu     sync.Mutex
	gen    atomic.Int64
	last   string
	self   map[string]struct{}
}

// New returns the platform clipboard source.
func New() Source {
	return &linuxSource{
		ch:   make(chan domain.Generation, 8),
		self: make(map[string]struct{}),
	}
}

func (l *linuxSource) SupportsBackgroundWatch() bool { return true }

func (l *linuxSource) Changes() <-chan domain.Generation { return l.ch }

func (l *linuxSource) CurrentGeneration() (domain.Generation, error) {
	return domain.Generation(l.gen.Load()), nil
}

func (l *linuxSource) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cancel != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel

	// Baseline — do not emit the current clipboard on startup (KQ-001).
	if text, ok := readClipboardText(); ok {
		l.last = hashText(text)
	}
	go l.poll(ctx)
	return nil
}

func (l *linuxSource) Stop() error {
	l.mu.Lock()
	cancel := l.cancel
	l.cancel = nil
	l.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (l *linuxSource) poll(ctx context.Context) {
	t := time.NewTicker(400 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			text, ok := readClipboardText()
			if !ok {
				continue
			}
			h := hashText(text)
			l.mu.Lock()
			if h == l.last {
				l.mu.Unlock()
				continue
			}
			l.last = h
			_, owned := l.self[h]
			if owned {
				delete(l.self, h)
				l.mu.Unlock()
				continue
			}
			l.mu.Unlock()
			g := domain.Generation(l.gen.Add(1))
			select {
			case l.ch <- g:
			default:
			}
		}
	}
}

func (l *linuxSource) Read(gen domain.Generation) (domain.RawPayload, error) {
	payload := domain.RawPayload{Generation: gen}
	text, textOK := readClipboardText()
	if textOK && text != "" {
		payload.Text = text
		payload.OSTypes = append(payload.OSTypes, "text/plain")
	}
	if png, ok := readClipboardPNG(); ok && len(png) > 0 {
		payload.ImageBytes = png
		payload.ImageType = "png"
		payload.OSTypes = append(payload.OSTypes, "image/png")
	}
	if payload.Text == "" && len(payload.ImageBytes) == 0 {
		if !textOK {
			return payload, fmt.Errorf("linux clipboard: install wl-clipboard or xclip")
		}
		return payload, fmt.Errorf("clipboard empty")
	}
	return payload, nil
}

func (l *linuxSource) Write(text string) (domain.Generation, error) {
	if err := writeClipboardText(text); err != nil {
		return 0, err
	}
	h := hashText(text)
	l.mu.Lock()
	l.self[h] = struct{}{}
	l.last = h
	l.mu.Unlock()
	g := domain.Generation(l.gen.Add(1))
	return g, nil
}

func (l *linuxSource) WriteImage(png []byte) (domain.Generation, error) {
	if err := writeClipboardPNG(png); err != nil {
		return 0, err
	}
	h := "img:" + hashBytes(png)
	l.mu.Lock()
	l.self[h] = struct{}{}
	l.last = h
	l.mu.Unlock()
	g := domain.Generation(l.gen.Add(1))
	return g, nil
}

func hashText(s string) string {
	return hashBytes([]byte(s))
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:16])
}

func readClipboardText() (string, bool) {
	if out, err := runCapture("wl-paste", "--no-newline", "--type", "text"); err == nil {
		return out, true
	}
	if out, err := runCapture("xclip", "-selection", "clipboard", "-o"); err == nil {
		return out, true
	}
	if out, err := runCapture("xsel", "--clipboard", "--output"); err == nil {
		return out, true
	}
	return "", false
}

func writeClipboardText(text string) error {
	if err := runFeed("wl-copy", text); err == nil {
		return nil
	}
	if err := runFeedArgs([]string{"xclip", "-selection", "clipboard"}, text); err == nil {
		return nil
	}
	if err := runFeedArgs([]string{"xsel", "--clipboard", "--input"}, text); err == nil {
		return nil
	}
	return fmt.Errorf("linux clipboard write: install wl-clipboard or xclip")
}

func readClipboardPNG() ([]byte, bool) {
	if out, err := runCaptureBytes("wl-paste", "--type", "image/png"); err == nil && len(out) > 0 {
		return out, true
	}
	if out, err := runCaptureBytes("xclip", "-selection", "clipboard", "-t", "image/png", "-o"); err == nil && len(out) > 0 {
		return out, true
	}
	return nil, false
}

func writeClipboardPNG(png []byte) error {
	if err := runFeedBytes([]string{"wl-copy", "--type", "image/png"}, png); err == nil {
		return nil
	}
	if err := runFeedBytes([]string{"xclip", "-selection", "clipboard", "-t", "image/png"}, png); err == nil {
		return nil
	}
	return fmt.Errorf("linux clipboard image write: install wl-clipboard or xclip")
}

func runCapture(name string, args ...string) (string, error) {
	if _, err := exec.LookPath(name); err != nil {
		return "", err
	}
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func runCaptureBytes(name string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(name); err != nil {
		return nil, err
	}
	return exec.Command(name, args...).Output()
}

func runFeed(name, input string) error {
	if _, err := exec.LookPath(name); err != nil {
		return err
	}
	cmd := exec.Command(name)
	cmd.Stdin = strings.NewReader(input)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runFeedArgs(argv []string, input string) error {
	if len(argv) == 0 {
		return fmt.Errorf("empty argv")
	}
	if _, err := exec.LookPath(argv[0]); err != nil {
		return err
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = strings.NewReader(input)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	return cmd.Run()
}

func runFeedBytes(argv []string, input []byte) error {
	if len(argv) == 0 {
		return fmt.Errorf("empty argv")
	}
	if _, err := exec.LookPath(argv[0]); err != nil {
		return err
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = bytes.NewReader(input)
	return cmd.Run()
}
