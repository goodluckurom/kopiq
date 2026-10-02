//go:build !darwin && !windows

package icons

func extractPNG(bundleID string) ([]byte, error) {
	return nil, nil
}
