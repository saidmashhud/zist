package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	zistauth "github.com/saidmashhud/zist/internal/auth"
	"github.com/saidmashhud/zist/internal/httputil"
	"github.com/saidmashhud/zist/services/bookings/domain"
	"github.com/saidmashhud/zist/services/bookings/store"
)

// ListHostBookings returns all bookings on the authenticated host's listings.
// GET /bookings/host
func (h *Handler) ListHostBookings(w http.ResponseWriter, r *http.Request) {
	principal := zistauth.FromContext(r.Context())
	if principal == nil || principal.TenantID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	bookings, err := h.Store.ListByHost(r.Context(), principal.TenantID, principal.UserID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "db query failed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"bookings": bookings})
}

// ApproveBooking lets a host approve a pending-approval request.
// Reserves dates and transitions to payment_pending.
// POST /bookings/{id}/approve
func (h *Handler) ApproveBooking(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	principal := zistauth.FromContext(r.Context())
	if principal == nil || principal.TenantID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	b, err := h.Store.Get(r.Context(), principal.TenantID, id)
	if err == store.ErrNotFound {
		httputil.WriteError(w, http.StatusNotFound, "booking not found")
		return
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	if b.HostID != principal.UserID {
		httputil.WriteError(w, http.StatusForbidden, "not your listing")
		return
	}
	if b.Status != domain.StatusPendingHostApproval {
		httputil.WriteError(w, http.StatusConflict, "booking is not pending host approval")
		return
	}

	// Reserve dates now.
	ciDate, _ := time.Parse("2006-01-02", b.CheckIn)
	coDate, _ := time.Parse("2006-01-02", b.CheckOut)
	var dates []string
	for d := ciDate; d.Before(coDate); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("2006-01-02"))
	}

	conflicts, err := h.Listings.MarkDatesBooked(r.Context(), principal.TenantID, b.ListingID, b.ID, dates)
	if err != nil {
		httputil.WriteError(w, http.StatusBadGateway, "could not reach listings service")
		return
	}
	if len(conflicts) > 0 {
		httputil.WriteJSON(w, http.StatusConflict, map[string]any{
			"error":     "dates no longer available",
			"conflicts": conflicts,
		})
		return
	}

	// Guest has 24 h to pay.
	expiresAt := time.Now().Unix() + 86400
	ok, err := h.Store.Approve(r.Context(), principal.TenantID, id, expiresAt)
	if err != nil {
		h.Listings.ReleaseDates(r.Context(), principal.TenantID, b.ListingID, b.ID) //nolint:errcheck
		httputil.WriteError(w, http.StatusInternalServerError, "update failed")
		return
	}
	if !ok {
		h.Listings.ReleaseDates(r.Context(), principal.TenantID, b.ListingID, b.ID) //nolint:errcheck
		httputil.WriteError(w, http.StatusConflict, "booking state changed concurrently")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":    domain.StatusPaymentPending,
		"expiresAt": expiresAt,
	})
}

// RejectBooking lets a host reject a pending-approval request.
// POST /bookings/{id}/reject
func (h *Handler) RejectBooking(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	principal := zistauth.FromContext(r.Context())
	if principal == nil || principal.TenantID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	b, err := h.Store.Get(r.Context(), principal.TenantID, id)
	if err == store.ErrNotFound {
		httputil.WriteError(w, http.StatusNotFound, "booking not found")
		return
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	if b.HostID != principal.UserID {
		httputil.WriteError(w, http.StatusForbidden, "not your listing")
		return
	}
	if b.Status != domain.StatusPendingHostApproval {
		httputil.WriteError(w, http.StatusConflict, "booking is not pending host approval")
		return
	}

	if err := h.Store.Reject(r.Context(), principal.TenantID, id); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "update failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
