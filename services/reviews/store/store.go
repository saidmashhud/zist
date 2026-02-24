// Package store implements PostgreSQL persistence for the reviews service.
package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/saidmashhud/zist/services/reviews/domain"
)

// ErrNotFound is returned when a resource does not exist.
var ErrNotFound = errors.New("not found")

// ErrAlreadyReviewed is returned when a booking already has a review.
var ErrAlreadyReviewed = errors.New("booking already reviewed")

// Store wraps a PostgreSQL connection and provides typed review queries.
type Store struct {
	db *sql.DB
}

// New creates a Store backed by db.
func New(db *sql.DB) *Store { return &Store{db: db} }

func scanReview(scan func(dest ...any) error) (domain.Review, error) {
	var r domain.Review
	return r, scan(
		&r.ID, &r.BookingID, &r.ListingID,
		&r.GuestID, &r.HostID, &r.TenantID,
		&r.Rating, &r.Comment, &r.Reply,
		&r.CreatedAt, &r.UpdatedAt,
	)
}

// Create inserts a new review. Returns ErrAlreadyReviewed if the booking already has one.
func (s *Store) Create(ctx context.Context, in domain.CreateReviewInput) (domain.Review, error) {
	id := uuid.NewString()
	now := time.Now().Unix()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO reviews
			(id, booking_id, listing_id, guest_id, host_id, tenant_id, rating, comment, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		id, in.BookingID, in.ListingID, in.GuestID, in.HostID, in.TenantID,
		in.Rating, in.Comment, now, now,
	)
	if err != nil {
		// Unique constraint on booking_id
		if isUniqueViolation(err) {
			return domain.Review{}, ErrAlreadyReviewed
		}
		return domain.Review{}, err
	}
	return s.GetByID(ctx, id)
}

// GetByID returns a review by its ID.
func (s *Store) GetByID(ctx context.Context, id string) (domain.Review, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id,booking_id,listing_id,guest_id,host_id,tenant_id,rating,comment,reply,created_at,updated_at
		 FROM reviews WHERE id=$1`, id)
	r, err := scanReview(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

// ListByListing returns all reviews for a listing, newest first.
func (s *Store) ListByListing(ctx context.Context, listingID string, limit int) ([]domain.Review, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id,booking_id,listing_id,guest_id,host_id,tenant_id,rating,comment,reply,created_at,updated_at
		 FROM reviews WHERE listing_id=$1 ORDER BY created_at DESC LIMIT $2`,
		listingID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectReviews(rows)
}

// ListByGuest returns reviews written by a guest within a tenant.
func (s *Store) ListByGuest(ctx context.Context, tenantID, guestID string) ([]domain.Review, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id,booking_id,listing_id,guest_id,host_id,tenant_id,rating,comment,reply,created_at,updated_at
		 FROM reviews WHERE tenant_id=$1 AND guest_id=$2 ORDER BY created_at DESC LIMIT 100`,
		tenantID, guestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectReviews(rows)
}

// SetReply allows a host to reply to a review.
func (s *Store) SetReply(ctx context.Context, reviewID, hostID, reply string) (domain.Review, error) {
	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx,
		`UPDATE reviews SET reply=$1, updated_at=$2 WHERE id=$3 AND host_id=$4`,
		reply, now, reviewID, hostID)
	if err != nil {
		return domain.Review{}, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return domain.Review{}, ErrNotFound
	}
	return s.GetByID(ctx, reviewID)
}

// RatingSummary returns average rating and count for a listing.
func (s *Store) RatingSummary(ctx context.Context, listingID string) (avg float64, count int, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT COALESCE(AVG(rating),0), COUNT(*) FROM reviews WHERE listing_id=$1`, listingID).
		Scan(&avg, &count)
	return
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func collectReviews(rows *sql.Rows) ([]domain.Review, error) {
	var reviews []domain.Review
	for rows.Next() {
		r, err := scanReview(rows.Scan)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, r)
	}
	if reviews == nil {
		reviews = []domain.Review{}
	}
	return reviews, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "unique") || contains(err.Error(), "duplicate")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
