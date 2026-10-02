package transform_test

import (
	"strings"
	"testing"

	"kopiq/internal/domain"
	"kopiq/internal/transform"
)

func itemJSON(s string) *domain.ClipboardItem {
	return &domain.ClipboardItem{
		Kind:         "text",
		DetectedType: "json",
		TextContent:  s,
		TextPreview:  s,
		Sensitivity:  "none",
	}
}

func TestJSONPrettyAndMinify(t *testing.T) {
	reg := transform.Default()
	item := itemJSON(`{"b":1,"a":2}`)
	pretty := reg.Apply(item, "json.pretty")
	if pretty.Kind != "payload" || !strings.Contains(pretty.Text, "\n") {
		t.Fatalf("pretty %#v", pretty)
	}
	mini := reg.Apply(item, "json.minify")
	if mini.Kind != "payload" || strings.Contains(mini.Text, "\n") {
		t.Fatalf("minify %#v", mini)
	}
	// Original unchanged
	if item.TextContent != `{"b":1,"a":2}` {
		t.Fatal("original mutated")
	}
}

func TestJSONToGo(t *testing.T) {
	reg := transform.Default()
	item := itemJSON(`{"user_id":1,"name":"Ada","tags":["a","b"],"meta":{"ok":true}}`)
	res := reg.Apply(item, "json.to_go")
	if res.Kind != "payload" {
		t.Fatalf("%#v", res)
	}
	if !strings.Contains(res.Text, "type Root struct") || !strings.Contains(res.Text, "UserId") {
		t.Fatalf("go output:\n%s", res.Text)
	}
}

func TestJSONToTS(t *testing.T) {
	reg := transform.Default()
	item := itemJSON(`{"id":"1","active":true}`)
	res := reg.Apply(item, "json.to_ts")
	if res.Kind != "payload" || !strings.Contains(res.Text, "export interface Root") {
		t.Fatalf("%#v", res)
	}
}

func TestMixedArrayRefused(t *testing.T) {
	reg := transform.Default()
	item := itemJSON(`[1, "x"]`)
	res := reg.Apply(item, "json.to_go")
	if res.Kind != "error" || res.Error == "" {
		t.Fatalf("expected error, got %#v", res)
	}
}

func TestCaseTransforms(t *testing.T) {
	reg := transform.Default()
	item := &domain.ClipboardItem{Kind: "text", TextContent: "user_id", Sensitivity: "none"}
	camel := reg.Apply(item, "case.camel")
	if camel.Text != "userId" {
		t.Fatalf("camel %q", camel.Text)
	}
	snake := reg.Apply(&domain.ClipboardItem{Kind: "text", TextContent: "userId", Sensitivity: "none"}, "case.snake")
	if snake.Text != "user_id" {
		t.Fatalf("snake %q", snake.Text)
	}
	upper := reg.Apply(item, "case.upper")
	if upper.Text != "USER_ID" {
		t.Fatalf("upper %q", upper.Text)
	}
}

func TestJSONYAMLRoundTrip(t *testing.T) {
	reg := transform.Default()
	item := itemJSON(`{"name":"Ada","n":1}`)
	y := reg.Apply(item, "json.to_yaml")
	if y.Kind != "payload" || !strings.Contains(y.Text, "name:") {
		t.Fatalf("to yaml %#v", y)
	}
	back := reg.Apply(&domain.ClipboardItem{Kind: "text", TextContent: y.Text, Sensitivity: "none"}, "yaml.to_json")
	if back.Kind != "payload" || !strings.Contains(back.Text, `"name"`) {
		t.Fatalf("to json %#v", back)
	}
	ok := reg.Apply(item, "json.validate")
	if ok.Kind != "payload" || !strings.Contains(ok.Text, "Valid") {
		t.Fatalf("validate %#v", ok)
	}
	bad := reg.Apply(&domain.ClipboardItem{Kind: "text", TextContent: "{nope", Sensitivity: "none"}, "json.validate")
	if bad.Kind != "payload" || !strings.Contains(bad.Text, "Invalid") {
		t.Fatalf("invalid %#v", bad)
	}
}

