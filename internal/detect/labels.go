package detect

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"net"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"kopiq/internal/domain"
)

type emailDetector struct{}

func (emailDetector) Name() string { return "email" }

var emailOnly = regexp.MustCompile(`(?i)^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)

func (emailDetector) Detect(item *domain.CanonicalItem) bool {
	s := strings.TrimSpace(item.TextContent)
	if strings.ContainsAny(s, " \n\t") || len(s) > 254 {
		return false
	}
	if !emailOnly.MatchString(s) {
		return false
	}
	item.DetectedType = "email"
	return true
}

type uuidDetector struct{}

func (uuidDetector) Name() string { return "uuid" }

var uuidRE = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func (uuidDetector) Detect(item *domain.CanonicalItem) bool {
	s := strings.TrimSpace(item.TextContent)
	if !uuidRE.MatchString(s) {
		return false
	}
	item.DetectedType = "uuid"
	return true
}

type ipDetector struct{}

func (ipDetector) Name() string { return "ip" }

func (ipDetector) Detect(item *domain.CanonicalItem) bool {
	s := strings.TrimSpace(item.TextContent)
	if strings.ContainsAny(s, " \n\t/") || len(s) > 45 {
		return false
	}
	if net.ParseIP(s) == nil {
		return false
	}
	item.DetectedType = "ip"
	return true
}

type timestampDetector struct{}

func (timestampDetector) Name() string { return "timestamp" }

func (timestampDetector) Detect(item *domain.CanonicalItem) bool {
	s := strings.TrimSpace(item.TextContent)
	if len(s) < 10 || len(s) > 13 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return false
	}
	// Seconds ~2001–2286 or millis in similar range.
	if len(s) == 10 && (n < 1_000_000_000 || n > 9_999_999_999) {
		return false
	}
	if len(s) == 13 && (n < 1_000_000_000_000 || n > 9_999_999_999_999) {
		return false
	}
	if len(s) != 10 && len(s) != 13 {
		return false
	}
	item.DetectedType = "timestamp"
	return true
}

type yamlDetector struct{}

func (yamlDetector) Name() string { return "yaml" }

func (yamlDetector) Detect(item *domain.CanonicalItem) bool {
	s := strings.TrimSpace(item.TextContent)
	if len(s) < 3 || looksJSON(s) {
		return false
	}
	// Require at least one "key: value" line; refuse Python/indent-only blobs.
	if !regexp.MustCompile(`(?m)^[A-Za-z_][\w.-]*:\s+\S`).MatchString(s) {
		return false
	}
	if strings.HasPrefix(strings.TrimSpace(s), "def ") || strings.HasPrefix(strings.TrimSpace(s), "class ") {
		return false
	}
	var v any
	if err := yaml.Unmarshal([]byte(s), &v); err != nil {
		return false
	}
	switch v.(type) {
	case map[string]any, []any:
		item.DetectedType = "yaml"
		return true
	default:
		return false
	}
}

type xmlDetector struct{}

func (xmlDetector) Name() string { return "xml" }

func (xmlDetector) Detect(item *domain.CanonicalItem) bool {
	s := strings.TrimSpace(item.TextContent)
	if len(s) < 5 || (!strings.HasPrefix(s, "<") && !strings.HasPrefix(s, "<?xml")) {
		return false
	}
	dec := xml.NewDecoder(strings.NewReader(s))
	var got bool
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if _, ok := tok.(xml.StartElement); ok {
			got = true
			break
		}
	}
	if !got {
		return false
	}
	item.DetectedType = "xml"
	return true
}

type csvDetector struct{}

func (csvDetector) Name() string { return "csv" }

func (csvDetector) Detect(item *domain.CanonicalItem) bool {
	s := strings.TrimSpace(item.TextContent)
	if !strings.Contains(s, ",") || !strings.Contains(s, "\n") {
		return false
	}
	r := csv.NewReader(strings.NewReader(s))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil || len(rows) < 2 {
		return false
	}
	cols := len(rows[0])
	if cols < 2 {
		return false
	}
	same := 0
	for _, row := range rows {
		if len(row) == cols {
			same++
		}
	}
	if same < len(rows)-1 {
		return false
	}
	item.DetectedType = "csv"
	return true
}

func looksJSON(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	if !(strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")) {
		return false
	}
	var v any
	return json.Unmarshal([]byte(s), &v) == nil
}
