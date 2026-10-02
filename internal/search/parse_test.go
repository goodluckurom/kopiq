package search

import (
	"testing"
	"time"
)

func TestParseOperators(t *testing.T) {
	q := Parse(`type:json app:cursor auth`)
	if q.Type != "json" {
		t.Fatalf("type %q", q.Type)
	}
	if q.App != "cursor" {
		t.Fatalf("app %q", q.App)
	}
	if len(q.Terms) != 1 || q.Terms[0] != "auth" {
		t.Fatalf("terms %#v", q.Terms)
	}
}

func TestParsePinnedQuotedApp(t *testing.T) {
	q := Parse(`pinned:true app:"Visual Studio Code"`)
	if q.Pinned == nil || !*q.Pinned {
		t.Fatal("pinned")
	}
	if q.App != "visual studio code" {
		t.Fatalf("app %q", q.App)
	}
}

func TestParseUnknownFieldIsTerm(t *testing.T) {
	q := Parse(`typo:json hello`)
	if q.Type != "" {
		t.Fatal("type should be empty")
	}
	if len(q.Terms) != 2 {
		t.Fatalf("terms %#v", q.Terms)
	}
}

func TestFTSMatch(t *testing.T) {
	q := Parse(`edge dashboard`)
	got := q.FTSMatch()
	if got != "edge* AND dashboard*" {
		t.Fatalf("got %q", got)
	}
}

func TestParseM3Operators(t *testing.T) {
	now := time.Date(2026, 10, 1, 15, 0, 0, 0, time.Local)
	q := ParseAt(`domain:github.com lang:go before:today after:2026-09-01`, now)
	if q.Domain != "github.com" {
		t.Fatalf("domain %q", q.Domain)
	}
	if q.Lang != "go" {
		t.Fatalf("lang %q", q.Lang)
	}
	if q.Before == nil || q.After == nil {
		t.Fatal("missing date bounds")
	}
	startToday := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local).UnixMilli()
	if *q.Before != startToday {
		t.Fatalf("before %d want %d", *q.Before, startToday)
	}
	after := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local).UnixMilli()
	if *q.After != after {
		t.Fatalf("after %d want %d", *q.After, after)
	}
}

func TestParseCollectionAndAlias(t *testing.T) {
	q := Parse(`collection:PickKnow auth`)
	if q.Collection != "pickknow" {
		t.Fatalf("collection %q", q.Collection)
	}
	if len(q.Terms) != 1 || q.Terms[0] != "auth" {
		t.Fatalf("terms %#v", q.Terms)
	}

	q2 := Parse(`:go-handler`)
	if q2.Alias != "go-handler" {
		t.Fatalf("alias %q", q2.Alias)
	}
	if q2.BareAlias() != "go-handler" {
		t.Fatalf("bare %q", q2.BareAlias())
	}
	if q2.Empty() {
		t.Fatal("expected non-empty alias query")
	}
}
