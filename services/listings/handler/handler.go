// Package handler implements HTTP handlers for the listings service.
// Each handler is a thin layer: parse request → call store → write response.
package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	zistauth "github.com/saidmashhud/zist/internal/auth"
	httputil "github.com/saidmashhud/zist/internal/httputil"
	"github.com/saidmashhud/zist/services/listings/store"
)

// Handler holds dependencies shared across all listing HTTP handlers.
type Handler struct {
	Store       *store.Store
	FeeGuestPct float64 // e.g. 12.0 → 12%
}

// New creates a Handler with the given store and platform fee percentage.
func New(s *store.Store, feeGuestPct float64) *Handler {
	return &Handler{Store: s, FeeGuestPct: feeGuestPct}
}

// requireOwner verifies the authenticated user is the listing's host.
// Returns the hostID on success; writes an error response and returns "" on failure.
func (h *Handler) requireOwner(w http.ResponseWriter, r *http.Request, listingID string) string {
	p := zistauth.FromContext(r.Context())
	if p == nil || strings.TrimSpace(p.TenantID) == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return ""
	}

	hostID, err := h.Store.GetHostIDForTenant(r.Context(), p.TenantID, listingID)
	if errors.Is(err, store.ErrNotFound) {
		httputil.WriteError(w, http.StatusNotFound, "listing not found")
		return ""
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "db error")
		return ""
	}
	if p.UserID != hostID {
		httputil.WriteError(w, http.StatusForbidden, "not the listing owner")
		return ""
	}
	return hostID
}

// listingID extracts and returns the {id} URL parameter.
func listingID(r *http.Request) string { return chi.URLParam(r, "id") }

func tenantFromRequest(r *http.Request) string {
	if p := zistauth.FromContext(r.Context()); p != nil && strings.TrimSpace(p.TenantID) != "" {
		return strings.TrimSpace(p.TenantID)
	}
	return strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
}
