package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
)

type oidcConfig struct {
	mgIDURL      string
	clientID     string
	clientSecret string
	redirectURI  string
}

const (
	pkceCookieName    = "zist_pkce"
	sessionCookieName = "zist_session"
	cookieMaxAge5min  = 5 * 60
	cookieMaxAge7days = 7 * 24 * 60 * 60
)

func mountOIDC(r chi.Router, cfg oidcConfig) {
	r.Get("/api/auth/login", handleLogin(cfg))
	r.Get("/api/auth/callback", handleCallback(cfg))
	r.Post("/api/auth/logout", handleLogout)
	r.Get("/api/auth/me", handleMe(cfg))
}

// handleLogin initiates the PKCE Authorization Code flow.
// Generates a code_verifier, derives the SHA-256 challenge, stores the
// verifier + state in a short-lived httpOnly cookie, then redirects to
// the mgID authorization endpoint.
func handleLogin(cfg oidcConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		verifier, err := randomBase64(32)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		state, err := randomBase64(16)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		// code_challenge = BASE64URL(SHA256(verifier))
		h := sha256.Sum256([]byte(verifier))
		challenge := base64.RawURLEncoding.EncodeToString(h[:])

		// Store verifier + state so we can verify on callback
		pkceCookieVal := verifier + ":" + state
		http.SetCookie(w, &http.Cookie{
			Name:     pkceCookieName,
			Value:    pkceCookieVal,
			Path:     "/",
			MaxAge:   cookieMaxAge5min,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		params := url.Values{
			"response_type":         {"code"},
			"client_id":             {cfg.clientID},
			"redirect_uri":          {cfg.redirectURI},
			"scope":                 {"openid profile email zist.listings.read zist.listings.manage zist.bookings.read zist.bookings.manage zist.payments.create"},
			"state":                 {state},
			"code_challenge":        {challenge},
			"code_challenge_method": {"S256"},
		}

		http.Redirect(w, r, cfg.mgIDURL+"/oauth/authorize?"+params.Encode(), http.StatusFound)
	}
}

// handleCallback exchanges the authorization code for tokens using the
// stored PKCE verifier, then sets a session cookie.
func handleCallback(cfg oidcConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" {
			writeJSONError(w, http.StatusBadRequest, "missing code")
			return
		}

		// Read and validate PKCE cookie
		cookie, err := r.Cookie(pkceCookieName)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "missing pkce cookie")
			return
		}
		parts := strings.SplitN(cookie.Value, ":", 2)
		if len(parts) != 2 || parts[1] != state {
			writeJSONError(w, http.StatusBadRequest, "state mismatch")
			return
		}
		verifier := parts[0]

		// Clear PKCE cookie
		http.SetCookie(w, &http.Cookie{
			Name:   pkceCookieName,
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})

		// Exchange code + verifier for tokens
		tokenResp, err := exchangeCode(cfg, code, verifier)
		if err != nil {
			writeJSONError(w, http.StatusBadGateway, "token exchange failed")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    tokenResp.AccessToken,
			Path:     "/",
			MaxAge:   cookieMaxAge7days,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		// Redirect to homepage after successful login
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// handleMe proxies the userinfo request to mgID using the session cookie token.
func handleMe(cfg oidcConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "not authenticated")
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, cfg.mgIDURL+"/oauth/userinfo", nil)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "server error")
			return
		}
		req.Header.Set("Authorization", "Bearer "+cookie.Value)

		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			writeJSONError(w, http.StatusUnauthorized, "invalid session")
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body) //nolint:errcheck
		w.Write(buf.Bytes())    //nolint:errcheck
	}
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func exchangeCode(cfg oidcConfig, code, verifier string) (*tokenResponse, error) {
	body := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {cfg.redirectURI},
		"client_id":     {cfg.clientID},
		"code_verifier": {verifier},
	}
	if cfg.clientSecret != "" {
		body.Set("client_secret", cfg.clientSecret)
	}

	resp, err := http.Post(
		cfg.mgIDURL+"/oauth/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(body.Encode()),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d", resp.StatusCode)
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}
	return &tr, nil
}

func randomBase64(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
