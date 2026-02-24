package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	zistauth "github.com/saidmashhud/zist/internal/auth"
	httputil "github.com/saidmashhud/zist/internal/httputil"
	"github.com/saidmashhud/zist/services/listings/domain"
	"github.com/saidmashhud/zist/services/listings/store"
)

func (h *Handler) ListMyListings(w http.ResponseWriter, r *http.Request) {
	p := zistauth.FromContext(r.Context())
	if p == nil || p.TenantID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	listings, err := h.Store.ListByHost(r.Context(), p.TenantID, p.UserID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"listings": listings})
}

func (h *Handler) ListListings(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	city := q.Get("city")
	statusFilter := q.Get("status")
	limit := 50
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n <= 100 {
		limit = n
	}
	listings, err := h.Store.List(r.Context(), statusFilter, city, limit)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"listings": listings})
}

func (h *Handler) GetListing(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	tenantID := tenantFromRequest(r)
	var (
		l   domain.Listing
		err error
	)
	if tenantID != "" {
		l, err = h.Store.GetForTenant(r.Context(), tenantID, id)
	} else {
		l, err = h.Store.Get(r.Context(), id)
	}
	if errors.Is(err, store.ErrNotFound) {
		httputil.WriteError(w, http.StatusNotFound, "listing not found")
		return
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	photos, _ := h.Store.GetPhotos(r.Context(), id)
	if photos != nil {
		l.Photos = photos
	}

	// Analytics: track listing view for host dashboard.
	h.Analytics.TrackListingView(r.Context(), tenantID, id, l.HostID)

	httputil.WriteJSON(w, http.StatusOK, l)
}

func (h *Handler) CreateListing(w http.ResponseWriter, r *http.Request) {
	p := zistauth.FromContext(r.Context())
	if p == nil || p.TenantID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Title              string            `json:"title"`
		Description        string            `json:"description"`
		City               string            `json:"city"`
		Country            string            `json:"country"`
		Address            string            `json:"address"`
		Type               string            `json:"type"`
		Bedrooms           int               `json:"bedrooms"`
		Beds               int               `json:"beds"`
		Bathrooms          int               `json:"bathrooms"`
		MaxGuests          int               `json:"maxGuests"`
		Amenities          []string          `json:"amenities"`
		Rules              domain.HouseRules `json:"rules"`
		PricePerNight      string            `json:"pricePerNight"`
		Currency           string            `json:"currency"`
		CleaningFee        string            `json:"cleaningFee"`
		Deposit            string            `json:"deposit"`
		MinNights          int               `json:"minNights"`
		MaxNights          int               `json:"maxNights"`
		CancellationPolicy string            `json:"cancellationPolicy"`
		InstantBook        bool              `json:"instantBook"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.City) == "" || req.PricePerNight == "" {
		httputil.WriteError(w, http.StatusUnprocessableEntity, "title, city, and pricePerNight are required")
		return
	}

	if req.Amenities == nil {
		req.Amenities = []string{}
	}

	in := domain.CreateListingInput{
		TenantID:           p.TenantID,
		HostID:             p.UserID,
		Title:              req.Title,
		Description:        req.Description,
		City:               req.City,
		Country:            httputil.OrDefault(req.Country, ""),
		Address:            req.Address,
		Type:               httputil.OrDefault(req.Type, "apartment"),
		Bedrooms:           atLeast1(req.Bedrooms),
		Beds:               atLeast1(req.Beds),
		Bathrooms:          atLeast1(req.Bathrooms),
		MaxGuests:          atLeast1(req.MaxGuests),
		Amenities:          req.Amenities,
		Rules:              req.Rules,
		PricePerNight:      req.PricePerNight,
		Currency:           httputil.OrDefault(req.Currency, "USD"),
		CleaningFee:        httputil.OrDefault(req.CleaningFee, "0"),
		Deposit:            httputil.OrDefault(req.Deposit, "0"),
		MinNights:          atLeast1(req.MinNights),
		MaxNights:          positiveOrDefault(req.MaxNights, 365),
		CancellationPolicy: httputil.OrDefault(req.CancellationPolicy, "moderate"),
		InstantBook:        req.InstantBook,
	}
	l, err := h.Store.Create(r.Context(), in)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "create failed")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, l)
}

func (h *Handler) UpdateListing(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	if h.requireOwner(w, r, id) == "" {
		return
	}

	var req domain.UpdateListingInput
	// Parse JSON into a raw map so we can distinguish missing vs null fields.
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Helper: decode field if present in JSON.
	decode := func(key string, dst any) {
		if v, ok := raw[key]; ok {
			json.Unmarshal(v, dst) //nolint:errcheck
		}
	}
	decode("title", &req.Title)
	decode("description", &req.Description)
	decode("address", &req.Address)
	decode("type", &req.Type)
	decode("bedrooms", &req.Bedrooms)
	decode("beds", &req.Beds)
	decode("bathrooms", &req.Bathrooms)
	decode("maxGuests", &req.MaxGuests)
	decode("amenities", &req.Amenities)
	decode("rules", &req.Rules)
	decode("pricePerNight", &req.PricePerNight)
	decode("currency", &req.Currency)
	decode("cleaningFee", &req.CleaningFee)
	decode("deposit", &req.Deposit)
	decode("minNights", &req.MinNights)
	decode("maxNights", &req.MaxNights)
	decode("cancellationPolicy", &req.CancellationPolicy)
	decode("instantBook", &req.InstantBook)
	decode("status", &req.Status)

	l, err := h.Store.Update(r.Context(), id, req)
	if errors.Is(err, store.ErrNotFound) {
		httputil.WriteError(w, http.StatusNotFound, "listing not found")
		return
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "update failed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, l)
}

func (h *Handler) DeleteListing(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	if h.requireOwner(w, r, id) == "" {
		return
	}
	if err := h.Store.Delete(r.Context(), id); errors.Is(err, store.ErrNotFound) {
		httputil.WriteError(w, http.StatusNotFound, "listing not found")
		return
	} else if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) PublishListing(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	if h.requireOwner(w, r, id) == "" {
		return
	}
	count, _ := h.Store.PhotoCount(r.Context(), id)
	if count == 0 {
		httputil.WriteError(w, http.StatusUnprocessableEntity, "at least one photo is required to publish")
		return
	}
	if err := h.Store.SetStatus(r.Context(), id, "active"); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "publish failed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "active"})
}

func (h *Handler) UnpublishListing(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	if h.requireOwner(w, r, id) == "" {
		return
	}
	if err := h.Store.SetStatus(r.Context(), id, "paused"); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "unpublish failed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "paused"})
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func atLeast1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func positiveOrDefault(n, def int) int {
	if n <= 0 {
		return def
	}
	return n
}

// unused import guard
var _ = time.Now
