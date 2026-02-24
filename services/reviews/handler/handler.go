// Package handler implements HTTP handlers for the reviews service.
package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	zistauth "github.com/saidmashhud/zist/internal/auth"
	"github.com/saidmashhud/zist/internal/httputil"
	"github.com/saidmashhud/zist/services/reviews/store"
)

// Handler holds shared dependencies for all reviews HTTP handlers.
type Handler struct {
	Store         *store.Store
	ListingsURL   string
	InternalToken string
}

// New creates a Handler.
func New(s *store.Store, listingsURL, internalToken string) *Handler {
	return &Handler{Store: s, ListingsURL: listingsURL, InternalToken: internalToken}
}

// updateListingStats fires an internal call to the listings service to
// recalculate average_rating + review_count. Best-effort: errors are logged.
func (h *Handler) updateListingStats(listingID string, avg float64, count int) {
	body, _ := json.Marshal(map[string]any{
		"averageRating": avg,
		"reviewCount":   count,
	})
	url := fmt.Sprintf("%s/listings/%s/rating", h.ListingsURL, listingID)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", h.InternalToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// tenantFromRequest extracts tenant_id from the request context.
func tenantFromRequest(r *http.Request) string {
	if p := zistauth.FromContext(r.Context()); p != nil && p.TenantID != "" {
		return p.TenantID
	}
	return r.Header.Get("X-Tenant-ID")
}

// requireAuth returns the principal or writes 401 and returns nil.
func requireAuth(w http.ResponseWriter, r *http.Request) *zistauth.Principal {
	p := zistauth.FromContext(r.Context())
	if p == nil || p.UserID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return nil
	}
	return p
}
