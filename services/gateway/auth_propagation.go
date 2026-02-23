package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// propagateAuth returns a middleware that:
//  1. Strips inbound X-User-* headers to prevent header injection attacks.
//  2. Reads the session cookie and validates it against mgID's /v1/auth/validate.
//  3. If valid, sets X-User-ID, X-Tenant-ID, X-User-Email, X-User-Scopes on the
//     forwarded request so downstream services can trust them.
//  4. Anonymous requests (no cookie or invalid token) pass through with no user headers.
func propagateAuth(mgIDURL, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Strip any client-supplied user headers (injection prevention)
			r = r.Clone(r.Context())
			r.Header.Del("X-User-ID")
			r.Header.Del("X-Tenant-ID")
			r.Header.Del("X-User-Email")
			r.Header.Del("X-User-Scopes")

			// 2. Read session cookie
			cookie, err := r.Cookie(cookieName)
			if err != nil {
				// Anonymous request — pass through
				next.ServeHTTP(w, r)
				return
			}

			// 3. Validate token via mgID
			claims, err := validateSessionToken(mgIDURL, cookie.Value)
			if err != nil || claims == nil {
				if err != nil {
					slog.Debug("auth validate failed", "err", err)
				}
				next.ServeHTTP(w, r)
				return
			}

			// 4. Propagate validated identity as trusted headers
			r.Header.Set("X-User-ID", claims.UserID)
			r.Header.Set("X-Tenant-ID", claims.TenantID)
			r.Header.Set("X-User-Email", claims.Email)
			r.Header.Set("X-User-Scopes", claims.Scope)

			next.ServeHTTP(w, r)
		})
	}
}

type validateClaims struct {
	Valid    bool     `json:"valid"`
	UserID   string   `json:"userId"`
	TenantID string   `json:"tenantId"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	Scope    string   `json:"scope"`
}

func validateSessionToken(mgIDURL, token string) (*validateClaims, error) {
	req, err := http.NewRequest(http.MethodGet, mgIDURL+"/v1/auth/validate", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var claims validateClaims
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, err
	}
	if !claims.Valid {
		return nil, nil
	}
	return &claims, nil
}
