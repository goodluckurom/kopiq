package multicopy_test

import (
	"encoding/json"
	"testing"

	"kopiq/internal/domain"
	"kopiq/internal/multicopy"
)

func TestMultiCopyJSON(t *testing.T) {
	items := []domain.ClipboardItem{
		{Kind: "text", TextContent: "alpha"},
		{Kind: "text", TextContent: "beta"},
		{Kind: "image", DetectedType: "image"},
	}
	res, err := multicopy.Build(items, multicopy.ShapeJSON)
	if err != nil {
		t.Fatal(err)
	}
	if res.ImagesOmitted != 1 || res.ItemsUsed != 2 {
		t.Fatalf("%#v", res)
	}
	var arr []string
	if err := json.Unmarshal([]byte(res.Text), &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 2 || arr[0] != "alpha" || arr[1] != "beta" {
		t.Fatalf("%v", arr)
	}
}

func TestMultiCopyMarkdown(t *testing.T) {
	items := []domain.ClipboardItem{
		{Kind: "text", TextContent: "one"},
		{Kind: "text", TextContent: "two"},
	}
	res, err := multicopy.Build(items, multicopy.ShapeMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "- one\n- two" {
		t.Fatalf("%q", res.Text)
	}
}
