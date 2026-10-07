package search

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
)

const (
	DefaultPageSize  = 10
	MaxPageSize      = 50
	MaxKeywordLength = 100
	MaxAddressLength = 100
)

// Query is a validated search request. Optional numeric filters are pointers
// so "not set" is different from zero.
type Query struct {
	MinPrice     *float64
	MaxPrice     *float64
	MinBedrooms  *int
	City         string
	Keyword      string
	Address      string // substring of the address, any case
	TargetBudget *float64
	Statuses     []string // empty means any status
	Page         int      // 1-based
	PageSize     int
}

// FieldError says which input was wrong and why, so the UI can point at it.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ParseQuery turns URL query parameters into a Query. It collects every
// problem instead of stopping at the first, so the user can fix them all at once.
func ParseQuery(v url.Values) (Query, []FieldError) {
	p := parser{values: v}
	q := Query{
		MinPrice:     p.money("minPrice", false),
		MaxPrice:     p.money("maxPrice", false),
		TargetBudget: p.money("targetBudget", true),
		MinBedrooms:  p.nonNegativeInt("minBedrooms"),
		City:         strings.TrimSpace(v.Get("city")),
		Keyword:      strings.TrimSpace(v.Get("keyword")),
		Address:      strings.TrimSpace(v.Get("address")),
		Statuses:     p.statuses("status"),
		Page:         p.intInRange("page", 1, 1, math.MaxInt32),
		PageSize:     p.intInRange("pageSize", DefaultPageSize, 1, MaxPageSize),
	}

	if q.MinPrice != nil && q.MaxPrice != nil && *q.MinPrice > *q.MaxPrice {
		p.fail("minPrice", fmt.Sprintf("minPrice (%s) can't be greater than maxPrice (%s)",
			fmtNum(*q.MinPrice), fmtNum(*q.MaxPrice)))
	}
	if len(q.Keyword) > MaxKeywordLength {
		p.fail("keyword", fmt.Sprintf("keyword must be %d characters or fewer", MaxKeywordLength))
	} else if q.Keyword != "" && len(Tokenize(q.Keyword)) == 0 {
		p.fail("keyword", "keyword needs at least one letter or number")
	}
	if len(q.Address) > MaxAddressLength {
		p.fail("address", fmt.Sprintf("address must be %d characters or fewer", MaxAddressLength))
	}
	return q, p.errs
}

type parser struct {
	values url.Values
	errs   []FieldError
}

func (p *parser) fail(field, msg string) {
	p.errs = append(p.errs, FieldError{Field: field, Message: msg})
}

// raw returns the trimmed value and whether the parameter was provided at all.
func (p *parser) raw(name string) (string, bool) {
	s := strings.TrimSpace(p.values.Get(name))
	return s, s != ""
}

func (p *parser) money(name string, mustBePositive bool) *float64 {
	s, ok := p.raw(name)
	if !ok {
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	switch {
	case err != nil || math.IsNaN(f) || math.IsInf(f, 0):
		p.fail(name, fmt.Sprintf("%s must be a number", name))
		return nil
	case mustBePositive && f <= 0:
		p.fail(name, fmt.Sprintf("%s must be greater than 0", name))
		return nil
	case f < 0:
		p.fail(name, fmt.Sprintf("%s can't be negative", name))
		return nil
	}
	return &f
}

func (p *parser) nonNegativeInt(name string) *int {
	s, ok := p.raw(name)
	if !ok {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		p.fail(name, fmt.Sprintf("%s must be a whole number", name))
		return nil
	}
	if n < 0 {
		p.fail(name, fmt.Sprintf("%s can't be negative", name))
		return nil
	}
	return &n
}

func (p *parser) intInRange(name string, def, min, max int) int {
	s, ok := p.raw(name)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		p.fail(name, fmt.Sprintf("%s must be a whole number", name))
		return def
	}
	if n < min || n > max {
		switch {
		case max == math.MaxInt32 && n < min:
			p.fail(name, fmt.Sprintf("%s must be %d or greater", name, min))
		case max == math.MaxInt32:
			p.fail(name, fmt.Sprintf("%s must be %d or less", name, max))
		default:
			p.fail(name, fmt.Sprintf("%s must be between %d and %d", name, min, max))
		}
		return def
	}
	return n
}

// statuses accepts "active" or a comma list like "active,pending".
func (p *parser) statuses(name string) []string {
	s, ok := p.raw(name)
	if !ok {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		st := strings.ToLower(strings.TrimSpace(part))
		if st == "" {
			continue
		}
		if !validStatuses[st] {
			p.fail(name, fmt.Sprintf("unknown status %q (use active, pending or sold)", st))
			continue
		}
		out = append(out, st)
	}
	return out
}

func fmtNum(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
