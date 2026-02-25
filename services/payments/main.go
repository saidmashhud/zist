// Package main implements the Zist payments microservice.
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	mashgate "github.com/saidmashhud/mashgate/packages/sdk-go"
	zistauth "github.com/saidmashhud/zist/internal/auth"
	"github.com/saidmashhud/zist/internal/dedup"
	"github.com/saidmashhud/zist/services/payments/handler"
)

func main() {
	shutdownOTel, err := setupOpenTelemetry(context.Background(), "zist-payments")
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

	if cfg.WebhookSecret == "" {
		slog.Error("MASHGATE_WEBHOOK_SECRET env var is required")
		os.Exit(1)
	}
	if cfg.InternalToken == "" {
		slog.Error("INTERNAL_TOKEN env var is required")
		os.Exit(1)
	}

	// Dedup store: PostgreSQL if DATABASE_URL is set, else in-memory.
	var dedupStore handler.DedupChecker
	if cfg.DatabaseURL != "" {
		db, err := sql.Open("postgres", cfg.DatabaseURL)
		if err != nil {
			slog.Error("failed to open dedup DB", "err", err)
			os.Exit(1)
		}
		defer db.Close()
		pgDedup, err := dedup.NewPgStore(db, 48*time.Hour)
		if err != nil {
			slog.Error("failed to init PgStore", "err", err)
			os.Exit(1)
		}
		dedupStore = pgDedup
		slog.Info("using PostgreSQL-backed dedup store")
	} else {
		dedupStore = dedup.New(24 * time.Hour)
		slog.Warn("DATABASE_URL not set — using in-memory dedup (not crash-safe)")
	}

	mg := mashgate.New(cfg.MashgateURL, cfg.MashgateKey)

	// Service JWT client (optional — falls back to X-Internal-Token if not configured)
	var tokenClient *zistauth.ServiceTokenClient
	if cfg.AuthServiceURL != "" && cfg.AuthServiceKey != "" {
		tokenClient = zistauth.NewServiceTokenClient(cfg.AuthServiceURL, cfg.ServiceName, cfg.AuthServiceKey)
		slog.Info("service JWT auth enabled", "authService", cfg.AuthServiceURL)
	}

	bc := handler.NewBookingsClient(cfg.BookingsURL, cfg.InternalToken, tokenClient)
	h := handler.New(mg, cfg.WebhookSecret, bc, dedupStore)
	srv := &server{cfg: cfg, h: h}

	slog.Info("Payments service starting",
		"port", cfg.Port,
		"mashgate", cfg.MashgateURL,
		"bookings", cfg.BookingsURL)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		slog.Error("payments service failed", "err", err)
		os.Exit(1)
	}
}
