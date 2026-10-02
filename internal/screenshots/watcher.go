package screenshots

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"kopiq/internal/normalize"
)

// Handler is called for each new screenshot file path.
type Handler func(path string)

// DetectHandler is called as soon as a new screenshot file is noticed,
// before waiting for a stable write — used to freeze frontmost-app context.
type DetectHandler func(path string)

// Watcher polls macOS screenshot folders for new Screenshot*.png files.
type Watcher struct {
	handler  Handler
	onDetect DetectHandler
	stopCh   chan struct{}
	wg       sync.WaitGroup

	mu      sync.Mutex
	seen    map[string]time.Time
	started time.Time
}

// New builds a watcher. Call Start after wiring the handler.
func New(handler Handler) *Watcher {
	return &Watcher{
		handler: handler,
		stopCh:  make(chan struct{}),
		seen:    make(map[string]time.Time),
		started: time.Now(),
	}
}

// OnDetect sets a callback invoked when a new file is first seen (before waitStable).
func (w *Watcher) OnDetect(fn DetectHandler) {
	w.onDetect = fn
}

// Start begins polling. Existing files are baselined without ingest so we
// only capture context for screenshots that appear while Kopiq is running.
func (w *Watcher) Start() {
	w.baseline()
	w.wg.Add(1)
	go w.loop()
}

// Stop ends polling.
func (w *Watcher) Stop() {
	select {
	case <-w.stopCh:
	default:
		close(w.stopCh)
	}
	w.wg.Wait()
}

func (w *Watcher) baseline() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, dir := range normalize.ScreenshotDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !isScreenshotFile(e.Name()) {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			w.seen[filepath.Join(dir, e.Name())] = info.ModTime()
		}
	}
}

func (w *Watcher) loop() {
	defer w.wg.Done()
	t := time.NewTicker(400 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-w.stopCh:
			return
		case <-t.C:
			w.scan()
		}
	}
}

func (w *Watcher) scan() {
	for _, dir := range normalize.ScreenshotDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !isScreenshotFile(e.Name()) {
				continue
			}
			path := filepath.Join(dir, e.Name())
			info, err := e.Info()
			if err != nil {
				continue
			}
			mod := info.ModTime()
			w.mu.Lock()
			prev, ok := w.seen[path]
			if ok && !mod.After(prev) {
				w.mu.Unlock()
				continue
			}
			if mod.Before(w.started.Add(-2 * time.Second)) && !ok {
				w.seen[path] = mod
				w.mu.Unlock()
				continue
			}
			w.seen[path] = mod
			w.mu.Unlock()

			if w.handler != nil || w.onDetect != nil {
				// Freeze context immediately; ingest waits for a stable file size.
				go func(p string) {
					if w.onDetect != nil {
						w.onDetect(p)
					}
					if err := waitStable(p, 12); err != nil {
						log.Printf("screenshots: wait %s: %v", p, err)
						return
					}
					if w.handler != nil {
						w.handler(p)
					}
				}(path)
			}
		}
	}
}

func isScreenshotFile(name string) bool {
	lower := strings.ToLower(name)
	lower = strings.ReplaceAll(lower, "\u202f", " ")
	if !(strings.HasPrefix(lower, "screenshot") || strings.Contains(lower, "screen shot")) {
		return false
	}
	return normalize.IsImagePath(name)
}

func waitStable(path string, attempts int) error {
	var last int64 = -1
	for i := 0; i < attempts; i++ {
		st, err := os.Stat(path)
		if err != nil {
			return err
		}
		sz := st.Size()
		if sz > 0 && sz == last {
			return nil
		}
		last = sz
		time.Sleep(100 * time.Millisecond)
	}
	if last > 0 {
		return nil
	}
	return os.ErrNotExist
}
