// Package main implements the Zist bookings microservice.
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/saidmashhud/zist/services/bookings/handler"
	"github.com/saidmashhud/zist/services/bookings/store"
)

func main() {
	shutdownOTel, err := setupOpenTelemetry(context.Background(), "zist-bookings")
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

	if cfg.InternalToken == "" {
		slog.Error("INTERNAL_TOKEN env var is required")
		os.Exit(1)
	}

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

	lc := handler.NewListingsClient(cfg.ListingsURL, cfg.InternalToken)
	h := handler.New(store.New(db), lc, cfg.FeeGuestPct)
	srv := &server{cfg: cfg, h: h}

	slog.Info("Bookings service starting", "port", cfg.Port)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		slog.Error("bookings service failed", "err", err)
		os.Exit(1)
	}
}
