package store

import (
	"path/filepath"
	"testing"

	"kopiq/internal/domain"
	"kopiq/internal/stack"
)

func TestStackPersistAndResolve(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := domain.CopyContext{ApplicationName: "T", ApplicationID: "t", CapturedAt: 1}
	mk := func(hash, text string, ts int64) int64 {
		id, _, err := st.InsertOrBump(&domain.CanonicalItem{
			ContentHash:  hash,
			Kind:         "text",
			DetectedType: "text",
			TextContent:  text,
			TextPreview:  text,
			MimeType:     "text/plain",
			ByteSize:     int64(len(text)),
			Sensitivity:  "none",
			MetadataJSON: "{}",
		}, ctx, ts)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	id1 := mk("h1", "one", 1)
	id2 := mk("h2", "two", 2)

	saved, err := st.CreateStack("creds", []stack.Step{
		{ItemID: id1, Text: "one", Label: "one"},
		{ItemID: id2, Text: "two", Label: "two"},
	})
	if err != nil {
		t.Fatal(err)
	}
	list, err := st.ListStacks()
	if err != nil || len(list) != 1 || list[0].ID != saved.ID {
		t.Fatalf("list %#v err=%v", list, err)
	}
	steps, err := st.ResolveStackSteps(saved.ID)
	if err != nil || len(steps) != 2 || steps[0].Text != "one" {
		t.Fatalf("resolve %#v err=%v", steps, err)
	}
	if err := st.Delete(id1); err != nil {
		t.Fatal(err)
	}
	if _, err = st.ResolveStackSteps(saved.ID); err == nil {
		t.Fatal("expected missing step to block")
	}
}
