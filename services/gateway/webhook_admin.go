package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	mashgate "github.com/saidmashhud/mashgate/packages/sdk-go"
)

// mashgateWebhookAdmin returns an http.Handler that routes webhook admin
// operations through the Mashgate SDK (mg-events gRPC → HookLine).
// Zist never talks to HookLine directly — all calls go through the canonical
// events control-plane.
func mashgateWebhookAdmin(mg *mashgate.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/admin/webhooks")
		if path == "" {
			path = "/"
		}

		switch {
		// GET /api/admin/webhooks → list endpoints
		case r.Method == http.MethodGet && path == "/":
			eps, err := mg.Events.ListEndpoints(r.Context())
			if err != nil {
				slog.Warn("webhook admin: list endpoints", "err", err)
				writeAdminError(w, err)
				return
			}
			writeAdminJSON(w, http.StatusOK, map[string]any{"endpoints": eps})

		// POST /api/admin/webhooks → create endpoint
		case r.Method == http.MethodPost && path == "/":
			var req mashgate.CreateEndpointRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeAdminJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
				return
			}
			ep, err := mg.Events.CreateEndpoint(r.Context(), req)
			if err != nil {
				slog.Warn("webhook admin: create endpoint", "err", err)
				writeAdminError(w, err)
				return
			}
			writeAdminJSON(w, http.StatusCreated, ep)

		// DELETE /api/admin/webhooks/{id} → delete endpoint
		case r.Method == http.MethodDelete && len(path) > 1 && !strings.Contains(path[1:], "/"):
			id := path[1:]
			if err := mg.Events.DeleteEndpoint(r.Context(), id); err != nil {
				slog.Warn("webhook admin: delete endpoint", "id", id, "err", err)
				writeAdminError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		// POST /api/admin/webhooks/{id}/rotate-secret → rotate signing secret
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/rotate-secret"):
			id := strings.TrimSuffix(strings.TrimPrefix(path, "/"), "/rotate-secret")
			ep, err := mg.Events.RotateSecret(r.Context(), id)
			if err != nil {
				slog.Warn("webhook admin: rotate secret", "id", id, "err", err)
				writeAdminError(w, err)
				return
			}
			writeAdminJSON(w, http.StatusOK, ep)

		// GET /api/admin/webhooks/{id}/deliveries → list deliveries
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/deliveries"):
			id := strings.TrimSuffix(strings.TrimPrefix(path, "/"), "/deliveries")
			deliveries, err := mg.Events.ListDeliveries(r.Context(), id)
			if err != nil {
				slog.Warn("webhook admin: list deliveries", "id", id, "err", err)
				writeAdminError(w, err)
				return
			}
			writeAdminJSON(w, http.StatusOK, map[string]any{"deliveries": deliveries})

		// POST /api/admin/webhooks/{id}/deliveries/{did}/retry → retry delivery
		case r.Method == http.MethodPost && strings.Contains(path, "/deliveries/") && strings.HasSuffix(path, "/retry"):
			parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
			// parts: [id, deliveries, did, retry]
			if len(parts) == 4 {
				if err := mg.Events.RetryDelivery(r.Context(), parts[0], parts[2]); err != nil {
					slog.Warn("webhook admin: retry delivery", "endpoint", parts[0], "delivery", parts[2], "err", err)
					writeAdminError(w, err)
					return
				}
				writeAdminJSON(w, http.StatusOK, map[string]string{"status": "retried"})
			} else {
				writeAdminJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			}

		default:
			writeAdminJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		}
	})
}

func writeAdminJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeAdminError(w http.ResponseWriter, err error) {
	switch err.(type) {
	case *mashgate.EndpointNotFoundError:
		writeAdminJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case *mashgate.SubscriptionConflictError:
		writeAdminJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	case *mashgate.QuotaExceededError:
		writeAdminJSON(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
	case *mashgate.EventsServiceUnavailableError:
		writeAdminJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
	default:
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
}
