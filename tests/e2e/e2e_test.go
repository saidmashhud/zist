// Package e2e contains integration tests for the Zist microservices stack.
//
// These tests call the individual services directly (bypassing the gateway)
// and simulate authentication via X-User-* headers. For gateway-level tests
// (OIDC, proxy routing), use smoke.sh against a running docker-compose stack.
//
// Usage:
//
//	go test -v -count=1 ./tests/e2e/...
//
// Environment:
//
//	LISTINGS_URL   (default http://localhost:8001)
//	BOOKINGS_URL   (default http://localhost:8002)
//	PAYMENTS_URL   (default http://localhost:8003)
//	GATEWAY_URL    (default http://localhost:8000)
//	INTERNAL_TOKEN (default dev-internal-token)
package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// ==========================================================================
// Health Checks
// ==========================================================================

func TestHealthChecks(t *testing.T) {
	endpoints := []struct {
		name string
		url  string
	}{
		{"gateway", gatewayURL() + "/healthz"},
		{"listings", listingsURL() + "/healthz"},
		{"bookings", bookingsURL() + "/healthz"},
		{"payments", paymentsURL() + "/healthz"},
	}

	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			status, _ := get(t, ep.url, nil)
			if status != http.StatusOK {
				t.Errorf("GET %s: want 200, got %d", ep.url, status)
			}
		})
	}
}

// ==========================================================================
// Listings Lifecycle
// ==========================================================================

func TestListingsLifecycle(t *testing.T) {
	base := listingsURL()

	// Create
	body := map[string]any{
		"title":         "E2E Test Villa",
		"description":   "A test listing",
		"city":          "Tashkent",
		"country":       "UZ",
		"pricePerNight": "250000.00",
		"currency":      "UZS",
		"maxGuests":     4,
	}
	status, resp := post(t, base+"/listings", body, authHeaders(defaultUser))
	if status != http.StatusCreated {
		t.Fatalf("create listing: want 201, got %d: %s", status, resp)
	}
	listingID := jsonField(t, resp, "id")
	if listingID == "" {
		t.Fatal("create listing: missing id in response")
	}

	// List
	status, resp = get(t, base+"/listings", nil)
	if status != http.StatusOK {
		t.Fatalf("list listings: want 200, got %d", status)
	}

	// Get by ID
	status, resp = get(t, base+"/listings/"+listingID, nil)
	if status != http.StatusOK {
		t.Fatalf("get listing: want 200, got %d", status)
	}
	if jsonField(t, resp, "id") != listingID {
		t.Error("get listing: returned wrong ID")
	}

	// Update
	updateBody := map[string]any{
		"title":         "Updated Villa",
		"description":   "Updated description",
		"pricePerNight": "300000.00",
		"maxGuests":     6,
	}
	status, _ = put(t, base+"/listings/"+listingID, updateBody, authHeaders(defaultUser))
	if status != http.StatusOK {
		t.Errorf("update listing: want 200, got %d", status)
	}

	// Add at least one photo before publishing (publish precondition).
	status, _ = post(t, base+"/listings/"+listingID+"/photos", map[string]any{
		"url":     "https://example.com/e2e-listing.jpg",
		"caption": "cover",
	}, authHeaders(defaultUser))
	if status != http.StatusCreated {
		t.Fatalf("add listing photo: want 201, got %d", status)
	}

	// Publish
	status, _ = post(t, base+"/listings/"+listingID+"/publish", nil, authHeaders(defaultUser))
	if status != http.StatusOK {
		t.Errorf("publish listing: want 200, got %d", status)
	}

	// Verify published
	status, resp = get(t, base+"/listings/"+listingID, nil)
	if status != http.StatusOK {
		t.Fatalf("get listing after publish: want 200, got %d", status)
	}
	if jsonField(t, resp, "status") != "active" {
		t.Errorf("publish: want status=active, got %s", jsonField(t, resp, "status"))
	}

	// Delete
	status, _ = del(t, base+"/listings/"+listingID, authHeaders(defaultUser))
	if status != http.StatusNoContent {
		t.Errorf("delete listing: want 204, got %d", status)
	}

	// Verify deleted
	status, _ = get(t, base+"/listings/"+listingID, nil)
	if status != http.StatusNotFound {
		t.Errorf("get deleted listing: want 404, got %d", status)
	}
}

