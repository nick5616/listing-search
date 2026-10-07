package search

import (
	"math"
	"time"
)

// Scoring holds the knobs for relevance. Keeping them in one struct makes the
// formula easy to explain, test, and tune.
type Scoring struct {
	BudgetWeight      float64 // share of the score from closeness to targetBudget
	RecencyWeight     float64 // share of the score from how new the listing is
	HalfLifeDays      float64 // a listing this many days old gets half the recency credit
	OverBudgetPenalty float64 // being over budget hurts this many times more than being under
}

var DefaultScoring = Scoring{
	BudgetWeight:      0.7,
	RecencyWeight:     0.3,
	HalfLifeDays:      30,
	OverBudgetPenalty: 2,
}

// Breakdown shows where a score came from, so the UI (and an interviewer) can see the reasoning.
type Breakdown struct {
	BudgetFit *float64 `json:"budgetFit,omitempty"` // 0..1, absent when no targetBudget
	Recency   float64  `json:"recency"`             // 0..1
}

// BudgetFit is 1.0 when price == target and falls off linearly with the
// percentage difference. Over-budget listings fall off faster:
//
//	10% under budget -> 0.90
//	10% over budget  -> 0.80 (with the default 2x penalty)
//	50%+ over budget -> 0.00
func (s Scoring) BudgetFit(price, target float64) float64 {
	diff := (price - target) / target
	if diff > 0 {
		diff *= s.OverBudgetPenalty
	}
	return clamp01(1 - math.Abs(diff))
}

// Recency decays by half every HalfLifeDays. Dates in the future count as brand new.
func (s Scoring) Recency(listed, now time.Time) float64 {
	ageDays := now.Sub(listed).Hours() / 24
	if ageDays < 0 {
		ageDays = 0
	}
	return math.Pow(0.5, ageDays/s.HalfLifeDays)
}

// Score returns relevance on a 0..100 scale, rounded to one decimal.
// Without a targetBudget, relevance is recency alone.
func (s Scoring) Score(l Listing, targetBudget *float64, now time.Time) (float64, Breakdown) {
	recency := s.Recency(l.listed, now)
	b := Breakdown{Recency: round(recency, 3)}

	if targetBudget == nil {
		return round(recency*100, 1), b
	}
	fit := s.BudgetFit(l.Price, *targetBudget)
	fitRounded := round(fit, 3)
	b.BudgetFit = &fitRounded

	total := s.BudgetWeight + s.RecencyWeight
	score := (s.BudgetWeight*fit + s.RecencyWeight*recency) / total
	return round(score*100, 1), b
}

func clamp01(x float64) float64 { return math.Max(0, math.Min(1, x)) }

func round(x float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(x*p) / p
}
