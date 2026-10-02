package normalize

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"

	"kopiq/internal/domain"
)

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestFromRawPrefersImageWhenScreenshotFilenameHasSpaces(t *testing.T) {
	item, ok := FromRaw(domain.RawPayload{
		Text:       "Screenshot 2026-10-01 at 6.42.21 AM.png",
		ImageBytes: tinyPNG(t),
		ImageType:  "png",
	})
	if !ok {
		t.Fatal("expected item")
	}
	if item.Kind != "image" {
		t.Fatalf("kind=%q want image", item.Kind)
	}
	if item.Title != "Screenshot 2026-10-01 at 6.42.21 AM.png" {
		t.Fatalf("title=%q", item.Title)
	}
}

func TestPreferImageOverText(t *testing.T) {
	if !preferImageOverText("", 100) {
		t.Fatal("empty should prefer image")
	}
	if !preferImageOverText("Screenshot 2026-10-01 at 6.42.21 AM.png", 100) {
		t.Fatal("screenshot filename should prefer image")
	}
	if !preferImageOverText("https://cdn.example.com/photo.jpg", 500) {
		t.Fatal("image URL companion should prefer image")
	}
	if !preferImageOverText("A short alt caption\nsecond line", 500) {
		t.Fatal("short multiline caption should still prefer image")
	}
	// Substantial image bytes always win, even with long body text.
	long := strings.Repeat("word ", 80)
	if !preferImageOverText(long, 20*1024) {
		t.Fatal("large image should prefer image over long text")
	}
	// Tiny decorative icon + long text → keep text.
	if preferImageOverText(long, 500) {
		t.Fatal("tiny image with long text should prefer text")
	}
}

func TestFromRawPrefersCopiedImageWithURL(t *testing.T) {
	item, ok := FromRaw(domain.RawPayload{
		Text:       "https://example.com/assets/hero.png",
		ImageBytes: tinyPNG(t),
		ImageType:  "png",
	})
	if !ok {
		t.Fatal("expected item")
	}
	if item.Kind != "image" {
		t.Fatalf("kind=%q want image (copied image with URL companion)", item.Kind)
	}
}

func TestResolveScreenshotOnDisk(t *testing.T) {
	p, ok := ResolveScreenshotOnDisk("Screenshot 2026-10-01 at 6.48.10 AM")
	if !ok {
		t.Skip("no matching Desktop screenshot in this environment")
	}
	item, ok := FromImageFile(p)
	if !ok {
		t.Fatalf("FromImageFile failed for %s", p)
	}
	if item.Kind != "image" {
		t.Fatalf("kind=%s", item.Kind)
	}
	if item.Width == 0 || item.Height == 0 {
		t.Fatalf("missing dims")
	}
}
