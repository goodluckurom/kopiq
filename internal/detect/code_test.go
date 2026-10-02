package detect_test

import (
	"strings"
	"testing"

	"kopiq/internal/detect"
	"kopiq/internal/domain"
)

func apply(text, fileName string) *domain.CanonicalItem {
	item := &domain.CanonicalItem{
		Kind:         "text",
		DetectedType: "text",
		TextContent:  text,
		TextPreview:  text,
	}
	detect.Default().Apply(item, "", "", fileName)
	return item
}

func TestCodeGoFixture(t *testing.T) {
	src := `package main

import "fmt"

func Handle(w http.ResponseWriter, r *http.Request) {
	fmt.Println("ok")
}
`
	item := apply(src, "")
	if item.DetectedType != "code" || item.Language != "go" {
		t.Fatalf("got type=%q lang=%q", item.DetectedType, item.Language)
	}
}

func TestCodeTypeScriptFixture(t *testing.T) {
	src := `export async function loadUser(id: string): Promise<User> {
  const res = await fetch('/api/users/' + id);
  return res.json();
}
`
	item := apply(src, "")
	if item.DetectedType != "code" || item.Language != "typescript" {
		t.Fatalf("got type=%q lang=%q", item.DetectedType, item.Language)
	}
}

func TestCodePythonFixture(t *testing.T) {
	src := `from __future__ import annotations

def greet(name: str) -> None:
    print(f"hello {name}")

if __name__ == "__main__":
    greet("kopiq")
`
	item := apply(src, "")
	if item.DetectedType != "code" || item.Language != "python" {
		t.Fatalf("got type=%q lang=%q", item.DetectedType, item.Language)
	}
}

func TestCodeSQLFixture(t *testing.T) {
	src := `SELECT id, email FROM users WHERE active = 1 ORDER BY created_at DESC LIMIT 20;`
	item := apply(src, "")
	if item.DetectedType != "code" || item.Language != "sql" {
		t.Fatalf("got type=%q lang=%q", item.DetectedType, item.Language)
	}
}

func TestCodeEnglishStaysText(t *testing.T) {
	src := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 8)
	item := apply(src, "")
	if item.DetectedType != "text" {
		t.Fatalf("english labeled %q / %q", item.DetectedType, item.Language)
	}
}

func TestCodeExtensionWinsOverHeuristic(t *testing.T) {
	// Looks like Python but came from a .go file in the editor.
	src := "def something():\n    return 1\n"
	item := apply(src, "handler.go")
	if item.DetectedType != "code" || item.Language != "go" {
		t.Fatalf("extension should win, got type=%q lang=%q", item.DetectedType, item.Language)
	}
}

func TestCodeShebangPython(t *testing.T) {
	src := "#!/usr/bin/env python3\nprint('hi')\n"
	item := apply(src, "")
	if item.Language != "python" {
		t.Fatalf("lang=%q", item.Language)
	}
}

func TestLanguageFromExt(t *testing.T) {
	if got := detect.LanguageFromExt("foo/bar.ts"); got != "typescript" {
		t.Fatalf("got %q", got)
	}
	if got := detect.LanguageFromExt("main.go"); got != "go" {
		t.Fatalf("got %q", got)
	}
}
