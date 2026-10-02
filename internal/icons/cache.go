package icons

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"kopiq/internal/appdata"
)

var (
	safeKey = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	mu      sync.Mutex
)

// KeyFromAppID turns a bundle id / app id into a filesystem-safe cache key.
func KeyFromAppID(appID string) string {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return ""
	}
	return safeKey.ReplaceAllString(appID, "_")
}

// Dir returns the on-disk icon cache directory.
func Dir() (string, error) {
	root, err := appdata.EnsureLayout()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "icons"), nil
}

// PathFor returns the PNG path for a cache key (may not exist yet).
func PathFor(key string) (string, error) {
	if key == "" {
		return "", nil
	}
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, key+".png"), nil
}

// Exists reports whether a cached PNG is present.
func Exists(key string) bool {
	p, err := PathFor(key)
	if err != nil || p == "" {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// Ensure caches the app icon for appID (bundle id). Safe to call often;
// returns the cache key when a file is available.
func Ensure(appID string) (string, error) {
	key := KeyFromAppID(appID)
	if key == "" {
		return "", nil
	}
	mu.Lock()
	defer mu.Unlock()

	path, err := PathFor(key)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		return key, nil
	}

	png, err := extractPNG(appID)
	if err != nil || len(png) == 0 {
		return "", err
	}
	if err := os.WriteFile(path, png, 0o644); err != nil {
		return "", err
	}
	return key, nil
}

// ReadPNG loads a cached icon by key.
func ReadPNG(key string) ([]byte, error) {
	path, err := PathFor(key)
	if err != nil || path == "" {
		return nil, err
	}
	return os.ReadFile(path)
}
