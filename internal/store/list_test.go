package store_test

import (
  "path/filepath"
  "testing"
  "kopiq/internal/domain"
  "kopiq/internal/store"
)

func TestListDoesNotDeadlockWithContext(t *testing.T) {
  dir := t.TempDir()
  st, err := store.Open(filepath.Join(dir, "t.db"))
  if err != nil { t.Fatal(err) }
  defer st.Close()

  for i := 0; i < 5; i++ {
    item := &domain.CanonicalItem{
      ContentHash: "h"+string(rune('a'+i)), Kind: "text", DetectedType: "text",
      TextContent: "hello", TextPreview: "hello", MimeType: "text/plain", ByteSize: 5, Sensitivity: "none", MetadataJSON: "{}",
    }
    // fix hash
    item.ContentHash = "hash-" + string(rune('0'+i))
    ctx := domain.CopyContext{ApplicationName: "Test", ApplicationID: "test.app", CapturedAt: 1}
    if _, _, err := st.InsertOrBump(item, ctx, int64(i+1)); err != nil {
      t.Fatalf("insert %d: %v", i, err)
    }
  }
  items, err := st.List(domain.HistoryFilter{Limit: 100})
  if err != nil { t.Fatal(err) }
  if len(items) != 5 { t.Fatalf("want 5 got %d", len(items)) }
  if items[0].LastContext == nil || items[0].LastContext.ApplicationName != "Test" {
    t.Fatalf("missing context: %+v", items[0].LastContext)
  }
  // list must not include full body (lean payload)
  if items[0].TextContent != "" {
    t.Fatalf("list should omit text_content, got %q", items[0].TextContent)
  }
  full, err := st.Get(items[0].ID)
  if err != nil { t.Fatal(err) }
  if full.TextContent != "hello" { t.Fatalf("get missing body") }
  counts, err := st.Counts()
  if err != nil { t.Fatal(err) }
  if counts.Total != 5 { t.Fatalf("counts %d", counts.Total) }
}
