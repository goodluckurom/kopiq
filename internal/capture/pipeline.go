package capture

import (
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"kopiq/internal/blob"
	"kopiq/internal/clipboard"
	appctx "kopiq/internal/context"
	"kopiq/internal/detect"
	"kopiq/internal/domain"
	"kopiq/internal/normalize"
	"kopiq/internal/sensitive"
	"kopiq/internal/settings"
	"kopiq/internal/store"
)

// EventSink receives history change notifications.
type EventSink func(evt domain.HistoryChangedEvent)

// Pipeline owns the ordered capture goroutine (architecture §Capture pipeline).
type Pipeline struct {
	src       clipboard.Source
	ctx       appctx.Provider
	store     *store.Store
	settings  *settings.Settings
	detectors *detect.Registry
	selfOwned *clipboard.Ring
	ignored   *clipboard.Ring
	emit      EventSink

	mu             sync.Mutex
	stopCh         chan struct{}
	wg             sync.WaitGroup
	lastUsefulSnap domain.Snapshot            // last frontmost app that isn't Kopiq / screenshot UI
	pendingShot    map[string]domain.Snapshot // path → snap frozen when file first appeared
}

// New builds a capture pipeline.
func New(
	src clipboard.Source,
	ctx appctx.Provider,
	st *store.Store,
	set *settings.Settings,
	emit EventSink,
) *Pipeline {
	return &Pipeline{
		src:         src,
		ctx:         ctx,
		store:       st,
		settings:    set,
		detectors:   detect.Default(),
		selfOwned:   clipboard.NewRing(64),
		ignored:     clipboard.NewRing(64),
		emit:        emit,
		stopCh:      make(chan struct{}),
		pendingShot: make(map[string]domain.Snapshot),
	}
}

// MarkSelfOwned records a generation Kopiq wrote.
func (p *Pipeline) MarkSelfOwned(g domain.Generation) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.selfOwned.Add(g)
}

// WriteText writes text to the clipboard and marks the generation self-owned
// under the same lock so the poller cannot re-ingest our own write.
func (p *Pipeline) WriteText(text string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	gen, err := p.src.Write(text)
	if err != nil {
		return err
	}
	p.selfOwned.Add(gen)
	return nil
}

// WriteImage writes PNG bytes to the clipboard and marks the generation self-owned.
func (p *Pipeline) WriteImage(png []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	gen, err := p.src.WriteImage(png)
	if err != nil {
		return err
	}
	p.selfOwned.Add(gen)
	return nil
}

// Start begins watching the clipboard.
func (p *Pipeline) Start() error {
	if err := p.src.Start(); err != nil {
		return err
	}
	p.wg.Add(1)
	go p.loop()
	p.wg.Add(1)
	go p.contextSampler()
	return nil
}

// contextSampler remembers the last "useful" frontmost app so Desktop
// screenshot ingest can attribute context even after the screenshot UI steals focus.
func (p *Pipeline) contextSampler() {
	defer p.wg.Done()
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-p.stopCh:
			return
		case <-t.C:
			deep := p.settings.Bool("context.deep")
			snap, err := p.ctx.Capture(deep)
			if err != nil || isNoiseApp(snap.ApplicationID, snap.ApplicationName) {
				continue
			}
			p.mu.Lock()
			p.lastUsefulSnap = snap
			p.mu.Unlock()
		}
	}
}

func isNoiseApp(id, name string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	name = strings.ToLower(strings.TrimSpace(name))
	if id == "" && name == "" {
		return true
	}
	if strings.Contains(id, "kopiq") || name == "kopiq" {
		return true
	}
	switch id {
	case "com.apple.screencapture",
		"com.apple.screenshot.launcher",
		"com.apple.loginwindow",
		"com.apple.dock",
		"com.apple.notificationcenterui",
		"com.apple.systemuiserver",
		"com.apple.controlcenter",
		"com.apple.Spotlight",
		"com.apple.finder", // Desktop screenshot writes often surface Finder briefly
		"com.apple.WindowManager",
		"com.apple.wallpaper":
		return true
	}
	// Windows Snipping Tool / Screen Sketch hosts focus during capture.
	if strings.Contains(id, "snippingtool") ||
		strings.Contains(id, "screenclippinghost") ||
		strings.Contains(id, "screensketch") ||
		strings.Contains(id, "shell experience host") {
		return true
	}
	if name == "finder" || name == "explorer" || name == "dwm" {
		return true
	}
	if strings.Contains(name, "screenshot") ||
		strings.Contains(name, "screen shot") ||
		strings.Contains(name, "snipping tool") ||
		strings.Contains(name, "snip & sketch") ||
		strings.Contains(name, "notification") ||
		strings.Contains(name, "control center") {
		return true
	}
	return false
}