// ==========================================================================
// Listings Auth
// ==========================================================================

func TestListingsAuth(t *testing.T) {
	base := listingsURL()
	body := map[string]any{
		"title":         "Auth Test Listing",
		"city":          "Samarkand",
		"country":       "UZ",
		"pricePerNight": "100000.00",
		"currency":      "UZS",
	}

	t.Run("create without auth returns 401", func(t *testing.T) {
		status, _ := post(t, base+"/listings", body, noAuthHeaders())
		if status != http.StatusUnauthorized {
			t.Errorf("want 401, got %d", status)
		}
	})

	t.Run("create without manage scope returns 403", func(t *testing.T) {
		status, _ := post(t, base+"/listings", body, authHeaders(readOnlyUser))
		if status != http.StatusForbidden {
			t.Errorf("want 403, got %d", status)
		}
	})
}

// ==========================================================================
// Bookings Lifecycle
// ==========================================================================

func TestBookingsLifecycle(t *testing.T) {
	// Create and publish a listing first (must be active before booking).
	listingBody := map[string]any{
		"title":         "Booking Test Villa",
		"city":          "Bukhara",
		"country":       "UZ",
		"pricePerNight": "180000.00",
		"currency":      "UZS",
		"maxGuests":     4,
		"instantBook":   false,
	}
	_, listingResp := post(t, listingsURL()+"/listings", listingBody, authHeaders(hostUser))
	listingID := jsonField(t, listingResp, "id")
	if listingID == "" {
		t.Fatal("create listing for booking test: missing id")
	}

	status, _ := post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url":     "https://example.com/booking-test.jpg",
		"caption": "cover",
	}, authHeaders(hostUser))
	if status != http.StatusCreated {
		t.Fatalf("add listing photo for booking test: want 201, got %d", status)
	}

	status, _ = post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))
	if status != http.StatusOK {
		t.Fatalf("publish listing for booking test: want 200, got %d", status)
	}

	// Create booking (non-instant → pending_host_approval).
	bookingBody := map[string]any{
		"listingId": listingID,
		"checkIn":   "2027-03-10",
		"checkOut":  "2027-03-15",
		"guests":    2,
		"message":   "E2E test booking",
	}
	status, resp := post(t, bookingsURL()+"/bookings", bookingBody, authHeaders(defaultUser))
	if status != http.StatusCreated {
		t.Fatalf("create booking: want 201, got %d: %s", status, resp)
	}
	bookingID := jsonField(t, resp, "id")
	if bookingID == "" {
		t.Fatal("create booking: missing id")
	}
	if jsonField(t, resp, "status") != "pending_host_approval" {
		t.Errorf("create booking: want status=pending_host_approval, got %s", jsonField(t, resp, "status"))
	}

	// Get booking (guest can view own booking).
	status, _ = get(t, bookingsURL()+"/bookings/"+bookingID, authHeaders(defaultUser))
	if status != http.StatusOK {
		t.Fatalf("get booking: want 200, got %d", status)
	}

	// List bookings (filtered by authenticated user).
	status, resp = get(t, bookingsURL()+"/bookings", authHeaders(defaultUser))
	if status != http.StatusOK {
		t.Fatalf("list bookings: want 200, got %d", status)
	}
	bookings := jsonArray(t, resp, "bookings")
	found := false
	for _, b := range bookings {
		if m, ok := b.(map[string]any); ok && m["id"] == bookingID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("list bookings: created booking %s not found", bookingID)
	}
}

// ==========================================================================
// Bookings Internal Token
// ==========================================================================

