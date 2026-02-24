// Package analytics provides a fire-and-forget client for mgLogs (ClickHouse analytics).
package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Client sends events to the mgLogs ingestion endpoint.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// New creates a Client. Returns a no-op client if baseURL is empty.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 3 * time.Second},
	}
}

// Track records an event. Always fire-and-forget — errors are only logged.
func (c *Client) Track(ctx context.Context, event string, props map[string]any) {
	if c.baseURL == "" {
		return
	}
	if props == nil {
		props = map[string]any{}
	}
	props["event"] = event
	props["ts"] = time.Now().UnixMilli()

	body, _ := json.Marshal(props)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/logs/ingest", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		slog.Debug("analytics: track failed", "event", event, "err", err)
		return
	}
	resp.Body.Close()
}

// TrackListingView records a listing_view event for host analytics.
func (c *Client) TrackListingView(ctx context.Context, tenantID, listingID, hostID string) {
	go c.Track(ctx, "listing_view", map[string]any{
		"tenant_id":  tenantID,
		"listing_id": listingID,
		"host_id":    hostID,
	})
}

// TrackBookingCreated records a booking_created event.
func (c *Client) TrackBookingCreated(ctx context.Context, tenantID, listingID, bookingID, guestID string) {
	go c.Track(ctx, "booking_created", map[string]any{
		"tenant_id":  tenantID,
		"listing_id": listingID,
		"booking_id": bookingID,
		"guest_id":   guestID,
	})
}

// TrackSearchQuery records a search query for analytics.
func (c *Client) TrackSearchQuery(ctx context.Context, tenantID, city string, guests int, resultCount int) {
	go c.Track(ctx, "search_query", map[string]any{
		"tenant_id":    tenantID,
		"city":         city,
		"guests":       guests,
		"result_count": resultCount,
	})
}
