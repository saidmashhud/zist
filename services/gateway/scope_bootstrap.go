package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type appScopeDefinition struct {
	ScopeCode   string `json:"scope_code"`
	Description string `json:"description"`
}

type appScope struct {
	ClientID    string `json:"client_id"`
	ScopeCode   string `json:"scope_code"`
	Description string `json:"description"`
}

type listAppScopesResponse struct {
	Scopes []appScope `json:"scopes"`
}

var scopeSyncHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

// zistScopes are the app-scoped permissions Zist declares on mgID.
// Code is the source of truth; sync is additive/upsert and does not remove
// orphan scopes from mgID automatically.
var zistScopes = []appScopeDefinition{
	{"zist.listings.read", "Read property listings"},
	{"zist.listings.manage", "Create and update property listings"},
	{"zist.bookings.read", "View own bookings"},
	{"zist.bookings.manage", "Create and manage bookings"},
	{"zist.payments.create", "Initiate payment checkout"},
	{"zist.webhooks.manage", "Manage webhook endpoint configuration"},
}

func scopeSyncRequired() bool {
	return getenvBool("ZIST_SCOPE_SYNC_REQUIRED", false)
}

// registerZistScopes synchronizes code-defined app scopes with mgID.
//
// Behavior:
// - Upsert only (POST /v1/iam/app-scopes, idempotent on client_id+scope_code)
// - No automatic delete of orphan scopes
// - Retries to handle startup ordering when mgID is not yet ready
func registerZistScopes(mgIDURL, clientID, adminToken string) error {
	if !getenvBool("ZIST_SCOPE_SYNC_ENABLED", true) {
		slog.Info("zist scope sync disabled")
		return nil
	}
	if strings.TrimSpace(clientID) == "" {
		return errors.New("scope sync: MGID_CLIENT_ID is required")
	}
	if strings.TrimSpace(adminToken) == "" {
		return errors.New("scope sync: MGID_ADMIN_TOKEN is required")
	}

	attempts := getenvInt("ZIST_SCOPE_SYNC_ATTEMPTS", 5)
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		err := syncScopesOnce(mgIDURL, clientID, adminToken)
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt < attempts {
			slog.Warn("scope sync attempt failed, retrying",
				"attempt", attempt,
				"max_attempts", attempts,
				"err", err,
			)
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
	}
	return fmt.Errorf("scope sync failed after %d attempts: %w", attempts, lastErr)
}

func syncScopesOnce(mgIDURL, clientID, adminToken string) error {
	existing, err := listAppScopes(mgIDURL, clientID, adminToken)
	if err != nil {
		return err
	}

	desired := make(map[string]string, len(zistScopes))
	for _, s := range zistScopes {
		desired[s.ScopeCode] = s.Description
	}

	created := 0
	updated := 0
	unchanged := 0

	for _, s := range zistScopes {
		current, ok := existing[s.ScopeCode]
		if ok && strings.TrimSpace(current.Description) == s.Description {
			unchanged++
			continue
		}
		if err := upsertAppScope(mgIDURL, clientID, adminToken, s); err != nil {
			return err
		}
		if ok {
			updated++
		} else {
			created++
		}
	}

	var orphan []string
	for code := range existing {
		if _, ok := desired[code]; !ok {
			orphan = append(orphan, code)
		}
	}
	sort.Strings(orphan)

	slog.Info("zist app scopes synced",
		"client_id", clientID,
		"declared", len(zistScopes),
		"created", created,
		"updated", updated,
		"unchanged", unchanged,
		"orphan_count", len(orphan),
	)
	if len(orphan) > 0 {
		slog.Warn("orphan app scopes exist in mgID (auto-delete disabled)",
			"client_id", clientID,
			"orphans_preview", joinFirst(orphan, 8),
		)
	}
	return nil
}

func listAppScopes(mgIDURL, clientID, adminToken string) (map[string]appScope, error) {
	params := url.Values{}
	params.Set("client_id", clientID)
	endpoint := strings.TrimRight(mgIDURL, "/") + "/v1/iam/app-scopes?" + params.Encode()

	req, err := newScopeSyncRequest(http.MethodGet, endpoint, nil, adminToken)
	if err != nil {
		return nil, err
	}
	resp, err := scopeSyncHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("list app scopes failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out listAppScopesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode app scopes response: %w", err)
	}
	result := make(map[string]appScope, len(out.Scopes))
	for _, s := range out.Scopes {
		result[s.ScopeCode] = s
	}
	return result, nil
}

func upsertAppScope(mgIDURL, clientID, adminToken string, scope appScopeDefinition) error {
	body := map[string]string{
		"client_id":   clientID,
		"scope_code":  scope.ScopeCode,
		"description": scope.Description,
	}
	data, _ := json.Marshal(body)

	req, err := newScopeSyncRequest(
		http.MethodPost,
		strings.TrimRight(mgIDURL, "/")+"/v1/iam/app-scopes",
		bytes.NewReader(data),
		adminToken,
	)
	if err != nil {
		return err
	}

	resp, err := scopeSyncHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("upsert app scope %s failed: status=%d body=%s", scope.ScopeCode, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func newScopeSyncRequest(method, endpoint string, body io.Reader, adminToken string) (*http.Request, error) {
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	return req, nil
}

func getenvBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(getenv(key, ""))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func getenvInt(key string, fallback int) int {
	raw := strings.TrimSpace(getenv(key, ""))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func joinFirst(items []string, n int) string {
	if len(items) == 0 {
		return ""
	}
	if n > len(items) {
		n = len(items)
	}
	joined := strings.Join(items[:n], ",")
	if len(items) > n {
		return joined + ",..."
	}
	return joined
}
