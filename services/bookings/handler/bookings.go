package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	zistauth "github.com/saidmashhud/zist/internal/auth"
	"github.com/saidmashhud/zist/internal/httputil"
	"github.com/saidmashhud/zist/services/bookings/domain"
	"github.com/saidmashhud/zist/services/bookings/store"
)

// mustFloat parses a decimal string to float64; returns 0 on error.
func mustFloat(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

// ListBookings returns the authenticated guest's bookings.
// GET /bookings/
func (h *Handler) ListBookings(w http.ResponseWriter, r *http.Request) {
	principal := zistauth.FromContext(r.Context())
	if principal == nil || principal.TenantID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	bookings, err := h.Store.ListByGuest(r.Context(), principal.TenantID, principal.UserID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "db query failed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"bookings": bookings})
}

// GetBooking returns a single booking. The caller must be the guest or host.
// GET /bookings/{id}
func (h *Handler) GetBooking(w http.ResponseWriter, r *http.Request) {
	principal := zistauth.FromContext(r.Context())
	if principal == nil || principal.TenantID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	b, err := h.Store.Get(r.Context(), principal.TenantID, id)
	if err == store.ErrNotFound {
		httputil.WriteError(w, http.StatusNotFound, "booking not found")
		return
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}

	if principal.UserID != b.GuestID && principal.UserID != b.HostID {
		httputil.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, b)
}

// CreateBooking creates a new booking request.
// Instant-book listings: dates reserved immediately → payment_pending.
// Request-approval listings: no reservation → pending_host_approval.
// POST /bookings/
func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	principal := zistauth.FromContext(r.Context())
	if principal == nil || principal.TenantID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		ListingID string `json:"listingId"`
		CheckIn   string `json:"checkIn"`
		CheckOut  string `json:"checkOut"`
		Guests    int    `json:"guests"`
		Message   string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ListingID == "" || req.CheckIn == "" || req.CheckOut == "" {
		httputil.WriteError(w, http.StatusUnprocessableEntity, "listingId, checkIn, checkOut are required")
		return
	}

	ciDate, err1 := time.Parse("2006-01-02", req.CheckIn)
	coDate, err2 := time.Parse("2006-01-02", req.CheckOut)
	if err1 != nil || err2 != nil || !coDate.After(ciDate) {
		httputil.WriteError(w, http.StatusBadRequest, "invalid dates: checkOut must be after checkIn")
		return
	}
	nights := int(coDate.Sub(ciDate).Hours() / 24)

	listing, err := h.Listings.GetListing(r.Context(), principal.TenantID, req.ListingID)
	if err != nil {
		httputil.WriteError(w, http.StatusBadGateway, "could not reach listings service")
		return
	}
	if listing == nil {
		httputil.WriteError(w, http.StatusNotFound, "listing not found")
		return
	}
	if listing.Status != "active" {
		httputil.WriteError(w, http.StatusUnprocessableEntity, "listing is not active")
		return
	}
	if req.Guests > listing.MaxGuests {
		httputil.WriteError(w, http.StatusUnprocessableEntity,
			fmt.Sprintf("listing capacity is %d guests", listing.MaxGuests))
		return
	}
	if nights < listing.MinNights {
		httputil.WriteError(w, http.StatusUnprocessableEntity,
			fmt.Sprintf("minimum stay is %d nights", listing.MinNights))
		return
	}
	if listing.MaxNights > 0 && nights > listing.MaxNights {
		httputil.WriteError(w, http.StatusUnprocessableEntity,
			fmt.Sprintf("maximum stay is %d nights", listing.MaxNights))
		return
	}

	ppn := mustFloat(listing.PricePerNight)
	cleaning := mustFloat(listing.CleaningFee)
	subtotal := ppn * float64(nights)
	platformFee := math.Round((subtotal+cleaning)*h.FeeGuestPct) / 100.0
	total := subtotal + cleaning + platformFee

	var dates []string
	for d := ciDate; d.Before(coDate); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("2006-01-02"))
	}

	now := time.Now().Unix()
	bookingID := uuid.NewString()

	var initialStatus string
	if listing.InstantBook {
		conflicts, err := h.Listings.MarkDatesBooked(r.Context(), principal.TenantID, req.ListingID, bookingID, dates)
		if err != nil {
			httputil.WriteError(w, http.StatusBadGateway, "could not reach listings service")
			return
		}
		if len(conflicts) > 0 {
			httputil.WriteJSON(w, http.StatusConflict, map[string]any{
				"error":     "dates not available",
				"conflicts": conflicts,
			})
			return
		}
		initialStatus = domain.StatusPaymentPending
	} else {
		initialStatus = domain.StatusPendingHostApproval
	}

	b := domain.Booking{
		ID:                 bookingID,
		ListingID:          req.ListingID,
		GuestID:            principal.UserID,
		HostID:             listing.HostID,
		CheckIn:            req.CheckIn,
		CheckOut:           req.CheckOut,
		Guests:             req.Guests,
		TotalAmount:        fmt.Sprintf("%.2f", total),
		PlatformFee:        fmt.Sprintf("%.2f", platformFee),
		CleaningFee:        fmt.Sprintf("%.2f", cleaning),
		Currency:           listing.Currency,
		Status:             initialStatus,
		CancellationPolicy: listing.CancellationPolicy,
		Message:            req.Message,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := h.Store.Create(r.Context(), principal.TenantID, b); err != nil {
		if listing.InstantBook {
			h.Listings.ReleaseDates(r.Context(), principal.TenantID, req.ListingID, bookingID) //nolint:errcheck
		}
		httputil.WriteError(w, http.StatusInternalServerError, "insert failed")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, b)
}
