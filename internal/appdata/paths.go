package appdata

import (
	"os"
	"path/filepath"
)

const appName = "Kopiq"

// Root returns the application data directory.
// Override with KOPIQ_DATA_DIR for tests or sandboxed runs.
// Default: ~/Library/Application Support/Kopiq on macOS,
// %AppData%/Kopiq on Windows, and ~/.local/share/Kopiq elsewhere.
func Root() (string, error) {
	if override := os.Getenv("KOPIQ_DATA_DIR"); override != "" {
		return filepath.Clean(override), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch {
	case fileExists(filepath.Join(home, "Library", "Application Support")):
		return filepath.Join(home, "Library", "Application Support", appName), nil
	case os.Getenv("APPDATA") != "":
		return filepath.Join(os.Getenv("APPDATA"), appName), nil
	default:
		return filepath.Join(home, ".local", "share", appName), nil
	}
}

// EnsureLayout creates the app support directory and blobs/icons subdirs.
func EnsureLayout() (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	for _, sub := range []string{
		"",
		"blobs",
		"blobs/original",
		"blobs/thumbs",
		"icons",
	} {
		p := root
		if sub != "" {
			p = filepath.Join(root, sub)
		}
		if err := os.MkdirAll(p, 0o755); err != nil {
			return "", err
		}
	}
	return root, nil
}

// DBPath returns the SQLite file path.
func DBPath() (string, error) {
	root, err := EnsureLayout()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "kopiq.db"), nil
}

// BlobRoot returns the blobs directory.
func BlobRoot() (string, error) {
	root, err := EnsureLayout()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "blobs"), nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
