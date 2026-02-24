// Package dedup provides deduplication stores for webhook event IDs.
// PgStore is a PostgreSQL-backed persistent store that survives restarts.
package dedup

import (
	"context"
	"database/sql"
	"time"
)

// PgStore is a PostgreSQL-backed dedup store. It persists seen event IDs in a
// table so dedup state survives process restarts (important for at-least-once
// webhook delivery).
type PgStore struct {
	db  *sql.DB
	ttl time.Duration
}

// NewPgStore creates a persistent dedup store backed by PostgreSQL.
// It auto-creates the dedup table and starts a background cleanup goroutine.
func NewPgStore(db *sql.DB, ttl time.Duration) (*PgStore, error) {
	s := &PgStore{db: db, ttl: ttl}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	go s.cleanup()
	return s, nil
}

func (s *PgStore) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS webhook_dedup (
			event_id   TEXT PRIMARY KEY,
			seen_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

// Check returns true if eventID has already been seen (duplicate).
// If new, the ID is recorded and false is returned.
// Uses INSERT ... ON CONFLICT for atomic check-and-insert.
func (s *PgStore) Check(eventID string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to insert; if it already exists, the ON CONFLICT clause does nothing
	// and we detect the duplicate via rows affected.
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO webhook_dedup (event_id) VALUES ($1) ON CONFLICT (event_id) DO NOTHING`,
		eventID)
	if err != nil {
		// On error, assume not duplicate to avoid dropping events.
		return false
	}
	rows, _ := result.RowsAffected()
	return rows == 0 // 0 rows = already existed = duplicate
}

func (s *PgStore) cleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		s.db.ExecContext(ctx,
			`DELETE FROM webhook_dedup WHERE seen_at < NOW() - $1::interval`,
			s.ttl.String())
		cancel()
	}
}
