package search

import (
	"testing"
	"time"
)

// asOf is the fixed "today" used by tests so recency scores never drift.
var asOf = time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)

func listing(source, id string, price float64, beds int, city, date, status, desc string) Listing {
	return Listing{ID: id, Source: source, Price: price, Bedrooms: beds, City: city,
		ListedDate: date, Status: status, Description: desc, Address: id + " Test St"}
}

func newTestIndex(t *testing.T, raw ...Listing) *Index {
	t.Helper()
	ls, problems := Prepare(raw)
	if len(problems) > 0 {
		t.Fatalf("fixture problems: %v", problems)
	}
	ix := NewIndex(ls)
	ix.Now = func() time.Time { return asOf }
	return ix
}

func ptr[T any](v T) *T { return &v }

func keys(rs []Result) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Key
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
