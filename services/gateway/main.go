// Package main implements the Zist API gateway.
//
// Listens on two ports:
//   - HTTP/1.1+2  :8000  (GATEWAY_PORT)
//   - HTTP/3 QUIC :8443  (GATEWAY_TLS_PORT) — self-signed cert generated at startup
//
// Advertises HTTP/3 via Alt-Svc header on every HTTP/1.1 response.
// Routes:
//
//	/api/auth/*      → mgID OIDC flow (login, callback, logout, me)
//	/api/listings/*  → listings service  (strips /api prefix)
//	/api/bookings/*  → bookings service  (strips /api prefix)
//	/api/payments/*  → payments service  (strips /api prefix)
//	/*               → SvelteKit frontend (web)
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/quic-go/quic-go/http3"
)

func main() {
	httpPort := getenv("GATEWAY_PORT", "8000")
	tlsPort  := getenv("GATEWAY_TLS_PORT", "8443")

	listingsURL := getenv("LISTINGS_URL", "http://listings:8001")
	bookingsURL := getenv("BOOKINGS_URL", "http://bookings:8002")
	paymentsURL := getenv("PAYMENTS_URL", "http://payments:8003")
	webURL      := getenv("WEB_URL", "http://web:3000")

	mgIDURL     := getenv("MGID_URL", "http://host.docker.internal:9661")
	clientID    := getenv("MGID_CLIENT_ID", "zist-local")
	clientSecret := getenv("MGID_CLIENT_SECRET", "")
	redirectURI := getenv("MGID_REDIRECT_URI", "http://localhost:8000/api/auth/callback")
	mgIDAdminToken := getenv("MGID_ADMIN_TOKEN", "")
	hooklineURL := getenv("HOOKLINE_URL", "http://hookline:8080")
	hooklineKey := getenv("HOOKLINE_API_KEY", "dev-secret")

	oidcCfg := oidcConfig{
		mgIDURL:      mgIDURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Advertise HTTP/3 on every response so browsers upgrade automatically
	r.Use(func(next http.Handler) http.Handler {
		alt := fmt.Sprintf(`h3=":%s"; ma=86400`, tlsPort)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Alt-Svc", alt)
			next.ServeHTTP(w, r)
		})
	})

	// Auth propagation: validate session cookie → inject X-User-* headers
	// Runs on all /api/* requests (strips injection, sets headers from mgID).
	r.Use(propagateAuth(mgIDURL, sessionCookieName))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})

	// OIDC auth routes (handled locally, not proxied)
	mountOIDC(r, oidcCfg)

	// API routes — /api prefix is stripped before forwarding so upstreams
	// see their own path space: /listings/*, /bookings/*, /payments/*
	mountAPI(r, "listings", proxyTo(listingsURL))
	mountAPI(r, "bookings", proxyTo(bookingsURL))
	mountAPI(r, "payments", proxyTo(paymentsURL))

	// Admin webhook management — proxied to HookLine with API key injection.
	// Requires zist.webhooks.manage scope (enforced at gateway level).
	r.Handle("/api/admin/webhooks", adminWebhookProxy(hooklineURL, hooklineKey))
	r.Handle("/api/admin/webhooks/*", adminWebhookProxy(hooklineURL, hooklineKey))

	// SvelteKit frontend — catch-all (all non-API routes)
	r.Mount("/", proxyTo(webURL))

	// Register Zist's app-scoped permissions with mgID (idempotent)
	go registerZistScopes(mgIDURL, clientID, mgIDAdminToken)

	// Generate ephemeral self-signed TLS cert for HTTP/3 (local dev only)
	cert, err := selfSignedCert()
	if err != nil {
		slog.Error("failed to generate TLS cert", "err", err)
		os.Exit(1)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3"},
	}

	h3srv := &http3.Server{
		Addr:      ":" + tlsPort,
		Handler:   r,
		TLSConfig: tlsCfg,
	}

	slog.Info("Zist gateway starting",
		"http", ":"+httpPort,
		"http3/quic", ":"+tlsPort,
		"mgid", mgIDURL,
	)

	// HTTP/3 (UDP) in background
	go func() {
		if err := h3srv.ListenAndServe(); err != nil {
			slog.Error("HTTP/3 server error", "err", err)
		}
	}()

	// HTTP/1.1+2 in foreground
	if err := http.ListenAndServe(":"+httpPort, r); err != nil {
		slog.Error("HTTP server error", "err", err)
		os.Exit(1)
	}
}

// mountAPI registers /api/{name} and /api/{name}/* routes on r.
// The /api prefix is stripped from the URL before forwarding so the upstream
// service receives its native path (e.g. /listings/123, not /api/listings/123).
func mountAPI(r chi.Router, name string, h http.Handler) {
	stripped := http.StripPrefix("/api", h)
	r.Handle("/api/"+name, stripped)
	r.Handle("/api/"+name+"/*", stripped)
}

func proxyTo(target string) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		panic(fmt.Sprintf("invalid proxy target %q: %v", target, err))
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Warn("proxy error", "target", target, "path", r.URL.Path, "err", err)
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}
	return proxy
}

// selfSignedCert generates an in-memory ECDSA P-256 certificate valid for 1 year.
// Suitable for local development only — browsers will show a TLS warning.
func selfSignedCert() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "zist.local",
			Organization: []string{"Zist Local Dev"},
		},
		NotBefore:   time.Now().Add(-time.Minute),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:    []string{"localhost", "gateway"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM  := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return tls.X509KeyPair(certPEM, keyPEM)
}

// adminWebhookProxy creates a reverse proxy to HookLine that:
//   - strips /api/admin/webhooks prefix → /v1/... (HookLine native paths)
//   - injects the HookLine API key as Authorization header
func adminWebhookProxy(hooklineURL, hooklineKey string) http.Handler {
	u, err := url.Parse(hooklineURL)
	if err != nil {
		panic(fmt.Sprintf("invalid hookline URL %q: %v", hooklineURL, err))
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Warn("hookline proxy error", "path", r.URL.Path, "err", err)
		http.Error(w, "webhook service unavailable", http.StatusBadGateway)
	}
	stripped := http.StripPrefix("/api/admin/webhooks", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Map /endpoints/... → /v1/endpoints/...
		if r.URL.Path == "" || r.URL.Path == "/" {
			r.URL.Path = "/v1/endpoints"
		} else {
			r.URL.Path = "/v1" + r.URL.Path
		}
		r.Header.Set("Authorization", "Bearer "+hooklineKey)
		proxy.ServeHTTP(w, r)
	}))
	return stripped
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
