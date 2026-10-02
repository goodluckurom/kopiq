package detect_test

import (
	"testing"

	"kopiq/internal/detect"
	"kopiq/internal/domain"
)

func run(t *testing.T, text string) string {
	t.Helper()
	item := &domain.CanonicalItem{
		Kind:         "text",
		DetectedType: "text",
		TextContent:  text,
		TextPreview:  text,
	}
	detect.Default().Apply(item, "", "", "")
	return item.DetectedType
}

func TestFixtureCorpusLabels(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`{"a":1}`, "json"},
		{`https://example.com/x`, "url"},
		{`#001646`, "color"},
		{`550e8400-e29b-41d4-a716-446655440000`, "uuid"},
		{`1700000000`, "timestamp"},
		{`ada@example.com`, "email"},
		{`127.0.0.1`, "ip"},
		{`2001:db8::1`, "ip"},
		{"name: Ada\nage: 1\n", "yaml"},
		{`<root><a/></root>`, "xml"},
		{"a,b\n1,2\n3,4\n", "csv"},
		{"eyJhbGciOiJub25lIn0.eyJzdWIiOiIxIn0.sig", "jwt"},
	}
	for _, tc := range cases {
		got := run(t, tc.in)
		if got != tc.want {
			t.Fatalf("%q → %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestEnglishParagraphStaysText(t *testing.T) {
	para := stringsRepeat("The quick brown fox jumps over the lazy dog. ", 12)
	got := run(t, para)
	if got != "text" && got != "code" {
		// code detector may claim long text; prefer not code — accept text only
	}
	if got != "text" {
		t.Fatalf("paragraph labeled %q, want text", got)
	}
}

func stringsRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
