package group_test

import (
	"testing"
	"time"

	"kopiq/internal/group"
)

func ms(t time.Time) int64 { return t.UnixMilli() }

func TestOneItemNoMultiCluster(t *testing.T) {
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.Local)
	got := group.ClusterItems([]group.Item{
		{ID: 1, AppID: "a", Place: "p", At: ms(base)},
	})
	if len(got) != 1 || len(got[0].Items) != 1 {
		t.Fatalf("%#v", got)
	}
}

func TestDifferentAppSplits(t *testing.T) {
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.Local)
	got := group.ClusterItems([]group.Item{
		{ID: 1, AppID: "edge", Place: "github.com", At: ms(base)},
		{ID: 2, AppID: "cursor", Place: "github.com", At: ms(base.Add(30 * time.Second))},
		{ID: 3, AppID: "edge", Place: "github.com", At: ms(base.Add(60 * time.Second))},
	})
	if len(got) != 3 {
		t.Fatalf("want 3 clusters, got %d %#v", len(got), got)
	}
}

func TestGapOverThreeMinutesSplits(t *testing.T) {
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.Local)
	got := group.ClusterItems([]group.Item{
		{ID: 1, AppID: "edge", Place: "p", At: ms(base)},
		{ID: 2, AppID: "edge", Place: "p", At: ms(base.Add(3*time.Minute + time.Second))},
	})
	if len(got) != 2 {
		t.Fatalf("want split, got %#v", got)
	}
}

func TestWithinThreeMinutesGroups(t *testing.T) {
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.Local)
	got := group.ClusterItems([]group.Item{
		{ID: 1, AppID: "edge", Place: "p", At: ms(base)},
		{ID: 2, AppID: "edge", Place: "p", At: ms(base.Add(2 * time.Minute))},
		{ID: 3, AppID: "edge", Place: "p", At: ms(base.Add(3 * time.Minute))},
	})
	if len(got) != 1 || len(got[0].Items) != 3 {
		t.Fatalf("%#v", got)
	}
}

func TestDayBoundarySplits(t *testing.T) {
	// 23:59 and 00:01 next day — under 3 min but different calendar day
	d1 := time.Date(2026, 10, 1, 23, 59, 0, 0, time.Local)
	d2 := time.Date(2026, 10, 2, 0, 1, 0, 0, time.Local)
	got := group.ClusterItems([]group.Item{
		{ID: 1, AppID: "edge", Place: "p", At: ms(d1)},
		{ID: 2, AppID: "edge", Place: "p", At: ms(d2)},
	})
	if len(got) != 2 {
		t.Fatalf("day boundary should split, got %#v", got)
	}
}