func TestBookingsMutationAuth(t *testing.T) {
	// Use instant-book listing so booking goes directly to payment_pending.
	listingBody := map[string]any{
		"title":         "Instant Book Test",
		"city":          "Fergana",
		"pricePerNight": "120000.00",
		"currency":      "UZS",
		"maxGuests":     2,
		"instantBook":   true,
	}
	_, listingResp := post(t, listingsURL()+"/listings", listingBody, authHeaders(hostUser))
	listingID := jsonField(t, listingResp, "id")
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url":     "https://example.com/mutation-test.jpg",
		"caption": "cover",
	}, authHeaders(hostUser)) //nolint:errcheck
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser)) //nolint:errcheck

	// Create booking → payment_pending.
	bookingBody := map[string]any{
		"listingId": listingID,
		"checkIn":   "2027-04-01",
		"checkOut":  "2027-04-04",
		"guests":    1,
	}
	_, resp := post(t, bookingsURL()+"/bookings", bookingBody, authHeaders(defaultUser))
	bookingID := jsonField(t, resp, "id")
	if bookingID == "" {
		t.Fatal("create booking for mutation test: missing id")
	}

	t.Run("confirm without X-Internal-Token returns 403", func(t *testing.T) {
		status, _ := post(t, bookingsURL()+"/bookings/"+bookingID+"/confirm", nil, noAuthHeaders())
		if status != http.StatusForbidden {
			t.Errorf("want 403, got %d", status)
		}
	})

	t.Run("confirm with wrong token returns 403", func(t *testing.T) {
		headers := map[string]string{"X-Internal-Token": "wrong-token"}
		status, _ := post(t, bookingsURL()+"/bookings/"+bookingID+"/confirm", nil, headers)
		if status != http.StatusForbidden {
			t.Errorf("want 403, got %d", status)
		}
	})

	t.Run("confirm with valid token returns 204", func(t *testing.T) {
		status, _ := post(t, bookingsURL()+"/bookings/"+bookingID+"/confirm", nil, internalHeaders())
		if status != http.StatusNoContent {
			t.Errorf("want 204, got %d", status)
		}
	})
}

func TestBookingsInternalTokenRequired(t *testing.T) {
	// Create a shared instant-book listing for all sub-tests.
	listingBody := map[string]any{
		"title":         "Token Test Listing",
		"city":          "Namangan",
		"pricePerNight": "100000.00",
		"currency":      "UZS",
		"maxGuests":     2,
		"instantBook":   true,
	}
	_, listingResp := post(t, listingsURL()+"/listings", listingBody, authHeaders(hostUser))
	listingID := jsonField(t, listingResp, "id")
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url":     "https://example.com/token-test.jpg",
		"caption": "cover",
	}, authHeaders(hostUser)) //nolint:errcheck
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser)) //nolint:errcheck

	// Each internal route requires X-Internal-Token.
	internalRoutes := []struct {
		method string
		route  string
		body   any
	}{
		{"POST", "confirm", nil},
		{"POST", "fail", nil},
		{"PUT", "checkout", map[string]any{"checkoutId": "sess_test"}},
	}

	for _, rt := range internalRoutes {
		t.Run(fmt.Sprintf("%s /{id}/%s without token → 403", rt.method, rt.route), func(t *testing.T) {
			// Fresh booking per sub-test.
			bb := map[string]any{
				"listingId": listingID,
				"checkIn":   "2027-05-01",
				"checkOut":  "2027-05-03",
				"guests":    1,
			}
			_, bResp := post(t, bookingsURL()+"/bookings", bb, authHeaders(defaultUser))
			bookingID := jsonField(t, bResp, "id")

			var status int
			if rt.method == "PUT" {
				status, _ = put(t, bookingsURL()+"/bookings/"+bookingID+"/"+rt.route, rt.body, noAuthHeaders())
			} else {
				status, _ = post(t, bookingsURL()+"/bookings/"+bookingID+"/"+rt.route, rt.body, noAuthHeaders())
			}
			if status != http.StatusForbidden {
				t.Errorf("want 403, got %d", status)
			}
		})
	}
}

// ==========================================================================
// Webhook Scope Enforcement (via gateway)
// ==========================================================================

func TestWebhookScopeEnforcement(t *testing.T) {
	base := gatewayURL()

	t.Run("without auth returns 401", func(t *testing.T) {
		status, _ := get(t, base+"/api/admin/webhooks", noAuthHeaders())
		if status != http.StatusUnauthorized {
			t.Errorf("want 401, got %d", status)
		}
	})

	t.Run("forged read-only headers are not trusted by gateway", func(t *testing.T) {
		status, _ := get(t, base+"/api/admin/webhooks", authHeaders(readOnlyUser))
		if status != http.StatusUnauthorized {
			t.Errorf("want 401, got %d", status)
		}
	})

	t.Run("forged admin-like headers not trusted by gateway", func(t *testing.T) {
		status, _ := get(t, base+"/api/admin/webhooks", authHeaders(defaultUser))
		if status != http.StatusUnauthorized {
			t.Errorf("want 401, got %d", status)
		}
	})
}

