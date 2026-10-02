package store

import (
	"path/filepath"
	"testing"

	"kopiq/internal/domain"
)

func TestSearchAliasAndCollection(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	item := &domain.CanonicalItem{
		ContentHash:  "hash-auth",
		Kind:         "text",
		DetectedType: "text",
		TextContent:  "auth token payload",
		TextPreview:  "auth token payload",
		MimeType:     "text/plain",
		ByteSize:     18,
		Sensitivity:  "none",
		MetadataJSON: "{}",
	}
	ctx := domain.CopyContext{
		ApplicationName: "TestApp",
		ApplicationID:   "test.app",
		CapturedAt:      1,
	}
	itemID, _, err := st.InsertOrBump(item, ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	col, err := st.CreateCollection("PickKnow")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AddToCollection(col.ID, itemID); err != nil {
		t.Fatal(err)
	}

	sn, err := st.CreateSnippet(nil, "Go handler", "func Handle() {}", "go", "go-handler")
	if err != nil {
		t.Fatal(err)
	}

	got, err := st.Search(":go-handler")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != -sn.ID {
		t.Fatalf("alias-only want snippet id %d, got %#v", -sn.ID, got)
	}
	if got[0].MetadataJSON == "" || got[0].TextContent != "func Handle() {}" {
		t.Fatalf("snippet payload %#v", got[0])
	}

	got, err = st.Search("collection:PickKnow")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != itemID {
		t.Fatalf("collection want item %d, got %#v", itemID, got)
	}

	raw, err := st.GetRaw(-sn.ID)
	if err != nil || raw == nil || raw.TextContent != "func Handle() {}" {
		t.Fatalf("GetRaw snippet: %#v err=%v", raw, err)
	}
}

func TestCollectionDualMembershipAndDeleteKeepsItems(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	item := &domain.CanonicalItem{
		ContentHash:  "hash-dual",
		Kind:         "text",
		DetectedType: "text",
		TextContent:  "shared",
		TextPreview:  "shared",
		MimeType:     "text/plain",
		ByteSize:     6,
		Sensitivity:  "none",
		MetadataJSON: "{}",
	}
	ctx := domain.CopyContext{ApplicationName: "App", ApplicationID: "app", CapturedAt: 1}
	itemID, _, err := st.InsertOrBump(item, ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	a, err := st.CreateCollection("Alpha")
	if err != nil {
		t.Fatal(err)
	}
	b, err := st.CreateCollection("Beta")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AddToCollection(a.ID, itemID); err != nil {
		t.Fatal(err)
	}
	if err := st.AddToCollection(b.ID, itemID); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteCollection(a.ID); err != nil {
		t.Fatal(err)
	}
	got, err := st.Get(itemID)
	if err != nil || got == nil {
		t.Fatalf("item should remain after collection delete: %v", err)
	}
	betaItems, err := st.ListCollectionItems(b.ID, 10)
	if err != nil || len(betaItems) != 1 {
		t.Fatalf("beta items %#v err=%v", betaItems, err)
	}
}

func TestSnippetSurvivesHistoryDeleteAndDuplicateAlias(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	item := &domain.CanonicalItem{
		ContentHash:  "hash-snip-src",
		Kind:         "text",
		DetectedType: "code",
		Language:     "go",
		TextContent:  "package x\n",
		TextPreview:  "package x",
		MimeType:     "text/plain",
		ByteSize:     10,
		Sensitivity:  "none",
		MetadataJSON: "{}",
	}
	ctx := domain.CopyContext{ApplicationName: "App", ApplicationID: "app", CapturedAt: 1}
	itemID, _, err := st.InsertOrBump(item, ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	sn, err := st.CreateSnippetFromItem(itemID, "", "go-handler")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateSnippet(nil, "Other", "body", "go", "go-handler"); err == nil {
		t.Fatal("expected duplicate alias error")
	}
	if err := st.Delete(itemID); err != nil {
		t.Fatal(err)
	}
	found, err := st.FindSnippetByAlias("go-handler")
	if err != nil || found == nil || found.ID != sn.ID {
		t.Fatalf("snippet should survive history delete: %#v err=%v", found, err)
	}
}
