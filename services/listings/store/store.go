// Package store implements PostgreSQL persistence for the listings service.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/saidmashhud/zist/services/listings/domain"
)

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// Store wraps a PostgreSQL connection and provides typed query methods.
type Store struct {
	db *sql.DB
}

// New creates a Store backed by db.
func New(db *sql.DB) *Store { return &Store{db: db} }

// ─── SELECT helper ────────────────────────────────────────────────────────────

const listingColumns = `
	id, title, description, city, country, address,
	type, bedrooms, beds, bathrooms, max_guests,
	amenities, rules,
	price_per_night, currency, cleaning_fee, deposit,
	min_nights, max_nights,
	cancellation_policy, instant_book,
	status, average_rating, review_count,
	host_id, created_at, updated_at`

func scanListing(scan func(dest ...any) error) (domain.Listing, error) {
	var l domain.Listing
	var amenitiesRaw, rulesRaw []byte
	err := scan(
		&l.ID, &l.Title, &l.Description, &l.City, &l.Country, &l.Address,
		&l.Type, &l.Bedrooms, &l.Beds, &l.Bathrooms, &l.MaxGuests,
		&amenitiesRaw, &rulesRaw,
		&l.PricePerNight, &l.Currency, &l.CleaningFee, &l.Deposit,
		&l.MinNights, &l.MaxNights,
		&l.CancellationPolicy, &l.InstantBook,
		&l.Status, &l.AverageRating, &l.ReviewCount,
		&l.HostID, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return l, err
	}
	if len(amenitiesRaw) > 0 {
		json.Unmarshal(amenitiesRaw, &l.Amenities) //nolint:errcheck
	}
	if len(rulesRaw) > 0 {
		json.Unmarshal(rulesRaw, &l.Rules) //nolint:errcheck
	}
	if l.Amenities == nil {
		l.Amenities = []string{}
	}
	return l, nil
}

// ─── Listing queries ──────────────────────────────────────────────────────────

// Get returns a single listing by ID. Returns ErrNotFound if it doesn't exist.
func (s *Store) Get(ctx context.Context, id string) (domain.Listing, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+listingColumns+` FROM listings WHERE id = $1`, id)
	l, err := scanListing(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return l, ErrNotFound
	}
	return l, err
}

