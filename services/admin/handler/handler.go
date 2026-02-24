// Package handler implements HTTP handlers for the admin service.
package handler

import (
	zistauth "github.com/saidmashhud/zist/internal/auth"
	"github.com/saidmashhud/zist/services/admin/store"
)

// Handler holds shared dependencies for all admin HTTP handlers.
type Handler struct {
	Store *store.Store
}

// New creates a Handler.
func New(s *store.Store) *Handler {
	return &Handler{Store: s}
}

// requireAdmin returns the principal or writes 401/403. Requires the
// zist.admin scope which is only granted to platform operators.
func requireAdmin(p *zistauth.Principal) bool {
	if p == nil {
		return false
	}
	for _, s := range p.Scopes {
		if s == "zist.admin" {
			return true
		}
	}
	return false
}
