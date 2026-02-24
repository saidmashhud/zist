package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// jwksCache fetches and caches the JSON Web Key Set from the mgID OIDC provider.
// Keys are refreshed automatically when the cache TTL expires.
type jwksCache struct {
	mu      sync.RWMutex
	keys    map[string]crypto.PublicKey // kid → public key
	fetched time.Time
	ttl     time.Duration
	jwksURL string
}

func newJWKSCache(mgIDURL string, ttl time.Duration) *jwksCache {
	return &jwksCache{
		keys:    make(map[string]crypto.PublicKey),
		ttl:     ttl,
		jwksURL: mgIDURL + "/.well-known/jwks.json",
	}
}

// getKey returns the public key for the given kid, refreshing the cache if needed.
func (c *jwksCache) getKey(kid string) (crypto.PublicKey, error) {
	c.mu.RLock()
	if time.Since(c.fetched) < c.ttl {
		if k, ok := c.keys[kid]; ok {
			c.mu.RUnlock()
			return k, nil
		}
	}
	c.mu.RUnlock()

	// Cache miss or expired — refresh
	if err := c.refresh(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	k, ok := c.keys[kid]
	if !ok {
		return nil, fmt.Errorf("unknown kid %q", kid)
	}
	return k, nil
}

func (c *jwksCache) refresh() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(c.fetched) < c.ttl {
		return nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(c.jwksURL)
	if err != nil {
		return fmt.Errorf("jwks fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks fetch: status %d", resp.StatusCode)
	}

	var jwks struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("jwks decode: %w", err)
	}

	keys := make(map[string]crypto.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		pub, err := k.toPublicKey()
		if err != nil {
			continue // skip unsupported key types
		}
		keys[k.Kid] = pub
	}

	c.keys = keys
	c.fetched = time.Now()
	return nil
}

// jwk is a minimal JSON Web Key representation supporting RSA and EC keys.
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	// RSA fields
	N string `json:"n"`
	E string `json:"e"`
	// EC fields
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func (k *jwk) toPublicKey() (crypto.PublicKey, error) {
	switch k.Kty {
	case "RSA":
		return k.toRSA()
	case "EC":
		return k.toEC()
	default:
		return nil, fmt.Errorf("unsupported key type %q", k.Kty)
	}
}

func (k *jwk) toRSA() (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e*256 + int(b)
	}
	return &rsa.PublicKey{N: n, E: e}, nil
}

func (k *jwk) toEC() (*ecdsa.PublicKey, error) {
	var curve elliptic.Curve
	switch k.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported curve %q", k.Crv)
	}
	xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
	if err != nil {
		return nil, err
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
	if err != nil {
		return nil, err
	}
	return &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

// jwtClaims holds the standard claims we need from the mgID access token.
type jwtClaims struct {
	Sub      string      `json:"sub"`       // user ID
	TenantID string      `json:"tenant_id"` // tenant
	Email    string      `json:"email"`
	Scope    string      `json:"scope"` // space-separated scopes
	Roles    []string    `json:"roles"`
	Iss      string      `json:"iss"`
	Aud      jwtAudience `json:"aud"`
	Exp      int64       `json:"exp"`
	Iat      int64       `json:"iat"`
	Nbf      int64       `json:"nbf"`
}

// verifyJWT parses and verifies a JWT using the JWKS cache.
// Returns the validated claims or an error.
func verifyJWT(cache *jwksCache, tokenStr, expectedIssuer, expectedAudience string) (*jwtClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed JWT: expected 3 parts")
	}

	// Decode header to get kid and alg
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}

	// Get public key
	pubKey, err := cache.getKey(header.Kid)
	if err != nil {
		return nil, fmt.Errorf("key lookup: %w", err)
	}

	// Verify signature
	signingInput := parts[0] + "." + parts[1]
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	switch header.Alg {
	case "RS256":
		rsaKey, ok := pubKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("key type mismatch for RS256")
		}
		hash := sha256.Sum256([]byte(signingInput))
		if err := rsa.VerifyPKCS1v15(rsaKey, crypto.SHA256, hash[:], sigBytes); err != nil {
			return nil, fmt.Errorf("RS256 verify: %w", err)
		}
	case "ES256":
		ecKey, ok := pubKey.(*ecdsa.PublicKey)
		if !ok {
			return nil, errors.New("key type mismatch for ES256")
		}
		hash := sha256.Sum256([]byte(signingInput))
		if !ecdsa.VerifyASN1(ecKey, hash[:], sigBytes) {
			return nil, errors.New("ES256 signature verification failed")
		}
	default:
		return nil, fmt.Errorf("unsupported algorithm %q", header.Alg)
	}

	// Decode and validate claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}
	var claims jwtClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	// Check expiration
	now := time.Now().Unix()
	if claims.Exp > 0 && now > claims.Exp {
		return nil, errors.New("token expired")
	}
	if claims.Nbf > 0 && now < claims.Nbf {
		return nil, errors.New("token is not valid yet")
	}

	if strings.TrimSpace(claims.Sub) == "" || strings.TrimSpace(claims.TenantID) == "" {
		return nil, errors.New("missing required claims")
	}

	if expectedIssuer != "" && normalizeIssuer(claims.Iss) != normalizeIssuer(expectedIssuer) {
		return nil, fmt.Errorf("unexpected issuer %q", claims.Iss)
	}
	if expectedAudience != "" && !claims.Aud.Contains(expectedAudience) {
		return nil, fmt.Errorf("unexpected audience: %v", []string(claims.Aud))
	}

	if claims.Scope == "" && len(claims.Roles) > 0 {
		claims.Scope = strings.Join(claims.Roles, " ")
	}

	return &claims, nil
}

type jwtAudience []string

func (a *jwtAudience) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*a = nil
		return nil
	}

	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*a = jwtAudience{single}
		return nil
	}

	var multi []string
	if err := json.Unmarshal(data, &multi); err == nil {
		*a = jwtAudience(multi)
		return nil
	}

	return fmt.Errorf("invalid aud claim")
}

func (a jwtAudience) Contains(expected string) bool {
	expected = strings.TrimSpace(expected)
	for _, v := range a {
		if strings.TrimSpace(v) == expected {
			return true
		}
	}
	return false
}

func normalizeIssuer(v string) string {
	return strings.TrimRight(strings.TrimSpace(v), "/")
}
