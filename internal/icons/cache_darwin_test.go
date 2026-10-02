//go:build darwin

package icons

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureSafariIcon(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KOPIQ_DATA_DIR", dir)
	key, err := Ensure("com.apple.Safari")
	if err != nil {
		t.Skipf("Safari icon unavailable in this environment: %v", err)
	}
	if key == "" {
		t.Skip("Safari icon not extractable here")
	}
	p := filepath.Join(dir, "icons", key+".png")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 100 || string(b[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("not a png: len=%d", len(b))
	}
}
