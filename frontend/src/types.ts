// These mirror the Go API's JSON. If the API changes, this is the one place to update.

export type Status = "active" | "pending" | "sold";

export interface Listing {
  key: string; // "MLS_A:A1" (ids are only unique per source)
  id: string;
  source: string;
  address: string;
  city: string;
  state: string;
  zip: string;
  price: number;
  bedrooms: number;
  bathrooms: number;
  sqft: number;
  listedDate: string;
  status: Status;
  description: string;
  score: number;
  scoreBreakdown: { budgetFit?: number; recency: number };
}

export interface FieldError {
  field: string;
  message: string;
}

export interface SearchResponse {
  results: Listing[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
  warnings: FieldError[];
}

// Form values stay as strings so a half-typed input never gets coerced.
// The server is the source of truth for validation.
export interface SearchForm {
  keyword: string;
  address: string;
  city: string;
  minPrice: string;
  maxPrice: string;
  minBedrooms: string;
  targetBudget: string;
  status: "" | Status;
  pageSize: string;
}

export interface SearchParams extends SearchForm {
  page: number;
}
