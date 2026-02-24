package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	httputil "github.com/saidmashhud/zist/internal/httputil"
	"github.com/saidmashhud/zist/services/listings/store"
)

func (h *Handler) GetCalendar(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}

	calendar, err := h.Store.GetCalendar(r.Context(), id, month)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "month must be YYYY-MM")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"month": month, "days": calendar})
}

func (h *Handler) BlockDates(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	if h.requireOwner(w, r, id) == "" {
		return
	}

	var req struct {
		Dates []string `json:"dates"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Dates) == 0 {
		httputil.WriteError(w, http.StatusUnprocessableEntity, "dates required")
		return
	}
	for _, d := range req.Dates {
		if _, err := time.Parse("2006-01-02", d); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid date format: "+d)
			return
		}
	}

	if err := h.Store.BlockDates(r.Context(), id, req.Dates); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "block dates failed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"blocked": len(req.Dates)})
}

func (h *Handler) UnblockDates(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	if h.requireOwner(w, r, id) == "" {
		return
	}

	var req struct {
		Dates []string `json:"dates"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Store.UnblockDates(r.Context(), id, req.Dates); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "unblock failed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"unblocked": len(req.Dates)})
}

func (h *Handler) SetPriceOverride(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	if h.requireOwner(w, r, id) == "" {
		return
	}

	var req struct {
		Entries []struct {
			Date  string `json:"date"`
			Price string `json:"price"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	entries := make([]struct {
		Date  string
		Price string
	}, len(req.Entries))
	for i, e := range req.Entries {
		entries[i] = struct {
			Date  string
			Price string
		}{e.Date, e.Price}
	}

	if err := h.Store.SetPriceOverride(r.Context(), id, entries); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "price override failed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"updated": len(req.Entries)})
}

func (h *Handler) CheckAvailability(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	checkIn := r.URL.Query().Get("check_in")
	checkOut := r.URL.Query().Get("check_out")
	if checkIn == "" || checkOut == "" {
		httputil.WriteError(w, http.StatusBadRequest, "check_in and check_out required")
		return
	}

	conflicts, err := h.Store.CheckAvailability(r.Context(), id, checkIn, checkOut)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	if conflicts == nil {
		conflicts = []string{}
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"available": len(conflicts) == 0,
		"conflicts": conflicts,
	})
}

func (h *Handler) MarkDatesBooked(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	var req struct {
		Dates     []string `json:"dates"`
		BookingID string   `json:"bookingId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Dates) == 0 || req.BookingID == "" {
		httputil.WriteError(w, http.StatusUnprocessableEntity, "dates and bookingId required")
		return
	}

	conflicts, err := h.Store.MarkDatesBooked(r.Context(), tenantID, id, req.BookingID, req.Dates)
	if err != nil {
		if err == store.ErrNotFound {
			httputil.WriteError(w, http.StatusNotFound, "listing not found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "mark booked failed")
		return
	}
	if len(conflicts) > 0 {
		httputil.WriteJSON(w, http.StatusConflict, map[string]any{
			"error":     "dates not available",
			"conflicts": conflicts,
		})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"marked": len(req.Dates)})
}

func (h *Handler) UnmarkDatesBooked(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}

	var req struct {
		BookingID string `json:"bookingId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Store.UnmarkDatesBooked(r.Context(), tenantID, id, req.BookingID); err != nil {
		if err == store.ErrNotFound {
			httputil.WriteError(w, http.StatusNotFound, "listing not found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "unmark failed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "released"})
}
