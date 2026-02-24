package store

import "database/sql"

// Migrate runs idempotent DDL for the bookings table.
func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS bookings (
			id                   TEXT PRIMARY KEY,
			listing_id           TEXT NOT NULL,
			guest_id             TEXT NOT NULL,
			host_id              TEXT NOT NULL DEFAULT '',
			check_in             DATE NOT NULL,
			check_out            DATE NOT NULL,
			guests               INT  NOT NULL DEFAULT 1,
			total_amount         TEXT NOT NULL,
			platform_fee         TEXT NOT NULL DEFAULT '0',
			cleaning_fee         TEXT NOT NULL DEFAULT '0',
			currency             TEXT NOT NULL DEFAULT 'USD',
			status               TEXT NOT NULL DEFAULT 'payment_pending',
			cancellation_policy  TEXT NOT NULL DEFAULT 'flexible',
			message              TEXT NOT NULL DEFAULT '',
			checkout_id          TEXT,
			approved_at          BIGINT,
			expires_at           BIGINT,
			created_at           BIGINT NOT NULL,
			updated_at           BIGINT NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	cols := []string{
		`ALTER TABLE bookings ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE bookings ADD COLUMN IF NOT EXISTS host_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE bookings ADD COLUMN IF NOT EXISTS platform_fee TEXT NOT NULL DEFAULT '0'`,
		`ALTER TABLE bookings ADD COLUMN IF NOT EXISTS cleaning_fee TEXT NOT NULL DEFAULT '0'`,
		`ALTER TABLE bookings ADD COLUMN IF NOT EXISTS cancellation_policy TEXT NOT NULL DEFAULT 'flexible'`,
		`ALTER TABLE bookings ADD COLUMN IF NOT EXISTS message TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE bookings ADD COLUMN IF NOT EXISTS checkout_id TEXT`,
		`ALTER TABLE bookings ADD COLUMN IF NOT EXISTS approved_at BIGINT`,
		`ALTER TABLE bookings ADD COLUMN IF NOT EXISTS expires_at BIGINT`,
		`ALTER TABLE bookings ADD COLUMN IF NOT EXISTS payment_id TEXT`,
	}
	for _, col := range cols {
		if _, err := db.Exec(col); err != nil {
			return err
		}
	}

	// Normalize legacy statuses before enforcing strict lifecycle constraint.
	if _, err := db.Exec(`
		UPDATE bookings
		SET status = CASE
			WHEN status = 'pending' THEN 'pending_host_approval'
			WHEN status = 'cancelled' THEN 'cancelled_by_guest'
			WHEN status = 'canceled' THEN 'cancelled_by_guest'
			WHEN status = 'paid' THEN 'confirmed'
			ELSE status
		END
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_bookings_tenant_guest ON bookings(tenant_id, guest_id, created_at DESC)`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_bookings_tenant_host ON bookings(tenant_id, host_id, created_at DESC)`); err != nil {
		return err
	}

	_, _ = db.Exec(`ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_status_check`)
	_, err = db.Exec(`
		ALTER TABLE bookings ADD CONSTRAINT bookings_status_check
		CHECK (status IN (
			'pending_host_approval','payment_pending','confirmed',
			'cancelled_by_guest','cancelled_by_host','rejected','failed','completed'
		))
	`)
	return err
}
