package detect_test

import (
	"runtime"
	"strings"
	"testing"

	"kopiq/internal/detect"
	"kopiq/internal/domain"
)

func TestPathDetectorUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix paths")
	}
	reg := detect.Default()
	item := &domain.CanonicalItem{
		Kind:        "text",
		TextContent: "/Users/me/Documents/Personal/kopiq/desktop/frontend/src/components/onboarding/OnboardingPane.tsx",
		DetectedType: "text",
	}
	reg.Apply(item, "", "", "")
	if item.DetectedType != "file" {
		t.Fatalf("detectedType=%q want file", item.DetectedType)
	}
	if item.Title != "OnboardingPane.tsx" {
		t.Fatalf("title=%q", item.Title)
	}
}

func TestPathDetectorHome(t *testing.T) {
	reg := detect.Default()
	item := &domain.CanonicalItem{
		Kind:         "text",
		TextContent:  "~/Documents/notes.txt",
		DetectedType: "text",
	}
	reg.Apply(item, "", "", "")
	if item.DetectedType != "file" {
		t.Fatalf("detectedType=%q want file", item.DetectedType)
	}
}

func TestPathDetectorRejectsURL(t *testing.T) {
	reg := detect.Default()
	item := &domain.CanonicalItem{
		Kind:         "text",
		TextContent:  "https://example.com/path/to/file",
		DetectedType: "text",
	}
	reg.Apply(item, "", "", "")
	if item.DetectedType != "url" {
		t.Fatalf("detectedType=%q want url", item.DetectedType)
	}
}

func TestPathDetectorRejectsShellCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix paths")
	}
	reg := detect.Default()
	item := &domain.CanonicalItem{
		Kind:         "text",
		TextContent:  "/usr/bin/git clone https://example.com/repo.git",
		DetectedType: "text",
	}
	reg.Apply(item, "", "", "")
	if item.DetectedType == "file" {
		t.Fatal("shell command should not be file")
	}
}

func TestParseFilePathsMulti(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix paths")
	}
	paths, ok := detect.ParseFilePaths("/tmp/a\n/tmp/b")
	if !ok || len(paths) != 2 {
		t.Fatalf("ok=%v paths=%v", ok, paths)
	}
}

func TestParseFilePathsWindows(t *testing.T) {
	paths, ok := detect.ParseFilePaths(`C:\Users\me\file.txt`)
	if !ok || len(paths) != 1 {
		t.Fatalf("ok=%v paths=%v", ok, paths)
	}
	if !strings.Contains(paths[0], "file.txt") {
		t.Fatalf("path=%q", paths[0])
	}
}

func TestFilesKindSetsFileType(t *testing.T) {
	reg := detect.Default()
	item := &domain.CanonicalItem{Kind: "files", DetectedType: "text"}
	reg.Apply(item, "", "", "")
	if item.DetectedType != "file" {
		t.Fatalf("detectedType=%q", item.DetectedType)
	}
}
