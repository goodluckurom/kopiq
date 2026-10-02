package transform

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"kopiq/internal/domain"
)

type colorView struct{}

func (colorView) ID() string    { return "view.color" }
func (colorView) Label() string { return "Color channels" }
func (colorView) Group() string { return "view" }
func (colorView) Applies(item *domain.ClipboardItem) bool {
	if item == nil {
		return false
	}
	if strings.EqualFold(item.DetectedType, "color") {
		return true
	}
	_, _, _, _, ok := parseHexColor(strings.TrimSpace(textOf(item)))
	return ok
}
func (t colorView) Apply(item *domain.ClipboardItem) Result {
	s := strings.TrimSpace(textOf(item))
	r, g, b, a, ok := parseHexColor(s)
	if !ok {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: "Not a hex color."}
	}
	var bld strings.Builder
	fmt.Fprintf(&bld, "HEX  %s\n", normalizeHex(r, g, b, a))
	fmt.Fprintf(&bld, "RGB  %d, %d, %d\n", r, g, b)
	fmt.Fprintf(&bld, "RGBA %d, %d, %d, %.2f\n", r, g, b, float64(a)/255)
	h, s2, l := rgbToHSL(r, g, b)
	fmt.Fprintf(&bld, "HSL  %.0f°, %.0f%%, %.0f%%\n", h, s2*100, l*100)
	fmt.Fprintf(&bld, "\nCopy forms:\n")
	fmt.Fprintf(&bld, "  %s\n", normalizeHex(r, g, b, a))
	fmt.Fprintf(&bld, "  rgb(%d, %d, %d)\n", r, g, b)
	fmt.Fprintf(&bld, "  hsl(%.0f, %.0f%%, %.0f%%)", h, s2*100, l*100)
	return viewResult(t, bld.String())
}

type timestampView struct{}

func (timestampView) ID() string    { return "view.timestamp" }
func (timestampView) Label() string { return "Timestamp" }
func (timestampView) Group() string { return "view" }
func (timestampView) Applies(item *domain.ClipboardItem) bool {
	s := strings.TrimSpace(textOf(item))
	return looksUnixTimestamp(s)
}
func (t timestampView) Apply(item *domain.ClipboardItem) Result {
	s := strings.TrimSpace(textOf(item))
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: "Not a timestamp."}
	}
	var sec, ms int64
	if len(s) >= 13 {
		ms = n
		sec = n / 1000
	} else {
		sec = n
		ms = n * 1000
	}
	utc := time.Unix(sec, 0).UTC()
	local := time.Unix(sec, 0).Local()
	var bld strings.Builder
	fmt.Fprintf(&bld, "Input   %s\n", s)
	fmt.Fprintf(&bld, "Seconds %d\n", sec)
	fmt.Fprintf(&bld, "Millis  %d\n", ms)
	fmt.Fprintf(&bld, "UTC     %s\n", utc.Format(time.RFC3339))
	fmt.Fprintf(&bld, "Local   %s\n", local.Format(time.RFC3339))
	fmt.Fprintf(&bld, "Relative %s", relativeFromNow(utc))
	return viewResult(t, bld.String())
}

type jwtView struct{}

func (jwtView) ID() string    { return "view.jwt" }
func (jwtView) Label() string { return "Inspect JWT" }
func (jwtView) Group() string { return "view" }
func (jwtView) Applies(item *domain.ClipboardItem) bool {
	if item == nil {
		return false
	}
	if strings.EqualFold(item.DetectedType, "jwt") {
		return true
	}
	return looksJWT(strings.TrimSpace(textOf(item)))
}
func (t jwtView) Apply(item *domain.ClipboardItem) Result {
	s := strings.TrimSpace(textOf(item))
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: "Not a JWT."}
	}
	header, err1 := decodeJWTPart(parts[0])
	payload, err2 := decodeJWTPart(parts[1])
	if err1 != nil || err2 != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: "Could not decode JWT segments."}
	}
	var bld strings.Builder
	bld.WriteString("Signature not verified.\n\n")
	bld.WriteString("Header\n")
	bld.WriteString(prettyJSONBytes(header))
	bld.WriteString("\n\nPayload\n")
	bld.WriteString(prettyJSONBytes(payload))
	if exp := jwtExp(payload); exp > 0 {
		when := time.Unix(exp, 0)
		bld.WriteString("\n\nExpiry\n")
		fmt.Fprintf(&bld, "  %s (%s)", when.UTC().Format(time.RFC3339), relativeFromNow(when.UTC()))
	}
	return viewResult(t, bld.String())
}

