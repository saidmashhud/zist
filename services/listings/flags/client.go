// Package flags provides a lightweight client for mgFlags feature flags.
package flags

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Client fetches feature flags from mgFlags and caches them locally.
type Client struct {
	baseURL  string
	apiKey   string
	http     *http.Client
	mu       sync.RWMutex
	cache    map[string]any
	cachedAt time.Time
	cacheTTL time.Duration
}

// New creates a Client. If baseURL is empty, all flags return their default.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:  baseURL,
		apiKey:   apiKey,
		http:     &http.Client{Timeout: 2 * time.Second},
		cache:    map[string]any{},
		cacheTTL: 30 * time.Second,
	}
}

// Bool returns a boolean flag value, or defaultVal if the flag is unset or fetch fails.
func (c *Client) Bool(ctx context.Context, flag string, defaultVal bool) bool {
	val := c.get(ctx, flag)
	if val == nil {
		return defaultVal
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return defaultVal
}

// String returns a string flag value, or defaultVal if unset.
func (c *Client) String(ctx context.Context, flag string, defaultVal string) string {
	val := c.get(ctx, flag)
	if val == nil {
		return defaultVal
	}
	if s, ok := val.(string); ok {
		return s
	}
	return defaultVal
}

func (c *Client) get(ctx context.Context, flag string) any {
	if c.baseURL == "" {
		return nil
	}
	c.mu.RLock()
	if time.Since(c.cachedAt) < c.cacheTTL {
		v, ok := c.cache[flag]
		c.mu.RUnlock()
		if ok {
			return v
		}
		return nil
	}
	c.mu.RUnlock()

	// Refresh cache
	c.refresh(ctx)

	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache[flag]
}

func (c *Client) refresh(ctx context.Context) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/flags", c.baseURL), nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		slog.Debug("flags: refresh failed", "err", err)
		return
	}
	defer resp.Body.Close()

	var result struct {
		Flags map[string]any `json:"flags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}

	c.mu.Lock()
	c.cache = result.Flags
	c.cachedAt = time.Now()
	c.mu.Unlock()
}
