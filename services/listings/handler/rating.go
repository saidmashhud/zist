package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/saidmashhud/zist/internal/httputil"
)

// UpdateRating handles PUT /listings/{id}/rating (internal).
// Called by the reviews service after aggregating new review stats.
func (h *Handler) UpdateRating(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		AverageRating float64 `json:"averageRating"`
		ReviewCount   int     `json:"reviewCount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Store.UpdateRating(r.Context(), id, req.AverageRating, req.ReviewCount); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update rating")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