// GetForTenant returns a single listing by ID within tenant scope.
func (s *Store) GetForTenant(ctx context.Context, tenantID, id string) (domain.Listing, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+listingColumns+` FROM listings WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	l, err := scanListing(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return l, ErrNotFound
	}
	return l, err
}

// List returns active listings with optional city/status filter.
func (s *Store) List(ctx context.Context, statusFilter, city string, limit int) ([]domain.Listing, error) {
	if statusFilter == "" {
		statusFilter = "active"
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+listingColumns+`
		 FROM listings
		 WHERE ($1 = '' OR status = $1)
		   AND ($2 = '' OR LOWER(city) = LOWER($2))
		 ORDER BY created_at DESC LIMIT $3`,
		statusFilter, city, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectListings(rows)
}

// ListByHost returns all listings owned by hostID within tenant scope.
func (s *Store) ListByHost(ctx context.Context, tenantID, hostID string) ([]domain.Listing, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+listingColumns+`
		 FROM listings WHERE tenant_id = $1 AND host_id = $2
		 ORDER BY created_at DESC LIMIT 100`,
		tenantID, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectListings(rows)
}

// Search runs the full search query with availability filtering.
func (s *Store) Search(ctx context.Context, f domain.SearchFilters) ([]domain.Listing, error) {
	args := []any{}
	conditions := []string{"l.status = 'active'"}
	argN := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	if f.City != "" {
		conditions = append(conditions, "LOWER(l.city) = LOWER("+argN(f.City)+")")
	}
	if f.Guests > 1 {
		conditions = append(conditions, "l.max_guests >= "+argN(f.Guests))
	}
	if f.Type != "" {
		conditions = append(conditions, "l.type = "+argN(f.Type))
	}
	if f.MinPrice != "" {
		conditions = append(conditions, "l.price_per_night::numeric >= "+argN(f.MinPrice)+"::numeric")
	}
	if f.MaxPrice != "" {
		conditions = append(conditions, "l.price_per_night::numeric <= "+argN(f.MaxPrice)+"::numeric")
	}
	if f.InstantBookOnly {
		conditions = append(conditions, "l.instant_book = true")
	}
	for _, amenity := range f.Amenities {
		amenity = strings.TrimSpace(amenity)
		if amenity != "" {
			conditions = append(conditions, "l.amenities @> "+argN(`["`+amenity+`"]`)+"::jsonb")
		}
	}

	if f.CheckIn != "" && f.CheckOut != "" {
		ciArg := argN(f.CheckIn)
		coArg := argN(f.CheckOut)
		conditions = append(conditions, `
			NOT EXISTS (
				SELECT 1 FROM listing_availability av
				WHERE av.listing_id = l.id
				  AND av.date >= `+ciArg+`::date
				  AND av.date < `+coArg+`::date
				  AND av.status IN ('blocked', 'booked')
			)`)
	}

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `SELECT ` + listingColumns + `
		FROM listings l
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY l.average_rating DESC, l.created_at DESC
		LIMIT ` + argN(limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectListings(rows)
}

// Create inserts a new listing and returns the persisted record.
func (s *Store) Create(ctx context.Context, in domain.CreateListingInput) (domain.Listing, error) {
	amenitiesJSON, _ := json.Marshal(in.Amenities)
	rulesJSON, _ := json.Marshal(in.Rules)
	now := time.Now().Unix()
	id := uuid.NewString()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO listings (
			tenant_id, id, title, description, city, country, address,
			type, bedrooms, beds, bathrooms, max_guests,
			amenities, rules,
			price_per_night, currency, cleaning_fee, deposit,
			min_nights, max_nights,
			cancellation_policy, instant_book,
			status, host_id, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,
			$8,$9,$10,$11,$12,
			$13,$14,
			$15,$16,$17,$18,
			$19,$20,
			$21,$22,
			'draft',$23,$24,$25
		)`,
		in.TenantID, id, in.Title, in.Description, in.City, in.Country, in.Address,
		in.Type, in.Bedrooms, in.Beds, in.Bathrooms, in.MaxGuests,
		amenitiesJSON, rulesJSON,
		in.PricePerNight, in.Currency, in.CleaningFee, in.Deposit,
		in.MinNights, in.MaxNights,
		in.CancellationPolicy, in.InstantBook,
		in.HostID, now, now,
	)
	if err != nil {
		return domain.Listing{}, err
	}
	return s.GetForTenant(ctx, in.TenantID, id)
}

