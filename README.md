# Listing Search

A full-stack search over property listings from multiple MLS feeds. A Go API filters, scores, ranks and paginates the sample data, and a React + TypeScript UI calls it and renders the ranked results.

## Running it

Requirements: Go 1.22+ and Node 20.19+ (or 22.12+).

```bash
# Terminal 1: API on http://localhost:8080
cd backend && go run .

# Terminal 2: UI on http://localhost:5173 (proxies /api to the Go server)
cd frontend && npm install && npm run dev
```

Recency depends on today's date, so scores drift from day to day. To pin "today" for a demo:

```bash
cd backend && go run . -as-of 2026-09-05
```

## Tests

```bash
make test            # or run each suite on its own:
cd backend && go test ./...
cd frontend && npm test
```

The Go tests are table-driven and run against a fixed clock. They cover the edge cases the brief names:

- **No matches:** an empty page 1 is a normal result, not an error.
- **Tied scores:** ties break by newer listing, then lower price, then `source:id`, so the order never changes between requests.
- **Invalid filters:** non-numbers, NaN, negatives, `minPrice > maxPrice`, fractional bedrooms, unknown status, and a keyword with no letters or digits. Every problem is reported at once, not just the first.
- **Pagination boundaries:** first, middle and last partial page; page size equal to and larger than the total; a page past the end.

They also cover the scoring math, word-prefix keyword matching, case and spacing in city names, the same `id` arriving from two feeds, bad records in the data file, and the HTTP layer (status codes and error shape). The frontend tests cover query building and request cancellation.

## Layout

```
backend/
  main.go          flags, data loading, server startup and graceful shutdown
  api.go           HTTP handlers, the single error shape, request logging
  search/          pure search logic, no net/http
    listing.go     Listing type, validation, precomputed fields
    query.go       query params -> validated Query, collecting every error
    score.go       relevance scoring (weights live in one Scoring struct)
    search.go      filter, score, sort, paginate
frontend/src/
  App.tsx          state and data fetching
  SearchForm.tsx   inputs, with the server's error shown under each field
  Results.tsx      ranked list, pagination, loading / empty / error states
  api.ts           fetch wrapper and query building
  types.ts         TypeScript mirror of the API's JSON
```

## API

`GET /api/listings/search`

| Param                  | Meaning                                                                              |
| ---------------------- | ------------------------------------------------------------------------------------ |
| `minPrice`, `maxPrice` | Inclusive price bounds                                                               |
| `minBedrooms`          | Whole number, inclusive                                                              |
| `city`                 | Ignores case and extra spaces                                                        |
| `keyword`              | Every word must start a word in the description ("pet" matches "pets", not "carpet") |
| `address`              | Address contains this text, ignoring case and extra spaces ("main st" matches "123 Main St, Apt 4B") |
| `targetBudget`         | Adds budget fit to the ranking                                                       |
| `status`               | `active`, `pending`, `sold`, or a comma list. Default: any                           |
| `page`                 | 1-based, default 1                                                                   |
| `pageSize`             | 1 to 50, default 10                                                                  |

A successful response:

```json
{
    "results": [
        {
            "key": "MLS_A:A1",
            "address": "123 Main St, Apt 4B",
            "price": 450000,
            "bedrooms": 2,
            "score": 95.5,
            "scoreBreakdown": { "budgetFit": 1, "recency": 0.851 }
        }
    ],
    "page": 1,
    "pageSize": 10,
    "total": 12,
    "totalPages": 2,
    "warnings": []
}
```

Every 400 uses one shape, so the UI can show each message next to the field it belongs to:

```json
{
    "error": "invalid_query",
    "details": [
        {
            "field": "minPrice",
            "message": "minPrice (600000) can't be greater than maxPrice (500000)"
        }
    ]
}
```

Also: `GET /api/cities` (city suggestions in the UI) and `GET /healthz`.

## Scoring

Relevance is a score from 0 to 100, built from two parts.

**Budget fit (70%)**, used only when `targetBudget` is set. It is 1.0 when the price equals the budget and falls off with the percentage difference. Going over budget costs twice as much as coming in under, because buyers tend to treat a budget as a ceiling:

| Price vs. budget | Budget fit |
| ---------------- | ---------- |
| 10% under        | 0.90       |
| 10% over         | 0.80       |
| 50% or more over | 0          |

