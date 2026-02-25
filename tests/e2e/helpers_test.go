package e2e

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

func gatewayURL() string       { return envOr("GATEWAY_URL", "http://localhost:8000") }
func listingsURL() string      { return envOr("LISTINGS_URL", "http://localhost:8001") }
func bookingsURL() string      { return envOr("BOOKINGS_URL", "http://localhost:8002") }
func paymentsURL() string      { return envOr("PAYMENTS_URL", "http://localhost:8003") }
func internalToken() string    { return envOr("INTERNAL_TOKEN", "dev-internal-token") }
func internalTenantID() string { return envOr("INTERNAL_TENANT_ID", defaultUser.TenantID) }
func webhookSecret() string    { return envOr("MASHGATE_WEBHOOK_SECRET", "") }

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

var httpClient = &http.Client{Timeout: 10 * time.Second}

// doRequest sends an HTTP request and returns status code + body.
func doRequest(t *testing.T, method, url string, body any, headers map[string]string) (int, []byte) {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("create request %s %s: %v", method, url, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("execute request %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return resp.StatusCode, respBody
}

// get performs a GET request.
func get(t *testing.T, url string, headers map[string]string) (int, []byte) {
	t.Helper()
	return doRequest(t, http.MethodGet, url, nil, headers)
}

// post performs a POST request with a JSON body.
func post(t *testing.T, url string, body any, headers map[string]string) (int, []byte) {
	t.Helper()
	return doRequest(t, http.MethodPost, url, body, headers)
}

// put performs a PUT request with a JSON body.
func put(t *testing.T, url string, body any, headers map[string]string) (int, []byte) {
	t.Helper()
	return doRequest(t, http.MethodPut, url, body, headers)
}

// del performs a DELETE request.
func del(t *testing.T, url string, headers map[string]string) (int, []byte) {
	t.Helper()
	return doRequest(t, http.MethodDelete, url, nil, headers)
}

// ---------------------------------------------------------------------------
// Auth simulation helpers
//
// In integration tests against running services, the gateway propagates
// X-User-* headers. When testing services directly (bypassing the gateway),
// we inject these headers to simulate an authenticated user.
// ---------------------------------------------------------------------------

type testUser struct {
	UserID   string
	TenantID string
	Email    string
	Scopes   string // space-separated
}

var defaultUser = testUser{
	UserID:   "e2e-user-001",
	TenantID: "e2e-tenant-001",
	Email:    "e2e@zist.test",
	Scopes:   "zist.listings.read zist.listings.manage zist.bookings.read zist.bookings.manage zist.payments.create zist.webhooks.manage",
}

var readOnlyUser = testUser{
	UserID:   "e2e-user-002",
	TenantID: "e2e-tenant-001",
	Email:    "readonly@zist.test",
	Scopes:   "zist.listings.read zist.bookings.read",
}

// hostUser is a separate account acting as the property host.
// Using a different ID from defaultUser ensures ownership checks work correctly.
var hostUser = testUser{
	UserID:   "e2e-host-001",
	TenantID: "e2e-tenant-001",
	Email:    "host@zist.test",
	Scopes:   "zist.listings.read zist.listings.manage zist.bookings.read zist.bookings.manage",
}

// authHeaders returns HTTP headers simulating an authenticated user.
func authHeaders(u testUser) map[string]string {
	return map[string]string{
		"X-User-ID":     u.UserID,
		"X-Tenant-ID":   u.TenantID,
		"X-User-Email":  u.Email,
		"X-User-Scopes": u.Scopes,
	}
}

// noAuthHeaders returns empty headers (anonymous request).
func noAuthHeaders() map[string]string {
	return map[string]string{}
}

// internalHeaders returns headers with the internal service token.
func internalHeaders() map[string]string {
	return map[string]string{
		"X-Internal-Token": internalToken(),
		"X-Tenant-ID":      internalTenantID(),
	}
}

func webhookHeaders(body []byte) map[string]string {
	secret := strings.TrimSpace(webhookSecret())
	if secret == "" {
		return nil
	}
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "." + string(body)))
	return map[string]string{
		"x-hl-timestamp": ts,
		"x-hl-signature": "v1=" + hex.EncodeToString(mac.Sum(nil)),
	}
}

// ---------------------------------------------------------------------------
// JSON helpers
// ---------------------------------------------------------------------------

func jsonField(t *testing.T, data []byte, field string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal JSON for field %q: %v", field, err)
	}
	v, ok := m[field]
	if !ok {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%v", val)
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}

func jsonArray(t *testing.T, data []byte, field string) []any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal JSON for array %q: %v", field, err)
	}
	arr, ok := m[field].([]any)
	if !ok {
		return nil
	}
	return arr
}
