package search

import (
	"strings"
	"testing"
)

func activeIn(city string) Query { return Query{City: city, Page: 1, PageSize: 10} }

func TestSearch_NoMatches(t *testing.T) {
	ix := newTestIndex(t, listing("MLS_A", "1", 300000, 2, "Fairfax", "2026-09-01", "active", "Cozy"))
	page, errs := ix.Search(Query{MinPrice: ptr(900000.0), Page: 1, PageSize: 10})
	if errs != nil {
		t.Fatalf("no matches is not an error, got %v", errs)
	}
	if page.Total != 0 || page.TotalPages != 0 || len(page.Results) != 0 {
		t.Errorf("want empty page, got %+v", page)
	}
	if page.Results == nil {
		t.Error("results should be an empty list, not null, so clients don't crash")
	}
}

func TestSearch_UnknownCityReturnsAWarning(t *testing.T) {
	ix := newTestIndex(t,
		listing("MLS_A", "1", 300000, 2, "Fairfax", "2026-09-01", "active", ""),
		listing("MLS_A", "2", 300000, 2, "Vienna", "2026-09-01", "active", ""),
	)
	page, errs := ix.Search(activeIn("Seatle"))
	if errs != nil || page.Total != 0 {
		t.Fatalf("got errs=%v total=%d", errs, page.Total)
	}
	if len(page.Warnings) != 1 || page.Warnings[0].Field != "city" ||
		!strings.Contains(page.Warnings[0].Message, "Fairfax, Vienna") {
		t.Errorf("want a city warning listing known cities, got %+v", page.Warnings)
	}
}

func TestSearch_KnownCityWithNoMatchesHasNoWarning(t *testing.T) {
	ix := newTestIndex(t, listing("MLS_A", "1", 300000, 2, "Fairfax", "2026-09-01", "active", ""))
	q := activeIn("Fairfax")
	q.MinBedrooms = ptr(5)
	page, _ := ix.Search(q)
	if page.Total != 0 || len(page.Warnings) != 0 {
		t.Errorf("filters excluding everything is a normal empty result, got %+v", page)
	}
}

func TestSearch_CityMatchIgnoresCaseAndSpacing(t *testing.T) {
	ix := newTestIndex(t,
		listing("MLS_A", "1", 300000, 2, "Fairfax", "2026-09-01", "active", ""),
		listing("MLS_B", "1", 300000, 2, " fairfax ", "2026-09-01", "active", ""),
	)
	page, _ := ix.Search(activeIn("FAIRFAX"))
	if page.Total != 2 {
		t.Errorf("want both spellings to match, got %d", page.Total)
	}
	got := ix.Cities()
	if !equal(got, []string{"Fairfax"}) {
		t.Errorf("Cities() should de-duplicate spellings, got %v", got)
	}
}

func TestSearch_SameIDFromDifferentSourcesAreDifferentListings(t *testing.T) {
	ix := newTestIndex(t,
		listing("MLS_A", "7", 300000, 2, "X", "2026-09-01", "active", ""),
		listing("MLS_B", "7", 310000, 2, "X", "2026-09-01", "active", ""),
	)
	page, _ := ix.Search(Query{Page: 1, PageSize: 10})
	if page.Total != 2 {
		t.Fatalf("want 2 listings, got %d", page.Total)
	}
}

func TestSearch_FiltersAreInclusiveAndCombine(t *testing.T) {
	ix := newTestIndex(t,
		listing("MLS_A", "low", 400000, 2, "X", "2026-09-01", "active", ""),
		listing("MLS_A", "mid", 450000, 3, "X", "2026-09-01", "active", ""),
		listing("MLS_A", "high", 500000, 4, "X", "2026-09-01", "active", ""),
		listing("MLS_A", "pend", 450000, 3, "X", "2026-09-01", "pending", ""),
	)
	page, _ := ix.Search(Query{
		MinPrice: ptr(400000.0), MaxPrice: ptr(450000.0), MinBedrooms: ptr(3),
		Statuses: []string{"active"}, Page: 1, PageSize: 10,
	})
	if !equal(keys(page.Results), []string{"MLS_A:mid"}) {
		t.Errorf("got %v", keys(page.Results))
	}
}

