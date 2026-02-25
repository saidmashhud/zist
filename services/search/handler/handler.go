package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	httputil "github.com/saidmashhud/zist/internal/httputil"
	"github.com/saidmashhud/zist/services/search/domain"
	"github.com/saidmashhud/zist/services/search/store"
)

// Handler serves HTTP search endpoints.
type Handler struct {
	Store *store.Store
}

// New creates a Handler.
func New(s *store.Store) *Handler { return &Handler{Store: s} }

// Search handles GET /search with query params.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	lat, _ := strconv.ParseFloat(q.Get("lat"), 64)
	lng, _ := strconv.ParseFloat(q.Get("lng"), 64)
	radiusKM, _ := strconv.ParseFloat(q.Get("radius_km"), 64)
	guests, _ := strconv.Atoi(q.Get("guests"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	var amenities []string
	if a := q.Get("amenities"); a != "" {
		amenities = strings.Split(a, ",")
	}

	filters := domain.SearchFilters{
		City:            q.Get("city"),
		Lat:             lat,
		Lng:             lng,
		RadiusKM:        radiusKM,
		CheckIn:         q.Get("check_in"),
		CheckOut:        q.Get("check_out"),
		Guests:          guests,
		Type:            q.Get("type"),
		MinPrice:        q.Get("min_price"),
		MaxPrice:        q.Get("max_price"),
		Amenities:       amenities,
		InstantBookOnly: q.Get("instant_book") == "true",
		SortBy:          q.Get("sort_by"),
		Limit:           limit,
		Offset:          offset,
	}

	results, total, err := h.Store.Search(r.Context(), filters)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, domain.SearchResponse{
		Listings: results,
		Total:    total,
		Limit:    filters.Limit,
		Offset:   filters.Offset,
	})
}

// UpdateLocation handles PUT /search/locations/{id} (internal).
func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing listing id")
		return
	}

	var body struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.Lat == 0 && body.Lng == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "lat and lng required")
		return
	}

	if err := h.Store.UpdateLocation(r.Context(), id, body.Lat, body.Lng); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
