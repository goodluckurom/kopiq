package normalize

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"strings"

	_ "image/gif"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"kopiq/internal/domain"
)

const maxImageBytes = 25 * 1024 * 1024
const thumbLongEdge = 320
const thumbQuality = 80
const jpegQuality = 90

// PrepareImage decodes clipboard image bytes, re-encodes a canonical original,
// and fills width/height/mime on the item. For oversized images it keeps only
// a thumbnail (ImageKept=false) and hashes thumb+dimensions.
func PrepareImage(item *domain.CanonicalItem, originalType string) error {
	if len(item.ImageBytes) == 0 {
		return fmt.Errorf("empty image")
	}

	img, format, err := image.Decode(bytes.NewReader(item.ImageBytes))
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return fmt.Errorf("invalid image dimensions")
	}
	item.Width = w
	item.Height = h

	srcType := strings.ToLower(originalType)
	if srcType == "" {
		srcType = strings.ToLower(format)
	}

	lossy := srcType == "jpeg" || srcType == "jpg"
	var encoded []byte
	var mime string
	var ext string

	if lossy && !hasAlpha(img) {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
			return err
		}
		encoded = buf.Bytes()
		mime = "image/jpeg"
		ext = ".jpg"
	} else {
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return err
		}
		encoded = buf.Bytes()
		mime = "image/png"
		ext = ".png"
	}

	thumb, err := EncodeThumb(img)
	if err != nil {
		return err
	}

	if len(encoded) > maxImageBytes {
		// Oversized: keep thumbnail only; hash thumb + dimensions for dedupe.
		item.ImageBytes = thumb
		item.MimeType = "image/jpeg"
		item.ByteSize = int64(len(thumb))
		item.ImageKept = false
		item.ContentHash = hashBytes(append(thumb, []byte(fmt.Sprintf("\n%d\n%d", w, h))...))
		item.TextPreview = fmt.Sprintf("%d × %d · original not kept", w, h)
		item.MetadataJSON = mergeJSONMeta(item.MetadataJSON, map[string]any{
			"original_type": srcType,
			"original_ext":  ext,
			"oversized":     true,
			"thumb_only":    true,
		})
		return nil
	}

	item.ImageBytes = encoded
	item.MimeType = mime
	item.ByteSize = int64(len(encoded))
	item.ImageKept = true
	item.ContentHash = hashBytes(encoded)
	item.TextPreview = fmt.Sprintf("%d × %d", w, h)
	item.MetadataJSON = mergeJSONMeta(item.MetadataJSON, map[string]any{
		"original_type": srcType,
		"stored_ext":    strings.TrimPrefix(ext, "."),
	})
	_ = thumb // thumb generated async for kept originals
	return nil
}

// EncodeThumb writes a JPEG whose longest edge is thumbLongEdge.
func EncodeThumb(img image.Image) ([]byte, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("invalid bounds")
	}
	long := w
	if h > long {
		long = h
	}
	nw, nh := w, h
	if long > thumbLongEdge {
		scale := float64(thumbLongEdge) / float64(long)
		nw = int(float64(w) * scale)
		nh = int(float64(h) * scale)
		if nw < 1 {
			nw = 1
		}
		if nh < 1 {
			nh = 1
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: thumbQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EncodeThumbFromBytes decodes stored original bytes then thumbs them.
func EncodeThumbFromBytes(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return EncodeThumb(img)
}

func hasAlpha(img image.Image) bool {
	switch img.(type) {
	case *image.NRGBA, *image.NRGBA64, *image.RGBA, *image.RGBA64:
		return true
	default:
		return false
	}
}

func mergeJSONMeta(existing string, patch map[string]any) string {
	var base map[string]any
	_ = json.Unmarshal([]byte(existing), &base)
	if base == nil {
		base = map[string]any{}
	}
	for k, v := range patch {
		base[k] = v
	}
	b, err := json.Marshal(base)
	if err != nil {
		return existing
	}
	return string(b)
}