func TestSearch_KeywordMatchesWordStartsOnly(t *testing.T) {
	ix := newTestIndex(t,
		listing("MLS_A", "pets", 1, 1, "X", "2026-09-01", "active", "Pets allowed."),
		listing("MLS_A", "carpet", 1, 1, "X", "2026-09-01", "active", "New carpet throughout."),
	)
	page, _ := ix.Search(Query{Keyword: "PET", Page: 1, PageSize: 10})
	if !equal(keys(page.Results), []string{"MLS_A:pets"}) {
		t.Errorf("'pet' should match 'Pets' but not 'carpet', got %v", keys(page.Results))
	}
}

func TestSearch_KeywordRequiresAllTerms(t *testing.T) {
	ix := newTestIndex(t,
		listing("MLS_A", "both", 1, 1, "X", "2026-09-01", "active", "Walkable to Metro, quiet street."),
		listing("MLS_A", "one", 1, 1, "X", "2026-09-01", "active", "Close to Metro."),
	)
	page, _ := ix.Search(Query{Keyword: "metro quiet", Page: 1, PageSize: 10})
	if !equal(keys(page.Results), []string{"MLS_A:both"}) {
		t.Errorf("got %v", keys(page.Results))
	}
}

func TestSearch_AddressMatchesSubstringIgnoringCaseAndSpaces(t *testing.T) {
	a := listing("MLS_A", "main", 1, 1, "X", "2026-09-01", "active", "")
	a.Address = "123 Main St, Apt 4B"
	b := listing("MLS_A", "oak", 1, 1, "X", "2026-09-01", "active", "")
	b.Address = "456 Oak Ave"
	ix := newTestIndex(t, a, b)

	for _, term := range []string{"main st", "  MAIN   st ", "apt 4", "23 ma"} {
		page, _ := ix.Search(Query{Address: term, Page: 1, PageSize: 10})
		if !equal(keys(page.Results), []string{"MLS_A:main"}) {
			t.Errorf("address %q: got %v", term, keys(page.Results))
		}
	}
}

func TestSearch_RanksCloserToBudgetFirst(t *testing.T) {
	ix := newTestIndex(t,
		listing("MLS_A", "far", 700000, 3, "X", "2026-09-01", "active", ""),
		listing("MLS_A", "close", 455000, 3, "X", "2026-09-01", "active", ""),
		listing("MLS_A", "under", 380000, 3, "X", "2026-09-01", "active", ""),
	)
	page, _ := ix.Search(Query{TargetBudget: ptr(450000.0), Page: 1, PageSize: 10})
	if !equal(keys(page.Results), []string{"MLS_A:close", "MLS_A:under", "MLS_A:far"}) {
		t.Errorf("got %v", keys(page.Results))
	}
}

func TestSearch_TiedScoresHaveAStableOrder(t *testing.T) {
	// Same price and date means identical scores. Ties break on newer date,
	// then lower price, then key, so the order never changes between requests.
	ix := newTestIndex(t,
		listing("MLS_B", "2", 450000, 3, "X", "2026-09-01", "active", ""),
		listing("MLS_A", "9", 450000, 3, "X", "2026-09-01", "active", ""),
		listing("MLS_A", "1", 450000, 3, "X", "2026-09-01", "active", ""),
	)
	want := []string{"MLS_A:1", "MLS_A:9", "MLS_B:2"}
	for i := 0; i < 5; i++ {
		page, _ := ix.Search(Query{TargetBudget: ptr(450000.0), Page: 1, PageSize: 10})
		if page.Results[0].Score != page.Results[2].Score {
			t.Fatal("fixture should produce tied scores")
		}
		if !equal(keys(page.Results), want) {
			t.Fatalf("run %d: got %v, want %v", i, keys(page.Results), want)
		}
	}
}

func TestSearch_TieBreaksOnNewerThenCheaper(t *testing.T) {
	// No budget, so score is recency only. Same date = tie, broken by price.
	ix := newTestIndex(t,
		listing("MLS_A", "pricey", 500000, 3, "X", "2026-09-01", "active", ""),
		listing("MLS_A", "cheap", 400000, 3, "X", "2026-09-01", "active", ""),
		listing("MLS_A", "newest", 900000, 3, "X", "2026-09-04", "active", ""),
	)
	page, _ := ix.Search(Query{Page: 1, PageSize: 10})
	if !equal(keys(page.Results), []string{"MLS_A:newest", "MLS_A:cheap", "MLS_A:pricey"}) {
		t.Errorf("got %v", keys(page.Results))
	}
}

func paginationIndex(t *testing.T, n int) *Index {
	var ls []Listing
	for i := 0; i < n; i++ {
		// Distinct prices give a deterministic order: cheapest first on ties.
		ls = append(ls, listing("MLS_A", string(rune('a'+i)), float64(100000+i*1000), 2, "X", "2026-09-01", "active", ""))
	}
	return newTestIndex(t, ls...)
}

