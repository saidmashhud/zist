// Package main implements the Zist booking management service.
//
// Handles reservation creation and status management.
// Runs on port 8002 (BOOKINGS_PORT env).
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
)

type Booking struct {
	ID          string `json:"id"`
	ListingID   string `json:"listingId"`
	GuestID     string `json:"guestId"`
	CheckIn     string `json:"checkIn"`  // YYYY-MM-DD
	CheckOut    string `json:"checkOut"` // YYYY-MM-DD
	Guests      int    `json:"guests"`
	TotalAmount string `json:"totalAmount"` // decimal string
	Currency    string `json:"currency"`
	Status      string `json:"status"` // pending | confirmed | cancelled
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
}

type server struct {
	db *sql.DB
}

func main() {
	port  := getenv("BOOKINGS_PORT", "8002")
	dbURL := getenv("DATABASE_URL", "postgres://dev:dev@db:5432/zist?sslmode=disable")

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

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	r.Route("/bookings", func(r chi.Router) {
		r.Get("/", s.listBookings)
		r.Post("/", s.createBooking)
		r.Get("/{id}", s.getBooking)
		r.Post("/{id}/cancel", s.cancelBooking)
		r.Post("/{id}/confirm", s.confirmBooking)
	})

	slog.Info("Bookings service starting", "port", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		slog.Error("bookings service failed", "err", err)
		os.Exit(1)
	}
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS bookings (
			id           TEXT PRIMARY KEY,
			listing_id   TEXT NOT NULL,
			guest_id     TEXT NOT NULL,
			check_in     DATE NOT NULL,
			check_out    DATE NOT NULL,
			guests       INT  NOT NULL DEFAULT 1,
			total_amount TEXT NOT NULL,
			currency     TEXT NOT NULL DEFAULT 'USD',
			status       TEXT NOT NULL DEFAULT 'pending'
			             CHECK (status IN ('pending','confirmed','cancelled')),
			created_at   BIGINT NOT NULL,
			updated_at   BIGINT NOT NULL
		)
	`)
	return err
}

func (s *server) listBookings(w http.ResponseWriter, r *http.Request) {
	guestID := r.URL.Query().Get("guestId")
	var rows *sql.Rows
	var err error

	if guestID != "" {
		rows, err = s.db.QueryContext(r.Context(),
			`SELECT id, listing_id, guest_id, check_in::text, check_out::text, guests, total_amount, currency, status, created_at, updated_at
			 FROM bookings WHERE guest_id = $1 ORDER BY created_at DESC LIMIT 50`, guestID)
	} else {
		rows, err = s.db.QueryContext(r.Context(),
			`SELECT id, listing_id, guest_id, check_in::text, check_out::text, guests, total_amount, currency, status, created_at, updated_at
			 FROM bookings ORDER BY created_at DESC LIMIT 50`)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db query failed")
		return
	}
	defer rows.Close()

	var bookings []Booking
	for rows.Next() {
		var b Booking
		if err := rows.Scan(&b.ID, &b.ListingID, &b.GuestID, &b.CheckIn, &b.CheckOut,
			&b.Guests, &b.TotalAmount, &b.Currency, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "scan failed")
			return
		}
		bookings = append(bookings, b)
	}
	if bookings == nil {
		bookings = []Booking{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"bookings": bookings})
}

func (s *server) createBooking(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ListingID   string `json:"listingId"`
		GuestID     string `json:"guestId"`
		CheckIn     string `json:"checkIn"`
		CheckOut    string `json:"checkOut"`
		Guests      int    `json:"guests"`
		TotalAmount string `json:"totalAmount"`
		Currency    string `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ListingID == "" || req.GuestID == "" || req.CheckIn == "" || req.CheckOut == "" {
		writeError(w, http.StatusUnprocessableEntity, "listingId, guestId, checkIn, checkOut are required")
		return
	}

	now := time.Now().Unix()
	b := Booking{
		ID:          uuid.NewString(),
		ListingID:   req.ListingID,
		GuestID:     req.GuestID,
		CheckIn:     req.CheckIn,
		CheckOut:    req.CheckOut,
		Guests:      req.Guests,
		TotalAmount: req.TotalAmount,
		Currency:    orDefault(req.Currency, "USD"),
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err := s.db.ExecContext(r.Context(),
		`INSERT INTO bookings (id, listing_id, guest_id, check_in, check_out, guests, total_amount, currency, status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		b.ID, b.ListingID, b.GuestID, b.CheckIn, b.CheckOut, b.Guests, b.TotalAmount, b.Currency, b.Status, b.CreatedAt, b.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "insert failed")
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *server) getBooking(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var b Booking
	err := s.db.QueryRowContext(r.Context(),
		`SELECT id, listing_id, guest_id, check_in::text, check_out::text, guests, total_amount, currency, status, created_at, updated_at
		 FROM bookings WHERE id = $1`, id).
		Scan(&b.ID, &b.ListingID, &b.GuestID, &b.CheckIn, &b.CheckOut,
			&b.Guests, &b.TotalAmount, &b.Currency, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "booking not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *server) confirmBooking(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	now := time.Now().Unix()
	result, err := s.db.ExecContext(r.Context(),
		`UPDATE bookings SET status='confirmed', updated_at=$1 WHERE id=$2 AND status='pending'`,
		now, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeError(w, http.StatusNotFound, "booking not found or not pending")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) cancelBooking(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	now := time.Now().Unix()
	result, err := s.db.ExecContext(r.Context(),
		`UPDATE bookings SET status='cancelled', updated_at=$1 WHERE id=$2 AND status != 'cancelled'`,
		now, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeError(w, http.StatusNotFound, "booking not found or already cancelled")
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