func (p *Pipeline) rememberSnap(snap domain.Snapshot) {
	if isNoiseApp(snap.ApplicationID, snap.ApplicationName) {
		return
	}
	p.mu.Lock()
	p.lastUsefulSnap = snap
	p.mu.Unlock()
}

// NoteScreenshotAppeared freezes context as soon as a Desktop screenshot file is
// detected — before waitStable and before Finder/thumbnail UI can steal focus.
func (p *Pipeline) NoteScreenshotAppeared(path string) {
	snap := p.snapForScreenshot()
	p.mu.Lock()
	p.pendingShot[path] = snap
	p.mu.Unlock()
}

func (p *Pipeline) takePendingShot(path string) (domain.Snapshot, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	snap, ok := p.pendingShot[path]
	if ok {
		delete(p.pendingShot, path)
	}
	return snap, ok
}

func (p *Pipeline) snapForScreenshot() domain.Snapshot {
	deep := p.settings.Bool("context.deep")
	now, err := p.ctx.Capture(deep)
	if err != nil {
		now = domain.Snapshot{CapturedAt: time.Now().UnixMilli()}
	}
	if !isNoiseApp(now.ApplicationID, now.ApplicationName) {
		p.rememberSnap(now)
		return now
	}
	p.mu.Lock()
	prev := p.lastUsefulSnap
	p.mu.Unlock()
	if prev.ApplicationName != "" && prev.ApplicationName != "Unknown" {
		prev.CapturedAt = time.Now().UnixMilli()
		return prev
	}
	// Last resort: keep whatever we have, but label clearly.
	if now.ApplicationName == "" || now.ApplicationName == "Unknown" || isNoiseApp(now.ApplicationID, now.ApplicationName) {
		now.ApplicationName = "Screenshot"
		now.ApplicationID = ""
	}
	return now
}

// Stop ends the pipeline.
func (p *Pipeline) Stop() {
	select {
	case <-p.stopCh:
	default:
		close(p.stopCh)
	}
	_ = p.src.Stop()
	p.wg.Wait()
}

func (p *Pipeline) loop() {
	defer p.wg.Done()
	for {
		select {
		case <-p.stopCh:
			return
		case gen, ok := <-p.src.Changes():
			if !ok {
				return
			}
			if err := p.handle(gen); err != nil {
				log.Printf("capture: %v", err)
			}
		}
	}
}