// Update applies a partial update and returns the updated record.
func (s *Store) Update(ctx context.Context, id string, in domain.UpdateListingInput) (domain.Listing, error) {
	setClauses := []string{"updated_at = $1"}
	args := []any{time.Now().Unix()}
	add := func(col string, val any) {
		args = append(args, val)
		setClauses = append(setClauses, col+" = $"+strconv.Itoa(len(args)))
	}

	if in.Title != nil {
		add("title", *in.Title)
	}
	if in.Description != nil {
		add("description", *in.Description)
	}
	if in.Address != nil {
		add("address", *in.Address)
	}
	if in.Type != nil {
		add("type", *in.Type)
	}
	if in.Bedrooms != nil {
		add("bedrooms", *in.Bedrooms)
	}
	if in.Beds != nil {
		add("beds", *in.Beds)
	}
	if in.Bathrooms != nil {
		add("bathrooms", *in.Bathrooms)
	}
	if in.MaxGuests != nil {
		add("max_guests", *in.MaxGuests)
	}
	if in.Amenities != nil {
		b, _ := json.Marshal(in.Amenities)
		add("amenities", b)
	}
	if in.Rules != nil {
		b, _ := json.Marshal(*in.Rules)
		add("rules", b)
	}
	if in.PricePerNight != nil {
		add("price_per_night", *in.PricePerNight)
	}
	if in.Currency != nil {
		add("currency", *in.Currency)
	}
	if in.CleaningFee != nil {
		add("cleaning_fee", *in.CleaningFee)
	}
	if in.Deposit != nil {
		add("deposit", *in.Deposit)
	}
	if in.MinNights != nil {
		add("min_nights", *in.MinNights)
	}
	if in.MaxNights != nil {
		add("max_nights", *in.MaxNights)
	}
	if in.CancellationPolicy != nil {
		add("cancellation_policy", *in.CancellationPolicy)
	}
	if in.InstantBook != nil {
		add("instant_book", *in.InstantBook)
	}
	if in.Status != nil {
		add("status", *in.Status)
	}

	args = append(args, id)
	query := "UPDATE listings SET " + strings.Join(setClauses, ", ") +
		" WHERE id = $" + strconv.Itoa(len(args))
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return domain.Listing{}, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return domain.Listing{}, ErrNotFound
	}
	return s.Get(ctx, id)
}

// SetStatus updates only the listing status (publish/unpublish/pause).
func (s *Store) SetStatus(ctx context.Context, id, status string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE listings SET status = $1, updated_at = $2 WHERE id = $3`,
		status, time.Now().Unix(), id)
	return err
}

// Delete removes a listing. Returns ErrNotFound if it doesn't exist.
func (s *Store) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM listings WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetHostID returns the host_id for id. Returns ErrNotFound if not found.
func (s *Store) GetHostID(ctx context.Context, id string) (string, error) {
	var hostID string
	err := s.db.QueryRowContext(ctx, `SELECT host_id FROM listings WHERE id = $1`, id).Scan(&hostID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return hostID, err
}

// GetHostIDForTenant returns the host_id for id scoped to tenant.
func (s *Store) GetHostIDForTenant(ctx context.Context, tenantID, id string) (string, error) {
	var hostID string
	err := s.db.QueryRowContext(ctx, `SELECT host_id FROM listings WHERE tenant_id = $1 AND id = $2`, tenantID, id).Scan(&hostID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return hostID, err
}

// GetPricingInfo returns price-relevant fields for price preview calculation.
func (s *Store) GetPricingInfo(ctx context.Context, id string) (pricePerNight, cleaningFee, currency string, minNights, maxNights int, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT price_per_night, cleaning_fee, currency, min_nights, max_nights
		 FROM listings WHERE id = $1`, id).
		Scan(&pricePerNight, &cleaningFee, &currency, &minNights, &maxNights)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return
}

// PhotoCount returns the number of photos attached to listing id.
func (s *Store) PhotoCount(ctx context.Context, id string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM listing_photos WHERE listing_id = $1`, id).Scan(&n)
	return n, err
}

// ─── Photos ───────────────────────────────────────────────────────────────────

// GetPhotos returns all photos for a listing ordered by sort_order.
func (s *Store) GetPhotos(ctx context.Context, listingID string) ([]domain.Photo, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, listing_id, url, caption, sort_order, created_at
		 FROM listing_photos WHERE listing_id = $1 ORDER BY sort_order ASC`, listingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var photos []domain.Photo
	for rows.Next() {
		var p domain.Photo
		if err := rows.Scan(&p.ID, &p.ListingID, &p.URL, &p.Caption, &p.SortOrder, &p.CreatedAt); err == nil {
			photos = append(photos, p)
		}
	}
	return photos, nil
}

