// Package auth provides HTTP middleware for reading propagated identity headers
// set by the Zist gateway after validating an mgID session.
//
// The gateway strips any inbound X-User-* headers (preventing injection),
// validates the zist_session cookie via mgID, and then sets:
//
//	X-User-ID      — authenticated user's UUID
//	X-Tenant-ID    — tenant the user belongs to
//	X-User-Email   — user's email address
//	X-User-Scopes  — space-separated granted scopes
//
// Services apply Middleware globally and then use RequireAuth / RequireScope
// to protect individual routes.
package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey int

const principalKey contextKey = iota

// Principal holds the authenticated identity extracted from gateway-injected headers.
type Principal struct {
	UserID   string
	TenantID string
	Email    string
	Scopes   []string
}

// HasScope reports whether the principal holds the given scope.
func (p *Principal) HasScope(scope string) bool {
	for _, s := range p.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// Middleware reads X-User-* headers injected by the gateway and stores a
// Principal in the request context. Anonymous requests (no headers) pass
// through; handlers call FromContext to check identity.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			next.ServeHTTP(w, r)
			return
		}

		scopeStr := r.Header.Get("X-User-Scopes")
		principal := &Principal{
			UserID:   userID,
			TenantID: r.Header.Get("X-Tenant-ID"),
			Email:    r.Header.Get("X-User-Email"),
			Scopes:   strings.Fields(scopeStr),
		}

		ctx := context.WithValue(r.Context(), principalKey, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FromContext returns the Principal stored in ctx by Middleware.
// Returns nil for anonymous (unauthenticated) requests.
func FromContext(ctx context.Context) *Principal {
	p, _ := ctx.Value(principalKey).(*Principal)
	return p
}

// RequireAuth is a middleware that returns 401 Unauthorized if no authenticated
// principal is present in the context.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if FromContext(r.Context()) == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`)) //nolint:errcheck
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireScope returns a middleware that responds 403 Forbidden if the
// authenticated principal does not hold the given scope.
// It implicitly requires authentication — anonymous requests receive 401.
func RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := FromContext(r.Context())
			if p == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`)) //nolint:errcheck
				return
			}
			if !p.HasScope(scope) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"insufficient_scope","required":"` + scope + `"}`)) //nolint:errcheck
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
