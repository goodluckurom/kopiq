package transform

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"kopiq/internal/domain"
)

type jsonToGo struct{}

func (jsonToGo) ID() string    { return "json.to_go" }
func (jsonToGo) Label() string { return "JSON → Go struct" }
func (jsonToGo) Group() string { return "generate" }

func (jsonToGo) Applies(item *domain.ClipboardItem) bool {
	if item == nil || isSuspected(item) {
		return false
	}
	return looksJSON(textOf(item))
}

func (t jsonToGo) Apply(item *domain.ClipboardItem) Result {
	raw := strings.TrimSpace(textOf(item))
	var v any
	if err := jsonUnmarshal(raw, &v); err != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: "Invalid JSON: " + err.Error()}
	}
	code, err := generateGo("Root", v)
	if err != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: err.Error()}
	}
	return Result{ID: t.ID(), Label: t.Label(), Kind: "payload", Text: code}
}

type jsonToTS struct{}

func (jsonToTS) ID() string    { return "json.to_ts" }
func (jsonToTS) Label() string { return "JSON → TypeScript" }
func (jsonToTS) Group() string { return "generate" }

func (jsonToTS) Applies(item *domain.ClipboardItem) bool {
	if item == nil || isSuspected(item) {
		return false
	}
	return looksJSON(textOf(item))
}

func (t jsonToTS) Apply(item *domain.ClipboardItem) Result {
	raw := strings.TrimSpace(textOf(item))
	var v any
	if err := jsonUnmarshal(raw, &v); err != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: "Invalid JSON: " + err.Error()}
	}
	code, err := generateTS("Root", v)
	if err != nil {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: err.Error()}
	}
	return Result{ID: t.ID(), Label: t.Label(), Kind: "payload", Text: code}
}

func generateGo(name string, v any) (string, error) {
	var b strings.Builder
	seen := map[string]bool{}
	switch x := v.(type) {
	case map[string]any:
		if err := writeGoStruct(&b, name, x, seen); err != nil {
			return "", err
		}
	case []any:
		elem, err := ensureGoSliceElem(&b, name, x, seen)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "type %s []%s\n", exportIdent(name), elem)
	default:
		return "", fmt.Errorf("top-level JSON must be an object or array")
	}
	return strings.TrimSpace(b.String()), nil
}

func writeGoStruct(b *strings.Builder, name string, m map[string]any, seen map[string]bool) error {
	name = exportIdent(name)
	if seen[name] {
		return nil
	}
	seen[name] = true
	keys := sortedKeys(m)

	for _, k := range keys {
		switch child := m[k].(type) {
		case map[string]any:
			if err := writeGoStruct(b, k, child, seen); err != nil {
				return err
			}
		case []any:
			if _, err := ensureGoSliceElem(b, k, child, seen); err != nil {
				return err
			}
		}
	}

	fmt.Fprintf(b, "type %s struct {\n", name)
	for _, k := range keys {
		field := exportIdent(k)
		typ, err := goTypeName(b, field, m[k], seen)
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "\t%s %s `json:%q`\n", field, typ, k)
	}
	b.WriteString("}\n\n")
	return nil
}

func ensureGoSliceElem(b *strings.Builder, name string, arr []any, seen map[string]bool) (string, error) {
	if len(arr) == 0 {
		return "any", nil
	}
	first := arr[0]
	for _, el := range arr[1:] {
		if kindOf(el) != kindOf(first) {
			return "", fmt.Errorf("array has mixed types, so Kopiq did not invent a struct")
		}
	}
	if _, ok := first.(map[string]any); ok {
		elem := singular(name)
		if err := writeGoStruct(b, elem, mergeObjectKeys(arr), seen); err != nil {
			return "", err
		}
		return exportIdent(elem), nil
	}
	return goTypeName(b, name+"Item", first, seen)
}

func goTypeName(b *strings.Builder, hint string, v any, seen map[string]bool) (string, error) {
	switch x := v.(type) {
	case nil:
		return "*string", nil
	case bool:
		return "bool", nil
	case float64:
		if x == float64(int64(x)) && x >= -9e15 && x <= 9e15 {
			return "int64", nil
		}
		return "float64", nil
	case string:
		return "string", nil
	case map[string]any:
		name := exportIdent(hint)
		if err := writeGoStruct(b, name, x, seen); err != nil {
			return "", err
		}
		return name, nil
	case []any:
		elem, err := ensureGoSliceElem(b, hint, x, seen)
		if err != nil {
			return "", err
		}
		return "[]" + elem, nil
	default:
		return "any", nil
	}
}