func (p *Pipeline) handle(gen domain.Generation) error {
	p.mu.Lock()
	owned := p.selfOwned.Contains(gen) || p.ignored.Contains(gen)
	p.mu.Unlock()
	if owned {
		return nil
	}

	// 3. pause / disabled
	if p.isPausedOrDisabled() {
		p.mu.Lock()
		p.ignored.Add(gen)
		p.mu.Unlock()
		return nil
	}

	// 4. context before read
	deep := p.settings.Bool("context.deep")
	snap, err := p.ctx.Capture(deep)
	if err != nil {
		snap = domain.Snapshot{ApplicationName: "Unknown", CapturedAt: time.Now().UnixMilli()}
	}
	p.rememberSnap(snap)

	// 5. exclusions
	excluded, err := p.store.IsExcluded(snap.ApplicationID, snap.ApplicationName)
	if err != nil {
		return err
	}
	if excluded {
		p.mu.Lock()
		p.ignored.Add(gen)
		p.mu.Unlock()
		return nil
	}

	// 6. read
	raw, err := p.src.Read(gen)
	if err != nil {
		p.mu.Lock()
		p.ignored.Add(gen)
		p.mu.Unlock()
		return err
	}

	// 7. normalize
	item, ok := normalize.FromRaw(raw)
	if !ok {
		p.mu.Lock()
		p.ignored.Add(gen)
		p.mu.Unlock()
		return nil
	}

	// 8. detect (screenshot tools use frontmost app)
	p.detectors.Apply(item, snap.ApplicationID, snap.ApplicationName, firstNonEmpty(snap.FileName, snap.FilePath))
	if item.Title == "" {
		switch item.DetectedType {
		case "screenshot":
			item.Title = "Screenshot"
		case "image":
			item.Title = "Image"
		}
	}

	// 9–10. sensitive
	policy := p.settings.Get("sensitive.policy")
	ttl := p.settings.Int("sensitive.ttl_seconds")
	if ttl <= 0 {
		ttl = 60
	}
	now := time.Now().UnixMilli()
	if !sensitive.Apply(item, policy, ttl, now) {
		p.mu.Lock()
		p.ignored.Add(gen)
		p.mu.Unlock()
		return nil
	}

	// 11. persist image blob (hot path: original only; thumb async)
	needThumb := false
	if item.Kind == "image" && len(item.ImageBytes) > 0 {
		if err := p.persistImageBlob(item); err != nil {
			return err
		}
		needThumb = item.ImageKept && item.ThumbnailPath == "" && item.BlobPath != ""
	}

	// 12–14. dedup + commit
	ctx := snap.ToCopyContext()
	id, inserted, err := p.store.InsertOrBump(item, ctx, now)
	if err != nil {
		return err
	}

	action := "update"
	if inserted {
		action = "insert"
	}
	if p.emit != nil {
		p.emit(domain.HistoryChangedEvent{ItemID: id, Action: action})
	}

	if inserted && needThumb {
		hash := item.ContentHash
		blobPath := item.BlobPath
		go p.generateThumb(id, hash, blobPath)
	}
	return nil
}

// IngestImageFile imports an on-disk image (Desktop screenshots, Finder copies).
func (p *Pipeline) IngestImageFile(path string) error {
	if p.isPausedOrDisabled() {
		return nil
	}
	item, ok := normalize.FromImageFile(path)
	if !ok || item == nil {
		return nil
	}

	snap, hadPending := p.takePendingShot(path)
	if !hadPending {
		snap = p.snapForScreenshot()
	}

	p.detectors.Apply(item, snap.ApplicationID, snap.ApplicationName, firstNonEmpty(snap.FileName, snap.FilePath))
	if item.DetectedType != "screenshot" {
		base := strings.ToLower(filepath.Base(path))
		base = strings.ReplaceAll(base, "\u202f", " ")
		if strings.HasPrefix(base, "screenshot") {
			item.DetectedType = "screenshot"
		}
	}
	if item.Title == "" {
		item.Title = filepath.Base(path)
	}
	if item.DetectedType == "screenshot" && (item.Title == "" || item.Title == "Image") {
		item.Title = filepath.Base(path)
	}

	policy := p.settings.Get("sensitive.policy")
	ttl := p.settings.Int("sensitive.ttl_seconds")
	if ttl <= 0 {
		ttl = 60
	}
	now := time.Now().UnixMilli()
	if !sensitive.Apply(item, policy, ttl, now) {
		return nil
	}

	needThumb := false
	if item.Kind == "image" && len(item.ImageBytes) > 0 {
		if err := p.persistImageBlob(item); err != nil {
			return err
		}
		needThumb = item.ImageKept && item.ThumbnailPath == "" && item.BlobPath != ""
	}

	ctx := snap.ToCopyContext()
	// Only attach the Desktop path when we lack a real source app.
	if ctx.ApplicationName == "" ||
		ctx.ApplicationName == "Unknown" ||
		ctx.ApplicationName == "Screenshot" {
		if ctx.FilePath == "" {
			ctx.FilePath = path
			ctx.FileName = filepath.Base(path)
			if ctx.PlaceKey == "" {
				ctx.PlaceKey = path
			}
		}
	}
	id, inserted, err := p.store.InsertOrBump(item, ctx, now)
	if err != nil {
		return err
	}
	action := "update"
	if inserted {
		action = "insert"
	}
	if p.emit != nil {
		p.emit(domain.HistoryChangedEvent{ItemID: id, Action: action})
	}
	if inserted && needThumb {
		go p.generateThumb(id, item.ContentHash, item.BlobPath)
	}
	return nil
}

