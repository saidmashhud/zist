package handler

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	httputil "github.com/saidmashhud/zist/internal/httputil"
	"github.com/saidmashhud/zist/services/listings/domain"
	"github.com/saidmashhud/zist/services/listings/store"
)

func (h *Handler) SearchListings(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	f := domain.SearchFilters{
		City:            q.Get("city"),
		CheckIn:         q.Get("check_in"),
		CheckOut:        q.Get("check_out"),
		Type:            q.Get("type"),
		MinPrice:        q.Get("min_price"),
		MaxPrice:        q.Get("max_price"),
		InstantBookOnly: q.Get("instant_book") == "true",
		Limit:           50,
	}
	if n, err := strconv.Atoi(q.Get("guests")); err == nil && n > 1 {
		f.Guests = n
	}
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n <= 100 {
		f.Limit = n
	}
	if amenities := q.Get("amenities"); amenities != "" {
		f.Amenities = strings.Split(amenities, ",")
	}

	// Validate date pair if provided.
	if f.CheckIn != "" && f.CheckOut != "" {
		ci, err1 := time.Parse("2006-01-02", f.CheckIn)
		co, err2 := time.Parse("2006-01-02", f.CheckOut)
		if err1 != nil || err2 != nil || !co.After(ci) {
			httputil.WriteError(w, http.StatusBadRequest, "check_in and check_out must be valid dates with check_out after check_in")
			return
		}
	}

	listings, err := h.Store.Search(r.Context(), f)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "search failed")
		return
	}

	// Attach cover photo for each result.
	for i := range listings {
		if p := h.Store.GetCoverPhoto(r.Context(), listings[i].ID); p != nil {
			listings[i].Photos = []domain.Photo{*p}
		}
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"listings": listings,
		"total":    len(listings),
	})
}

func (h *Handler) PricePreview(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	checkIn := r.URL.Query().Get("check_in")
	checkOut := r.URL.Query().Get("check_out")

	if checkIn == "" || checkOut == "" {
		httputil.WriteError(w, http.StatusBadRequest, "check_in and check_out are required")
		return
	}

	ciDate, err1 := time.Parse("2006-01-02", checkIn)
	coDate, err2 := time.Parse("2006-01-02", checkOut)
	if err1 != nil || err2 != nil || !coDate.After(ciDate) {
		httputil.WriteError(w, http.StatusBadRequest, "invalid dates: check_out must be after check_in")
		return
	}

	nights := int(coDate.Sub(ciDate).Hours() / 24)
	if nights <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "minimum stay is 1 night")
		return
	}

	ppn, cleaningFee, currency, minNights, maxNights, err := h.Store.GetPricingInfo(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			httputil.WriteError(w, http.StatusNotFound, "listing not found")
		} else {
			httputil.WriteError(w, http.StatusInternalServerError, "db error")
		}
		return
	}
	if nights < minNights {
		httputil.WriteError(w, http.StatusUnprocessableEntity, fmt.Sprintf("minimum stay is %d nights", minNights))
		return
	}
	if nights > maxNights {
		httputil.WriteError(w, http.StatusUnprocessableEntity, fmt.Sprintf("maximum stay is %d nights", maxNights))
		return
	}

	// Per-day prices from the store (uses price_override if set).
	pricesByDate, _ := h.Store.GetPricesByDate(r.Context(), id, ppn, checkIn, checkOut)

	basePPN := parseFloat(ppn)
	var subtotal float64
	effectivePPN := basePPN

	if len(pricesByDate) > 0 {
		for _, p := range pricesByDate {
			subtotal += parseFloat(p)
		}
		effectivePPN = subtotal / float64(len(pricesByDate))
	} else {
		subtotal = basePPN * float64(nights)
	}

	cleaning := parseFloat(cleaningFee)
	platformFee := math.Round((subtotal+cleaning)*h.FeeGuestPct) / 100.0
	total := subtotal + cleaning + platformFee

	httputil.WriteJSON(w, http.StatusOK, domain.PricePreview{
		Nights:           nights,
		PricePerNight:    fmt.Sprintf("%.2f", effectivePPN),
		Subtotal:         fmt.Sprintf("%.2f", subtotal),
		CleaningFee:      fmt.Sprintf("%.2f", cleaning),
		PlatformFeeGuest: fmt.Sprintf("%.2f", platformFee),
		Total:            fmt.Sprintf("%.2f", total),
		Currency:         currency,
	})
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}
