package detect

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"kopiq/internal/domain"
)

// LanguageFromExt maps a file name or path extension to a language id.
// .json stays empty here so the JSON detector can win first; callers that
// already know the content is not JSON may still force "json".
func LanguageFromExt(name string) string {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(name)))
	if base == "" {
		return ""
	}
	ext := strings.TrimPrefix(filepath.Ext(base), ".")
	switch base {
	case "dockerfile", "containerfile", "makefile", "cmakelists.txt", "rakefile", "gemfile", "procfile":
		return "shell"
	}
	switch ext {
	case "go":
		return "go"
	case "ts":
		return "typescript"
	case "tsx":
		return "tsx"
	case "js", "mjs", "cjs":
		return "javascript"
	case "jsx":
		return "jsx"
	case "py":
		return "python"
	case "rs":
		return "rust"
	case "dart":
		return "dart"
	case "swift":
		return "swift"
	case "kt", "kts":
		return "kotlin"
	case "sql":
		return "sql"
	case "html", "htm":
		return "html"
	case "css":
		return "css"
	case "yml", "yaml":
		return "yaml"
	case "sh", "bash", "zsh":
		return "shell"
	case "json":
		return "json"
	default:
		return ""
	}
}

type codeDetector struct {
	fileName string
}

func (codeDetector) Name() string { return "code" }

func (d codeDetector) Detect(item *domain.CanonicalItem) bool {
	if item.Kind != "text" && item.Kind != "" {
		return false
	}
	s := item.TextContent
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return false
	}

	// 1. Context file extension wins (KQ-015 / M2 rule).
	if lang := LanguageFromExt(d.fileName); lang != "" {
		if lang == "json" {
			// Leave JSON to jsonDetector; if we reached code, treat as code/json.
			item.DetectedType = "json"
			item.Language = "json"
			return true
		}
		item.DetectedType = "code"
		item.Language = lang
		return true
	}

	// 2. Shebang
	if lang, ok := languageFromShebang(trimmed); ok {
		item.DetectedType = "code"
		item.Language = lang
		return true
	}

	// 3. Leading package / module declarations
	if lang, ok := languageFromLead(trimmed); ok {
		item.DetectedType = "code"
		item.Language = lang
		return true
	}

	// 4. Scored keyword table — need enough signals; English prose must miss.
	if len(trimmed) < 24 {
		return false
	}
	lang, score := scoreLanguage(trimmed)
	if lang == "" || score < 3 {
		return false
	}
	item.DetectedType = "code"
	item.Language = lang
	return true
}

var shebangRe = regexp.MustCompile(`(?i)^#!\s*(?:/usr/bin/env\s+)?([a-z0-9._+-]+)`)

func languageFromShebang(s string) (string, bool) {
	line := s
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		line = s[:i]
	}
	m := shebangRe.FindStringSubmatch(strings.TrimSpace(line))
	if m == nil {
		return "", false
	}
	bin := strings.ToLower(m[1])
	switch {
	case strings.HasPrefix(bin, "python"):
		return "python", true
	case bin == "node" || bin == "nodejs":
		return "javascript", true
	case bin == "bash" || bin == "sh" || bin == "zsh" || bin == "dash" || bin == "ksh":
		return "shell", true
	case bin == "ruby":
		return "shell", true // not in primary list; treat as shell-ish script
	case bin == "perl":
		return "shell", true
	case bin == "deno" || bin == "bun":
		return "typescript", true
	default:
		return "", false
	}
}

func languageFromLead(s string) (string, bool) {
	head := s
	if len(head) > 400 {
		head = head[:400]
	}
	lower := strings.ToLower(head)
	switch {
	case strings.HasPrefix(strings.TrimSpace(lower), "package main") ||
		regexp.MustCompile(`(?m)^\s*package\s+[a-zA-Z_]\w*`).MatchString(head):
		if strings.Contains(head, "func ") || strings.Contains(head, "import ") || strings.Contains(head, "type ") {
			return "go", true
		}
		// bare package declaration is still strong for Go
		if regexp.MustCompile(`(?m)^\s*package\s+[a-zA-Z_]\w*\s*$`).MatchString(strings.Split(head, "\n")[0]) {
			return "go", true
		}
	case strings.Contains(lower, "package com.") || strings.HasPrefix(strings.TrimSpace(lower), "package "):
		if strings.Contains(head, "fun ") || strings.Contains(head, "class ") && strings.Contains(lower, "kotlin") {
			return "kotlin", true
		}
	case strings.HasPrefix(strings.TrimSpace(lower), "module ") && (strings.Contains(head, "fn ") || strings.Contains(lower, "use ")):
		return "rust", true
	case strings.HasPrefix(strings.TrimSpace(head), "import Swift") || strings.HasPrefix(strings.TrimSpace(head), "import Foundation"):
		return "swift", true
	case strings.HasPrefix(strings.TrimSpace(lower), "#!/") :
		// handled by shebang
	}
	// Rust: fn + (let mut | use std::)
	if strings.Contains(head, "fn ") && (strings.Contains(head, "let mut ") || strings.Contains(head, "use std::")) {
		return "rust", true
	}
	return "", false
}