// GetCoverPhoto returns the first photo of a listing (for search cards). Returns nil if none.
func (s *Store) GetCoverPhoto(ctx context.Context, listingID string) *domain.Photo {
	var p domain.Photo
	err := s.db.QueryRowContext(ctx,
		`SELECT id, listing_id, url, caption, sort_order, created_at
		 FROM listing_photos WHERE listing_id = $1 ORDER BY sort_order ASC LIMIT 1`, listingID).
		Scan(&p.ID, &p.ListingID, &p.URL, &p.Caption, &p.SortOrder, &p.CreatedAt)
	if err != nil {
		return nil
	}
	return &p
}

// AddPhoto inserts a new photo and returns it.
func (s *Store) AddPhoto(ctx context.Context, listingID, url, caption string, sortOrder int) (domain.Photo, error) {
	id := uuid.NewString()
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO listing_photos (id, listing_id, url, caption, sort_order, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, listingID, url, caption, sortOrder, now)
	if err != nil {
		return domain.Photo{}, err
	}
	return domain.Photo{ID: id, ListingID: listingID, URL: url, Caption: caption, SortOrder: sortOrder, CreatedAt: now}, nil
}

// ReorderPhotos updates sort_order for each (photoID, sortOrder) pair in a transaction.
func (s *Store) ReorderPhotos(ctx context.Context, listingID string, items []struct {
	ID        string
	SortOrder int
}) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	for _, item := range items {
		if _, err := tx.ExecContext(ctx,
			`UPDATE listing_photos SET sort_order = $1 WHERE id = $2 AND listing_id = $3`,
			item.SortOrder, item.ID, listingID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// DeletePhoto removes a photo. Returns ErrNotFound if it doesn't exist for this listing.
func (s *Store) DeletePhoto(ctx context.Context, listingID, photoID string) error {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM listing_photos WHERE id = $1 AND listing_id = $2`, photoID, listingID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Availability ─────────────────────────────────────────────────────────────

// GetCalendar returns all availability days in the given month YYYY-MM,
// filling missing days with {status: "available"}.
func (s *Store) GetCalendar(ctx context.Context, listingID, month string) ([]domain.AvailabilityDay, error) {
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, err
	}
	end := start.AddDate(0, 1, 0)

	rows, err := s.db.QueryContext(ctx,
		`SELECT date::text, status, COALESCE(price_override,''), COALESCE(booking_id,'')
		 FROM listing_availability
		 WHERE listing_id = $1 AND date >= $2::date AND date < $3::date
		 ORDER BY date`,
		listingID, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	overrides := map[string]domain.AvailabilityDay{}
	for rows.Next() {
		var d domain.AvailabilityDay
		if err := rows.Scan(&d.Date, &d.Status, &d.PriceOverride, &d.BookingID); err == nil {
			overrides[d.Date] = d
		}
	}

	var calendar []domain.AvailabilityDay
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		if entry, ok := overrides[dateStr]; ok {
			calendar = append(calendar, entry)
		} else {
			calendar = append(calendar, domain.AvailabilityDay{Date: dateStr, Status: "available"})
		}
	}
	return calendar, nil
}

// CheckAvailability returns dates in [checkIn, checkOut) that are blocked or booked.
func (s *Store) CheckAvailability(ctx context.Context, listingID, checkIn, checkOut string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT date::text FROM listing_availability
		 WHERE listing_id = $1 AND date >= $2::date AND date < $3::date
		   AND status IN ('blocked', 'booked')`,
		listingID, checkIn, checkOut)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var conflicts []string
	for rows.Next() {
		var d string
		if rows.Scan(&d) == nil {
			conflicts = append(conflicts, d)
		}
	}
	return conflicts, nil
}

// BlockDates marks the given dates as 'blocked'.
func (s *Store) BlockDates(ctx context.Context, listingID string, dates []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	for _, d := range dates {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO listing_availability (id, listing_id, date, status)
			VALUES ($1, $2, $3::date, 'blocked')
			ON CONFLICT (listing_id, date)
			DO UPDATE SET status = 'blocked', booking_id = NULL, price_override = NULL`,
			uuid.NewString(), listingID, d); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UnblockDates removes blocked entries (restores availability).
func (s *Store) UnblockDates(ctx context.Context, listingID string, dates []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	for _, d := range dates {
		tx.ExecContext(ctx, //nolint:errcheck
			`DELETE FROM listing_availability WHERE listing_id = $1 AND date = $2::date AND status = 'blocked'`,
			listingID, d)
	}
	return tx.Commit()
}

// SetPriceOverride upserts per-date price overrides.
func (s *Store) SetPriceOverride(ctx context.Context, listingID string, entries []struct {
	Date  string
	Price string
}) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	for _, e := range entries {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO listing_availability (id, listing_id, date, status, price_override)
			VALUES ($1, $2, $3::date, 'available', $4)
			ON CONFLICT (listing_id, date)
			DO UPDATE SET price_override = $4`,
			uuid.NewString(), listingID, e.Date, e.Price); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// MarkDatesBooked reserves dates for bookingID.
// Returns a non-empty conflict slice if any dates are already blocked/booked.
func (s *Store) MarkDatesBooked(ctx context.Context, tenantID, listingID, bookingID string, dates []string) ([]string, error) {
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM listings WHERE tenant_id = $1 AND id = $2)`, tenantID, listingID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	conflictRows, err := tx.QueryContext(ctx,
		`SELECT date::text FROM listing_availability
		 WHERE listing_id = $1 AND date = ANY($2::date[]) AND status IN ('blocked','booked')`,
		listingID, "{"+strings.Join(dates, ",")+"}",
	)
	if err != nil {
		return nil, err
	}
	var conflicts []string
	for conflictRows.Next() {
		var d string
		conflictRows.Scan(&d) //nolint:errcheck
		conflicts = append(conflicts, d)
	}
	conflictRows.Close()

	if len(conflicts) > 0 {
		return conflicts, nil
	}

	for _, d := range dates {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO listing_availability (id, listing_id, date, status, booking_id)
			VALUES ($1, $2, $3::date, 'booked', $4)
			ON CONFLICT (listing_id, date)
			DO UPDATE SET status = 'booked', booking_id = $4`,
			uuid.NewString(), listingID, d, bookingID); err != nil {
			return nil, err
		}
	}
	return nil, tx.Commit()
}

