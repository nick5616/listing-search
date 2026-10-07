package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"listingsearch/search"
)

func testServer(t *testing.T) http.Handler {
	t.Helper()
	ls, _, err := search.LoadFile("data/sample_listings.json")
	if err != nil {
		t.Fatal(err)
	}
	ix := search.NewIndex(ls)
	ix.Now = func() time.Time { return time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC) }
	return (&server{index: ix, log: slog.New(slog.NewTextHandler(io.Discard, nil))}).routes()
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestAPI_SearchReturnsRankedPage(t *testing.T) {
	rec := get(t, testServer(t), "/api/listings/search?targetBudget=450000&pageSize=3")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var page search.Page
	err := json.NewDecoder(rec.Body).Decode(&page)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 12 || page.TotalPages != 4 || len(page.Results) != 3 {
		t.Fatalf("got total=%d pages=%d len=%d", page.Total, page.TotalPages, len(page.Results))
	}
	for i := 1; i < len(page.Results); i++ {
		if page.Results[i].Score > page.Results[i-1].Score {
			t.Errorf("results not sorted by score: %v", page.Results)
		}
	}
	if page.Results[0].Key == "" || page.Results[0].Breakdown.BudgetFit == nil {
		t.Errorf("results should include key and score breakdown: %+v", page.Results[0])
	}
}

func TestAPI_InvalidInputIs400WithDetails(t *testing.T) {
	rec := get(t, testServer(t), "/api/listings/search?minPrice=600000&maxPrice=500000&pageSize=0")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
	var body struct {
		Error   string              `json:"error"`
		Details []search.FieldError `json:"details"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body.Error != "invalid_query" || len(body.Details) != 2 {
		t.Errorf("got %+v", body)
	}
}

func TestAPI_PagePastEndIs400(t *testing.T) {
	rec := get(t, testServer(t), "/api/listings/search?pageSize=10&page=3")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", rec.Code, rec.Body)
	}
}

func TestAPI_CitiesAndHealth(t *testing.T) {
	h := testServer(t)
	health := get(t, h, "/healthz")
	if health.Code != http.StatusOK {
		t.Errorf("health: %d", health.Code)
	}
	rec := get(t, h, "/api/cities")
	var body struct{ Cities []string }
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if len(body.Cities) != 6 {
		t.Errorf("want 6 cities in sample data, got %v", body.Cities)
	}
}

func TestAPI_WrongMethodIsRejected(t *testing.T) {
	rec := httptest.NewRecorder()
	testServer(t).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/listings/search", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", rec.Code)
	}
}
