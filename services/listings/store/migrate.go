package store

import "database/sql"

// Migrate runs idempotent DDL to ensure all required tables exist.
func Migrate(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS listings (
			id              TEXT   PRIMARY KEY,
			title           TEXT   NOT NULL,
			description     TEXT   NOT NULL DEFAULT '',
			city            TEXT   NOT NULL,
			country         TEXT   NOT NULL DEFAULT '',
			price_per_night TEXT   NOT NULL,
			currency        TEXT   NOT NULL DEFAULT 'USD',
			max_guests      INT    NOT NULL DEFAULT 2,
			host_id         TEXT   NOT NULL,
			created_at      BIGINT NOT NULL,
			updated_at      BIGINT NOT NULL
		)
	`); err != nil {
		return err
	}

	newCols := []string{
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS tenant_id          TEXT    NOT NULL DEFAULT ''`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS address             TEXT    NOT NULL DEFAULT ''`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS type               TEXT    NOT NULL DEFAULT 'apartment'`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS bedrooms           INT     NOT NULL DEFAULT 1`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS beds               INT     NOT NULL DEFAULT 1`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS bathrooms          INT     NOT NULL DEFAULT 1`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS amenities          JSONB   NOT NULL DEFAULT '[]'`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS rules              JSONB   NOT NULL DEFAULT '{}'`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS cleaning_fee       TEXT    NOT NULL DEFAULT '0'`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS deposit            TEXT    NOT NULL DEFAULT '0'`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS min_nights         INT     NOT NULL DEFAULT 1`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS max_nights         INT     NOT NULL DEFAULT 365`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS cancellation_policy TEXT   NOT NULL DEFAULT 'moderate'`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS instant_book       BOOLEAN NOT NULL DEFAULT false`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS status             TEXT    NOT NULL DEFAULT 'active'`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS average_rating     NUMERIC(3,2) NOT NULL DEFAULT 0`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS review_count       INT     NOT NULL DEFAULT 0`,
	}
	for _, stmt := range newCols {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_listings_tenant_status_city ON listings(tenant_id, status, city, created_at DESC)`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS listing_photos (
			id         TEXT   PRIMARY KEY,
			listing_id TEXT   NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
			url        TEXT   NOT NULL,
			caption    TEXT   NOT NULL DEFAULT '',
			sort_order INT    NOT NULL DEFAULT 0,
			created_at BIGINT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_listing_photos_listing
			ON listing_photos(listing_id, sort_order);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS listing_availability (
			id             TEXT PRIMARY KEY,
			listing_id     TEXT NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
			date           DATE NOT NULL,
			status         TEXT NOT NULL DEFAULT 'available',
			price_override TEXT,
			booking_id     TEXT,
			UNIQUE(listing_id, date)
		);
		CREATE INDEX IF NOT EXISTS idx_availability_listing_date
			ON listing_availability(listing_id, date);
	`); err != nil {
		return err
	}

	return nil
}