func TestIdentifierCaseFixtureTable(t *testing.T) {
	reg := transform.Default()
	cases := []struct {
		in, id, want string
	}{
		{"user_id", "case.camel", "userId"},
		{"user_id", "case.pascal", "UserId"},
		{"userId", "case.snake", "user_id"},
		{"userId", "case.kebab", "user-id"},
		{"UserId", "case.snake", "user_id"},
		{"hello-world", "case.camel", "helloWorld"},
		{"helloWorld", "case.pascal", "HelloWorld"},
		{"API_KEY", "case.snake", "api_key"},
	}
	for _, tc := range cases {
		item := &domain.ClipboardItem{Kind: "text", TextContent: tc.in, Sensitivity: "none"}
		res := reg.Apply(item, tc.id)
		if res.Kind != "payload" || res.Text != tc.want {
			t.Fatalf("%s(%q)=%#v want %q", tc.id, tc.in, res, tc.want)
		}
		if item.TextContent != tc.in {
			t.Fatalf("mutated original for %q", tc.in)
		}
	}
	refuse := reg.Apply(&domain.ClipboardItem{
		Kind: "text", TextContent: "line1\nline2", Sensitivity: "none",
	}, "case.camel")
	if refuse.Kind != "error" || refuse.Error == "" {
		t.Fatalf("expected multi-line refuse, got %#v", refuse)
	}
}

func TestColorFixtureRGB(t *testing.T) {
	reg := transform.Default()
	color := reg.Apply(&domain.ClipboardItem{
		Kind: "text", DetectedType: "color", TextContent: "#001646", Sensitivity: "none",
	}, "view.color")
	if color.Kind != "view" {
		t.Fatalf("%#v", color)
	}
	if !strings.Contains(color.Text, "RGB  0, 22, 70") {
		t.Fatalf("want RGB 0, 22, 70 in:\n%s", color.Text)
	}
	if !strings.Contains(color.Text, "#001646") {
		t.Fatalf("want HEX #001646 in:\n%s", color.Text)
	}
}

func TestTimestampFixtureUTC(t *testing.T) {
	reg := transform.Default()
	ts := reg.Apply(&domain.ClipboardItem{
		Kind: "text", TextContent: "1700000000", Sensitivity: "none",
	}, "view.timestamp")
	if ts.Kind != "view" {
		t.Fatalf("%#v", ts)
	}
	wantUTC := "2023-11-14T22:13:20Z"
	if !strings.Contains(ts.Text, "UTC     "+wantUTC) {
		t.Fatalf("want UTC %s in:\n%s", wantUTC, ts.Text)
	}
}

func TestJWTViewDoesNotPersist(t *testing.T) {
	reg := transform.Default()
	// header.payload.sig — signature not verified; view only
	header := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0" // {"alg":"none","typ":"JWT"}
	payload := "eyJzdWIiOiJhZGEiLCJleHAiOjIwMDAwMDAwMDB9" // {"sub":"ada","exp":2000000000}
	raw := header + "." + payload + ".c2ln"
	item := &domain.ClipboardItem{
		Kind: "text", DetectedType: "jwt", TextContent: raw, TextPreview: raw, Sensitivity: "none",
	}
	res := reg.Apply(item, "view.jwt")
	if res.Kind != "view" {
		t.Fatalf("%#v", res)
	}
	if !strings.Contains(res.Text, "Signature not verified") {
		t.Fatalf("missing disclaimer:\n%s", res.Text)
	}
	if !strings.Contains(res.Text, `"sub"`) || !strings.Contains(res.Text, "ada") {
		t.Fatalf("missing payload:\n%s", res.Text)
	}
	if item.TextContent != raw {
		t.Fatal("JWT view must not mutate stored item text")
	}
	if strings.Contains(item.TextContent, `"sub"`) && item.TextContent != raw {
		t.Fatal("decoded JSON must not replace item body")
	}
}

func TestColorAndTimestampViews(t *testing.T) {
	reg := transform.Default()
	color := reg.Apply(&domain.ClipboardItem{Kind: "text", DetectedType: "color", TextContent: "#001646", Sensitivity: "none"}, "view.color")
	if color.Kind != "view" || !strings.Contains(color.Text, "RGB") {
		t.Fatalf("%#v", color)
	}
	ts := reg.Apply(&domain.ClipboardItem{Kind: "text", TextContent: "1700000000", Sensitivity: "none"}, "view.timestamp")
	if ts.Kind != "view" || !strings.Contains(ts.Text, "UTC") {
		t.Fatalf("%#v", ts)
	}
}
