package multicopy

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"kopiq/internal/domain"
)

// Shape IDs for multi-copy.
const (
	ShapePlain  = "plain"
	ShapeLines  = "lines"
	ShapeComma  = "comma"
	ShapeCSV    = "csv"
	ShapeJSON   = "json"
	ShapeMarkdown = "markdown"
)

// Result is the clipboard payload plus a note about omitted images.
type Result struct {
	Text           string `json:"text"`
	ImagesOmitted  int    `json:"imagesOmitted"`
	ItemsUsed      int    `json:"itemsUsed"`
}

// Build joins selected items into one text payload. Images are omitted.
func Build(items []domain.ClipboardItem, shape string) (Result, error) {
	var texts []string
	omitted := 0
	for _, it := range items {
		if it.Kind == "image" || it.DetectedType == "image" || it.DetectedType == "screenshot" {
			omitted++
			continue
		}
		t := strings.TrimRight(it.TextContent, "\n")
		if t == "" {
			t = strings.TrimSpace(it.TextPreview)
		}
		if t == "" {
			continue
		}
		texts = append(texts, t)
	}
	if len(texts) == 0 {
		msg := "No text items in the selection."
		if omitted > 0 {
			msg = fmt.Sprintf("No text items (%d image%s left out).", omitted, plural(omitted))
		}
		return Result{}, fmt.Errorf("%s", msg)
	}

	var out string
	switch shape {
	case ShapePlain:
		out = strings.Join(texts, "\n\n")
	case ShapeLines:
		// collapse internal newlines to spaces so one item = one line
		lines := make([]string, len(texts))
		for i, t := range texts {
			lines[i] = strings.Join(strings.Fields(strings.ReplaceAll(t, "\n", " ")), " ")
		}
		out = strings.Join(lines, "\n")
	case ShapeComma:
		parts := make([]string, len(texts))
		for i, t := range texts {
			parts[i] = quoteComma(t)
		}
		out = strings.Join(parts, ", ")
	case ShapeCSV:
		var b strings.Builder
		w := csv.NewWriter(&b)
		if err := w.Write(texts); err != nil {
			return Result{}, err
		}
		w.Flush()
		out = strings.TrimRight(b.String(), "\n")
	case ShapeJSON:
		b, err := json.Marshal(texts)
		if err != nil {
			return Result{}, err
		}
		out = string(b)
	case ShapeMarkdown:
		var b strings.Builder
		for _, t := range texts {
			line := strings.ReplaceAll(t, "\n", " ")
			fmt.Fprintf(&b, "- %s\n", line)
		}
		out = strings.TrimRight(b.String(), "\n")
	default:
		return Result{}, fmt.Errorf("unknown copy shape %q", shape)
	}

	return Result{Text: out, ImagesOmitted: omitted, ItemsUsed: len(texts)}, nil
}

func quoteComma(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
