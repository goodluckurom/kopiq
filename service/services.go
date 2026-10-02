package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"kopiq/internal/blob"
	"kopiq/internal/capture"
	"kopiq/internal/detect"
	"kopiq/internal/domain"
	"kopiq/internal/icons"
	"kopiq/internal/multicopy"
	"kopiq/internal/settings"
	"kopiq/internal/store"
)

// HistoryService exposes clipboard history to the React UI.
type HistoryService struct {
	Store      *store.Store
	Pipeline   *capture.Pipeline
	Emit       func(domain.HistoryChangedEvent)
	SaveDialog func(defaultName string) (path string, err error)
	RevealPath func(absPath string) error
	OpenPath   func(absPath string) error
}

func (h *HistoryService) emit(itemID int64, action string) {
	if h.Emit == nil {
		return
	}
	h.Emit(domain.HistoryChangedEvent{ItemID: itemID, Action: action})
}

// List returns items matching the filter.
func (h *HistoryService) List(filter domain.HistoryFilter) ([]domain.ClipboardItem, error) {
	return h.Store.List(filter)
}

// ListRecent returns the newest clipboard items.
func (h *HistoryService) ListRecent() ([]domain.ClipboardItem, error) {
	return h.Store.List(domain.HistoryFilter{Limit: 100})
}

// Counts returns sidebar tallies.
func (h *HistoryService) Counts() (domain.HistoryCounts, error) {
	counts, err := h.Store.Counts()
	if err != nil {
		return counts, err
	}
	for i := range counts.ByApp {
		app := &counts.ByApp[i]
		if app.IconCacheKey == "" && app.AppID != "" {
			if key, err := icons.Ensure(app.AppID); err == nil && key != "" {
				app.IconCacheKey = key
			}
		} else if app.IconCacheKey != "" && !icons.Exists(app.IconCacheKey) && app.AppID != "" {
			_, _ = icons.Ensure(app.AppID)
		}
	}
	return counts, nil
}

// AppIconDataURL returns a PNG data URL for a cached app icon key (or app id).
func (h *HistoryService) AppIconDataURL(keyOrAppID string) (string, error) {
	if keyOrAppID == "" {
		return "", nil
	}
	key := icons.KeyFromAppID(keyOrAppID)
	if key == "" {
		return "", nil
	}
	if !icons.Exists(key) {
		if _, err := icons.Ensure(keyOrAppID); err != nil {
			_, _ = icons.Ensure(key)
		}
	}
	pngBytes, err := icons.ReadPNG(key)
	if err != nil || len(pngBytes) == 0 {
		return "", nil
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes), nil
}

