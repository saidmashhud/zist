package store

import "database/sql"

// Migrate ensures PostGIS extension and geographic column exist.
// The search service reads the listings table owned by the listings service,
// so it only adds the geographic column — it never creates the table.
func Migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE EXTENSION IF NOT EXISTS postgis`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS location GEOMETRY(POINT, 4326)`,
		`CREATE INDEX IF NOT EXISTS idx_listings_location ON listings USING GIST(location) WHERE location IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_listings_search ON listings(status, city, max_guests, instant_book, average_rating DESC)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
