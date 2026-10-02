package detect

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	"kopiq/internal/domain"
)

// Detector inspects normalized content and may set type/language.
type Detector interface {
	Name() string
	Detect(item *domain.CanonicalItem) bool
}

// Registry runs detectors in order; first match wins for detected_type
// unless the type is still the default.
type Registry struct {
	detectors []Detector
}

// Default returns the M1 detector set (color, path, url, json, plus a few extras).
func Default() *Registry {
	return &Registry{detectors: []Detector{
		colorDetector{},
		pathDetector{},
		urlDetector{},
		jsonDetector{},
		jwtDetector{},
		emailDetector{},
		uuidDetector{},
		ipDetector{},
		timestampDetector{},
		yamlDetector{},
		xmlDetector{},
		csvDetector{},
		sqlDetector{},
		shellDetector{},
		codeDetector{},
	}}
}

// Apply runs detectors against the item. snapAppID/snapAppName classify screenshots.
// fileName is the context file name/path when known (extension wins for code).
func (r *Registry) Apply(item *domain.CanonicalItem, snapAppID, snapAppName, fileName string) {
	if item.Kind == "image" {
		if isScreenshotApp(snapAppID, snapAppName) || looksLikeScreenshotItem(item) {
			item.DetectedType = "screenshot"
		} else {
			item.DetectedType = "image"
		}
		return
	}
	if item.Kind == "files" {
		item.DetectedType = "file"
		return
	}
	// Context file extension wins over structured-text heuristics (KQ-003 / KQ-015).
	if fileName != "" {
		if lang := LanguageFromExt(fileName); lang != "" {
			cd := codeDetector{fileName: fileName}
			if cd.Detect(item) {
				return
			}
		}
	}
	for _, d := range r.detectors {
		if d.Name() == "code" {
			d = codeDetector{fileName: fileName}
		}
		if d.Detect(item) {
			return
		}
	}
}

func looksLikeScreenshotItem(item *domain.CanonicalItem) bool {
	candidates := []string{item.Title, item.TextPreview}
	var meta map[string]any
	if item.MetadataJSON != "" {
		_ = json.Unmarshal([]byte(item.MetadataJSON), &meta)
		if name, ok := meta["source_name"].(string); ok {
			candidates = append(candidates, name)
		}
		if name, ok := meta["companion_text"].(string); ok {
			candidates = append(candidates, name)
		}
	}
	for _, c := range candidates {
		lower := strings.ToLower(strings.TrimSpace(c))
		if lower == "" {
			continue
		}
		if strings.HasPrefix(lower, "screenshot") || strings.Contains(lower, "screen shot") {
			return true
		}
	}
	return false
}

func isScreenshotApp(appID, appName string) bool {
	id := strings.ToLower(strings.TrimSpace(appID))
	name := strings.ToLower(strings.TrimSpace(appName))
	switch id {
	case "com.apple.screencapture",
		"com.apple.screenshot.launcher",
		"com.apple.ScreenContinuity",
		"microsoft.screenSketch",
		"microsoft.screensketch":
		return true
	}
	if strings.Contains(id, "snippingtool") || strings.Contains(id, "screensketch") {
		return true
	}
	switch name {
	case "screenshot", "screen shot", "snip & sketch", "snipping tool", "flameshot", "spectacle", "grim":
		return true
	}
	if strings.Contains(name, "screenshot") || strings.Contains(name, "snipping") {
		return true
	}
	return false
}

type colorDetector struct{}

func (colorDetector) Name() string { return "color" }

var (
	hexColor = regexp.MustCompile(`(?i)^#([0-9a-f]{3}|[0-9a-f]{6}|[0-9a-f]{8})$`)
	rgbColor = regexp.MustCompile(`(?i)^rgba?\(\s*\d+\s*,\s*\d+\s*,\s*\d+`)
)

func (colorDetector) Detect(item *domain.CanonicalItem) bool {
	s := strings.TrimSpace(item.TextContent)
	if hexColor.MatchString(s) || rgbColor.MatchString(s) {
		item.DetectedType = "color"
		meta := map[string]any{"hex": s}
		if b, err := json.Marshal(meta); err == nil {
			item.MetadataJSON = mergeMeta(item.MetadataJSON, b)
		}
		return true
	}
	return false
}

type urlDetector struct{}

func (urlDetector) Name() string { return "url" }

func (urlDetector) Detect(item *domain.CanonicalItem) bool {
	s := strings.TrimSpace(item.TextContent)
	if strings.Contains(s, "\n") || len(s) > 2048 {
		return false
	}
	lower := strings.ToLower(s)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return false
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return false
	}
	item.DetectedType = "url"
	meta := map[string]any{"scheme": u.Scheme, "host": u.Host}
	if b, err := json.Marshal(meta); err == nil {
		item.MetadataJSON = mergeMeta(item.MetadataJSON, b)
	}
	return true
}

type jsonDetector struct{}

func (jsonDetector) Name() string { return "json" }

func (jsonDetector) Detect(item *domain.CanonicalItem) bool {
	s := strings.TrimSpace(item.TextContent)
	if len(s) < 2 {
		return false
	}
	if !(strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")) {
		return false
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return false
	}
	item.DetectedType = "json"
	return true
}

type jwtDetector struct{}

func (jwtDetector) Name() string { return "jwt" }

func (jwtDetector) Detect(item *domain.CanonicalItem) bool {
	s := strings.TrimSpace(item.TextContent)
	parts := strings.Split(s, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return false
	}
	if !looksBase64URL(parts[0]) || !looksBase64URL(parts[1]) {
		return false
	}
	item.DetectedType = "jwt"
	return true
}

func looksBase64URL(s string) bool {
	for _, c := range s {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			continue
		}
		return false
	}
	return len(s) >= 4
}

type sqlDetector struct{}

func (sqlDetector) Name() string { return "sql" }

var sqlStart = regexp.MustCompile(`(?is)^\s*(SELECT|INSERT|UPDATE|DELETE|CREATE|ALTER|DROP|WITH)\b`)

func (sqlDetector) Detect(item *domain.CanonicalItem) bool {
	s := item.TextContent
	if !sqlStart.MatchString(s) {
		return false
	}
	item.DetectedType = "code"
	item.Language = "sql"
	return true
}

type shellDetector struct{}

func (shellDetector) Name() string { return "shell" }

var shellHints = regexp.MustCompile(`(?i)^(docker|kubectl|npm|pnpm|yarn|git|curl|wget|ssh|scp|brew|go|cargo|make)\s`)

func (shellDetector) Detect(item *domain.CanonicalItem) bool {
	s := strings.TrimSpace(item.TextContent)
	if strings.HasPrefix(s, "#!") {
		// Shebangs belong to the code detector (language from interpreter).
		return false
	}
	if strings.Count(s, "\n") > 5 {
		return false
	}
	if shellHints.MatchString(s) {
		item.DetectedType = "shell"
		item.Language = "shell"
		return true
	}
	return false
}

func mergeMeta(existing string, extra []byte) string {
	var a, b map[string]any
	_ = json.Unmarshal([]byte(existing), &a)
	_ = json.Unmarshal(extra, &b)
	if a == nil {
		a = map[string]any{}
	}
	for k, v := range b {
		a[k] = v
	}
	out, err := json.Marshal(a)
	if err != nil {
		return existing
	}
	return string(out)
}
