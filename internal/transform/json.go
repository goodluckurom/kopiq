package transform

import (
	"bytes"
	"encoding/json"
	"strings"

	"kopiq/internal/domain"
)

func jsonUnmarshal(raw string, v *any) error {
	return json.Unmarshal([]byte(raw), v)
}

type jsonPretty struct{}

func (jsonPretty) ID() string    { return "json.pretty" }
func (jsonPretty) Label() string { return "Pretty JSON" }
func (jsonPretty) Group() string { return "json" }

func (jsonPretty) Applies(item *domain.ClipboardItem) bool {
	if item == nil || isSuspected(item) {
		return false
	}
	return looksJSON(textOf(item))
}

func (t jsonPretty) Apply(item *domain.ClipboardItem) Result {
	raw := strings.TrimSpace(textOf(item))
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: "Invalid JSON: " + err.Error()}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: err.Error()}
	}
	out := strings.TrimRight(buf.String(), "\n")
	return Result{ID: t.ID(), Label: t.Label(), Kind: "payload", Text: out}
}

type jsonMinify struct{}

func (jsonMinify) ID() string    { return "json.minify" }
func (jsonMinify) Label() string { return "Minify JSON" }
func (jsonMinify) Group() string { return "json" }

func (jsonMinify) Applies(item *domain.ClipboardItem) bool {
	if item == nil || isSuspected(item) {
		return false
	}
	return looksJSON(textOf(item))
}

func (t jsonMinify) Apply(item *domain.ClipboardItem) Result {
	raw := strings.TrimSpace(textOf(item))
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: "Invalid JSON: " + err.Error()}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: err.Error()}
	}
	return Result{ID: t.ID(), Label: t.Label(), Kind: "payload", Text: string(b)}
}

func looksJSON(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	if !(strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")) {
		return false
	}
	var v any
	return json.Unmarshal([]byte(s), &v) == nil
}
