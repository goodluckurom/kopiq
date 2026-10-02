package transform

import (
	"strings"
	"unicode"

	"kopiq/internal/domain"
)

type upperCase struct{}

func (upperCase) ID() string    { return "case.upper" }
func (upperCase) Label() string { return "UPPERCASE" }
func (upperCase) Group() string { return "case" }
func (upperCase) Applies(item *domain.ClipboardItem) bool {
	return textTransformOK(item)
}
func (t upperCase) Apply(item *domain.ClipboardItem) Result {
	return payload(t, strings.ToUpper(textOf(item)))
}

type lowerCase struct{}

func (lowerCase) ID() string    { return "case.lower" }
func (lowerCase) Label() string { return "lowercase" }
func (lowerCase) Group() string { return "case" }
func (lowerCase) Applies(item *domain.ClipboardItem) bool {
	return textTransformOK(item)
}
func (t lowerCase) Apply(item *domain.ClipboardItem) Result {
	return payload(t, strings.ToLower(textOf(item)))
}

type camelCaseT struct{}

func (camelCaseT) ID() string    { return "case.camel" }
func (camelCaseT) Label() string { return "camelCase" }
func (camelCaseT) Group() string { return "case" }
func (camelCaseT) Applies(item *domain.ClipboardItem) bool {
	return identTransformOK(item)
}
func (t camelCaseT) Apply(item *domain.ClipboardItem) Result {
	s := strings.TrimSpace(textOf(item))
	if err := refuseMultiLine(s); err != "" {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: err}
	}
	return payload(t, toCamel(s, false))
}

type pascalCaseT struct{}

func (pascalCaseT) ID() string    { return "case.pascal" }
func (pascalCaseT) Label() string { return "PascalCase" }
func (pascalCaseT) Group() string { return "case" }
func (pascalCaseT) Applies(item *domain.ClipboardItem) bool {
	return identTransformOK(item)
}
func (t pascalCaseT) Apply(item *domain.ClipboardItem) Result {
	s := strings.TrimSpace(textOf(item))
	if err := refuseMultiLine(s); err != "" {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: err}
	}
	return payload(t, toCamel(s, true))
}

type snakeCaseT struct{}

func (snakeCaseT) ID() string    { return "case.snake" }
func (snakeCaseT) Label() string { return "snake_case" }
func (snakeCaseT) Group() string { return "case" }
func (snakeCaseT) Applies(item *domain.ClipboardItem) bool {
	return identTransformOK(item)
}
func (t snakeCaseT) Apply(item *domain.ClipboardItem) Result {
	s := strings.TrimSpace(textOf(item))
	if err := refuseMultiLine(s); err != "" {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: err}
	}
	return payload(t, toSnakeOrKebab(s, '_'))
}

type kebabCaseT struct{}

func (kebabCaseT) ID() string    { return "case.kebab" }
func (kebabCaseT) Label() string { return "kebab-case" }
func (kebabCaseT) Group() string { return "case" }
func (kebabCaseT) Applies(item *domain.ClipboardItem) bool {
	return identTransformOK(item)
}
func (t kebabCaseT) Apply(item *domain.ClipboardItem) Result {
	s := strings.TrimSpace(textOf(item))
	if err := refuseMultiLine(s); err != "" {
		return Result{ID: t.ID(), Label: t.Label(), Kind: "error", Error: err}
	}
	return payload(t, toSnakeOrKebab(s, '-'))
}

type trimWS struct{}

func (trimWS) ID() string    { return "text.trim" }
func (trimWS) Label() string { return "Trim whitespace" }
func (trimWS) Group() string { return "case" }
func (trimWS) Applies(item *domain.ClipboardItem) bool {
	return textTransformOK(item)
}
func (t trimWS) Apply(item *domain.ClipboardItem) Result {
	return payload(t, strings.TrimSpace(textOf(item)))
}

type dedupeLines struct{}

func (dedupeLines) ID() string    { return "text.dedupe_lines" }
func (dedupeLines) Label() string { return "Remove duplicate lines" }
func (dedupeLines) Group() string { return "case" }
func (dedupeLines) Applies(item *domain.ClipboardItem) bool {
	s := textOf(item)
	return textTransformOK(item) && strings.Contains(s, "\n")
}
func (t dedupeLines) Apply(item *domain.ClipboardItem) Result {
	lines := strings.Split(textOf(item), "\n")
	seen := map[string]bool{}
	var out []string
	for _, line := range lines {
		if seen[line] {
			continue
		}
		seen[line] = true
		out = append(out, line)
	}
	return payload(t, strings.Join(out, "\n"))
}

type sortLinesT struct{}

func (sortLinesT) ID() string    { return "text.sort_lines" }
func (sortLinesT) Label() string { return "Sort lines" }
func (sortLinesT) Group() string { return "case" }
func (sortLinesT) Applies(item *domain.ClipboardItem) bool {
	s := textOf(item)
	return textTransformOK(item) && strings.Contains(s, "\n")
}
func (t sortLinesT) Apply(item *domain.ClipboardItem) Result {
	lines := strings.Split(textOf(item), "\n")
	trailing := len(lines) > 0 && lines[len(lines)-1] == ""
	body := lines
	if trailing {
		body = lines[:len(lines)-1]
	}
	sorted := append([]string{}, body...)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	if trailing {
		sorted = append(sorted, "")
	}
	return payload(t, strings.Join(sorted, "\n"))
}

func textTransformOK(item *domain.ClipboardItem) bool {
	if item == nil || isSuspected(item) {
		return false
	}
	s := textOf(item)
	return strings.TrimSpace(s) != "" && item.Kind != "image" && item.Kind != "files"
}

func identTransformOK(item *domain.ClipboardItem) bool {
	if !textTransformOK(item) {
		return false
	}
	s := strings.TrimSpace(textOf(item))
	if strings.Contains(s, "\n") {
		return false
	}
	return looksLikeIdent(s) || strings.ContainsAny(s, "_- ")
}

func refuseMultiLine(s string) string {
	if strings.Contains(s, "\n") {
		return "Identifier case transforms need a single line."
	}
	return ""
}

func looksLikeIdent(s string) bool {
	if s == "" || len(s) > 120 {
		return false
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func splitIdentParts(s string) []string {
	var parts []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		parts = append(parts, strings.ToLower(cur.String()))
		cur.Reset()
	}
	runes := []rune(strings.TrimSpace(s))
	for i, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if i > 0 && unicode.IsUpper(r) {
				prev := runes[i-1]
				nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
					flush()
				}
			}
			cur.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return parts
}

func toCamel(s string, pascal bool) string {
	parts := splitIdentParts(s)
	if len(parts) == 0 {
		return s
	}
	var b strings.Builder
	for i, p := range parts {
		if i == 0 && !pascal {
			b.WriteString(p)
			continue
		}
		if p == "" {
			continue
		}
		r := []rune(p)
		r[0] = unicode.ToUpper(r[0])
		b.WriteString(string(r))
	}
	return b.String()
}

func toSnakeOrKebab(s string, sep rune) string {
	parts := splitIdentParts(s)
	return strings.Join(parts, string(sep))
}

func payload(t Transformer, text string) Result {
	return Result{ID: t.ID(), Label: t.Label(), Kind: "payload", Text: text}
}

func viewResult(t Transformer, text string) Result {
	return Result{ID: t.ID(), Label: t.Label(), Kind: "view", Text: text}
}