// ==========================================================================
// Webhook Deduplication
// ==========================================================================

func TestWebhookDeduplication(t *testing.T) {
	base := paymentsURL()

	eventPayload := map[string]any{
		"event_id":     "evt_dedup_e2e_001",
		"event_type":   "checkout.completed",
		"aggregate_id": "sess_dedup_test",
		"tenant_id":    defaultUser.TenantID,
		"data":         map[string]any{},
	}
	payloadJSON, _ := json.Marshal(eventPayload)
	hdr := webhookHeaders(payloadJSON)

	// First call — should process normally.
	status, resp := post(t, base+"/webhooks/mashgate", eventPayload, hdr)
	if status != http.StatusOK {
		t.Fatalf("first webhook: want 200, got %d: %s", status, resp)
	}
	if jsonField(t, resp, "status") != "ok" {
		t.Errorf("first webhook: want status=ok, got %s", jsonField(t, resp, "status"))
	}

	// Second call (same event_id) — should be deduplicated.
	status, resp = post(t, base+"/webhooks/mashgate", eventPayload, hdr)
	if status != http.StatusOK {
		t.Fatalf("dedup webhook: want 200, got %d", status)
	}
	if jsonField(t, resp, "dedup") != "skipped" {
		t.Errorf("dedup webhook: want dedup=skipped, got %q", jsonField(t, resp, "dedup"))
	}
}

// ==========================================================================
// Checkout Flow
// ==========================================================================

func TestCheckoutFlow(t *testing.T) {
	base := paymentsURL()

	t.Run("checkout without auth returns 401", func(t *testing.T) {
		body := map[string]any{
			"bookingId": "bk-001",
			"amount":    "500000.00",
			"currency":  "UZS",
		}
		status, _ := post(t, base+"/checkout", body, noAuthHeaders())
		if status != http.StatusUnauthorized {
			t.Errorf("want 401, got %d", status)
		}
	})

	t.Run("checkout without payments.create scope returns 403", func(t *testing.T) {
		body := map[string]any{
			"bookingId": "bk-002",
			"amount":    "500000.00",
			"currency":  "UZS",
		}
		status, _ := post(t, base+"/checkout", body, authHeaders(readOnlyUser))
		if status != http.StatusForbidden {
			t.Errorf("want 403, got %d", status)
		}
	})

	t.Run("checkout with valid auth returns 201 or 502", func(t *testing.T) {
		body := map[string]any{
			"bookingId":     "bk-e2e-003",
			"listingId":     "lst-e2e-001",
			"amount":        "500000.00",
			"currency":      "UZS",
			"successUrl":    "http://localhost:3000/success",
			"cancelUrl":     "http://localhost:3000/cancel",
			"customerEmail": "guest@test.com",
		}
		status, resp := post(t, base+"/checkout", body, authHeaders(defaultUser))
		// 201 if Mashgate is running, 502 if unavailable — both are valid.
		if status != http.StatusCreated && status != http.StatusBadGateway {
			t.Errorf("want 201 or 502, got %d: %s", status, resp)
		}
		if status == http.StatusCreated {
			if jsonField(t, resp, "sessionId") == "" || jsonField(t, resp, "checkoutUrl") == "" {
				t.Error("checkout response missing sessionId or checkoutUrl")
			}
		}
	})
}

// ==========================================================================
// OIDC Routes (via gateway)
// ==========================================================================

func TestOIDCRoutes(t *testing.T) {
	base := gatewayURL()

	t.Run("POST /api/auth/login route exists", func(t *testing.T) {
		body := map[string]any{"email": "nobody@example.com", "password": "wrong"}
		status, _ := post(t, base+"/api/auth/login", body, nil)
		if status == http.StatusNotFound {
			t.Errorf("route not found: got 404")
		}
	})

	t.Run("GET /api/auth/me without cookie returns 401", func(t *testing.T) {
		status, _ := get(t, base+"/api/auth/me", noAuthHeaders())
		if status != http.StatusUnauthorized {
			t.Errorf("want 401, got %d", status)
		}
	})

	t.Run("POST /api/auth/logout returns 200", func(t *testing.T) {
		status, _ := post(t, base+"/api/auth/logout", nil, nil)
		if status != http.StatusOK {
			t.Errorf("want 200, got %d", status)
		}
	})
}
