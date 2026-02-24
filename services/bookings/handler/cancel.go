package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	zistauth "github.com/saidmashhud/zist/internal/auth"
	"github.com/saidmashhud/zist/internal/httputil"
	"github.com/saidmashhud/zist/services/bookings/domain"
	"github.com/saidmashhud/zist/services/bookings/store"
)

// CancelBooking handles cancellation by the guest or host.
// Computes a policy-based refund. Host cancellations always yield 100% refund.
// POST /bookings/{id}/cancel
func (h *Handler) CancelBooking(w http.ResponseWriter, r *http.Request) {
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

	var newStatus string
	switch principal.UserID {
	case b.GuestID:
		newStatus = domain.StatusCancelledByGuest
	case b.HostID:
		newStatus = domain.StatusCancelledByHost
	default:
		httputil.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}

	switch b.Status {
	case domain.StatusPendingHostApproval, domain.StatusPaymentPending, domain.StatusConfirmed:
		// allowed
	default:
		httputil.WriteError(w, http.StatusConflict, "booking cannot be cancelled in status: "+b.Status)
		return
	}

	var refund domain.RefundResult
	if newStatus == domain.StatusCancelledByHost {
		refund = domain.RefundResult{
			RefundAmount: b.TotalAmount,
			RefundPct:    100,
			Currency:     b.Currency,
		}
	} else {
		refund, err = domain.CalculateRefund(b.CancellationPolicy, b.TotalAmount, b.Currency, b.CheckIn)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "refund calculation failed")
			return
		}
	}

	if err := h.Store.Cancel(r.Context(), principal.TenantID, id, newStatus); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "update failed")
		return
	}

	// Release reserved dates only if they were reserved.
	if b.Status == domain.StatusPaymentPending || b.Status == domain.StatusConfirmed {
		h.Listings.ReleaseDates(r.Context(), principal.TenantID, b.ListingID, b.ID) //nolint:errcheck
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status": newStatus,
		"refund": refund,
	})
}