**Recency (30%)** halves every 30 days after `listedDate`. A listing posted today scores 1.0, one from a month ago 0.5, and one from two months ago 0.25. Future dates count as new.

```
score = 100 × (0.7 × budgetFit + 0.3 × recency)
score = 100 × recency        (when there's no targetBudget)
```

Why this shape:

- **Percentages, not dollars.** $50k off means something very different at $300k than at $3M.
- **Budget outweighs recency.** Price is the buyer's hard constraint; freshness is a nudge.
- **Exponential decay.** New listings matter most in their first weeks, and the score never goes negative.
- **One struct of knobs.** The weights, half-life and over-budget penalty live in `search.Scoring`, so they're easy to tune and test.

Each result includes `scoreBreakdown`, and the UI shows it next to the score so the ranking can be checked by eye.

## Invalid input

| Input                                                                                      | Response                                                    |
| ------------------------------------------------------------------------------------------ | ----------------------------------------------------------- |
| `minPrice > maxPrice`                                                                      | 400, naming both values                                     |
| `pageSize` of 0, negative, or over 50                                                      | 400                                                         |
| `page` past the last page                                                                  | 400 that says what the last page is                         |
| No matches on page 1                                                                       | 200 with an empty list                                      |
| A city with no listings at all                                                             | 200, empty, plus a warning listing the cities that do exist |
| Non-numbers, NaN, negatives, fractional bedrooms, unknown status, punctuation-only keyword | 400                                                         |
| Blank fields                                                                               | Treated as "no filter"                                      |

Validation lives only on the server. The UI sends exactly what was typed and shows the server's messages, so there is one set of rules to maintain. Records in the data file that fail validation are skipped and logged at startup, so one bad record doesn't take the service down.

## UI notes

- Results update live: the search runs 300 ms after typing pauses (Enter runs it right away), so typing doesn't fire a request on every keystroke. Changing any filter goes back to page 1, and changing pages keeps the filters that produced the results.
- When a new search starts, the previous request is aborted, so a slow old response can't overwrite a newer one.
- Results show the score and its parts, address, price, beds, baths, square feet, status, listed date and `source:id`.
- The loading, no-results, warning and error states are all handled. The page is kept deliberately simple, as the brief suggests.

## What I noticed in the data

- **The same home appears in both feeds.** `A1`/`B7`, `A2`/`B8`, `A3`/`B9` and `A5`/`B11` are the same properties with different address spellings (`St` vs `Street`, `Apt` vs `Unit`), slightly different prices and dates, and in one case a different zip (22150 vs 22151). For now they are separate results, identified by `source:id`. Merging them is the first item under "Next steps".
- **IDs collide across sources by design**, so the API returns a `key` of `source:id` and the UI uses it as the React key.
- **Keyword search can't read negation.** "pets" matches "No pets". Telling those apart would take structured data or a classifier.
- **Status defaults to "any".** Showing only `active` by default is a reasonable product call I'd want to confirm.

## Trade-offs

- **In-memory linear scan.** It's the simplest correct thing for a dozen listings. The `search` package doesn't import `net/http`, so swapping the slice for a database or search index leaves the handlers alone.
- **Offset pagination.** It's fine for static data. If listings changed between requests, cursor pagination keyed on `(score, listedDate, price, key)` would stop items from shifting between pages.
- **Simple keyword matching.** It matches word prefixes and requires every word. There's no stemming, synonyms or typo tolerance.
- **Scores are computed per request.** Recency depends on today's date and budget fit depends on the query, so there's nothing useful to precompute.
- **Out of scope:** auth, rate limiting and caching.

## Next steps

1. **Merge cross-feed duplicates.** Normalize addresses (lowercase, expand `St`/`Ave`/`Ct`, unify `Apt`/`Unit`) and treat two records as the same home when the normalized address and unit match, or when the coordinates are within a few meters. Keep one canonical record and remember which feeds it came from.
2. **Sorting options** (price, newest) next to relevance, and filters stored in the URL so a search can be shared and the back button works.
3. **Location search** using the latitude and longitude already in the data.
4. **Real search infrastructure at scale:** ingest the feeds into Postgres or Elasticsearch and push filtering and sorting into the query.
