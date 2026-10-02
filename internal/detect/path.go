package detect

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"kopiq/internal/domain"
)

type pathDetector struct{}

func (pathDetector) Name() string { return "path" }

// Detect labels absolute filesystem paths (and file:// URLs) as type "file".
// Finder/Explorer file lists already arrive as kind=files; this covers path-as-text
// copies from editors, terminals, and browsers.
func (pathDetector) Detect(item *domain.CanonicalItem) bool {
	paths, ok := ParseFilePaths(item.TextContent)
	if !ok {
		return false
	}
	item.DetectedType = "file"
	meta := map[string]any{
		"detector": "path",
		"paths":    paths,
		"count":    len(paths),
	}
	if b, err := json.Marshal(meta); err == nil {
		item.MetadataJSON = mergeMeta(item.MetadataJSON, b)
	}
	if strings.TrimSpace(item.Title) == "" {
		item.Title = filepath.Base(paths[0])
		if item.Title == "." || item.Title == string(filepath.Separator) || item.Title == "" {
			item.Title = paths[0]
		}
	}
	return true
}

// ParseFilePaths returns absolute paths when the entire text is one or more
// path lines (unix, ~/…, Windows drive/UNC, or file://). Otherwise ok is false.
func ParseFilePaths(text string) (paths []string, ok bool) {
	s := strings.TrimSpace(text)
	if s == "" || len(s) > 8192 {
		return nil, false
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return nil, false
	}

	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 || len(lines) > 100 {
		return nil, false
	}

	out := make([]string, 0, len(lines))
	for _, line := range lines {
		p, ok := normalizePathCandidate(line)
		if !ok {
			return nil, false
		}
		out = append(out, p)
	}
	return out, true
}

func normalizePathCandidate(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsRune(s, 0) {
		return "", false
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return "", false
	}

	if strings.HasPrefix(lower, "file://") {
		u, err := url.Parse(s)
		if err != nil {
			return "", false
		}
		p := u.Path
		if runtime.GOOS == "windows" && len(p) >= 3 && p[0] == '/' && p[2] == ':' {
			p = p[1:]
		}
		if u.Host != "" && u.Host != "localhost" {
			// UNC-style file://server/share
			p = `\\` + u.Host + filepath.FromSlash(p)
		}
		s = p
	}

	if s == "~" || strings.HasPrefix(s, "~/") || strings.HasPrefix(s, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return "", false
		}
		if s == "~" {
			s = home
		} else {
			s = filepath.Join(home, s[2:])
		}
	}

	if !isAbsolutePathShape(s) {
		return "", false
	}
	// Reject control chars
	if strings.ContainsAny(s, "\x00\r") {
		return "", false
	}
	cleaned := filepath.Clean(s)
	if cleaned == "." || cleaned == string(filepath.Separator) {
		return "", false
	}
	// Paths with whitespace are common on macOS ("My Documents"), but also look
	// like shell commands ("/usr/bin/git clone …"). Require the path to exist.
	if strings.ContainsAny(cleaned, " \t") {
		if _, err := os.Stat(cleaned); err != nil {
			return "", false
		}
	}
	return cleaned, true
}

func isAbsolutePathShape(s string) bool {
	if len(s) < 2 {
		return false
	}
	// Unix absolute
	if strings.HasPrefix(s, "/") {
		if s == "/" {
			return false
		}
		// Need a real path segment after root (e.g. /Users/..., /tmp/x)
		rest := strings.TrimPrefix(s, "/")
		if rest == "" {
			return false
		}
		// Reject bare "/bin" style only if it has no further slash AND no extension?
		// /tmp and /etc are valid dirs users copy. Allow any /x...
		for _, r := range rest {
			if r == 0 || (!unicode.IsPrint(r) && r != '\t') {
				return false
			}
		}
		return true
	}
	// Windows drive letter
	if len(s) >= 3 {
		drive := s[0]
		if ((drive >= 'A' && drive <= 'Z') || (drive >= 'a' && drive <= 'z')) &&
			s[1] == ':' && (s[2] == '\\' || s[2] == '/') {
			return true
		}
	}
	// UNC
	if strings.HasPrefix(s, `\\`) && len(s) > 3 {
		return true
	}
	return false
}

// FirstExistingPath returns the first path from text that exists on disk.
func FirstExistingPath(text string) (string, bool) {
	paths, ok := ParseFilePaths(text)
	if !ok {
		return "", false
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	// Still return first path so Reveal/Open can surface OS errors if none exist.
	return paths[0], false
}
