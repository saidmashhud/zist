package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	mashgate "github.com/saidmashhud/mashgate/packages/sdk-go"
)

const (
	sessionCookieName = "zist_session"
	refreshCookieName = "zist_refresh"
	cookieMaxAge7days = 7 * 24 * 60 * 60
)

// mountAuth registers credential-based auth routes using the Mashgate SDK.
//
//	POST /api/auth/login    – email+password → set session + refresh cookies
//	POST /api/auth/logout   – invalidate refresh token, clear cookies
//	POST /api/auth/refresh  – exchange refresh token for new token pair
//	GET  /api/auth/me       – return user info from propagateAuth headers
func mountAuth(r chi.Router, mgClient *mashgate.Client) {
	r.Post("/api/auth/login", handleLogin(mgClient))
	r.Post("/api/auth/logout", handleLogout(mgClient))
	r.Post("/api/auth/refresh", handleRefresh(mgClient))
	r.Get("/api/auth/me", handleMe())
}

func handleLogin(mgClient *mashgate.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Email == "" || req.Password == "" {
			writeJSONError(w, http.StatusBadRequest, "email and password required")
			return
		}

		pair, err := mgClient.Login(r.Context(), req.Email, req.Password)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

		setSessionCookies(w, r, pair)
		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	}
}

func handleLogout(mgClient *mashgate.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if rc, err := r.Cookie(refreshCookieName); err == nil {
			_ = mgClient.Logout(r.Context(), rc.Value)
		}
		clearSessionCookies(w)
		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	}
}

func handleRefresh(mgClient *mashgate.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rc, err := r.Cookie(refreshCookieName)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "no refresh token")
			return
		}

		pair, err := mgClient.RefreshToken(r.Context(), rc.Value)
		if err != nil {
			clearSessionCookies(w)
			writeJSONError(w, http.StatusUnauthorized, "refresh failed")
			return
		}

		setSessionCookies(w, r, pair)
		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	}
}

// handleMe reads X-User-* headers injected by the propagateAuth middleware.
// Returns 401 if the session cookie is missing or invalid.
func handleMe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			writeJSONError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"user_id":   userID,
			"tenant_id": r.Header.Get("X-Tenant-ID"),
			"email":     r.Header.Get("X-User-Email"),
			"scopes":    r.Header.Get("X-User-Scopes"),
		})
	}
}

// setSessionCookies writes the access token + refresh token into httpOnly cookies.
func setSessionCookies(w http.ResponseWriter, r *http.Request, pair *mashgate.TokenPair) {
	secure := isSecureRequest(r)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    pair.AccessToken,
		Path:     "/",
		MaxAge:   cookieMaxAge7days,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    pair.RefreshToken,
		Path:     "/api/auth",
		MaxAge:   cookieMaxAge7days,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: refreshCookieName, Value: "", Path: "/api/auth", MaxAge: -1})
}

func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
