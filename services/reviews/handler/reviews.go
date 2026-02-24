package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/saidmashhud/zist/internal/httputil"
	"github.com/saidmashhud/zist/services/reviews/domain"
	"github.com/saidmashhud/zist/services/reviews/store"
)

// CreateReview handles POST /reviews.
// Only guests who completed a booking may submit a review.
func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
	p := requireAuth(w, r)
	if p == nil {
		return
	}

	var req struct {
		BookingID string `json:"bookingId"`
		ListingID string `json:"listingId"`
		HostID    string `json:"hostId"`
		Rating    int    `json:"rating"`
		Comment   string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.BookingID == "" || req.ListingID == "" {
		httputil.WriteError(w, http.StatusUnprocessableEntity, "bookingId and listingId are required")
		return
	}
	if req.Rating < 1 || req.Rating > 5 {
		httputil.WriteError(w, http.StatusUnprocessableEntity, "rating must be between 1 and 5")
		return
	}

	rev, err := h.Store.Create(r.Context(), domain.CreateReviewInput{
		BookingID: req.BookingID,
		ListingID: req.ListingID,
		GuestID:   p.UserID,
		HostID:    req.HostID,
		TenantID:  p.TenantID,
		Rating:    req.Rating,
		Comment:   req.Comment,
	})
	if err == store.ErrAlreadyReviewed {
		httputil.WriteError(w, http.StatusConflict, "booking already reviewed")
		return
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create review")
		return
	}

	// Fire-and-forget: update listing's aggregate rating
	avg, count, _ := h.Store.RatingSummary(r.Context(), req.ListingID)
	go h.updateListingStats(req.ListingID, avg, count)

	httputil.WriteJSON(w, http.StatusCreated, rev)
}

// ListReviewsByListing handles GET /reviews/listing/{id}.
func (h *Handler) ListReviewsByListing(w http.ResponseWriter, r *http.Request) {
	listingID := chi.URLParam(r, "id")
	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if n, err := strconv.Atoi(lStr); err == nil && n > 0 {
			limit = n
		}
	}

	reviews, err := h.Store.ListByListing(r.Context(), listingID, limit)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "db query failed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"reviews": reviews})
}

// ListMyReviews handles GET /reviews/my — reviews written by the authenticated guest.
func (h *Handler) ListMyReviews(w http.ResponseWriter, r *http.Request) {
	p := requireAuth(w, r)
	if p == nil {
		return
	}

	reviews, err := h.Store.ListByGuest(r.Context(), p.TenantID, p.UserID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "db query failed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"reviews": reviews})
}

// ReplyToReview handles POST /reviews/{id}/reply — host replies to a review.
func (h *Handler) ReplyToReview(w http.ResponseWriter, r *http.Request) {
	p := requireAuth(w, r)
	if p == nil {
		return
	}

	reviewID := chi.URLParam(r, "id")
	var req struct {
		Reply string `json:"reply"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Reply == "" {
		httputil.WriteError(w, http.StatusBadRequest, "reply text is required")
		return
	}

	rev, err := h.Store.SetReply(r.Context(), reviewID, p.UserID, req.Reply)
	if err == store.ErrNotFound {
		httputil.WriteError(w, http.StatusNotFound, "review not found or not owned by you")
		return
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update review")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, rev)
}
