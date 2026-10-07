package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"listingsearch/search"
)

type server struct {
	index *search.Index
	log   *slog.Logger
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/listings/search", s.handleSearch)
	mux.HandleFunc("GET /api/cities", s.handleCities)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return s.logRequests(mux)
}

// errorResponse is the one shape every 4xx/5xx uses, so the UI can handle it in one place.
type errorResponse struct {
	Error   string              `json:"error"`
	Details []search.FieldError `json:"details,omitempty"`
}

func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q, errs := search.ParseQuery(r.URL.Query())
	if len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid_query", Details: errs})
		return
	}
	page, errs := s.index.Search(q)
	if len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid_query", Details: errs})
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *server) handleCities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string][]string{"cities": s.index.Cities()})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// statusRecorder lets the logging middleware see the response code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (s *server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		s.log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
