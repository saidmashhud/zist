package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// buildTestJWT creates a signed JWT for testing using ES256.
func buildTestJWT(t *testing.T, key *ecdsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()

	header := map[string]string{"alg": "ES256", "typ": "JWT", "kid": kid}
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := headerB64 + "." + claimsB64

	hash := sha256.Sum256([]byte(signingInput))
	sigBytes, err := ecdsa.SignASN1(rand.Reader, key, hash[:])
	if err != nil {
		t.Fatal(err)
	}
	sigB64 := base64.RawURLEncoding.EncodeToString(sigBytes)

	return signingInput + "." + sigB64
}

// serveJWKS starts a test HTTP server serving a JWKS with the given EC key.
func serveJWKS(t *testing.T, key *ecdsa.PrivateKey, kid string) *httptest.Server {
	t.Helper()
	xBytes := key.PublicKey.X.Bytes()
	yBytes := key.PublicKey.Y.Bytes()

	// Pad to 32 bytes for P-256
	byteLen := (key.PublicKey.Curve.Params().BitSize + 7) / 8
	xPadded := make([]byte, byteLen)
	yPadded := make([]byte, byteLen)
	copy(xPadded[byteLen-len(xBytes):], xBytes)
	copy(yPadded[byteLen-len(yBytes):], yBytes)

	jwks := map[string]any{
		"keys": []map[string]string{
			{
				"kty": "EC",
				"crv": "P-256",
				"kid": kid,
				"use": "sig",
				"alg": "ES256",
				"x":   base64.RawURLEncoding.EncodeToString(xPadded),
				"y":   base64.RawURLEncoding.EncodeToString(yPadded),
			},
		},
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
}

func TestVerifyJWT_ValidToken(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	kid := "test-key-1"
	srv := serveJWKS(t, key, kid)
	defer srv.Close()

	// The JWKS cache expects /.well-known/jwks.json path, but our test server
	// serves on any path. Override the jwksURL.
	cache := &jwksCache{
		keys:    make(map[string]crypto.PublicKey),
		ttl:     5 * time.Minute,
		jwksURL: srv.URL,
	}

	claims := map[string]any{
		"sub":       "user-123",
		"tenant_id": "tenant-456",
		"email":     "user@example.com",
		"scope":     "admin.read admin.write",
		"iss":       "http://issuer.test",
		"aud":       []string{"zist-local", "other-aud"},
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}

	token := buildTestJWT(t, key, kid, claims)
	result, err := verifyJWT(cache, token, "http://issuer.test", "zist-local")
	if err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}
	if result.Sub != "user-123" {
		t.Fatalf("expected sub=user-123, got %s", result.Sub)
	}
	if result.TenantID != "tenant-456" {
		t.Fatalf("expected tenant_id=tenant-456, got %s", result.TenantID)
	}
	if result.Email != "user@example.com" {
		t.Fatalf("expected email=user@example.com, got %s", result.Email)
	}
	if result.Scope != "admin.read admin.write" {
		t.Fatalf("expected scope='admin.read admin.write', got %s", result.Scope)
	}
}

func TestVerifyJWT_ExpiredToken(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	kid := "test-key-1"
	srv := serveJWKS(t, key, kid)
	defer srv.Close()

	cache := &jwksCache{
		keys:    make(map[string]crypto.PublicKey),
		ttl:     5 * time.Minute,
		jwksURL: srv.URL,
	}

	claims := map[string]any{
		"sub":       "user-123",
		"tenant_id": "tenant-456",
		"iss":       "http://issuer.test",
		"aud":       "zist-local",
		"exp":       time.Now().Add(-time.Hour).Unix(), // expired
	}

	token := buildTestJWT(t, key, kid, claims)
	_, err = verifyJWT(cache, token, "http://issuer.test", "zist-local")
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestVerifyJWT_WrongKey(t *testing.T) {
	// Sign with one key, serve a different key in JWKS
	signingKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	jwksKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	kid := "test-key-1"
	srv := serveJWKS(t, jwksKey, kid)
	defer srv.Close()

	cache := &jwksCache{
		keys:    make(map[string]crypto.PublicKey),
		ttl:     5 * time.Minute,
		jwksURL: srv.URL,
	}

	claims := map[string]any{
		"sub":       "user-123",
		"tenant_id": "tenant-456",
		"iss":       "http://issuer.test",
		"aud":       "zist-local",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}

	token := buildTestJWT(t, signingKey, kid, claims)
	_, err := verifyJWT(cache, token, "http://issuer.test", "zist-local")
	if err == nil {
		t.Fatal("expected error for wrong key")
	}
}

func TestVerifyJWT_MalformedToken(t *testing.T) {
	cache := &jwksCache{
		keys: make(map[string]crypto.PublicKey),
		ttl:  5 * time.Minute,
	}

	_, err := verifyJWT(cache, "not-a-jwt", "", "")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}

	_, err = verifyJWT(cache, "a.b", "", "")
	if err == nil {
		t.Fatal("expected error for 2-part token")
	}
}

