package search

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"
)

// Listing is one property listing as it arrives from an MLS feed.
// IDs are only unique within a source, so Key() (source + id) is the real identity.
type Listing struct {
	ID          string  `json:"id"`
	Source      string  `json:"source"`
	Address     string  `json:"address"`
	City        string  `json:"city"`
	State       string  `json:"state"`
	Zip         string  `json:"zip"`
	Price       float64 `json:"price"`
	Bedrooms    int     `json:"bedrooms"`
	Bathrooms   float64 `json:"bathrooms"`
	Sqft        int     `json:"sqft"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	ListedDate  string  `json:"listedDate"`
	Status      string  `json:"status"`
	Description string  `json:"description"`

	// Precomputed at load time so each search doesn't redo the work.
	listed     time.Time
	cityKey    string
	addressKey string
	descTokens []string
}

// Key is the identity of a listing across all feeds, e.g. "MLS_A:A1".
func (l Listing) Key() string { return l.Source + ":" + l.ID }

var validStatuses = map[string]bool{"active": true, "pending": true, "sold": true}

// LoadFile reads a JSON array of listings and prepares them for searching.
// Records that fail validation are skipped and reported in problems, so one
// bad record in a feed doesn't take down the whole service.
func LoadFile(path string) (listings []Listing, problems []error, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read listings: %w", err)
	}
	var raw []Listing
	err = json.Unmarshal(data, &raw)
	if err != nil {
		return nil, nil, fmt.Errorf("parse listings: %w", err)
	}
	listings, problems = Prepare(raw)
	return listings, problems, nil
}

// Prepare validates raw listings and fills in the precomputed fields.
func Prepare(raw []Listing) ([]Listing, []error) {
	var out []Listing
	var problems []error
	seen := map[string]bool{}

	for i, l := range raw {
		err := validate(l)
		if err != nil {
			problems = append(problems, fmt.Errorf("record %d (%s): %w", i, l.Key(), err))
			continue
		}
		if seen[l.Key()] {
			problems = append(problems, fmt.Errorf("record %d: duplicate key %s, keeping the first", i, l.Key()))
			continue
		}
		seen[l.Key()] = true

		l.listed, _ = time.Parse(time.DateOnly, l.ListedDate) // already validated
		l.cityKey = NormalizeCity(l.City)
		l.addressKey = NormalizeText(l.Address)
		l.descTokens = Tokenize(l.Description)
		out = append(out, l)
	}
	return out, problems
}

func validate(l Listing) error {
	switch {
	case strings.TrimSpace(l.ID) == "":
		return fmt.Errorf("missing id")
	case strings.TrimSpace(l.Source) == "":
		return fmt.Errorf("missing source")
	case l.Price < 0:
		return fmt.Errorf("negative price")
	case l.Bedrooms < 0:
		return fmt.Errorf("negative bedrooms")
	case !validStatuses[l.Status]:
		return fmt.Errorf("unknown status %q", l.Status)
	}
	_, err := time.Parse(time.DateOnly, l.ListedDate)
	if err != nil {
		return fmt.Errorf("bad listedDate %q", l.ListedDate)
	}
	return nil
}

// NormalizeCity makes "  springfield ", "Springfield" and "SPRINGFIELD" compare equal.
func NormalizeCity(s string) string { return NormalizeText(s) }

// NormalizeText lowercases s and collapses runs of whitespace to one space.
func NormalizeText(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// Tokenize lowercases text and splits it into words on anything that isn't a
// letter or digit. "Pet friendly." -> ["pet", "friendly"].
func Tokenize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}