// ItemImageDataURL returns a data URL for an item's thumbnail or original.
func (h *HistoryService) ItemImageDataURL(id int64, kind string) (string, error) {
	item, err := h.Store.GetRaw(id)
	if err != nil {
		return "", err
	}
	if item.Kind != "image" {
		return "", nil
	}
	rel := item.ThumbnailPath
	mime := "image/jpeg"
	if kind == "original" && item.ImageKept && item.BlobPath != "" {
		rel = item.BlobPath
		mime = item.MimeType
		if mime == "" {
			mime = "image/png"
		}
	}
	if rel == "" {
		if item.ThumbnailPath != "" {
			rel = item.ThumbnailPath
			mime = "image/jpeg"
		} else if item.BlobPath != "" {
			rel = item.BlobPath
			mime = item.MimeType
			if mime == "" {
				mime = "image/png"
			}
		} else {
			return "", nil
		}
	}
	data, err := blob.Read(rel)
	if err != nil || len(data) == 0 {
		return "", nil
	}
	if strings.HasSuffix(strings.ToLower(rel), ".png") {
		mime = "image/png"
	} else if strings.HasSuffix(strings.ToLower(rel), ".jpg") || strings.HasSuffix(strings.ToLower(rel), ".jpeg") {
		mime = "image/jpeg"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// Get returns one item by id.
func (h *HistoryService) Get(id int64) (*domain.ClipboardItem, error) {
	return h.Store.Get(id)
}

// Delete removes an item.
func (h *HistoryService) Delete(id int64) error {
	if err := h.Store.Delete(id); err != nil {
		return err
	}
	h.emit(id, "delete")
	return nil
}

// Pin sets or clears the pin flag.
func (h *HistoryService) Pin(id int64, pinned bool) error {
	if err := h.Store.SetPinned(id, pinned); err != nil {
		return err
	}
	h.emit(id, "update")
	return nil
}

// Rename sets the item title.
func (h *HistoryService) Rename(id int64, title string) error {
	if err := h.Store.SetTitle(id, title); err != nil {
		return err
	}
	h.emit(id, "update")
	return nil
}

// Keep clears expiry on a suspected item.
func (h *HistoryService) Keep(id int64) error {
	if err := h.Store.ClearExpiresAt(id); err != nil {
		return err
	}
	h.emit(id, "update")
	return nil
}

// ClearHistory removes unpinned items.
func (h *HistoryService) ClearHistory() (int, error) {
	n, err := h.Store.ClearHistory()
	if err != nil {
		return 0, err
	}
	h.emit(0, "delete")
	return n, nil
}

// ListExclusions returns capture exclusions with icon cache keys warmed.
func (h *HistoryService) ListExclusions() ([]store.Exclusion, error) {
	rows, err := h.Store.ListExclusions()
	if err != nil {
		return nil, err
	}
	for i := range rows {
		id := rows[i].ApplicationID
		if id == "" && rows[i].ApplicationName != "" {
			info := icons.LookupByName(rows[i].ApplicationName)
			if info.Found && info.ApplicationID != "" {
				rows[i].ApplicationID = info.ApplicationID
				id = info.ApplicationID
				if rows[i].ApplicationName == "" && info.ApplicationName != "" {
					rows[i].ApplicationName = info.ApplicationName
				}
				rows[i].IconCacheKey = info.IconCacheKey
				continue
			}
		}
		if id == "" {
			continue
		}
		if key, err := icons.Ensure(id); err == nil && key != "" {
			rows[i].IconCacheKey = key
		} else {
			rows[i].IconCacheKey = icons.KeyFromAppID(id)
		}
	}
	return rows, nil
}

// SetExclusionEnabled toggles an exclusion.
func (h *HistoryService) SetExclusionEnabled(id int64, enabled bool) error {
	return h.Store.SetExclusionEnabled(id, enabled)
}

// DeleteExclusion permanently removes a user-added exclusion.
func (h *HistoryService) DeleteExclusion(id int64) error {
	return h.Store.DeleteExclusion(id)
}

// AddExclusion adds a user exclusion by app id/name. Resolves icons from bundle
// id or display name when possible, and fills missing id/name from the OS.
func (h *HistoryService) AddExclusion(applicationID, applicationName string) error {
	applicationID = strings.TrimSpace(applicationID)
	applicationName = strings.TrimSpace(applicationName)

	var info icons.AppInfo
	switch {
	case applicationID != "":
		info = icons.Resolve(applicationID)
	case applicationName != "":
		info = icons.LookupByName(applicationName)
		if !info.Found {
			info = icons.Resolve(applicationName)
		}
	}
	if info.Found {
		if applicationID == "" && info.ApplicationID != "" {
			applicationID = info.ApplicationID
		}
		if applicationName == "" && info.ApplicationName != "" {
			applicationName = info.ApplicationName
		}
	}
	if applicationName == "" && applicationID != "" {
		applicationName = applicationID
	}
	if applicationName == "" {
		return fmt.Errorf("app name or bundle id required")
	}
	return h.Store.AddExclusion(applicationID, applicationName)
}

// ResolveApp looks up an app by bundle id / path or display name and caches its icon.
func (h *HistoryService) ResolveApp(query string) icons.AppInfo {
	return icons.Resolve(strings.TrimSpace(query))
}

// CopyToClipboard writes the item's text or image to the system clipboard (self-owned).
func (h *HistoryService) CopyToClipboard(id int64) error {
	item, err := h.Store.GetRaw(id)
	if err != nil {
		return err
	}
	if h.Pipeline == nil {
		return fmt.Errorf("capture pipeline not ready")
	}
	if item.Kind == "image" {
		pngBytes, err := imageItemPNG(item)
		if err != nil {
			return err
		}
		return h.Pipeline.WriteImage(pngBytes)
	}
	if item.TextContent == "" {
		return fmt.Errorf("item has no text content")
	}
	return h.Pipeline.WriteText(item.TextContent)
}

// MultiCopyResult is returned after copying several items as one payload.
type MultiCopyResult struct {
	Text          string `json:"text"`
	ImagesOmitted int    `json:"imagesOmitted"`
	ItemsUsed     int    `json:"itemsUsed"`
	Note          string `json:"note,omitempty"`
}

// MultiCopy builds a combined payload from item IDs and writes it (self-owned).
// shape: plain | lines | comma | csv | json | markdown
func (h *HistoryService) MultiCopy(ids []int64, shape string) (MultiCopyResult, error) {
	if len(ids) == 0 {
		return MultiCopyResult{}, fmt.Errorf("select at least one item")
	}
	if h.Pipeline == nil {
		return MultiCopyResult{}, fmt.Errorf("capture pipeline not ready")
	}
	items := make([]domain.ClipboardItem, 0, len(ids))
	for _, id := range ids {
		it, err := h.Store.GetRaw(id)
		if err != nil || it == nil {
			continue
		}
		items = append(items, *it)
	}
	res, err := multicopy.Build(items, shape)
	if err != nil {
		return MultiCopyResult{}, err
	}
	if err := h.Pipeline.WriteText(res.Text); err != nil {
		return MultiCopyResult{}, err
	}
	out := MultiCopyResult{
		Text:          res.Text,
		ImagesOmitted: res.ImagesOmitted,
		ItemsUsed:     res.ItemsUsed,
	}
	if res.ImagesOmitted > 0 {
		out.Note = fmt.Sprintf("%d images left out", res.ImagesOmitted)
		if res.ImagesOmitted == 1 {
			out.Note = "1 image left out"
		}
	}
	return out, nil
}

// SaveImage opens a native save dialog and writes a copy of the blob.
func (h *HistoryService) SaveImage(id int64) error {
	item, err := h.Store.GetRaw(id)
	if err != nil {
		return err
	}
	if item.Kind != "image" {
		return fmt.Errorf("not an image")
	}
	if !item.ImageKept || item.BlobPath == "" {
		return fmt.Errorf("original was not kept")
	}
	data, err := blob.Read(item.BlobPath)
	if err != nil {
		return err
	}
	ext := filepath.Ext(item.BlobPath)
	if ext == "" {
		ext = ".png"
	}
	name := fmt.Sprintf("kopiq-%s%s", time.Now().Format("2006-01-02-150405"), ext)
	if h.SaveDialog == nil {
		return fmt.Errorf("save dialog unavailable")
	}
	dest, err := h.SaveDialog(name)
	if err != nil || dest == "" {
		return err
	}
	return os.WriteFile(dest, data, 0o644)
}

// RevealImage shows the blob in the file manager. Disabled when original not kept.
func (h *HistoryService) RevealImage(id int64) error {
	item, err := h.Store.GetRaw(id)
	if err != nil {
		return err
	}
	if item.Kind != "image" {
		return fmt.Errorf("not an image")
	}
	if !item.ImageKept || item.BlobPath == "" {
		return fmt.Errorf("original was not kept")
	}
	abs, err := blob.Abs(item.BlobPath)
	if err != nil {
		return err
	}
	if h.RevealPath == nil {
		return fmt.Errorf("reveal unavailable")
	}
	return h.RevealPath(abs)
}

// OpenFile opens the first filesystem path from a file item with the OS default app.
func (h *HistoryService) OpenFile(id int64) error {
	path, err := h.filePathForItem(id)
	if err != nil {
		return err
	}
	if h.OpenPath == nil {
		return fmt.Errorf("open unavailable")
	}
	return h.OpenPath(path)
}

// RevealFile shows the first filesystem path from a file item in the file manager.
func (h *HistoryService) RevealFile(id int64) error {
	path, err := h.filePathForItem(id)
	if err != nil {
		return err
	}
	if h.RevealPath == nil {
		return fmt.Errorf("reveal unavailable")
	}
	return h.RevealPath(path)
}

// OpenContextFile opens the copy-context file path in the OS default app (editor).
func (h *HistoryService) OpenContextFile(id int64) error {
	item, err := h.Store.Get(id)
	if err != nil {
		return err
	}
	if item == nil || item.LastContext == nil || strings.TrimSpace(item.LastContext.FilePath) == "" {
		return fmt.Errorf("no file path on this item")
	}
	path := item.LastContext.FilePath
	if !filepath.IsAbs(path) {
		return fmt.Errorf("file path is not absolute")
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("file not found: %w", err)
	}
	if h.OpenPath == nil {
		return fmt.Errorf("open unavailable")
	}
	return h.OpenPath(path)
}

func (h *HistoryService) filePathForItem(id int64) (string, error) {
	item, err := h.Store.GetRaw(id)
	if err != nil {
		return "", err
	}
	path, _ := detect.FirstExistingPath(item.TextContent)
	if path == "" {
		return "", fmt.Errorf("no path found")
	}
	return path, nil
}

func imageItemPNG(item *domain.ClipboardItem) ([]byte, error) {
	rel := item.BlobPath
	if rel == "" {
		rel = item.ThumbnailPath
	}
	if rel == "" {
		return nil, fmt.Errorf("image has no stored file")
	}
	data, err := blob.Read(rel)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(strings.ToLower(rel), ".png") || item.MimeType == "image/png" {
		return data, nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// SearchService runs clipboard search.
type SearchService struct {
	Store *store.Store
}

// Search finds items matching query.
func (s *SearchService) Search(query string) ([]domain.ClipboardItem, error) {
	return s.Store.Search(query)
}

// CaptureService controls pause / status.
type CaptureService struct {
	Pipeline *capture.Pipeline
}

// Status returns current capture status.
func (c *CaptureService) Status() domain.CaptureStatus {
	if c.Pipeline == nil {
		return domain.CaptureStatus{}
	}
	return c.Pipeline.Status()
}

// Pause pauses capture for the given number of minutes (0 = until resume).
func (c *CaptureService) Pause(minutes int) error {
	return c.Pipeline.Pause(minutes)
}

// Resume clears pause.
func (c *CaptureService) Resume() error {
	return c.Pipeline.Resume()
}

// IsPaused reports whether capture is paused.
func (c *CaptureService) IsPaused() bool {
	return c.Pipeline.IsPaused()
}

// SettingsService exposes settings to the UI.
type SettingsService struct {
	Settings *settings.Settings
	Store    *store.Store
	// OnSet runs after a successful write (e.g. rebind global shortcut).
	OnSet func(key, value string) error

	mu              sync.RWMutex
	launchFrontmost FrontmostAppDTO
}

// Get returns all settings.
func (s *SettingsService) Get() map[string]string {
	return s.Settings.GetAll()
}

// Set writes a single setting and runs optional side effects.
func (s *SettingsService) Set(key, value string) error {
	if err := s.Settings.Set(key, value); err != nil {
		return err
	}
	if s.OnSet != nil {
		return s.OnSet(key, value)
	}
	return nil
}

// StorageStats returns disk usage for the Settings storage screen.
func (s *SettingsService) StorageStats() (store.StorageBreakdown, error) {
	if s.Store == nil {
		return store.StorageBreakdown{}, nil
	}
	return s.Store.StorageStats()
}

// CleanUp runs retention immediately and returns how many items were removed.
func (s *SettingsService) CleanUp() (store.RetentionResult, error) {
	if s.Store == nil || s.Settings == nil {
		return store.RetentionResult{}, nil
	}
	cfg := store.RetentionConfig{
		TextCap:          s.Settings.Int("retention.text_cap"),
		ImageDays:        s.Settings.Int("retention.image_days"),
		ImageBudgetBytes: 2147483648,
		Now:              time.Now().UnixMilli(),
	}
	if cfg.TextCap <= 0 {
		cfg.TextCap = 50000
	}
	if cfg.ImageDays < 0 {
		cfg.ImageDays = 30
	}
	if raw := s.Settings.Get("retention.image_budget_bytes"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 {
			cfg.ImageBudgetBytes = n
		}
	}
	return s.Store.ApplyRetention(cfg)
}

// ClearDeepFields removes window titles / URLs / paths from stored contexts.
func (s *SettingsService) ClearDeepFields() (int, error) {
	if s.Store == nil {
		return 0, nil
	}
	return s.Store.ClearDeepFields()
}