func TestVerifyJWT_UnknownKid(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	// Serve JWKS with kid "key-A" but sign with kid "key-B"
	srv := serveJWKS(t, key, "key-A")
	defer srv.Close()

	cache := &jwksCache{
		keys:    make(map[string]crypto.PublicKey),
		ttl:     5 * time.Minute,
		jwksURL: srv.URL,
	}

	claims := map[string]any{
		"sub":       "user-123",
		"tenant_id": "tenant-456",
		"iss":       "http://issuer.test",
		"aud":       "zist-local",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}

	token := buildTestJWT(t, key, "key-B", claims) // different kid
	_, err := verifyJWT(cache, token, "http://issuer.test", "zist-local")
	if err == nil {
		t.Fatal("expected error for unknown kid")
	}
}

func TestJWKSCache_Refresh(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		xBytes := key.PublicKey.X.Bytes()
		yBytes := key.PublicKey.Y.Bytes()
		byteLen := 32
		xPadded := make([]byte, byteLen)
		yPadded := make([]byte, byteLen)
		copy(xPadded[byteLen-len(xBytes):], xBytes)
		copy(yPadded[byteLen-len(yBytes):], yBytes)

		jwks := map[string]any{
			"keys": []map[string]string{{
				"kty": "EC", "crv": "P-256", "kid": "k1", "alg": "ES256",
				"x": base64.RawURLEncoding.EncodeToString(xPadded),
				"y": base64.RawURLEncoding.EncodeToString(yPadded),
			}},
		}
		json.NewEncoder(w).Encode(jwks)
	}))
	defer srv.Close()

	cache := &jwksCache{
		keys:    make(map[string]crypto.PublicKey),
		ttl:     time.Hour,
		jwksURL: srv.URL,
	}

	// First call — should fetch
	_, _ = cache.getKey("k1")
	if callCount != 1 {
		t.Fatalf("expected 1 fetch, got %d", callCount)
	}

	// Second call within TTL — should use cache
	_, _ = cache.getKey("k1")
	if callCount != 1 {
		t.Fatalf("expected still 1 fetch (cached), got %d", callCount)
	}

	// Expire cache
	cache.mu.Lock()
	cache.fetched = time.Time{}
	cache.mu.Unlock()

	_, _ = cache.getKey("k1")
	if callCount != 2 {
		t.Fatalf("expected 2 fetches after expiry, got %d", callCount)
	}
}

func TestJWK_RSA(t *testing.T) {
	n := big.NewInt(0)
	n.SetString("00b3510a2f7c7aeeb18c5f0b2cfc625c038b0faa6c78dfe983e0f2c7b8e1b6e4d8f72c23d8e9b6c24a6b6d0e5b6c24a6b6d0e5b6c24a6b", 16)

	k := &jwk{
		Kty: "RSA",
		N:   base64.RawURLEncoding.EncodeToString(n.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(65537).Bytes()),
	}

	pub, err := k.toPublicKey()
	if err != nil {
		t.Fatalf("failed to parse RSA key: %v", err)
	}
	if pub == nil {
		t.Fatal("expected non-nil public key")
	}
	_ = fmt.Sprintf("RSA key parsed: %T", pub)
}
