package transform

import "kopiq/internal/domain"

// Result is either a new text payload or an error reason.
type Result struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Text    string `json:"text,omitempty"`
	Error   string `json:"error,omitempty"`
	Kind    string `json:"kind"` // payload | error
}

// Action describes an available transform for the UI.
type Action struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Group string `json:"group"` // json | case | encode | generate
}

// Transformer converts item content.
type Transformer interface {
	ID() string
	Label() string
	Group() string
	Applies(item *domain.ClipboardItem) bool
	Apply(item *domain.ClipboardItem) Result
}

// Registry holds transformers in display order.
type Registry struct {
	all []Transformer
}

// Default returns the M4 transformers.
func Default() *Registry {
	return &Registry{all: []Transformer{
		jsonPretty{},
		jsonMinify{},
		jsonValidate{},
		jsonToYAML{},
		yamlToJSON{},
		jsonToGo{},
		jsonToTS{},
		upperCase{},
		lowerCase{},
		camelCaseT{},
		pascalCaseT{},
		snakeCaseT{},
		kebabCaseT{},
		trimWS{},
		dedupeLines{},
		sortLinesT{},
		b64Encode{},
		b64Decode{},
		urlEncode{},
		urlDecode{},
		htmlEscapeT{},
		htmlUnescapeT{},
		colorView{},
		timestampView{},
		jwtView{},
	}}
}

// ActionsFor lists transforms that apply to this item.
func (r *Registry) ActionsFor(item *domain.ClipboardItem) []Action {
	var out []Action
	for _, t := range r.all {
		if t.Applies(item) {
			out = append(out, Action{ID: t.ID(), Label: t.Label(), Group: t.Group()})
		}
	}
	return out
}

// Apply runs a transform by id.
func (r *Registry) Apply(item *domain.ClipboardItem, id string) Result {
	for _, t := range r.all {
		if t.ID() == id {
			if !t.Applies(item) {
				return Result{ID: id, Label: t.Label(), Kind: "error", Error: "This transform does not apply to this item."}
			}
			return t.Apply(item)
		}
	}
	return Result{ID: id, Kind: "error", Error: "Unknown transform."}
}

func textOf(item *domain.ClipboardItem) string {
	if item.TextContent != "" {
		return item.TextContent
	}
	return item.TextPreview
}

func isSuspected(item *domain.ClipboardItem) bool {
	return item.Sensitivity == "suspected" || item.Masked
}