func TestSearch_PaginationBoundaries(t *testing.T) {
	ix := paginationIndex(t, 5)

	cases := []struct {
		name           string
		page, size     int
		wantKeys       []string
		wantTotalPages int
	}{
		{"first page", 1, 2, []string{"MLS_A:a", "MLS_A:b"}, 3},
		{"middle page", 2, 2, []string{"MLS_A:c", "MLS_A:d"}, 3},
		{"last partial page", 3, 2, []string{"MLS_A:e"}, 3},
		{"page size equals total", 1, 5, []string{"MLS_A:a", "MLS_A:b", "MLS_A:c", "MLS_A:d", "MLS_A:e"}, 1},
		{"page size larger than total", 1, 50, []string{"MLS_A:a", "MLS_A:b", "MLS_A:c", "MLS_A:d", "MLS_A:e"}, 1},
		{"page size of one", 5, 1, []string{"MLS_A:e"}, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page, errs := ix.Search(Query{Page: tc.page, PageSize: tc.size})
			if errs != nil {
				t.Fatalf("unexpected errors %v", errs)
			}
			if !equal(keys(page.Results), tc.wantKeys) || page.TotalPages != tc.wantTotalPages || page.Total != 5 {
				t.Errorf("got keys=%v totalPages=%d total=%d", keys(page.Results), page.TotalPages, page.Total)
			}
		})
	}
}

func TestSearch_PagePastTheEndIsAnError(t *testing.T) {
	ix := paginationIndex(t, 5)
	_, errs := ix.Search(Query{Page: 4, PageSize: 2})
	if len(errs) != 1 || errs[0].Field != "page" || !strings.Contains(errs[0].Message, "last page (3)") {
		t.Errorf("got %v", errs)
	}
}

func TestSearch_PageOneOfEmptyResultIsNotAnError(t *testing.T) {
	ix := paginationIndex(t, 5)
	page, errs := ix.Search(Query{MinPrice: ptr(1e9), Page: 1, PageSize: 2})
	if errs != nil || page.TotalPages != 0 {
		t.Errorf("got page=%+v errs=%v", page, errs)
	}
}

func TestSearch_SampleDataLoads(t *testing.T) {
	ls, problems, err := LoadFile("../data/sample_listings.json")
	if err != nil || len(problems) != 0 {
		t.Fatalf("err=%v problems=%v", err, problems)
	}
	if len(ls) != 12 {
		t.Errorf("want 12 listings, got %d", len(ls))
	}
}

func TestPrepare_SkipsBadRecordsAndKeepsTheRest(t *testing.T) {
	good := listing("MLS_A", "ok", 1, 1, "X", "2026-09-01", "active", "")
	badStatus := listing("MLS_A", "s", 1, 1, "X", "2026-09-01", "archived", "")
	badDate := listing("MLS_A", "d", 1, 1, "X", "09/01/2026", "active", "")
	noSource := listing("", "n", 1, 1, "X", "2026-09-01", "active", "")
	dup := listing("MLS_A", "ok", 2, 2, "X", "2026-09-01", "active", "")

	ls, problems := Prepare([]Listing{good, badStatus, badDate, noSource, dup})
	if len(ls) != 1 || ls[0].Key() != "MLS_A:ok" || ls[0].Price != 1 {
		t.Errorf("want only the first good record kept, got %+v", ls)
	}
	if len(problems) != 4 {
		t.Errorf("want 4 problems, got %v", problems)
	}
}

func TestSearch_PagePastTheEndOfEmptyResultSaysThereAreNoResults(t *testing.T) {
	ix := paginationIndex(t, 5)
	_, errs := ix.Search(Query{MinPrice: ptr(1e9), Page: 2, PageSize: 2})
	if len(errs) != 1 || errs[0].Field != "page" || !strings.Contains(errs[0].Message, "no results") {
		t.Errorf("got %v", errs)
	}
}

func TestCities_LeavesOutListingsWithNoCity(t *testing.T) {
	ix := newTestIndex(t,
		listing("MLS_A", "1", 1, 1, "Vienna", "2026-09-01", "active", ""),
		listing("MLS_A", "2", 1, 1, "  ", "2026-09-01", "active", ""),
	)
	got := ix.Cities()
	if !equal(got, []string{"Vienna"}) {
		t.Errorf("Cities() = %q, want [Vienna]", got)
	}
}
