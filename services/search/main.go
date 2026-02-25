package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/saidmashhud/zist/services/search/handler"
	"github.com/saidmashhud/zist/services/search/store"
)

func main() {
	shutdownOTel, err := setupOpenTelemetry(context.Background(), "zist-search")
	if err != nil {
		slog.Error("failed to initialize OpenTelemetry", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := shutdownOTel(context.Background()); err != nil {
			slog.Warn("OpenTelemetry shutdown failed", "err", err)
		}
	}()

	cfg := LoadConfig()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to open db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("failed to ping db", "err", err)
		os.Exit(1)
	}

	if err := store.Migrate(db); err != nil {
		slog.Error("migration failed", "err", err)
		os.Exit(1)
	}

	s := &server{
		cfg: cfg,
		h:   handler.New(store.New(db)),
	}

	slog.Info("search service starting", "port", cfg.Port)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           s.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("search service failed", "err", err)
		os.Exit(1)
	}
}
