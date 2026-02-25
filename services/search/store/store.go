package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/saidmashhud/zist/services/search/domain"
)

// Store provides read-only access to listings for search queries.
type Store struct{ db *sql.DB }

// New creates a new Store backed by the given database connection.
func New(db *sql.DB) *Store { return &Store{db: db} }

// Search executes a filtered, sorted search over active listings.
func (s *Store) Search(ctx context.Context, f domain.SearchFilters) ([]domain.SearchResult, int, error) {
	var (
		where []string
		args  []any
		idx   = 1
	)

	where = append(where, "l.status = 'active'")

	if f.City != "" {
		where = append(where, fmt.Sprintf("LOWER(l.city) = LOWER($%d)", idx))
		args = append(args, f.City)
		idx++
	}
	if f.Lat != 0 && f.Lng != 0 && f.RadiusKM > 0 {
		where = append(where, fmt.Sprintf(
			"l.location IS NOT NULL AND ST_DWithin(l.location::geography, ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography, $%d)",
			idx, idx+1, idx+2,
		))
		args = append(args, f.Lng, f.Lat, f.RadiusKM*1000) // metres
		idx += 3
	}
	if f.Guests > 0 {
		where = append(where, fmt.Sprintf("l.max_guests >= $%d", idx))
		args = append(args, f.Guests)
		idx++
	}
	if f.Type != "" {
		where = append(where, fmt.Sprintf("l.type = $%d", idx))
		args = append(args, f.Type)
		idx++
	}
	if f.MinPrice != "" {
		where = append(where, fmt.Sprintf("l.price_per_night::numeric >= $%d::numeric", idx))
		args = append(args, f.MinPrice)
		idx++
	}
	if f.MaxPrice != "" {
		where = append(where, fmt.Sprintf("l.price_per_night::numeric <= $%d::numeric", idx))
		args = append(args, f.MaxPrice)
		idx++
	}
	if f.InstantBookOnly {
		where = append(where, "l.instant_book = true")
	}
	if len(f.Amenities) > 0 {
		where = append(where, fmt.Sprintf("l.amenities @> $%d::jsonb", idx))
		b, _ := json.Marshal(f.Amenities)
		args = append(args, string(b))
		idx++
	}

	// Availability: exclude listings that have blocked/booked dates in the requested range.
	if f.CheckIn != "" && f.CheckOut != "" {
		where = append(where, fmt.Sprintf(`NOT EXISTS (
			SELECT 1 FROM listing_availability a
			WHERE a.listing_id = l.id
			  AND a.date >= $%d::date AND a.date < $%d::date
			  AND a.status IN ('blocked','booked')
		)`, idx, idx+1))
		args = append(args, f.CheckIn, f.CheckOut)
		idx += 2
	}

	// Distance select expression
	distExpr := "NULL::float8"
	if f.Lat != 0 && f.Lng != 0 {
		distExpr = fmt.Sprintf(
			"ST_Distance(l.location::geography, ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography) / 1000.0",
			idx, idx+1,
		)
		args = append(args, f.Lng, f.Lat)
		idx += 2
	}

	orderBy := "l.average_rating DESC, l.created_at DESC"
	switch f.SortBy {
	case "price":
		orderBy = "l.price_per_night::numeric ASC"
	case "distance":
		if f.Lat != 0 && f.Lng != 0 {
			orderBy = "distance_km ASC NULLS LAST"
		}
	}

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM listings l WHERE %s`, strings.Join(where, " AND "))
	// Count uses the same args minus the distance-select args (last 2 if geo), but we reuse args here.
	// Build separate arg lists for count (without the final distance args).
	countArgs := args[:len(args)]
	if f.Lat != 0 && f.Lng != 0 {
		countArgs = args[:len(args)-2] // strip the trailing distance select args
	}

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT l.id, l.title, l.city, l.country, l.type,
		       l.price_per_night, l.currency, l.max_guests, l.instant_book,
		       l.average_rating, l.review_count, l.amenities,
		       %s AS distance_km,
		       (SELECT p.url FROM listing_photos p WHERE p.listing_id = l.id ORDER BY p.sort_order LIMIT 1) AS cover_photo
		FROM listings l
		WHERE %s
		ORDER BY %s
		LIMIT %d OFFSET %d
	`, distExpr, strings.Join(where, " AND "), orderBy, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()

	var results []domain.SearchResult
	for rows.Next() {
		var r domain.SearchResult
		var amenitiesJSON string
		var distKM sql.NullFloat64
		var coverPhoto sql.NullString
		if err := rows.Scan(
			&r.ID, &r.Title, &r.City, &r.Country, &r.Type,
			&r.PricePerNight, &r.Currency, &r.MaxGuests, &r.InstantBook,
			&r.AverageRating, &r.ReviewCount, &amenitiesJSON,
			&distKM, &coverPhoto,
		); err != nil {
			return nil, 0, fmt.Errorf("scan: %w", err)
		}
		_ = json.Unmarshal([]byte(amenitiesJSON), &r.Amenities)
		if r.Amenities == nil {
			r.Amenities = []string{}
		}
		if distKM.Valid {
			d := distKM.Float64
			r.DistanceKM = &d
		}
		if coverPhoto.Valid {
			r.CoverPhoto = coverPhoto.String
		}
		results = append(results, r)
	}
	if results == nil {
		results = []domain.SearchResult{}
	}
	return results, total, rows.Err()
}

// UpdateLocation sets the PostGIS point for a listing (called via internal API).
func (s *Store) UpdateLocation(ctx context.Context, listingID string, lat, lng float64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE listings SET location = ST_SetSRID(ST_MakePoint($1, $2), 4326) WHERE id = $3`,
		lng, lat, listingID,
	)
	return err
}
