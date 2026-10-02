package store

import (
	"path/filepath"
	"testing"

	"kopiq/internal/domain"
)

func TestInsertOrBumpNewItem(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	item := &domain.CanonicalItem{
		ContentHash:  "abc123",
		Kind:         "text",
		DetectedType: "text",
		TextContent:  "hello",
		TextPreview:  "hello",
		MimeType:     "text/plain",
		ByteSize:     5,
		Sensitivity:  "none",
		MetadataJSON: "{}",
	}
	ctx := domain.CopyContext{
		ApplicationName: "TestApp",
		ApplicationID:   "test.app",
		CapturedAt:      1,
	}

	id, inserted, err := st.InsertOrBump(item, ctx, 1)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if !inserted || id == 0 {
		t.Fatalf("want new insert id>0, got inserted=%v id=%d", inserted, id)
	}

	id2, inserted2, err := st.InsertOrBump(item, ctx, 2)
	if err != nil {
		t.Fatalf("bump: %v", err)
	}
	if inserted2 || id2 != id {
		t.Fatalf("want bump of %d, got inserted=%v id=%d", id, inserted2, id2)
	}
}
