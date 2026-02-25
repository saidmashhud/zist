package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	zistauth "github.com/saidmashhud/zist/internal/auth"
	"github.com/saidmashhud/zist/services/bookings/domain"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// ListingsClient is an HTTP client for the listings service.
type ListingsClient struct {
	baseURL       string
	internalToken string
	tokenClient   *zistauth.ServiceTokenClient
	hc            *http.Client
}

// NewListingsClient creates a client for the listings service.
// If tokenClient is non-nil, JWT auth is preferred with X-Internal-Token as fallback.
func NewListingsClient(baseURL, internalToken string, tokenClient *zistauth.ServiceTokenClient) *ListingsClient {
	return &ListingsClient{
		baseURL:       strings.TrimRight(baseURL, "/"),
		internalToken: internalToken,
		tokenClient:   tokenClient,
		hc: &http.Client{
			Timeout:   5 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

// setAuth sets the appropriate auth header on the request.
func (c *ListingsClient) setAuth(req *http.Request) {
	if c.tokenClient != nil {
		tok, err := c.tokenClient.Token()
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+tok)
			return
		}
		slog.Warn("service JWT fetch failed, falling back to X-Internal-Token", "err", err)
	}
	req.Header.Set("X-Internal-Token", c.internalToken)
}

// GetListing fetches listing details. Returns (nil, nil) when not found.
func (c *ListingsClient) GetListing(ctx context.Context, tenantID, id string) (*domain.ListingInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/listings/%s", c.baseURL, id), nil)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(tenantID) != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("listings service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("listings service returned %d", resp.StatusCode)
	}

	var raw struct {
		ID                 string `json:"id"`
		HostID             string `json:"hostId"`
		InstantBook        bool   `json:"instantBook"`
		CancellationPolicy string `json:"cancellationPolicy"`
		PricePerNight      string `json:"pricePerNight"`
		CleaningFee        string `json:"cleaningFee"`
		Currency           string `json:"currency"`
		MinNights          int    `json:"minNights"`
		MaxNights          int    `json:"maxNights"`
		MaxGuests          int    `json:"maxGuests"`
		Status             string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode listing: %w", err)
	}
	return &domain.ListingInfo{
		ID:                 raw.ID,
		HostID:             raw.HostID,
		InstantBook:        raw.InstantBook,
		CancellationPolicy: raw.CancellationPolicy,
		PricePerNight:      raw.PricePerNight,
		CleaningFee:        raw.CleaningFee,
		Currency:           raw.Currency,
		MinNights:          raw.MinNights,
		MaxNights:          raw.MaxNights,
		MaxGuests:          raw.MaxGuests,
		Status:             raw.Status,
	}, nil
}

// MarkDatesBooked reserves dates on a listing for a booking.
// Returns non-empty conflict slice on 409.
func (c *ListingsClient) MarkDatesBooked(ctx context.Context, tenantID, listingID, bookingID string, dates []string) ([]string, error) {
	body, _ := json.Marshal(map[string]any{
		"dates":     dates,
		"bookingId": bookingID,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/listings/%s/availability/book", c.baseURL, listingID),
		strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("listings service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		var conflict struct {
			Conflicts []string `json:"conflicts"`
		}
		json.NewDecoder(resp.Body).Decode(&conflict) //nolint:errcheck
		return conflict.Conflicts, nil
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("listings service returned %d: %s", resp.StatusCode, b)
	}
	return nil, nil
}

// ReleaseDates releases dates previously reserved for a booking.
func (c *ListingsClient) ReleaseDates(ctx context.Context, tenantID, listingID, bookingID string) error {
	body, _ := json.Marshal(map[string]string{"bookingId": bookingID})
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/listings/%s/availability/book", c.baseURL, listingID),
		strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("listings service unavailable: %w", err)
	}
	resp.Body.Close()
	return nil
}
