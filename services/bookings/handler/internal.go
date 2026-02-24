package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/saidmashhud/zist/internal/httputil"
	"github.com/saidmashhud/zist/services/bookings/store"
)

// ConfirmBooking transitions a booking from payment_pending → confirmed.
// Called by the payments service after a successful payment.captured event.
// POST /bookings/{id}/confirm  (internal token required)
func (h *Handler) ConfirmBooking(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	var req struct {
		PaymentID string `json:"paymentId"`
	}
	json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck — body is optional

	ok, err := h.Store.Confirm(r.Context(), tenantID, id, req.PaymentID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "update failed")
		return
	}
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "booking not found or not in payment_pending status")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// FailBooking transitions a booking from payment_pending → failed and releases dates.
// Called by the payments service after a payment.failed event.
// POST /bookings/{id}/fail  (internal token required)
func (h *Handler) FailBooking(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	b, err := h.Store.Fail(r.Context(), tenantID, id)
	if err == store.ErrNotFound {
		httputil.WriteError(w, http.StatusNotFound, "booking not found or not in payment_pending status")
		return
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "update failed")
		return
	}

	h.Listings.ReleaseDates(r.Context(), tenantID, b.ListingID, b.ID) //nolint:errcheck
	w.WriteHeader(http.StatusNoContent)
}

// SetCheckoutID stores the Mashgate checkout session ID on the booking.
// Called by the payments service after creating a checkout session.
// PUT /bookings/{id}/checkout  (internal token required)
func (h *Handler) SetCheckoutID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	var req struct {
		CheckoutID string `json:"checkoutId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.CheckoutID == "" {
		httputil.WriteError(w, http.StatusUnprocessableEntity, "checkoutId is required")
		return
	}

	ok, err := h.Store.SetCheckoutID(r.Context(), tenantID, id, req.CheckoutID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "update failed")
		return
	}
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "booking not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
