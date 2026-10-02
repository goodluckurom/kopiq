package group

import "time"

// Item is a minimal row for clustering (KQ-007).
type Item struct {
	ID    int64
	AppID string
	Place string
	At    int64 // unix ms
}

// Cluster is one context group.
type Cluster struct {
	AppID string
	Place string
	Items []Item
	From  int64
	To    int64
}

const GapMS = 3 * 60 * 1000

// ClusterItems groups items that share app + place within 3 minutes,
// without crossing a local calendar day. Walks oldest→newest.
// One-item bursts are still returned (UI draws them without chrome).
func ClusterItems(items []Item) []Cluster {
	if len(items) == 0 {
		return nil
	}
	chron := append([]Item(nil), items...)
	// insertion sort by At ascending
	for i := 1; i < len(chron); i++ {
		j := i
		for j > 0 && chron[j-1].At > chron[j].At {
			chron[j-1], chron[j] = chron[j], chron[j-1]
			j--
		}
	}

	var out []Cluster
	for _, it := range chron {
		appID := it.AppID
		place := it.Place
		if len(out) == 0 {
			out = append(out, Cluster{AppID: appID, Place: place, Items: []Item{it}, From: it.At, To: it.At})
			continue
		}
		prev := &out[len(out)-1]
		sameApp := prev.AppID == appID
		samePlace := prev.Place == place
		gapOK := it.At-prev.To <= GapMS
		sameDay := sameLocalDay(it.At, prev.To)
		if sameApp && samePlace && gapOK && sameDay {
			dup := false
			for _, x := range prev.Items {
				if x.ID == it.ID {
					dup = true
					break
				}
			}
			if !dup {
				prev.Items = append(prev.Items, it)
			}
			if it.At > prev.To {
				prev.To = it.At
			}
			if it.At < prev.From {
				prev.From = it.At
			}
			continue
		}
		out = append(out, Cluster{AppID: appID, Place: place, Items: []Item{it}, From: it.At, To: it.At})
	}
	return out
}

func sameLocalDay(a, b int64) bool {
	ta := time.UnixMilli(a).Local()
	tb := time.UnixMilli(b).Local()
	y1, m1, d1 := ta.Date()
	y2, m2, d2 := tb.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