func generateTS(name string, v any) (string, error) {
	var b strings.Builder
	seen := map[string]bool{}
	switch x := v.(type) {
	case map[string]any:
		if err := writeTSInterface(&b, name, x, seen); err != nil {
			return "", err
		}
	case []any:
		if len(x) == 0 {
			fmt.Fprintf(&b, "export type %s = unknown[]\n", exportIdent(name))
			break
		}
		for _, el := range x[1:] {
			if kindOf(el) != kindOf(x[0]) {
				return "", fmt.Errorf("array has mixed types, so Kopiq did not invent a type")
			}
		}
		if _, ok := x[0].(map[string]any); ok {
			elem := singular(name)
			if err := writeTSInterface(&b, elem, mergeObjectKeys(x), seen); err != nil {
				return "", err
			}
			fmt.Fprintf(&b, "export type %s = %s[]\n", exportIdent(name), exportIdent(elem))
		} else {
			t, _, err := tsTypeName(&b, name+"Item", x[0], seen)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&b, "export type %s = %s[]\n", exportIdent(name), t)
		}
	default:
		return "", fmt.Errorf("top-level JSON must be an object or array")
	}
	return strings.TrimSpace(b.String()), nil
}

func writeTSInterface(b *strings.Builder, name string, m map[string]any, seen map[string]bool) error {
	name = exportIdent(name)
	if seen[name] {
		return nil
	}
	seen[name] = true
	keys := sortedKeys(m)

	for _, k := range keys {
		switch child := m[k].(type) {
		case map[string]any:
			if err := writeTSInterface(b, k, child, seen); err != nil {
				return err
			}
		case []any:
			if len(child) > 0 {
				if _, ok := child[0].(map[string]any); ok {
					if err := writeTSInterface(b, singular(k), mergeObjectKeys(child), seen); err != nil {
						return err
					}
				}
			}
		}
	}

	fmt.Fprintf(b, "export interface %s {\n", name)
	for _, k := range keys {
		field := tsField(k)
		typ, optional, err := tsTypeName(b, k, m[k], seen)
		if err != nil {
			return err
		}
		opt := ""
		if optional {
			opt = "?"
		}
		fmt.Fprintf(b, "  %s%s: %s\n", field, opt, typ)
	}
	b.WriteString("}\n\n")
	return nil
}

func tsTypeName(b *strings.Builder, hint string, v any, seen map[string]bool) (string, bool, error) {
	switch x := v.(type) {
	case nil:
		return "null", true, nil
	case bool:
		return "boolean", false, nil
	case float64:
		return "number", false, nil
	case string:
		return "string", false, nil
	case map[string]any:
		name := exportIdent(hint)
		if err := writeTSInterface(b, name, x, seen); err != nil {
			return "", false, err
		}
		return name, false, nil
	case []any:
		if len(x) == 0 {
			return "unknown[]", false, nil
		}
		for _, el := range x[1:] {
			if kindOf(el) != kindOf(x[0]) {
				return "", false, fmt.Errorf("array has mixed types, so Kopiq did not invent a type")
			}
		}
		if _, ok := x[0].(map[string]any); ok {
			elem := exportIdent(singular(hint))
			if err := writeTSInterface(b, elem, mergeObjectKeys(x), seen); err != nil {
				return "", false, err
			}
			return elem + "[]", false, nil
		}
		inner, _, err := tsTypeName(b, hint+"Item", x[0], seen)
		return inner + "[]", false, err
	default:
		return "unknown", false, nil
	}
}

func mergeObjectKeys(arr []any) map[string]any {
	out := map[string]any{}
	for _, el := range arr {
		m, ok := el.(map[string]any)
		if !ok {
			continue
		}
		for k, v := range m {
			if _, exists := out[k]; !exists || (out[k] == nil && v != nil) {
				out[k] = v
			}
		}
	}
	return out
}

func kindOf(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case float64:
		return "number"
	case string:
		return "string"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func exportIdent(s string) string {
	s = sanitizeIdent(s)
	if s == "" {
		return "Root"
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}

func tsField(s string) string {
	id := sanitizeIdent(s)
	if id == "" {
		return "field"
	}
	r, size := utf8.DecodeRuneInString(id)
	return string(unicode.ToLower(r)) + id[size:]
}

func sanitizeIdent(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '_' || r == '-' || r == ' ':
			b.WriteRune('_')
		}
	}
	parts := strings.Split(strings.Trim(b.String(), "_"), "_")
	var camel strings.Builder
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			camel.WriteString(p)
			continue
		}
		r, size := utf8.DecodeRuneInString(p)
		camel.WriteRune(unicode.ToUpper(r))
		camel.WriteString(p[size:])
	}
	out := camel.String()
	if out == "" {
		return "Field"
	}
	if unicode.IsDigit(rune(out[0])) {
		return "N" + out
	}
	return out
}

func singular(name string) string {
	name = exportIdent(name)
	if strings.HasSuffix(name, "ies") && len(name) > 3 {
		return name[:len(name)-3] + "y"
	}
	if strings.HasSuffix(name, "s") && len(name) > 1 && !strings.HasSuffix(name, "ss") {
		return name[:len(name)-1]
	}
	return name + "Item"
}
