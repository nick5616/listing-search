package search

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestBudgetFit(t *testing.T) {
	s := DefaultScoring
	cases := []struct {
		name          string
		price, target float64
		want          float64
	}{
		{"exact match", 500000, 500000, 1},
		{"10% under", 450000, 500000, 0.9},
		{"10% over costs double", 550000, 500000, 0.8},
		{"50% over bottoms out", 750000, 500000, 0},
		{"way over stays at 0", 2000000, 500000, 0},
		{"way under stays at 0", 0, 500000, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := s.BudgetFit(tc.price, tc.target)
			if !near(got, tc.want) {
				t.Errorf("BudgetFit(%v, %v) = %v, want %v", tc.price, tc.target, got, tc.want)
			}
		})
	}
}

func TestRecency(t *testing.T) {
	s := DefaultScoring
	got := s.Recency(asOf, asOf)
	if !near(got, 1) {
		t.Errorf("listed today: got %v, want 1", got)
	}
	got = s.Recency(asOf.AddDate(0, 0, -30), asOf)
	if !near(got, 0.5) {
		t.Errorf("one half-life old: got %v, want 0.5", got)
	}
	got = s.Recency(asOf.AddDate(0, 0, -60), asOf)
	if !near(got, 0.25) {
		t.Errorf("two half-lives old: got %v, want 0.25", got)
	}
	got = s.Recency(asOf.AddDate(0, 0, 10), asOf)
	if !near(got, 1) {
		t.Errorf("future date should count as new: got %v", got)
	}
}

func TestScore_WithoutBudgetIsRecencyOnly(t *testing.T) {
	l, _ := Prepare([]Listing{listing("MLS_A", "1", 500000, 3, "X", "2026-08-06", "active", "")})
	score, b := DefaultScoring.Score(l[0], nil, asOf) // 30 days old
	if score != 50 || b.BudgetFit != nil {
		t.Errorf("got score=%v breakdown=%+v, want 50 and no budget part", score, b)
	}
}

func TestScore_BlendsBudgetAndRecency(t *testing.T) {
	l, _ := Prepare([]Listing{listing("MLS_A", "1", 500000, 3, "X", "2026-08-06", "active", "")})
	score, b := DefaultScoring.Score(l[0], ptr(500000.0), asOf)
	// 0.7 * 1.0 (perfect budget fit) + 0.3 * 0.5 (one half-life) = 0.85
	if score != 85 || b.BudgetFit == nil || *b.BudgetFit != 1 {
		t.Errorf("got score=%v breakdown=%+v, want 85", score, b)
	}
}

func TestScore_StaysWithinZeroToHundred(t *testing.T) {
	l, _ := Prepare([]Listing{listing("MLS_A", "1", 9e9, 3, "X", "1990-01-01", "active", "")})
	score, _ := DefaultScoring.Score(l[0], ptr(100000.0), asOf)
	if score < 0 || score > 100 {
		t.Errorf("score out of range: %v", score)
	}
}
