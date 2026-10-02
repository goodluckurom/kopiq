package icons

import "strings"

// AppInfo is a resolved application identity plus cached icon key.
type AppInfo struct {
	ApplicationID   string `json:"applicationId"`
	ApplicationName string `json:"applicationName"`
	IconCacheKey    string `json:"iconCacheKey,omitempty"`
	Found           bool   `json:"found"`
}

// Lookup resolves a display name and caches the icon for appID (bundle id on
// macOS, executable path on Windows). Found is true when an icon or name was
// obtained from the OS.
func Lookup(appID string) AppInfo {
	appID = strings.TrimSpace(appID)
	info := AppInfo{ApplicationID: appID}
	if appID == "" {
		return info
	}
	if name := platformDisplayName(appID); name != "" {
		info.ApplicationName = name
		info.Found = true
	}
	if key, err := Ensure(appID); err == nil && key != "" {
		info.IconCacheKey = key
		info.Found = true
	} else {
		info.IconCacheKey = KeyFromAppID(appID)
	}
	return info
}

// LookupByName finds an installed/running app by display name, caches its icon,
// and returns the resolved bundle id (or empty when unresolved).
func LookupByName(name string) AppInfo {
	name = strings.TrimSpace(name)
	info := AppInfo{ApplicationName: name}
	if name == "" {
		return info
	}
	bid, display, ok := platformResolveByName(name)
	if !ok || bid == "" {
		return info
	}
	info.ApplicationID = bid
	if display != "" {
		info.ApplicationName = display
	}
	info.Found = true
	if key, err := Ensure(bid); err == nil && key != "" {
		info.IconCacheKey = key
	} else {
		info.IconCacheKey = KeyFromAppID(bid)
	}
	return info
}

// Resolve accepts either a bundle id / app path or a display name.
// Bundle-id shaped queries are tried first; otherwise (and on miss) name lookup runs.
func Resolve(query string) AppInfo {
	query = strings.TrimSpace(query)
	if query == "" {
		return AppInfo{}
	}
	if looksLikeBundleID(query) {
		info := Lookup(query)
		if info.Found {
			return info
		}
	}
	if info := LookupByName(query); info.Found {
		return info
	}
	// Last resort: treat as id even when it does not look like one.
	return Lookup(query)
}

func looksLikeBundleID(s string) bool {
	if strings.ContainsAny(s, " /\\") {
		return false
	}
	if strings.Count(s, ".") < 1 {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}
