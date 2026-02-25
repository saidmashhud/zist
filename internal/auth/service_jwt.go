package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ServiceTokenClient obtains and caches service JWTs from the Mashgate auth-service.
// Thread-safe. Automatically refreshes the token 5 minutes before expiry.
type ServiceTokenClient struct {
	authURL     string // e.g. "http://auth-service:8080"
	serviceName string // e.g. "zist-payments"
	apiKey      string // Mashgate API key for auth

	mu       sync.RWMutex
	token    string
	expireAt time.Time
	client   *http.Client
}

// NewServiceTokenClient creates a client that fetches service JWTs.
func NewServiceTokenClient(authURL, serviceName, apiKey string) *ServiceTokenClient {
	return &ServiceTokenClient{
		authURL:     strings.TrimRight(authURL, "/"),
		serviceName: serviceName,
		apiKey:      apiKey,
		client:      &http.Client{Timeout: 10 * time.Second},
	}
}

// Token returns a valid service JWT, refreshing if needed.
func (c *ServiceTokenClient) Token() (string, error) {
	c.mu.RLock()
	if c.token != "" && time.Now().Before(c.expireAt.Add(-5*time.Minute)) {
		tok := c.token
		c.mu.RUnlock()
		return tok, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock.
	if c.token != "" && time.Now().Before(c.expireAt.Add(-5*time.Minute)) {
		return c.token, nil
	}

	body := fmt.Sprintf(`{"callerService":"%s"}`, c.serviceName)
	req, err := http.NewRequest("POST", c.authURL+"/v1/auth/service-token", strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("service token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("service token request returned %d", resp.StatusCode)
	}

	var result struct {
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expiresAt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding service token response: %w", err)
	}

	c.token = result.Token
	c.expireAt = time.Unix(result.ExpiresAt, 0)
	return c.token, nil
}

// RequireServiceAuth returns a middleware that accepts either:
//   - Authorization: Bearer <service-jwt> (preferred, validated via JWKS)
//   - X-Internal-Token: <shared-secret> (legacy fallback)
//
// This allows gradual migration from shared secrets to JWT.
func RequireServiceAuth(legacyToken string, jwksValidator func(token string) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try JWT first (Authorization: Bearer <jwt>)
			if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				jwt := strings.TrimPrefix(auth, "Bearer ")
				if jwksValidator != nil && jwksValidator(jwt) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Fallback: legacy X-Internal-Token
			got := r.Header.Get("X-Internal-Token")
			if legacyToken != "" && got != "" && got == legacyToken {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"forbidden","code":"INVALID_SERVICE_AUTH"}`)) //nolint:errcheck
		})
	}
}
