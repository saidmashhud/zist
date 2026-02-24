package store

import "database/sql"

// Migrate runs idempotent DDL for admin tables.
func Migrate(db *sql.DB) error {
	// Platform-wide feature flags (backed by mgFlags, but also local overrides).
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS feature_flags (
			id         TEXT PRIMARY KEY,
			name       TEXT UNIQUE NOT NULL,
			enabled    BOOLEAN NOT NULL DEFAULT false,
			rollout    INT     NOT NULL DEFAULT 100,  -- percentage 0-100
			tenant_id  TEXT,                          -- NULL = global
			created_at BIGINT  NOT NULL,
			updated_at BIGINT  NOT NULL
		)
	`); err != nil {
		return err
	}

	// Audit log for admin actions.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS admin_audit_log (
			id         TEXT    PRIMARY KEY,
			actor_id   TEXT    NOT NULL,
			action     TEXT    NOT NULL,
			resource   TEXT    NOT NULL,
			detail     TEXT    NOT NULL DEFAULT '',
			tenant_id  TEXT    NOT NULL DEFAULT '',
			created_at BIGINT  NOT NULL
		)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_admin_audit_actor ON admin_audit_log(actor_id, created_at DESC)
	`); err != nil {
		return err
	}

	// Tenant configuration overrides.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tenant_configs (
			tenant_id         TEXT    PRIMARY KEY,
			platform_fee_pct  NUMERIC(5,2) NOT NULL DEFAULT 12.00,
			max_listings      INT     NOT NULL DEFAULT 50,
			verified          BOOLEAN NOT NULL DEFAULT false,
			created_at        BIGINT  NOT NULL,
			updated_at        BIGINT  NOT NULL
		)
	`); err != nil {
		return err
	}

	return nil
}