func scoreLanguage(s string) (string, int) {
	scores := map[string]int{}

	add := func(lang string, n int) { scores[lang] += n }

	// Go
	if word(s, "package") {
		add("go", 2)
	}
	if word(s, "func") {
		add("go", 1)
	}
	if strings.Contains(s, ":= ") || strings.Contains(s, ":=") {
		add("go", 1)
	}
	if word(s, "defer") || word(s, "goroutine") {
		add("go", 2)
	}
	if strings.Contains(s, "fmt.") || strings.Contains(s, "context.Context") {
		add("go", 1)
	}

	// TypeScript / TSX
	if word(s, "interface") && (strings.Contains(s, ": ") || strings.Contains(s, "export ")) {
		add("typescript", 2)
	}
	if word(s, "type") && strings.Contains(s, " = ") && (word(s, "export") || word(s, "import")) {
		add("typescript", 1)
	}
	if strings.Contains(s, ": React.") || strings.Contains(s, "React.FC") || strings.Contains(s, "</") && word(s, "return") {
		add("tsx", 2)
	}
	if word(s, "async") && word(s, "await") {
		add("typescript", 1)
		add("javascript", 1)
	}

	// JavaScript
	if word(s, "function") || strings.Contains(s, "=>") {
		add("javascript", 1)
	}
	if word(s, "const") || word(s, "let") {
		add("javascript", 1)
	}
	if word(s, "require") && strings.Contains(s, "(") {
		add("javascript", 1)
	}
	if word(s, "export") || word(s, "import") {
		add("javascript", 1)
		add("typescript", 1)
	}

	// Prefer TS over JS when typed signals present
	if scores["typescript"] > 0 && scores["javascript"] > 0 {
		if strings.Contains(s, ": string") || strings.Contains(s, ": number") || strings.Contains(s, ": boolean") ||
			strings.Contains(s, "interface ") || strings.Contains(s, " type ") ||
			strings.Contains(s, "Promise<") || strings.Contains(s, ": ") && strings.Contains(s, "=>") {
			scores["typescript"] += 2
		}
	}
	if scores["tsx"] >= 2 {
		scores["typescript"] = 0 // tsx wins over ts when JSX present
		scores["javascript"] = 0
	}

	// Python
	if word(s, "def") {
		add("python", 1)
	}
	if word(s, "import") || word(s, "from") && strings.Contains(s, " import ") {
		add("python", 1)
	}
	if colonIndent(s) {
		add("python", 1)
	}
	if word(s, "self") || word(s, "elif") || word(s, "None") || word(s, "True") || word(s, "False") {
		add("python", 1)
	}
	if strings.HasPrefix(strings.TrimSpace(s), "from __future__") {
		add("python", 2)
	}

	// Rust
	if word(s, "fn") {
		add("rust", 1)
	}
	if strings.Contains(s, "let mut ") {
		add("rust", 2)
	}
	if strings.Contains(s, "use std::") || word(s, "impl") {
		add("rust", 1)
	}
	if strings.Contains(s, "-> ") && word(s, "fn") {
		add("rust", 1)
	}

	// Dart
	if word(s, "Widget") || strings.Contains(s, "StatelessWidget") || strings.Contains(s, "BuildContext") {
		add("dart", 2)
	}
	if strings.Contains(s, "void main()") {
		add("dart", 1)
	}

	// Swift
	if word(s, "func") && (strings.Contains(s, "->") || word(s, "var") || word(s, "let")) {
		add("swift", 1)
	}
	if word(s, "guard") || word(s, "typealias") || strings.Contains(s, "UIView") {
		add("swift", 2)
	}

	// Kotlin
	if word(s, "fun") {
		add("kotlin", 2)
	}
	if word(s, "val") || word(s, "lateinit") {
		add("kotlin", 1)
	}
	if strings.Contains(s, "companion object") {
		add("kotlin", 2)
	}

	// HTML / CSS
	if strings.HasPrefix(strings.TrimSpace(s), "<!DOCTYPE") || strings.HasPrefix(strings.TrimSpace(s), "<html") ||
		(strings.Count(s, "<") >= 3 && strings.Count(s, ">") >= 3 && strings.Contains(s, "</")) {
		add("html", 3)
	}
	if regexp.MustCompile(`(?m)^\s*[.#]?[\w-]+\s*\{`).MatchString(s) && strings.Contains(s, ":") && strings.Contains(s, ";") {
		add("css", 3)
	}

	// YAML
	if strings.HasPrefix(strings.TrimSpace(s), "---") || yamlLike(s) {
		add("yaml", 2)
	}

	// SQL already has its own detector; light boost if we got here
	if sqlStart.MatchString(s) {
		add("sql", 3)
	}

	best := ""
	bestScore := 0
	for lang, sc := range scores {
		if sc > bestScore {
			best = lang
			bestScore = sc
		}
	}
	if bestScore > 0 {
		prefer := []string{"tsx", "typescript", "jsx", "javascript", "go", "python", "rust", "kotlin", "swift", "dart", "html", "css", "yaml", "sql", "shell"}
		for _, lang := range prefer {
			if scores[lang] == bestScore {
				return lang, bestScore
			}
		}
	}
	return best, bestScore
}

func word(s, w string) bool {
	re := regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9_])` + regexp.QuoteMeta(w) + `(?:$|[^A-Za-z0-9_])`)
	return re.MatchString(s)
}

func colonIndent(s string) bool {
	lines := strings.Split(s, "\n")
	hits := 0
	for _, line := range lines {
		t := strings.TrimRightFunc(line, unicode.IsSpace)
		if strings.HasSuffix(t, ":") && !strings.HasSuffix(t, "::") {
			hits++
		}
	}
	return hits >= 2
}

func yamlLike(s string) bool {
	lines := strings.Split(s, "\n")
	if len(lines) < 2 {
		return false
	}
	kv := 0
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if i := strings.Index(t, ":"); i > 0 && i < len(t)-1 && !strings.Contains(t[:i], " ") {
			kv++
		}
	}
	return kv >= 3
}
