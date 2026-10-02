package normalize

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"kopiq/internal/domain"
)

func TestPrepareImagePNG(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			img.Set(x, y, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	item := &domain.CanonicalItem{
		Kind:         "image",
		ImageBytes:   buf.Bytes(),
		DetectedType: "image",
		Sensitivity:  "none",
		ImageKept:    true,
	}
	if err := PrepareImage(item, "png"); err != nil {
		t.Fatal(err)
	}
	if item.Width != 40 || item.Height != 20 {
		t.Fatalf("dims %dx%d", item.Width, item.Height)
	}
	if !item.ImageKept {
		t.Fatal("expected kept")
	}
	if item.MimeType != "image/png" {
		t.Fatalf("mime %s", item.MimeType)
	}
	if item.ContentHash == "" {
		t.Fatal("empty hash")
	}
	thumb, err := EncodeThumbFromBytes(item.ImageBytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(thumb) == 0 {
		t.Fatal("empty thumb")
	}
}
