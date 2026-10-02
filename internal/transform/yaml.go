package transform

import (
	"encoding/json"
	"strings"

	"gopkg.in/yaml.v3"

	"kopiq/internal/domain"
)

type jsonToYAML struct{}

func (jsonToYAML) ID() string    { return "json.to_yaml" }
func (jsonToYAML) Label() string { return "JSON → YAML" }
func (jsonToYAML) Group() string { return "json" }

func (jsonToYAML) Applies(item *domain.ClipboardItem) bool {
	if item == nil || isSuspected(item) {
		return false
	}
	return looksJSON(textOf(item))
}

func (t jsonToYAML) Apply(item *domain.ClipboardItem) Result {
	raw := strings.TrimSpace(textOf(item))
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: "Invalid JSON: " + err.Error()}
	}
	b, err := yaml.Marshal(v)
	if err != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: err.Error()}
	}
	out := strings.TrimRight(string(b), "\n")
	return Result{ID: t.ID(), Label: t.Label(), Kind: "payload", Text: out}
}

type yamlToJSON struct{}

func (yamlToJSON) ID() string    { return "yaml.to_json" }
func (yamlToJSON) Label() string { return "YAML → JSON" }
func (yamlToJSON) Group() string { return "json" }

func (yamlToJSON) Applies(item *domain.ClipboardItem) bool {
	if item == nil || isSuspected(item) {
		return false
	}
	return looksYAML(textOf(item))
}

func (t yamlToJSON) Apply(item *domain.ClipboardItem) Result {
	raw := strings.TrimSpace(textOf(item))
	var v any
	if err := yaml.Unmarshal([]byte(raw), &v); err != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: "Invalid YAML: " + err.Error()}
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: err.Error()}
	}
	return Result{ID: t.ID(), Label: t.Label(), Kind: "payload", Text: string(b)}
}

type jsonValidate struct{}

func (jsonValidate) ID() string    { return "json.validate" }
func (jsonValidate) Label() string { return "Validate JSON" }
func (jsonValidate) Group() string { return "json" }

func (jsonValidate) Applies(item *domain.ClipboardItem) bool {
	if item == nil || isSuspected(item) {
		return false
	}
	s := strings.TrimSpace(textOf(item))
	if len(s) < 2 {
		return false
	}
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}

func (t jsonValidate) Apply(item *domain.ClipboardItem) Result {
	raw := strings.TrimSpace(textOf(item))
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return Result{
			ID:    t.ID(),
			Label: t.Label(),
			Kind:  "payload",
			Text:  "Invalid JSON\n\n" + err.Error(),
		}
	}
	return Result{
		ID:    t.ID(),
		Label: t.Label(),
		Kind:  "payload",
		Text:  "Valid JSON",
	}
}

func looksYAML(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || looksJSON(s) {
		return false
	}
	// Avoid plain single-line identifiers / URLs.
	if !strings.Contains(s, ":") && !strings.Contains(s, "- ") && !strings.Contains(s, "\n") {
		return false
	}
	var v any
	if err := yaml.Unmarshal([]byte(s), &v); err != nil {
		return false
	}
	switch v.(type) {
	case map[string]any, []any:
		return true
	default:
		// yaml.v3 may decode scalars; require structured docs.
		return false
	}
}