// UnmarkDatesBooked releases dates that were booked for bookingID.
func (s *Store) UnmarkDatesBooked(ctx context.Context, tenantID, listingID, bookingID string) error {
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM listings WHERE tenant_id = $1 AND id = $2)`, tenantID, listingID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	_, err := s.db.ExecContext(ctx,
		`DELETE FROM listing_availability WHERE listing_id = $1 AND booking_id = $2 AND status = 'booked'`,
		listingID, bookingID)
	return err
}

// GetPricesByDate returns per-day effective prices (using price_override where set) for [checkIn, checkOut).
func (s *Store) GetPricesByDate(ctx context.Context, listingID, basePrice, checkIn, checkOut string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT date::text, COALESCE(price_override, $1) AS effective_price
		 FROM (
		   SELECT generate_series($2::date, $3::date - interval '1 day', '1 day') AS date
		 ) dates
		 LEFT JOIN listing_availability av
		   ON av.listing_id = $4 AND av.date = dates.date`,
		basePrice, checkIn, checkOut, listingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	prices := map[string]string{}
	for rows.Next() {
		var dateStr, priceStr string
		if rows.Scan(&dateStr, &priceStr) == nil {
			prices[dateStr] = priceStr
		}
	}
	return prices, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func collectListings(rows *sql.Rows) ([]domain.Listing, error) {
	var listings []domain.Listing
	for rows.Next() {
		l, err := scanListing(rows.Scan)
		if err != nil {
			return nil, err
		}
		listings = append(listings, l)
	}
	if listings == nil {
		listings = []domain.Listing{}
	}
	return listings, nil
}
