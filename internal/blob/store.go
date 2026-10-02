package blob

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kopiq/internal/appdata"
)

// Relative path prefixes under the blobs root.
const (
	OriginalPrefix = "original"
	ThumbPrefix    = "thumbs"
)

// WriteOriginal writes normalized image bytes under blobs/original/{hash}{ext}.
// Returns the relative path stored in SQLite (e.g. original/abc.png).
func WriteOriginal(contentHash, ext string, data []byte) (rel string, err error) {
	ext = normalizeExt(ext)
	root, err := appdata.BlobRoot()
	if err != nil {
		return "", err
	}
	name := contentHash + ext
	rel = filepath.ToSlash(filepath.Join(OriginalPrefix, name))
	abs := filepath.Join(root, OriginalPrefix, name)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		return "", err
	}
	return rel, nil
}

// WriteThumb writes a JPEG thumbnail under blobs/thumbs/{hash}.jpg.
func WriteThumb(contentHash string, jpeg []byte) (rel string, err error) {
	root, err := appdata.BlobRoot()
	if err != nil {
		return "", err
	}
	name := contentHash + ".jpg"
	rel = filepath.ToSlash(filepath.Join(ThumbPrefix, name))
	abs := filepath.Join(root, ThumbPrefix, name)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, jpeg, 0o644); err != nil {
		return "", err
	}
	return rel, nil
}

// Abs resolves a relative blob path to an absolute file path.
func Abs(rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", fmt.Errorf("empty blob path")
	}
	if filepath.IsAbs(rel) {
		return rel, nil
	}
	root, err := appdata.BlobRoot()
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("invalid blob path")
	}
	return filepath.Join(root, clean), nil
}

// Read loads bytes for a relative blob path.
func Read(rel string) ([]byte, error) {
	abs, err := Abs(rel)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(abs)
}

// Remove deletes a relative blob path if present.
func Remove(rel string) {
	if rel == "" {
		return
	}
	abs, err := Abs(rel)
	if err != nil {
		return
	}
	_ = os.Remove(abs)
}

func normalizeExt(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" {
		return ".png"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	switch ext {
	case ".jpg", ".jpeg":
		return ".jpg"
	case ".png", ".webp", ".gif", ".bmp", ".tiff", ".tif":
		return ext
	default:
		return ".bin"
	}
}
