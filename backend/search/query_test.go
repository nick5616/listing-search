package search

import (
	"net/url"
	"strings"
	"testing"
)

func parse(raw string) (Query, []FieldError) {
	v, _ := url.ParseQuery(raw)
	return ParseQuery(v)
}

func TestParseQuery_Defaults(t *testing.T) {
	q, errs := parse("")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if q.Page != 1 || q.PageSize != DefaultPageSize {
		t.Errorf("got page=%d pageSize=%d, want 1 and %d", q.Page, q.PageSize, DefaultPageSize)
	}
	if q.MinPrice != nil || q.MaxPrice != nil || q.TargetBudget != nil || q.MinBedrooms != nil {
		t.Error("optional filters should be unset by default")
	}
}

func TestParseQuery_ValidValues(t *testing.T) {
	q, errs := parse("minPrice=400000&maxPrice=500000&minBedrooms=2&city=%20Vienna%20&keyword=metro&address=%20main%20st%20&targetBudget=450000&status=active,pending&page=2&pageSize=5")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if *q.MinPrice != 400000 || *q.MaxPrice != 500000 || *q.MinBedrooms != 2 || *q.TargetBudget != 450000 {
		t.Errorf("numeric filters parsed wrong: %+v", q)
	}
	if q.City != "Vienna" || q.Keyword != "metro" || q.Address != "main st" || q.Page != 2 || q.PageSize != 5 {
		t.Errorf("string or paging params parsed wrong: %+v", q)
	}
	if !equal(q.Statuses, []string{"active", "pending"}) {
		t.Errorf("statuses = %v", q.Statuses)
	}
}

func TestParseQuery_MinPriceEqualToMaxPriceIsFine(t *testing.T) {
	_, errs := parse("minPrice=450000&maxPrice=450000")
	if len(errs) != 0 {
		t.Fatalf("equal bounds should be valid, got %v", errs)
	}
}

func TestParseQuery_InvalidInput(t *testing.T) {
	cases := []struct {
		name, query, field, contains string
	}{
		{"min greater than max", "minPrice=600000&maxPrice=500000", "minPrice", "greater than maxPrice"},
		{"page size zero", "pageSize=0", "pageSize", "between 1 and"},
		{"page size negative", "pageSize=-5", "pageSize", "between 1 and"},
		{"page size too large", "pageSize=51", "pageSize", "between 1 and"},
		{"page zero", "page=0", "page", "1 or greater"},
		{"page too large", "page=9999999999", "page", "or less"},
		{"page not a number", "page=two", "page", "whole number"},
		{"price not a number", "minPrice=cheap", "minPrice", "must be a number"},
		{"price NaN", "maxPrice=NaN", "maxPrice", "must be a number"},
		{"negative price", "minPrice=-1", "minPrice", "can't be negative"},
		{"fractional bedrooms", "minBedrooms=2.5", "minBedrooms", "whole number"},
		{"negative bedrooms", "minBedrooms=-1", "minBedrooms", "can't be negative"},
		{"zero budget", "targetBudget=0", "targetBudget", "greater than 0"},
		{"unknown status", "status=archived", "status", "unknown status"},
		{"keyword with no letters", "keyword=!!!", "keyword", "at least one letter"},
		{"keyword too long", "keyword=" + strings.Repeat("a", MaxKeywordLength+1), "keyword", "characters or fewer"},
		{"address too long", "address=" + strings.Repeat("a", MaxAddressLength+1), "address", "characters or fewer"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, errs := parse(tc.query)
			if len(errs) != 1 {
				t.Fatalf("want exactly 1 error, got %v", errs)
			}
			if errs[0].Field != tc.field || !strings.Contains(errs[0].Message, tc.contains) {
				t.Errorf("got %+v, want field %q containing %q", errs[0], tc.field, tc.contains)
			}
		})
	}
}

func TestParseQuery_ReportsEveryProblemAtOnce(t *testing.T) {
	_, errs := parse("minPrice=abc&pageSize=0&minBedrooms=-2")
	if len(errs) != 3 {
		t.Fatalf("want 3 errors, got %d: %v", len(errs), errs)
	}
}

func TestParseQuery_BlankValuesAreTreatedAsUnset(t *testing.T) {
	q, errs := parse("minPrice=&maxPrice=%20%20&city=&keyword=")
	if len(errs) != 0 || q.MinPrice != nil || q.MaxPrice != nil {
		t.Fatalf("blank values should mean 'no filter', got q=%+v errs=%v", q, errs)
	}
}