func parseHexColor(s string) (r, g, b, a int, ok bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "#") {
		return
	}
	h := strings.TrimPrefix(s, "#")
	a = 255
	switch len(h) {
	case 3:
		r = hexNibble(h[0])*17
		g = hexNibble(h[1]) * 17
		b = hexNibble(h[2]) * 17
		ok = r >= 0 && g >= 0 && b >= 0
	case 6:
		r, g, b = hexByte(h[0:2]), hexByte(h[2:4]), hexByte(h[4:6])
		ok = r >= 0 && g >= 0 && b >= 0
	case 8:
		r, g, b, a = hexByte(h[0:2]), hexByte(h[2:4]), hexByte(h[4:6]), hexByte(h[6:8])
		ok = r >= 0 && g >= 0 && b >= 0 && a >= 0
	}
	return
}

func hexNibble(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	default:
		return -1
	}
}

func hexByte(s string) int {
	hi, lo := hexNibble(s[0]), hexNibble(s[1])
	if hi < 0 || lo < 0 {
		return -1
	}
	return hi*16 + lo
}

func normalizeHex(r, g, b, a int) string {
	if a < 255 {
		return fmt.Sprintf("#%02X%02X%02X%02X", r, g, b, a)
	}
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

func rgbToHSL(r, g, b int) (h, s, l float64) {
	rf, gf, bf := float64(r)/255, float64(g)/255, float64(b)/255
	max := rf
	if gf > max {
		max = gf
	}
	if bf > max {
		max = bf
	}
	min := rf
	if gf < min {
		min = gf
	}
	if bf < min {
		min = bf
	}
	l = (max + min) / 2
	if max == min {
		return 0, 0, l
	}
	d := max - min
	if l > 0.5 {
		s = d / (2 - max - min)
	} else {
		s = d / (max + min)
	}
	switch max {
	case rf:
		h = (gf - bf) / d
		if gf < bf {
			h += 6
		}
	case gf:
		h = (bf-rf)/d + 2
	default:
		h = (rf-gf)/d + 4
	}
	h *= 60
	return h, s, l
}

func looksUnixTimestamp(s string) bool {
	if len(s) != 10 && len(s) != 13 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return false
	}
	sec := n
	if len(s) == 13 {
		sec = n / 1000
	}
	// 2001-01-01 .. 2100-01-01
	return sec >= 978307200 && sec <= 4102444800
}

func relativeFromNow(t time.Time) string {
	d := time.Until(t)
	past := d < 0
	if past {
		d = -d
	}
	var s string
	switch {
	case d < time.Minute:
		s = fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		s = fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		s = fmt.Sprintf("%.1fh", d.Hours())
	default:
		s = fmt.Sprintf("%.1fd", d.Hours()/24)
	}
	if past {
		return s + " ago"
	}
	return "in " + s
}

func looksJWT(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return false
	}
	_, err1 := decodeJWTPart(parts[0])
	_, err2 := decodeJWTPart(parts[1])
	return err1 == nil && err2 == nil
}

func decodeJWTPart(seg string) ([]byte, error) {
	seg = strings.TrimRight(seg, "=")
	if b, err := base64.RawURLEncoding.DecodeString(seg); err == nil {
		return b, nil
	}
	return base64.URLEncoding.DecodeString(seg)
}

func prettyJSONBytes(b []byte) string {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return string(b)
	}
	out, err := json.MarshalIndent(v, "  ", "  ")
	if err != nil {
		return string(b)
	}
	return "  " + string(out)
}

func jwtExp(payload []byte) int64 {
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return 0
	}
	switch v := m["exp"].(type) {
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	default:
		return 0
	}
}
