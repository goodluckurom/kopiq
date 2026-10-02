//go:build windows

package icons

import (
	"path/filepath"
	"strings"
)

func platformDisplayName(appID string) string {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return ""
	}
	base := filepath.Base(appID)
	ext := filepath.Ext(base)
	if ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	return base
}

// Name lookup is limited on Windows without a Start Menu indexer; return false
// so callers keep the typed name and fall back to initials until an exe path is given.
func platformResolveByName(name string) (bundleID, displayName string, ok bool) {
	return "", "", false
}
