package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/saidmashhud/zist/services/bookings/domain"
)

// ErrNotFound is returned when a booking is not found.
var ErrNotFound = errors.New("not found")

// bookingColumns is the SELECT list used by all queries.
const bookingColumns = `id, listing_id, guest_id, host_id,
	check_in::text, check_out::text, guests,
	total_amount, platform_fee, cleaning_fee, currency,
	status, cancellation_policy, message,
	checkout_id, approved_at, expires_at, payment_id, created_at, updated_at`

// Store provides all SQL operations for the bookings service.
type Store struct {
	db *sql.DB
}

// New returns a new Store.
func New(db *sql.DB) *Store { return &Store{db: db} }

// ─── scan ─────────────────────────────────────────────────────────────────────

func scanBooking(scan func(...any) error) (domain.Booking, error) {
	var b domain.Booking
	err := scan(
		&b.ID, &b.ListingID, &b.GuestID, &b.HostID,
		&b.CheckIn, &b.CheckOut, &b.Guests,
		&b.TotalAmount, &b.PlatformFee, &b.CleaningFee, &b.Currency,
		&b.Status, &b.CancellationPolicy, &b.Message,
		&b.CheckoutID, &b.ApprovedAt, &b.ExpiresAt, &b.PaymentID,
		&b.CreatedAt, &b.UpdatedAt,
	)
	return b, err
}

// ─── queries ─────────────────────────────────────────────────────────────────

// Get fetches a single booking by ID within tenant scope.
// Returns ErrNotFound if not found.
func (s *Store) Get(ctx context.Context, tenantID, id string) (domain.Booking, error) {
	b, err := scanBooking(s.db.QueryRowContext(ctx,
		`SELECT `+bookingColumns+` FROM bookings WHERE tenant_id = $1 AND id = $2`,
		tenantID, id).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Booking{}, ErrNotFound
	}
	return b, err
}

// ListByGuest returns all bookings for a guest (newest first, limit 50).
func (s *Store) ListByGuest(ctx context.Context, tenantID, guestID string) ([]domain.Booking, error) {
	return s.list(ctx,
		`SELECT `+bookingColumns+` FROM bookings WHERE tenant_id = $1 AND guest_id = $2 ORDER BY created_at DESC LIMIT 50`,
		tenantID, guestID)
}

// ListByHost returns all bookings on a host's listings (newest first, limit 100).
func (s *Store) ListByHost(ctx context.Context, tenantID, hostID string) ([]domain.Booking, error) {
	return s.list(ctx,
		`SELECT `+bookingColumns+` FROM bookings WHERE tenant_id = $1 AND host_id = $2 ORDER BY created_at DESC LIMIT 100`,
		tenantID, hostID)
}

func (s *Store) list(ctx context.Context, query, tenantID, userID string) ([]domain.Booking, error) {
	rows, err := s.db.QueryContext(ctx, query, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Booking
	for rows.Next() {
		b, err := scanBooking(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if out == nil {
		out = []domain.Booking{}
	}
	return out, rows.Err()
}

// ─── mutations ───────────────────────────────────────────────────────────────

// Create inserts a new booking.
func (s *Store) Create(ctx context.Context, tenantID string, b domain.Booking) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO bookings
			(tenant_id, id, listing_id, guest_id, host_id, check_in, check_out, guests,
			 total_amount, platform_fee, cleaning_fee, currency, status,
			 cancellation_policy, message, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		tenantID, b.ID, b.ListingID, b.GuestID, b.HostID, b.CheckIn, b.CheckOut, b.Guests,
		b.TotalAmount, b.PlatformFee, b.CleaningFee, b.Currency, b.Status,
		b.CancellationPolicy, b.Message, b.CreatedAt, b.UpdatedAt)
	return err
}

// Approve transitions a booking from pending_host_approval → payment_pending.
// Sets approved_at and expires_at. Returns false if the transition was rejected (concurrent update).
func (s *Store) Approve(ctx context.Context, tenantID, id string, expiresAt int64) (bool, error) {
	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx,
		`UPDATE bookings SET status = $1, approved_at = $2, expires_at = $3, updated_at = $4
		 WHERE tenant_id = $5 AND id = $6 AND status = $7`,
		domain.StatusPaymentPending, now, expiresAt, now, tenantID, id, domain.StatusPendingHostApproval)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

// Reject transitions a booking from pending_host_approval → rejected.
func (s *Store) Reject(ctx context.Context, tenantID, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE bookings SET status = $1, updated_at = $2 WHERE tenant_id = $3 AND id = $4`,
		domain.StatusRejected, time.Now().Unix(), tenantID, id)
	return err
}

// Cancel transitions a booking to a cancelled status.
func (s *Store) Cancel(ctx context.Context, tenantID, id, newStatus string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE bookings SET status = $1, updated_at = $2 WHERE tenant_id = $3 AND id = $4`,
		newStatus, time.Now().Unix(), tenantID, id)
	return err
}

// Confirm transitions a booking from payment_pending → confirmed.
// paymentID may be empty. Returns false if booking was not in payment_pending.
func (s *Store) Confirm(ctx context.Context, tenantID, id, paymentID string) (bool, error) {
	now := time.Now().Unix()
	var result sql.Result
	var err error
	if paymentID != "" {
		result, err = s.db.ExecContext(ctx,
			`UPDATE bookings SET status = $1, payment_id = $2, updated_at = $3
			 WHERE tenant_id = $4 AND id = $5 AND status = $6`,
			domain.StatusConfirmed, paymentID, now, tenantID, id, domain.StatusPaymentPending)
	} else {
		result, err = s.db.ExecContext(ctx,
			`UPDATE bookings SET status = $1, updated_at = $2
			 WHERE tenant_id = $3 AND id = $4 AND status = $5`,
			domain.StatusConfirmed, now, tenantID, id, domain.StatusPaymentPending)
	}
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

// Fail transitions a booking from payment_pending → failed.
// Returns the booking (for date release) or ErrNotFound.
func (s *Store) Fail(ctx context.Context, tenantID, id string) (domain.Booking, error) {
	b, err := scanBooking(s.db.QueryRowContext(ctx,
		`SELECT `+bookingColumns+` FROM bookings WHERE tenant_id = $1 AND id = $2 AND status = $3`,
		tenantID, id, domain.StatusPaymentPending).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Booking{}, ErrNotFound
	}
	if err != nil {
		return domain.Booking{}, err
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE bookings SET status = $1, updated_at = $2 WHERE tenant_id = $3 AND id = $4`,
		domain.StatusFailed, time.Now().Unix(), tenantID, id)
	return b, err
}

// SetCheckoutID stores the Mashgate checkout session ID.
// Returns false if the booking was not found.
func (s *Store) SetCheckoutID(ctx context.Context, tenantID, id, checkoutID string) (bool, error) {
	result, err := s.db.ExecContext(ctx,
		`UPDATE bookings SET checkout_id = $1, updated_at = $2 WHERE tenant_id = $3 AND id = $4`,
		checkoutID, time.Now().Unix(), tenantID, id)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}
