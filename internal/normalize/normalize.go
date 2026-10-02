package normalize

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"kopiq/internal/domain"
)

const maxTextBytes = 2 * 1024 * 1024

var whitespaceCollapse = regexp.MustCompile(`\s+`)

// FromRaw turns a RawPayload into a CanonicalItem (without detection/sensitive).
func FromRaw(raw domain.RawPayload) (*domain.CanonicalItem, bool) {
	meta := map[string]any{
		"nxt": raw.OSTypes,
	}

	// Finder / Desktop screenshot copies often put file URLs, not inline image bytes.
	if len(raw.ImageBytes) == 0 && len(raw.FileURLs) > 0 {
		if path, ok := SingleImagePath(raw.FileURLs); ok {
			if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
				raw.ImageBytes = data
				raw.ImageType = ImageTypeFromPath(path)
				meta["source_path"] = path
				meta["source_name"] = filepath.Base(path)
			}
		}
	}

	if len(raw.FileURLs) > 0 && len(raw.ImageBytes) == 0 {
		paths := raw.FileURLs
		if len(paths) > 500 {
			paths = paths[:500]
			meta["truncated"] = true
		}
		text := strings.Join(paths, "\n")
		hash := hashBytes([]byte(text))
		b, _ := json.Marshal(meta)
		title := filepath.Base(paths[0])
		if len(paths) > 1 {
			title = fmt.Sprintf("%s +%d", title, len(paths)-1)
		}
		return &domain.CanonicalItem{
			Kind:         "files",
			Title:        title,
			TextContent:  text,
			TextPreview:  Preview(text),
			ByteSize:     int64(len(text)),
			ContentHash:  hash,
			MetadataJSON: string(b),
			DetectedType: "file",
			Sensitivity:  "none",
			ImageKept:    true,
		}, true
	}

	text := canonicalizeText(raw.Text)
	if text == "" && raw.HTML != "" {
		text = canonicalizeText(stripTags(raw.HTML))
		meta["html"] = true
	}

	// Text-only paste of a screenshot filename → try resolve from Desktop.
	if len(raw.ImageBytes) == 0 && looksLikeScreenshotName(text) {
		if path, ok := ResolveScreenshotOnDisk(text); ok {
			if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
				raw.ImageBytes = data
				raw.ImageType = ImageTypeFromPath(path)
				meta["source_path"] = path
				meta["source_name"] = filepath.Base(path)
				meta["resolved_from_text"] = true
			}
		}
	}

	hasImage := len(raw.ImageBytes) > 0
	if hasImage && preferImageOverText(text, len(raw.ImageBytes)) {
		if strings.TrimSpace(text) != "" {
			meta["had_text"] = true
			meta["companion_text"] = Preview(text)
			if isURLOrFilename(strings.TrimSpace(text)) || looksLikeImageFilename(strings.TrimSpace(text)) || looksLikeScreenshotName(text) {
				if strings.HasPrefix(strings.ToLower(strings.TrimSpace(text)), "http") {
					meta["source_url"] = strings.TrimSpace(text)
				} else if _, ok := meta["source_name"]; !ok {
					meta["source_name"] = strings.TrimSpace(text)
				}
			}
		}
		b, _ := json.Marshal(meta)
		item := &domain.CanonicalItem{
			Kind:         "image",
			ImageBytes:   raw.ImageBytes,
			DetectedType: "image",
			Sensitivity:  "none",
			ImageKept:    true,
			TextPreview:  "Image",
			MetadataJSON: string(b),
		}
		if name, ok := meta["source_name"].(string); ok && name != "" {
			item.Title = name
		}
		if err := PrepareImage(item, raw.ImageType); err != nil {
			// Image bytes present but undecodable — fall through to text if any,
			// otherwise drop (better than storing a corrupt blob).
			if text == "" {
				return nil, false
			}
			meta["image_decode_error"] = err.Error()
			hasImage = false
		} else {
			return item, true
		}
	}

	if text == "" {
		return nil, false
	}

	if len(text) > maxTextBytes {
		text = truncateUTF8(text, maxTextBytes)
		meta["truncated"] = true
	}
	if hasImage {
		meta["had_image"] = true
	}
	if raw.HTML != "" {
		meta["html"] = true
	}
	if raw.RTF != "" {
		meta["rtf"] = true
	}

	hash := hashBytes([]byte(text))
	b, _ := json.Marshal(meta)
	return &domain.CanonicalItem{
		Kind:         "text",
		TextContent:  text,
		TextPreview:  Preview(text),
		ByteSize:     int64(len(text)),
		ContentHash:  hash,
		MetadataJSON: string(b),
		DetectedType: "text",
		Sensitivity:  "none",
		ImageKept:    true,
	}, true
}

