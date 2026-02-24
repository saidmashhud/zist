// Package store implements PostgreSQL persistence for the admin service.
package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// FeatureFlag represents a platform-level feature toggle.
type FeatureFlag struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Enabled   bool    `json:"enabled"`
	Rollout   int     `json:"rollout"` // 0–100 %
	TenantID  *string `json:"tenantId,omitempty"`
	CreatedAt int64   `json:"createdAt"`
	UpdatedAt int64   `json:"updatedAt"`
}

// AuditEntry records an admin action.
type AuditEntry struct {
	ID        string `json:"id"`
	ActorID   string `json:"actorId"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	Detail    string `json:"detail"`
	TenantID  string `json:"tenantId"`
	CreatedAt int64  `json:"createdAt"`
}

// TenantConfig stores per-tenant platform configuration.
type TenantConfig struct {
	TenantID       string  `json:"tenantId"`
	PlatformFeePct float64 `json:"platformFeePct"`
	MaxListings    int     `json:"maxListings"`
	Verified       bool    `json:"verified"`
	CreatedAt      int64   `json:"createdAt"`
	UpdatedAt      int64   `json:"updatedAt"`
}

// Store wraps a PostgreSQL connection.
type Store struct {
	db *sql.DB
}

// New creates a Store backed by db.
func New(db *sql.DB) *Store { return &Store{db: db} }

// ─── Feature Flags ────────────────────────────────────────────────────────────

func (s *Store) ListFlags(ctx context.Context) ([]FeatureFlag, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, enabled, rollout, tenant_id, created_at, updated_at
		 FROM feature_flags ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var flags []FeatureFlag
	for rows.Next() {
		var f FeatureFlag
		if err := rows.Scan(&f.ID, &f.Name, &f.Enabled, &f.Rollout, &f.TenantID, &f.CreatedAt, &f.UpdatedAt); err == nil {
			flags = append(flags, f)
		}
	}
	if flags == nil {
		flags = []FeatureFlag{}
	}
	return flags, nil
}

func (s *Store) UpsertFlag(ctx context.Context, name string, enabled bool, rollout int, tenantID *string) (FeatureFlag, error) {
	now := time.Now().Unix()
	id := uuid.NewString()
	var f FeatureFlag

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO feature_flags (id, name, enabled, rollout, tenant_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (name) DO UPDATE
		  SET enabled=$3, rollout=$4, tenant_id=$5, updated_at=$7
		RETURNING id, name, enabled, rollout, tenant_id, created_at, updated_at`,
		id, name, enabled, rollout, tenantID, now, now,
	).Scan(&f.ID, &f.Name, &f.Enabled, &f.Rollout, &f.TenantID, &f.CreatedAt, &f.UpdatedAt)
	return f, err
}

// ─── Audit Log ────────────────────────────────────────────────────────────────

func (s *Store) AddAudit(ctx context.Context, actorID, action, resource, detail, tenantID string) error {
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO admin_audit_log (id, actor_id, action, resource, detail, tenant_id, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		uuid.NewString(), actorID, action, resource, detail, tenantID, now)
	return err
}

func (s *Store) ListAudit(ctx context.Context, actorID string, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var rows *sql.Rows
	var err error
	if actorID != "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, actor_id, action, resource, detail, tenant_id, created_at
			 FROM admin_audit_log WHERE actor_id=$1 ORDER BY created_at DESC LIMIT $2`,
			actorID, limit)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, actor_id, action, resource, detail, tenant_id, created_at
			 FROM admin_audit_log ORDER BY created_at DESC LIMIT $1`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.ActorID, &e.Action, &e.Resource, &e.Detail, &e.TenantID, &e.CreatedAt); err == nil {
			entries = append(entries, e)
		}
	}
	if entries == nil {
		entries = []AuditEntry{}
	}
	return entries, nil
}

// ─── Tenant Config ────────────────────────────────────────────────────────────

func (s *Store) GetTenantConfig(ctx context.Context, tenantID string) (TenantConfig, error) {
	var cfg TenantConfig
	err := s.db.QueryRowContext(ctx,
		`SELECT tenant_id, platform_fee_pct, max_listings, verified, created_at, updated_at
		 FROM tenant_configs WHERE tenant_id=$1`, tenantID).
		Scan(&cfg.TenantID, &cfg.PlatformFeePct, &cfg.MaxListings, &cfg.Verified, &cfg.CreatedAt, &cfg.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		// Return sensible defaults if not configured.
		return TenantConfig{
			TenantID:       tenantID,
			PlatformFeePct: 12.0,
			MaxListings:    50,
		}, nil
	}
	return cfg, err
}

func (s *Store) UpsertTenantConfig(ctx context.Context, cfg TenantConfig) (TenantConfig, error) {
	now := time.Now().Unix()
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO tenant_configs (tenant_id, platform_fee_pct, max_listings, verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id) DO UPDATE
		  SET platform_fee_pct=$2, max_listings=$3, verified=$4, updated_at=$6
		RETURNING tenant_id, platform_fee_pct, max_listings, verified, created_at, updated_at`,
		cfg.TenantID, cfg.PlatformFeePct, cfg.MaxListings, cfg.Verified, now, now,
	).Scan(&cfg.TenantID, &cfg.PlatformFeePct, &cfg.MaxListings, &cfg.Verified, &cfg.CreatedAt, &cfg.UpdatedAt)
	return cfg, err
}
