package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// propagateAuth returns a middleware that:
//  1. Strips inbound X-User-* headers to prevent header injection attacks.
//  2. Reads the session cookie and validates the JWT locally using JWKS.
//  3. If valid, sets X-User-ID, X-Tenant-ID, X-User-Email, X-User-Scopes on the
//     forwarded request so downstream services can trust them.
//  4. Anonymous requests (no cookie or invalid token) pass through with no user headers.
func propagateAuth(mgIDURL, clientID, cookieName string) func(http.Handler) http.Handler {
	jwks := newJWKSCache(mgIDURL, 5*time.Minute)

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
				next.ServeHTTP(w, r)
				return
			}

			// 3. Try local JWKS verification first (fast path)
			jwtClaims, err := verifyJWT(jwks, cookie.Value, mgIDURL, clientID)
			if err == nil && jwtClaims != nil {
				r.Header.Set("X-User-ID", jwtClaims.Sub)
				r.Header.Set("X-Tenant-ID", jwtClaims.TenantID)
				r.Header.Set("X-User-Email", jwtClaims.Email)
				r.Header.Set("X-User-Scopes", jwtClaims.Scope)
				next.ServeHTTP(w, r)
				return
			}

			if err != nil {
				slog.Debug("JWKS verify failed, falling back to HTTP", "err", err)
			}

			// 4. Fallback: POST /v1/auth/validate
			claims, httpErr := validateSessionTokenHTTP(mgIDURL, cookie.Value)
			if httpErr != nil || claims == nil {
				if httpErr != nil {
					slog.Debug("auth validate failed", "err", httpErr)
				}
				next.ServeHTTP(w, r)
				return
			}

			r.Header.Set("X-User-ID", claims.UserID)
			r.Header.Set("X-Tenant-ID", claims.TenantID)
			r.Header.Set("X-User-Email", claims.Email)
			r.Header.Set("X-User-Scopes", claims.Scope)

			next.ServeHTTP(w, r)
		})
	}
}

type validateClaims struct {
	Valid    bool   `json:"valid"`
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Email    string // populated from JWT payload
	Scope    string // populated from JWT payload roles
}

// validateSessionTokenHTTP validates a token via POST /v1/auth/validate.
// The endpoint requires the token in both the Authorization header and request body.
func validateSessionTokenHTTP(mgIDURL, token string) (*validateClaims, error) {
	body := fmt.Sprintf(`{"token":%q}`, token)
	req, err := http.NewRequest(http.MethodPost, mgIDURL+"/v1/auth/validate",
		strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var result struct {
		Valid    bool   `json:"valid"`
		UserID   string `json:"user_id"`
		TenantID string `json:"tenant_id"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if !result.Valid {
		return nil, nil
	}

	// Enrich with email and roles from the JWT payload (no signature check needed —
	// the validate call above already confirmed the token is authentic).
	email, roles := jwtPayloadFields(token)

	return &validateClaims{
		Valid:    true,
		UserID:   result.UserID,
		TenantID: result.TenantID,
		Email:    email,
		Scope:    roles,
	}, nil
}

// jwtPayloadFields base64-decodes the JWT payload and extracts email + roles.
// This is safe to call after the token has been validated server-side.
func jwtPayloadFields(token string) (email, roles string) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return
	}
	var claims struct {
		Email string   `json:"email"`
		Roles []string `json:"roles"`
		Scope string   `json:"scope"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return
	}
	email = claims.Email
	if claims.Scope != "" {
		roles = claims.Scope
	} else {
		roles = strings.Join(claims.Roles, " ")
	}
	return
}