// FromImageFile builds a canonical image item from a path on disk.
func FromImageFile(path string) (*domain.CanonicalItem, bool) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return FromRaw(domain.RawPayload{
		ImageBytes: data,
		ImageType:  ImageTypeFromPath(path),
		FileURLs:   []string{path},
		OSTypes:    []string{"public.file-url"},
		Text:       filepath.Base(path),
	})
}

// SingleImagePath returns the path when FileURLs is exactly one image file.
func SingleImagePath(urls []string) (string, bool) {
	if len(urls) != 1 {
		return "", false
	}
	p := urls[0]
	if !IsImagePath(p) {
		return "", false
	}
	return p, true
}

// IsImagePath reports whether path looks like a raster image we can store.
func IsImagePath(path string) bool {
	return looksLikeImageFilename(path)
}

// ImageTypeFromPath returns a normalize image type hint from extension.
func ImageTypeFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return "jpeg"
	case ".tif", ".tiff":
		return "tiff"
	case ".png":
		return "png"
	case ".bmp":
		return "bmp"
	default:
		return strings.TrimPrefix(ext, ".")
	}
}

func looksLikeScreenshotName(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" || strings.Contains(t, "\n") {
		return false
	}
	lower := strings.ToLower(t)
	return strings.HasPrefix(lower, "screenshot") || strings.Contains(lower, "screen shot")
}

// ResolveScreenshotOnDisk finds a Desktop (or configured) screenshot matching name.
func ResolveScreenshotOnDisk(name string) (string, bool) {
	base := strings.TrimSpace(name)
	if base == "" {
		return "", false
	}
	// Allow name without extension.
	candidates := []string{base}
	if !looksLikeImageFilename(base) {
		for _, ext := range []string{".png", ".jpg", ".jpeg"} {
			candidates = append(candidates, base+ext)
		}
	}
	dirs := ScreenshotDirs()
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			fn := e.Name()
			for _, c := range candidates {
				if namesMatchScreenshot(fn, c) {
					return filepath.Join(dir, fn), true
				}
			}
		}
	}
	return "", false
}

func namesMatchScreenshot(fileName, want string) bool {
	a := normalizeShotName(fileName)
	b := normalizeShotName(want)
	if a == b {
		return true
	}
	// want without extension vs file with extension
	aNo := strings.TrimSuffix(a, filepath.Ext(a))
	bNo := strings.TrimSuffix(b, filepath.Ext(b))
	return aNo == bNo
}

func normalizeShotName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// macOS uses narrow no-break space before AM/PM
	s = strings.ReplaceAll(s, "\u202f", " ")
	s = strings.ReplaceAll(s, "\u00a0", " ")
	return s
}

// ScreenshotDirs returns folders where macOS may write screenshots.
func ScreenshotDirs() []string {
	seen := map[string]bool{}
	var dirs []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		dirs = append(dirs, p)
	}
	add(macScreenshotLocation())
	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, "Desktop"))
	}
	return dirs
}

func canonicalizeText(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "\uFFFD")
	}
	return s
}

// Preview collapses whitespace and returns the first 280 characters.
func Preview(s string) string {
	p := whitespaceCollapse.ReplaceAllString(strings.TrimSpace(s), " ")
	if utf8.RuneCountInString(p) > 280 {
		runes := []rune(p)
		p = string(runes[:280])
	}
	return p
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func HashText(s string) string {
	return hashBytes([]byte(s))
}

func stripTags(html string) string {
	re := regexp.MustCompile(`(?s)<[^>]*>`)
	return re.ReplaceAllString(html, "")
}

func isURLOrFilename(s string) bool {
	if s == "" {
		return false
	}
	if strings.Contains(s, "\n") {
		return false
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return true
	}
	return !strings.Contains(s, " ") && strings.Contains(s, ".")
}

// preferImageOverText decides whether clipboard image bytes should become an
// image item when text is also present (browsers attach URLs/alt text; macOS
// attaches "Screenshot ….png"). Real image payloads always win; only tiny
// decorative icons attached to long rich text are deferred to the text path.
func preferImageOverText(text string, imageLen int) bool {
	if imageLen <= 0 {
		return false
	}
	t := strings.TrimSpace(text)
	if t == "" {
		return true
	}
	// Photos, screenshots, and "Copy Image" payloads are never treated as icons.
	if imageLen >= 8*1024 {
		return true
	}
	if isURLOrFilename(t) || looksLikeImageFilename(t) || looksLikeScreenshotName(t) {
		return true
	}
	// Short companion label / caption (single- or multi-line).
	if utf8.RuneCountInString(t) <= 200 {
		return true
	}
	// Tiny image + long text → likely a decorative icon on a rich copy.
	return false
}

func looksLikeImageFilename(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".tif", ".tiff", ".heic", ".bmp"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	for maxBytes > 0 && !utf8.ValidString(s[:maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}
