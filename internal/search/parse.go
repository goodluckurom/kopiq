package search

import (
	"strings"
	"time"
	"unicode"
)

// Query is a parsed clipboard search.
type Query struct {
	Type       string // detected_type filter; image matches screenshot too
	App        string // contains match on app name/id
	Pinned     *bool  // nil = any
	Lang       string
	Domain     string
	Collection string // collection name (contains, case-insensitive)
	Before     *int64 // exclusive upper bound on last_copied_at (unix ms)
	After      *int64 // inclusive lower bound on last_copied_at (unix ms)
	Terms      []string
	// Alias is set when the whole query is ":name" or a single token that looks like an alias.
	Alias string
}

// Parse splits a user query into operators and free-text terms.
func Parse(raw string) Query {
	return ParseAt(raw, time.Now())
}

// ParseAt is Parse with a fixed clock (tests).
func ParseAt(raw string, now time.Time) Query {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Query{}
	}
	tokens := tokenize(raw)
	var q Query
	for _, tok := range tokens {
		if field, value, ok := splitFilter(tok); ok {
			switch field {
			case "type":
				q.Type = strings.ToLower(value)
			case "app":
				q.App = strings.ToLower(value)
			case "lang":
				q.Lang = strings.ToLower(value)
			case "domain":
				q.Domain = strings.ToLower(strings.TrimPrefix(value, "www."))
			case "pinned":
				v := strings.ToLower(value)
				if v == "true" || v == "1" || v == "yes" {
					b := true
					q.Pinned = &b
				} else if v == "false" || v == "0" || v == "no" {
					b := false
					q.Pinned = &b
				}
			case "before":
				if ms, ok := parseDateBound(value, now, true); ok {
					q.Before = &ms
				}
			case "after":
				if ms, ok := parseDateBound(value, now, false); ok {
					q.After = &ms
				}
			case "collection":
				q.Collection = strings.ToLower(value)
			default:
				q.Terms = append(q.Terms, tok)
			}
			continue
		}
		// Leading-colon alias token (:go-handler)
		if strings.HasPrefix(tok, ":") && len(tok) > 1 && !strings.Contains(tok[1:], ":") {
			q.Alias = strings.ToLower(strings.TrimPrefix(tok, ":"))
			continue
		}
		// Bare date words as free filters when alone? Spec uses domain:… yesterday
		// as free text "yesterday" OR after:yesterday. Treat bare today/yesterday/week as after:/before helpers.
		lower := strings.ToLower(tok)
		switch lower {
		case "today":
			if ms, ok := parseDateBound("today", now, false); ok {
				q.After = &ms
			}
		case "yesterday":
			start, _ := parseDateBound("yesterday", now, false)
			end, _ := parseDateBound("today", now, false)
			if start > 0 {
				q.After = &start
			}
			if end > 0 {
				q.Before = &end
			}
		case "week":
			if ms, ok := parseDateBound("week", now, false); ok {
				q.After = &ms
			}
		case "older":
			if ms, ok := parseDateBound("week", now, true); ok {
				q.Before = &ms
			}
		default:
			q.Terms = append(q.Terms, tok)
		}
	}
	return q
}

// parseDateBound interprets YYYY-MM-DD or today|yesterday|week|older.
// before=true → start of that day (exclusive upper for before:).
// before=false → start of that day (inclusive lower for after:).
func parseDateBound(value string, now time.Time, beforeOp bool) (int64, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	loc := now.Location()
	y, m, d := now.Date()
	startOfToday := time.Date(y, m, d, 0, 0, 0, 0, loc)

	var t time.Time
	switch value {
	case "today":
		t = startOfToday
	case "yesterday":
		t = startOfToday.AddDate(0, 0, -1)
	case "week":
		// Start of this week (Monday).
		wd := int(startOfToday.Weekday())
		if wd == 0 {
			wd = 7
		}
		t = startOfToday.AddDate(0, 0, -(wd - 1))
	case "older":
		wd := int(startOfToday.Weekday())
		if wd == 0 {
			wd = 7
		}
		t = startOfToday.AddDate(0, 0, -(wd - 1))
		beforeOp = true
	default:
		parsed, err := time.ParseInLocation("2006-01-02", value, loc)
		if err != nil {
			return 0, false
		}
		t = parsed
	}
	if beforeOp {
		return t.UnixMilli(), true
	}
	return t.UnixMilli(), true
}

func splitFilter(tok string) (field, value string, ok bool) {
	i := strings.IndexByte(tok, ':')
	if i <= 0 || i == len(tok)-1 {
		return "", "", false
	}
	field = strings.ToLower(tok[:i])
	value = tok[i+1:]
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}
	switch field {
	case "type", "app", "pinned", "lang", "domain", "before", "after", "tag", "collection":
		return field, value, true
	default:
		return "", "", false
	}
}

func tokenize(s string) []string {
	var out []string
	var b strings.Builder
	inQuote := false
	flush := func() {
		if b.Len() == 0 {
			return
		}
		out = append(out, b.String())
		b.Reset()
	}
	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
			b.WriteRune(r)
		case unicode.IsSpace(r) && !inQuote:
			flush()
		default:
			b.WriteRune(r)
		}
	}
	flush()
	return out
}

// Empty reports whether the query has no constraints.
func (q Query) Empty() bool {
	return q.Type == "" && q.App == "" && q.Pinned == nil &&
		q.Lang == "" && q.Domain == "" && q.Collection == "" &&
		q.Before == nil && q.After == nil && q.Alias == "" &&
		len(q.Terms) == 0
}

// BareAlias reports a query that is only an alias lookup (possibly with :).
func (q Query) BareAlias() string {
	if q.Alias != "" && q.Type == "" && q.App == "" && q.Pinned == nil &&
		q.Lang == "" && q.Domain == "" && q.Collection == "" &&
		q.Before == nil && q.After == nil && len(q.Terms) == 0 {
		return q.Alias
	}
	// Single free term may be an alias without colon.
	if q.Alias == "" && len(q.Terms) == 1 && q.Type == "" && q.App == "" &&
		q.Pinned == nil && q.Lang == "" && q.Domain == "" && q.Collection == "" &&
		q.Before == nil && q.After == nil {
		t := strings.TrimSpace(q.Terms[0])
		if t != "" && !strings.Contains(t, " ") {
			return strings.ToLower(strings.TrimPrefix(t, ":"))
		}
	}
	return ""
}

// FTSMatch builds an FTS5 MATCH expression (AND of prefix terms).
func (q Query) FTSMatch() string {
	var parts []string
	for _, t := range q.Terms {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if len([]rune(t)) < 2 && len(q.Terms) > 1 {
			continue
		}
		esc := escapeFTS(t)
		parts = append(parts, esc+"*")
	}
	return strings.Join(parts, " AND ")
}

func escapeFTS(s string) string {
	s = strings.ReplaceAll(s, `"`, `""`)
	if strings.ContainsAny(s, ` :*()"`) || strings.Contains(s, "-") {
		return `"` + s + `"`
	}
	return s
}
