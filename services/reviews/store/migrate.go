package store

import "database/sql"

// Migrate runs idempotent DDL for the reviews tables.
func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS reviews (
			id          TEXT PRIMARY KEY,
			booking_id  TEXT NOT NULL,
			listing_id  TEXT NOT NULL,
			guest_id    TEXT NOT NULL,
			host_id     TEXT NOT NULL DEFAULT '',
			tenant_id   TEXT NOT NULL DEFAULT '',
			rating      INT  NOT NULL CHECK (rating BETWEEN 1 AND 5),
			comment     TEXT NOT NULL DEFAULT '',
			reply       TEXT NOT NULL DEFAULT '',
			created_at  BIGINT NOT NULL,
			updated_at  BIGINT NOT NULL,
			UNIQUE (booking_id)
		)
	`)
	if err != nil {
		return err
	}

	addCols := []string{
		`ALTER TABLE reviews ADD COLUMN IF NOT EXISTS reply TEXT NOT NULL DEFAULT ''`,
	}
	for _, col := range addCols {
		if _, err := db.Exec(col); err != nil {
			return err
		}
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_reviews_listing ON reviews (listing_id, created_at DESC)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_reviews_guest ON reviews (tenant_id, guest_id, created_at DESC)`)
	return err
}
