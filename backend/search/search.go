package search

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Result is a listing plus its relevance score.
type Result struct {
	Listing
	Key       string    `json:"key"`
	Score     float64   `json:"score"`
	Breakdown Breakdown `json:"scoreBreakdown"`
}

// Page is one page of ranked results.
type Page struct {
	Results    []Result     `json:"results"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	Total      int          `json:"total"`
	TotalPages int          `json:"totalPages"`
	Warnings   []FieldError `json:"warnings"`
}

// Index holds the listings in memory. With ~12 listings a linear scan is the
// right call; see the README for what changes at real scale.
type Index struct {
	listings []Listing
	cities   map[string]string // normalized key -> display name
	Scoring  Scoring
	Now      func() time.Time // injectable so tests and demos are reproducible
}

func NewIndex(listings []Listing) *Index {
	cities := map[string]string{}
	for _, l := range listings {
		_, ok := cities[l.cityKey]
		if !ok && l.cityKey != "" {
			cities[l.cityKey] = strings.Join(strings.Fields(l.City), " ")
		}
	}
	return &Index{listings: listings, cities: cities, Scoring: DefaultScoring, Now: time.Now}
}

// Cities returns the distinct city names, sorted, for the UI's suggestions.
func (ix *Index) Cities() []string {
	out := make([]string, 0, len(ix.cities))
	for _, name := range ix.cities {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Search filters, scores, sorts and paginates. It returns field errors only
// for problems that depend on the data, like asking for a page past the end.
func (ix *Index) Search(q Query) (Page, []FieldError) {
	now := ix.Now()
	terms := Tokenize(q.Keyword)
	cityKey := NormalizeCity(q.City)
	addressPart := NormalizeText(q.Address)
	statuses := map[string]bool{}
	for _, s := range q.Statuses {
		statuses[s] = true
	}

	page := Page{Page: q.Page, PageSize: q.PageSize, Results: []Result{}, Warnings: []FieldError{}}

	if cityKey != "" {
		_, known := ix.cities[cityKey]
		if !known {
			page.Warnings = append(page.Warnings, FieldError{
				Field:   "city",
				Message: fmt.Sprintf("No listings in %q. Cities with listings: %s.", q.City, strings.Join(ix.Cities(), ", ")),
			})
		}
	}

	var matched []Result
	for _, l := range ix.listings {
		if !matches(l, q, cityKey, addressPart, terms, statuses) {
			continue
		}
		score, breakdown := ix.Scoring.Score(l, q.TargetBudget, now)
		matched = append(matched, Result{Listing: l, Key: l.Key(), Score: score, Breakdown: breakdown})
	}

	sortResults(matched)

	page.Total = len(matched)
	page.TotalPages = (page.Total + q.PageSize - 1) / q.PageSize

	// Page 1 of an empty result is a normal "no results". Any other page past
	// the end is a client mistake, and saying so beats a silent empty list.
	if q.Page > 1 && q.Page > page.TotalPages {
		msg := fmt.Sprintf("page %d is past the last page (%d)", q.Page, page.TotalPages)
		if page.TotalPages == 0 {
			msg = fmt.Sprintf("page %d is past the end: there are no results, so only page 1 exists", q.Page)
		}
		return page, []FieldError{{Field: "page", Message: msg}}
	}

	start := (q.Page - 1) * q.PageSize
	end := min(start+q.PageSize, page.Total)
	if start < end {
		page.Results = matched[start:end]
	}
	return page, nil
}

func matches(l Listing, q Query, cityKey, addressPart string, terms []string, statuses map[string]bool) bool {
	if q.MinPrice != nil && l.Price < *q.MinPrice {
		return false
	}
	if q.MaxPrice != nil && l.Price > *q.MaxPrice {
		return false
	}
	if q.MinBedrooms != nil && l.Bedrooms < *q.MinBedrooms {
		return false
	}
	if cityKey != "" && l.cityKey != cityKey {
		return false
	}
	// Plain substring, so "main st" matches "123 Main St, Apt 4B" and "4b" matches too.
	if addressPart != "" && !strings.Contains(l.addressKey, addressPart) {
		return false
	}
	if len(statuses) > 0 && !statuses[l.Status] {
		return false
	}
	return containsAllTerms(l.descTokens, terms)
}

// containsAllTerms requires every search term to be the start of some word in
// the description: "pet" matches "pets" and "pet", but not "carpet".
func containsAllTerms(tokens, terms []string) bool {
	for _, term := range terms {
		found := false
		for _, tok := range tokens {
			if strings.HasPrefix(tok, term) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// sortResults orders by score, and breaks ties so pagination is stable:
// newer listing first, then lower price, then key alphabetically.
func sortResults(rs []Result) {
	sort.SliceStable(rs, func(i, j int) bool {
		a, b := rs[i], rs[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if !a.listed.Equal(b.listed) {
			return a.listed.After(b.listed)
		}
		if a.Price != b.Price {
			return a.Price < b.Price
		}
		return a.Key < b.Key
	})
}