func (p *Pipeline) persistImageBlob(item *domain.CanonicalItem) error {
	ext := ".png"
	if strings.Contains(item.MimeType, "jpeg") || strings.Contains(item.MimeType, "jpg") {
		ext = ".jpg"
	}
	if !item.ImageKept {
		rel, err := blob.WriteThumb(item.ContentHash, item.ImageBytes)
		if err != nil {
			return err
		}
		item.ThumbnailPath = rel
		item.BlobPath = ""
		item.ImageBytes = nil
		return nil
	}
	rel, err := blob.WriteOriginal(item.ContentHash, ext, item.ImageBytes)
	if err != nil {
		return err
	}
	item.BlobPath = rel
	item.ImageBytes = nil
	return nil
}

func (p *Pipeline) generateThumb(id int64, contentHash, blobPath string) {
	data, err := blob.Read(blobPath)
	if err != nil {
		log.Printf("capture: thumb read: %v", err)
		return
	}
	jpeg, err := normalize.EncodeThumbFromBytes(data)
	if err != nil {
		log.Printf("capture: thumb encode: %v", err)
		return
	}
	rel, err := blob.WriteThumb(contentHash, jpeg)
	if err != nil {
		log.Printf("capture: thumb write: %v", err)
		return
	}
	if err := p.store.SetThumbnailPath(id, rel); err != nil {
		log.Printf("capture: thumb path: %v", err)
		return
	}
	if p.emit != nil {
		p.emit(domain.HistoryChangedEvent{ItemID: id, Action: "update"})
	}
}

func (p *Pipeline) isPausedOrDisabled() bool {
	if !p.settings.Bool("capture.enabled") {
		return true
	}
	until := p.settings.Get("pause.until")
	if until == "" {
		return false
	}
	if until == settings.PauseUntilResume {
		return true
	}
	ms, err := strconv.ParseInt(until, 10, 64)
	if err != nil {
		return false
	}
	if time.Now().UnixMilli() >= ms {
		_ = p.settings.ClearPause()
		return false
	}
	return true
}

// Status builds a CaptureStatus snapshot.
func (p *Pipeline) Status() domain.CaptureStatus {
	st := domain.CaptureStatus{
		Enabled:         p.settings.Bool("capture.enabled"),
		BackgroundWatch: p.src.SupportsBackgroundWatch(),
	}
	until := p.settings.Get("pause.until")
	if until == "" {
		return st
	}
	st.Paused = true
	if until == settings.PauseUntilResume {
		st.PauseUntil = -1
		st.Message = "Paused until resumed"
		return st
	}
	ms, err := strconv.ParseInt(until, 10, 64)
	if err != nil {
		return st
	}
	st.PauseUntil = ms
	rem := ms - time.Now().UnixMilli()
	if rem < 0 {
		rem = 0
		st.Paused = false
		_ = p.settings.ClearPause()
	}
	st.RemainingMs = rem
	return st
}

// Pause pauses capture for minutes. minutes <= 0 means until resume.
func (p *Pipeline) Pause(minutes int) error {
	if minutes <= 0 {
		return p.settings.Set("pause.until", settings.PauseUntilResume)
	}
	until := time.Now().Add(time.Duration(minutes) * time.Minute).UnixMilli()
	return p.settings.Set("pause.until", strconv.FormatInt(until, 10))
}

// Resume clears pause.
func (p *Pipeline) Resume() error {
	return p.settings.ClearPause()
}

// IsPaused reports whether capture is currently paused.
func (p *Pipeline) IsPaused() bool {
	return p.Status().Paused
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
