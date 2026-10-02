package transform

import (
	"encoding/base64"
	"html"
	"net/url"
	"strings"
	"unicode/utf8"

	"kopiq/internal/domain"
)

type b64Encode struct{}

func (b64Encode) ID() string    { return "encode.base64" }
func (b64Encode) Label() string { return "Base64 encode" }
func (b64Encode) Group() string { return "encode" }
func (b64Encode) Applies(item *domain.ClipboardItem) bool {
	return textTransformOK(item)
}
func (t b64Encode) Apply(item *domain.ClipboardItem) Result {
	return payload(t, base64.StdEncoding.EncodeToString([]byte(textOf(item))))
}

type b64Decode struct{}

func (b64Decode) ID() string    { return "encode.base64_decode" }
func (b64Decode) Label() string { return "Base64 decode" }
func (b64Decode) Group() string { return "encode" }
func (b64Decode) Applies(item *domain.ClipboardItem) bool {
	if !textTransformOK(item) {
		return false
	}
	s := strings.TrimSpace(textOf(item))
	if len(s) < 8 || len(s)%4 != 0 {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}
func (t b64Decode) Apply(item *domain.ClipboardItem) Result {
	s := strings.TrimSpace(textOf(item))
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: "Not valid Base64."}
	}
	if !utf8.Valid(b) {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: "Decoded bytes are not UTF-8 text."}
	}
	return payload(t, string(b))
}

type urlEncode struct{}

func (urlEncode) ID() string    { return "encode.url" }
func (urlEncode) Label() string { return "URL encode" }
func (urlEncode) Group() string { return "encode" }
func (urlEncode) Applies(item *domain.ClipboardItem) bool {
	s := textOf(item)
	return textTransformOK(item) && !strings.Contains(s, "\n") && len(s) < 4096
}
func (t urlEncode) Apply(item *domain.ClipboardItem) Result {
	return payload(t, url.QueryEscape(textOf(item)))
}

type urlDecode struct{}

func (urlDecode) ID() string    { return "encode.url_decode" }
func (urlDecode) Label() string { return "URL decode" }
func (urlDecode) Group() string { return "encode" }
func (urlDecode) Applies(item *domain.ClipboardItem) bool {
	s := strings.TrimSpace(textOf(item))
	return textTransformOK(item) && strings.Contains(s, "%") && !strings.Contains(s, "\n")
}
func (t urlDecode) Apply(item *domain.ClipboardItem) Result {
	out, err := url.QueryUnescape(strings.TrimSpace(textOf(item)))
	if err != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: "Could not URL-decode."}
	}
	return payload(t, out)
}

type htmlEscapeT struct{}

func (htmlEscapeT) ID() string    { return "encode.html" }
func (htmlEscapeT) Label() string { return "HTML escape" }
func (htmlEscapeT) Group() string { return "encode" }
func (htmlEscapeT) Applies(item *domain.ClipboardItem) bool {
	return textTransformOK(item)
}
func (t htmlEscapeT) Apply(item *domain.ClipboardItem) Result {
	return payload(t, html.EscapeString(textOf(item)))
}

type htmlUnescapeT struct{}

func (htmlUnescapeT) ID() string    { return "encode.html_unescape" }
func (htmlUnescapeT) Label() string { return "HTML unescape" }
func (htmlUnescapeT) Group() string { return "encode" }
func (htmlUnescapeT) Applies(item *domain.ClipboardItem) bool {
	s := textOf(item)
	return textTransformOK(item) && (strings.Contains(s, "&lt;") || strings.Contains(s, "&amp;") || strings.Contains(s, "&quot;") || strings.Contains(s, "&#"))
}
func (t htmlUnescapeT) Apply(item *domain.ClipboardItem) Result {
	return payload(t, html.UnescapeString(textOf(item)))
}
