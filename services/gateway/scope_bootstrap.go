package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// zistScopes are the app-scoped permissions Zist declares on mgID.
// This registration is idempotent — safe to run on every startup.
var zistScopes = []struct {
	ScopeCode   string `json:"scope_code"`
	Description string `json:"description"`
}{
	{"zist.listings.read", "Read property listings"},
	{"zist.listings.manage", "Create and update property listings"},
	{"zist.bookings.read", "View own bookings"},
	{"zist.bookings.manage", "Create and manage bookings"},
	{"zist.payments.create", "Initiate payment checkout"},
}

// registerZistScopes idempotently registers Zist's app-scoped permissions
// with mgID. Called in a goroutine at gateway startup.
func registerZistScopes(mgIDURL, clientID, adminToken string) {
	// Retry a few times to handle mgID not being ready yet
	for attempt := range 5 {
		if err := tryRegisterScopes(mgIDURL, clientID, adminToken); err != nil {
			slog.Warn("scope registration attempt failed, retrying",
				"attempt", attempt+1,
				"err", err,
			)
			time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
			continue
		}
		slog.Info("zist app scopes registered", "count", len(zistScopes))
		return
	}
	slog.Error("failed to register zist app scopes after retries")
}

func tryRegisterScopes(mgIDURL, clientID, adminToken string) error {
	for _, s := range zistScopes {
		body := map[string]string{
			"client_id":   clientID,
			"scope_code":  s.ScopeCode,
			"description": s.Description,
		}
		data, _ := json.Marshal(body)

		req, err := http.NewRequest(http.MethodPost, mgIDURL+"/v1/iam/app-scopes", bytes.NewReader(data))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		if adminToken != "" {
			req.Header.Set("Authorization", "Bearer "+adminToken)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			slog.Warn("failed to register scope",
				"scope", s.ScopeCode,
				"status", resp.StatusCode,
			)
		}
	}
	return nil
}
