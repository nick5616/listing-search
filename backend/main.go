package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"listingsearch/search"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	dataPath := flag.String("data", "data/sample_listings.json", "path to the listings JSON file")
	asOf := flag.String("as-of", "", "optional YYYY-MM-DD to use as 'today' for recency scoring (handy for demos)")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	listings, problems, err := search.LoadFile(*dataPath)
	if err != nil {
		log.Error("could not load listings", "error", err)
		os.Exit(1)
	}
	for _, p := range problems {
		log.Warn("skipped listing", "reason", p.Error())
	}
	log.Info("loaded listings", "count", len(listings), "skipped", len(problems))

	index := search.NewIndex(listings)
	if *asOf != "" {
		day, err := time.Parse(time.DateOnly, *asOf)
		if err != nil {
			log.Error("bad -as-of date, expected YYYY-MM-DD", "value", *asOf)
			os.Exit(1)
		}
		index.Now = func() time.Time { return day }
	}

	srv := &http.Server{
		Addr:              *addr,
		Handler:           (&server{index: index, log: log}).routes(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	go func() {
		log.Info("listening", "addr", *addr)
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Shut down cleanly on Ctrl+C so in-flight requests finish.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Info("stopped")
}
