//go:build !darwin && !windows

package icons

func platformDisplayName(appID string) string { return "" }

func platformResolveByName(name string) (bundleID, displayName string, ok bool) {
	return "", "", false
}
