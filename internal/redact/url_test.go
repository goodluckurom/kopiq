package redact

import (
	"strings"
	"testing"
)

func TestURLStripsSecrets(t *testing.T) {
	got, domain := URL(`https://user:pass@www.example.com/path?token=abc&q=1`)
	if domain != "example.com" {
		t.Fatalf("domain %q", domain)
	}
	if strings.Contains(got, "token=") {
		t.Fatalf("token leaked: %q", got)
	}
	if strings.Contains(got, "user:") || strings.Contains(got, "pass@") {
		t.Fatalf("userinfo leaked: %q", got)
	}
	if !strings.Contains(got, "q=1") {
		t.Fatalf("kept query missing: %q", got)
	}
}

func TestURLRejectsGarbage(t *testing.T) {
	u, d := URL("not a url")
	if u != "" || d != "" {
		t.Fatalf("%q %q", u, d)
	}
}
