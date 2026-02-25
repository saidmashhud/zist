package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	zistauth "github.com/saidmashhud/zist/internal/auth"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// BookingsClient is an HTTP client for the bookings service.
type BookingsClient struct {
	baseURL       string
	internalToken string
	tokenClient   *zistauth.ServiceTokenClient
	hc            *http.Client
}

// NewBookingsClient creates a client for the bookings service.
// If tokenClient is non-nil, JWT auth is preferred with X-Internal-Token as fallback.
func NewBookingsClient(baseURL, internalToken string, tokenClient *zistauth.ServiceTokenClient) *BookingsClient {
	return &BookingsClient{
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
// Prefers JWT if ServiceTokenClient is configured, falls back to X-Internal-Token.
func (c *BookingsClient) setAuth(req *http.Request) {
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

// ConfirmBooking calls the bookings service to mark a booking as confirmed.
func (c *BookingsClient) ConfirmBooking(ctx context.Context, tenantID, bookingID, paymentID string) error {
	body, _ := json.Marshal(map[string]string{"paymentId": paymentID})
	return c.post(ctx, tenantID, "/bookings/"+bookingID+"/confirm", body)
}

// FailBooking calls the bookings service to mark a booking as failed.
func (c *BookingsClient) FailBooking(ctx context.Context, tenantID, bookingID string) error {
	return c.post(ctx, tenantID, "/bookings/"+bookingID+"/fail", nil)
}

// SetCheckoutID persists the Mashgate checkout session ID on the booking.
func (c *BookingsClient) SetCheckoutID(ctx context.Context, tenantID, bookingID, checkoutID string) error {
	if strings.TrimSpace(tenantID) == "" {
		return errors.New("tenant id is required")
	}
	body, _ := json.Marshal(map[string]string{"checkoutId": checkoutID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		c.baseURL+"/bookings/"+bookingID+"/checkout",
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)
	req.Header.Set("X-Tenant-ID", tenantID)
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("bookings service returned %d", resp.StatusCode)
	}
	return nil
}

func (c *BookingsClient) post(ctx context.Context, tenantID, path string, body []byte) error {
	if strings.TrimSpace(tenantID) == "" {
		return errors.New("tenant id is required")
	}
	var reqBody *bytes.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	} else {
		reqBody = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, reqBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.setAuth(req)
	req.Header.Set("X-Tenant-ID", tenantID)
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("bookings service returned %d", resp.StatusCode)
	}
	return nil
}
