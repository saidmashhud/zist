// Package main implements the Zist property listings service.
//
// Provides CRUD operations for Airbnb-style property listings backed by PostgreSQL.
// Runs on port 8001 (LISTINGS_PORT env).
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	zistauth "github.com/saidmashhud/zist/internal/auth"
)

type Listing struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	City        string  `json:"city"`
	Country     string  `json:"country"`
	PricePerNight string `json:"pricePerNight"` // decimal string e.g. "150000.00"
	Currency    string  `json:"currency"`
	MaxGuests   int     `json:"maxGuests"`
	HostID      string  `json:"hostId"`
	CreatedAt   int64   `json:"createdAt"`
	UpdatedAt   int64   `json:"updatedAt"`
}

type server struct {
	db *sql.DB
}

func main() {
	port   := getenv("LISTINGS_PORT", "8001")
	dbURL  := getenv("DATABASE_URL", "postgres://dev:dev@db:5432/zist?sslmode=disable")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		slog.Error("failed to open db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("failed to ping db", "err", err)
		os.Exit(1)
	}

	if err := migrate(db); err != nil {
		slog.Error("migration failed", "err", err)
		os.Exit(1)
	}

	s := &server{db: db}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(zistauth.Middleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	r.Route("/listings", func(r chi.Router) {
		r.Get("/", s.listListings)   // public
		r.Get("/{id}", s.getListing) // public
		r.With(zistauth.RequireAuth, zistauth.RequireScope("zist.listings.manage")).Post("/", s.createListing)
		r.With(zistauth.RequireAuth, zistauth.RequireScope("zist.listings.manage")).Put("/{id}", s.updateListing)
		r.With(zistauth.RequireAuth, zistauth.RequireScope("zist.listings.manage")).Delete("/{id}", s.deleteListing)
	})

	slog.Info("Listings service starting", "port", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		slog.Error("listings service failed", "err", err)
		os.Exit(1)
	}
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS listings (
			id               TEXT PRIMARY KEY,
			title            TEXT NOT NULL,
			description      TEXT NOT NULL DEFAULT '',
			city             TEXT NOT NULL,
			country          TEXT NOT NULL,
			price_per_night  TEXT NOT NULL,
			currency         TEXT NOT NULL DEFAULT 'USD',
			max_guests       INT  NOT NULL DEFAULT 2,
			host_id          TEXT NOT NULL,
			created_at       BIGINT NOT NULL,
			updated_at       BIGINT NOT NULL
		)
	`)
	return err
}

func (s *server) listListings(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(),
		`SELECT id, title, description, city, country, price_per_night, currency, max_guests, host_id, created_at, updated_at
		 FROM listings ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db query failed")
		return
	}
	defer rows.Close()

	var listings []Listing
	for rows.Next() {
		var l Listing
		if err := rows.Scan(&l.ID, &l.Title, &l.Description, &l.City, &l.Country,
			&l.PricePerNight, &l.Currency, &l.MaxGuests, &l.HostID, &l.CreatedAt, &l.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "scan failed")
			return
		}
		listings = append(listings, l)
	}
	if listings == nil {
		listings = []Listing{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"listings": listings})
}

func (s *server) createListing(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title         string `json:"title"`
		Description   string `json:"description"`
		City          string `json:"city"`
		Country       string `json:"country"`
		PricePerNight string `json:"pricePerNight"`
		Currency      string `json:"currency"`
		MaxGuests     int    `json:"maxGuests"`
		HostID        string `json:"hostId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" || req.City == "" || req.PricePerNight == "" {
		writeError(w, http.StatusUnprocessableEntity, "title, city, and pricePerNight are required")
		return
	}

	now := time.Now().Unix()
	l := Listing{
		ID:            uuid.NewString(),
		Title:         req.Title,
		Description:   req.Description,
		City:          req.City,
		Country:       req.Country,
		PricePerNight: req.PricePerNight,
		Currency:      orDefault(req.Currency, "USD"),
		MaxGuests:     req.MaxGuests,
		HostID:        req.HostID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	_, err := s.db.ExecContext(r.Context(),
		`INSERT INTO listings (id, title, description, city, country, price_per_night, currency, max_guests, host_id, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		l.ID, l.Title, l.Description, l.City, l.Country, l.PricePerNight, l.Currency, l.MaxGuests, l.HostID, l.CreatedAt, l.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "insert failed")
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

func (s *server) getListing(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var l Listing
	err := s.db.QueryRowContext(r.Context(),
		`SELECT id, title, description, city, country, price_per_night, currency, max_guests, host_id, created_at, updated_at
		 FROM listings WHERE id = $1`, id).
		Scan(&l.ID, &l.Title, &l.Description, &l.City, &l.Country,
			&l.PricePerNight, &l.Currency, &l.MaxGuests, &l.HostID, &l.CreatedAt, &l.UpdatedAt)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "listing not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func (s *server) updateListing(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Title         string `json:"title"`
		Description   string `json:"description"`
		PricePerNight string `json:"pricePerNight"`
		MaxGuests     int    `json:"maxGuests"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	now := time.Now().Unix()
	result, err := s.db.ExecContext(r.Context(),
		`UPDATE listings SET title=$1, description=$2, price_per_night=$3, max_guests=$4, updated_at=$5
		 WHERE id=$6`,
		req.Title, req.Description, req.PricePerNight, req.MaxGuests, now, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeError(w, http.StatusNotFound, "listing not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) deleteListing(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	result, err := s.db.ExecContext(r.Context(), `DELETE FROM listings WHERE id=$1`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeError(w, http.StatusNotFound, "listing not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
