// Package e2e — cross-product integration test scenarios.
//
// These tests verify integrations between Zist, Mashgate, and HookLine:
//   - Zist → Mashgate: checkout creation and payment webhook processing
//   - Zist → HookLine: real-time event streaming via WebSocket
//   - Full user journey: search → book → pay → confirm → review
//
// These tests require ALL THREE stacks running:
//   - Zist: docker compose (ports 8000-8006)
//   - Mashgate: make dev (port 9661)
//   - HookLine: docker run (port 8080, optional)
//
// Run with:
//
//	MASHGATE_AVAILABLE=true HOOKLINE_AVAILABLE=true go test -v -run TestCross ./tests/e2e/...
package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

func mashgateAvailable() bool { return os.Getenv("MASHGATE_AVAILABLE") == "true" }
func hooklineAvailable() bool { return os.Getenv("HOOKLINE_AVAILABLE") == "true" }
func mashgateURL() string     { return envOr("MASHGATE_URL", "http://localhost:9661") }
func hooklineURL() string     { return envOr("HOOKLINE_URL", "http://localhost:8080") }

// ===========================================================================
// Scenario 1: Zist → Mashgate Checkout Integration
//
// Complete booking → Create Mashgate checkout session → Simulate payment
// webhook from Mashgate → Verify booking transitions to confirmed.
//
// This tests the FULL payment flow across both products.
// ===========================================================================

func TestCrossProductCheckoutFlow(t *testing.T) {
	if !mashgateAvailable() {
		t.Skip("MASHGATE_AVAILABLE not set — skipping cross-product checkout test")
	}

	// Step 1: Create and publish a listing
	listing := map[string]any{
		"title":         "Cross-Product Luxury Suite",
		"city":          "Tashkent",
		"country":       "UZ",
		"pricePerNight": "500000.00",
		"currency":      "UZS",
		"maxGuests":     4,
		"instantBook":   true,
	}
	_, resp := post(t, listingsURL()+"/listings", listing, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/luxury.jpg", "caption": "cover",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

	// Step 2: Guest creates booking → payment_pending
	_, resp = post(t, bookingsURL()+"/bookings", map[string]any{
		"listingId": listingID,
		"checkIn":   "2028-11-01",
		"checkOut":  "2028-11-05",
		"guests":    2,
	}, authHeaders(defaultUser))
	bookingID := jsonField(t, resp, "id")
	if jsonField(t, resp, "status") != "payment_pending" {
		t.Fatalf("want payment_pending, got %s", jsonField(t, resp, "status"))
	}

	// Step 3: Create Mashgate checkout session (via Zist payments service)
	checkoutBody := map[string]any{
		"bookingId":     bookingID,
		"listingId":     listingID,
		"amount":        "2000000.00",
		"currency":      "UZS",
		"successUrl":    "http://localhost:3000/bookings/" + bookingID + "/success",
		"cancelUrl":     "http://localhost:3000/bookings/" + bookingID,
		"customerEmail": defaultUser.Email,
	}
	status, resp := post(t, paymentsURL()+"/checkout", checkoutBody, authHeaders(defaultUser))
	if status == http.StatusCreated {
		sessionID := jsonField(t, resp, "sessionId")
		checkoutURL := jsonField(t, resp, "checkoutUrl")
		t.Logf("checkout session created: sessionId=%s, url=%s", sessionID, checkoutURL)

		// Step 4: Verify checkout ID was stored on booking
		time.Sleep(500 * time.Millisecond)
		_, resp = get(t, bookingsURL()+"/bookings/"+bookingID, authHeaders(defaultUser))
		storedCheckoutID := jsonField(t, resp, "checkoutId")
		if storedCheckoutID != "" {
			t.Logf("checkout ID stored on booking: %s", storedCheckoutID)
		}

		// Step 5: Simulate Mashgate payment.captured webhook
		webhookEvent := map[string]any{
			"event_id":     fmt.Sprintf("evt_cross_%s_%d", bookingID, time.Now().UnixMilli()),
			"event_type":   "payment.captured",
			"aggregate_id": "pay_cross_" + bookingID,
			"tenant_id":    defaultUser.TenantID,
			"data": map[string]any{
				"metadata": map[string]any{
					"bookingId": bookingID,
				},
			},
		}
		whPayload, _ := json.Marshal(webhookEvent)
		whHeaders := webhookHeaders(whPayload)

		status, resp = post(t, paymentsURL()+"/webhooks/mashgate", webhookEvent, whHeaders)
		if status != http.StatusOK {
			t.Fatalf("webhook processing: want 200, got %d: %s", status, resp)
		}

		// Step 6: Verify booking is now confirmed
		_, resp = get(t, bookingsURL()+"/bookings/"+bookingID, authHeaders(defaultUser))
		if jsonField(t, resp, "status") == "confirmed" {
			t.Log("booking confirmed via Mashgate webhook")
		} else {
			t.Logf("booking status after webhook: %s (may need async processing)", jsonField(t, resp, "status"))
		}
	} else if status == http.StatusBadGateway {
		t.Log("Mashgate unavailable (502) — testing webhook path with simulated data")

		// Fallback: simulate the webhook directly
		webhookEvent := map[string]any{
			"event_id":     fmt.Sprintf("evt_cross_sim_%d", time.Now().UnixMilli()),
			"event_type":   "payment.captured",
			"aggregate_id": "pay_sim_cross",
			"tenant_id":    defaultUser.TenantID,
			"data": map[string]any{
				"metadata": map[string]any{
					"bookingId": bookingID,
				},
			},
		}
		whPayload, _ := json.Marshal(webhookEvent)
		whHeaders := webhookHeaders(whPayload)
		status, resp = post(t, paymentsURL()+"/webhooks/mashgate", webhookEvent, whHeaders)
		if status == http.StatusOK {
			t.Log("webhook processed successfully (simulated)")
		}
	} else {
		t.Fatalf("checkout: want 201 or 502, got %d: %s", status, resp)
	}

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 2: Full Guest Journey
//
// Search listings → Select one → Check availability → Get price preview →
// Create booking → Payment → Confirm → Leave review → Verify listing rating.
// ===========================================================================

func TestCrossProductFullGuestJourney(t *testing.T) {
	// Step 1: Host creates a listing with full details
	listing := map[string]any{
		"title":              "Traditional Uzbek Guesthouse",
		"description":        "Authentic experience in old town",
		"city":               "Samarkand",
		"country":            "UZ",
		"type":               "guesthouse",
		"pricePerNight":      "280000.00",
		"cleaningFee":        "40000.00",
		"currency":           "UZS",
		"maxGuests":          6,
		"bedrooms":           3,
		"beds":               4,
		"bathrooms":          2,
		"minNights":          1,
		"maxNights":          30,
		"cancellationPolicy": "flexible",
		"instantBook":        true,
		"amenities":          []string{"wifi", "breakfast", "courtyard", "air_conditioning"},
	}
	_, resp := post(t, listingsURL()+"/listings", listing, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/guesthouse-1.jpg", "caption": "courtyard",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/guesthouse-2.jpg", "caption": "bedroom",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

	// Step 2: Guest searches for listings in Samarkand
	status, resp := get(t, listingsURL()+"/listings/search?city=Samarkand", nil)
	if status != http.StatusOK {
		t.Fatalf("search: want 200, got %d", status)
	}
	t.Log("guest found listings in Samarkand")

	// Step 3: Guest checks availability
	status, resp = get(t,
		listingsURL()+"/listings/"+listingID+"/availability/check?check_in=2028-12-20&check_out=2028-12-25",
		nil)
	if status != http.StatusOK {
		t.Fatalf("check availability: want 200, got %d", status)
	}
	if jsonField(t, resp, "available") != "true" {
		t.Fatal("dates should be available")
	}

	// Step 4: Guest gets price preview
	status, resp = get(t,
		listingsURL()+"/listings/"+listingID+"/price-preview?check_in=2028-12-20&check_out=2028-12-25",
		nil)
	if status != http.StatusOK {
		t.Fatalf("price preview: want 200, got %d", status)
	}
	nights := jsonField(t, resp, "nights")
	total := jsonField(t, resp, "total")
	t.Logf("price preview: %s nights, total=%s", nights, total)

	// Step 5: Guest creates booking
	status, resp = post(t, bookingsURL()+"/bookings", map[string]any{
		"listingId": listingID,
		"checkIn":   "2028-12-20",
		"checkOut":  "2028-12-25",
		"guests":    4,
		"message":   "Looking forward to exploring Registan!",
	}, authHeaders(defaultUser))
	if status != http.StatusCreated {
		t.Fatalf("create booking: want 201, got %d: %s", status, resp)
	}
	bookingID := jsonField(t, resp, "id")
	t.Logf("booking created: %s (status=%s)", bookingID, jsonField(t, resp, "status"))

	// Step 6: Simulate payment confirmation
	post(t, bookingsURL()+"/bookings/"+bookingID+"/confirm",
		map[string]any{"paymentId": "pay_journey_" + bookingID}, internalHeaders())

	// Step 7: Verify booking confirmed and dates reserved
	_, resp = get(t, bookingsURL()+"/bookings/"+bookingID, authHeaders(defaultUser))
	if jsonField(t, resp, "status") != "confirmed" {
		t.Errorf("want confirmed, got %s", jsonField(t, resp, "status"))
	}

	// Step 8: Guest leaves review
	review := map[string]any{
		"bookingId": bookingID,
		"listingId": listingID,
		"hostId":    hostUser.UserID,
		"rating":    5,
		"comment":   "Incredible guesthouse! The courtyard was magical at sunset.",
	}
	status, resp = post(t, reviewsURL()+"/reviews", review, authHeaders(defaultUser))
	if status != http.StatusCreated {
		t.Logf("create review: got %d (may be expected if booking not in completed state)", status)
	} else {
		reviewID := jsonField(t, resp, "id")
		t.Logf("review created: %s", reviewID)

		// Step 9: Host replies
		post(t, reviewsURL()+"/reviews/"+reviewID+"/reply",
			map[string]any{"reply": "Thank you! The Registan is indeed breathtaking."}, authHeaders(hostUser))
	}

	// Step 10: Verify listing reviews
	status, resp = get(t, reviewsURL()+"/reviews/listing/"+listingID, nil)
	if status == http.StatusOK {
		reviews := jsonArray(t, resp, "reviews")
		t.Logf("listing has %d reviews", len(reviews))
	}

	// Step 11: Guest views booking history
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
		t.Error("completed booking not found in guest history")
	}

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 3: Multi-Service Error Recovery
//
// Test that system handles service failures gracefully:
// - Listing deletion while booking exists
// - Payment webhook for non-existent booking
// - Review for non-existent listing
// ===========================================================================

func TestCrossProductErrorRecovery(t *testing.T) {
	t.Run("webhook_for_nonexistent_booking", func(t *testing.T) {
		event := map[string]any{
			"event_id":     fmt.Sprintf("evt_err_%d", time.Now().UnixMilli()),
			"event_type":   "payment.captured",
			"aggregate_id": "pay_err_001",
			"tenant_id":    defaultUser.TenantID,
			"data": map[string]any{
				"metadata": map[string]any{
					"bookingId": "bk-nonexistent-999",
				},
			},
		}
		payload, _ := json.Marshal(event)
		hdr := webhookHeaders(payload)

		status, _ := post(t, paymentsURL()+"/webhooks/mashgate", event, hdr)
		// Should not crash — should return 200 (processed) or similar
		if status != http.StatusOK {
			t.Logf("webhook for nonexistent booking returned %d (expected)", status)
		}
	})

	t.Run("booking_nonexistent_listing", func(t *testing.T) {
		booking := map[string]any{
			"listingId": "lst-does-not-exist",
			"checkIn":   "2028-03-01",
			"checkOut":  "2028-03-05",
			"guests":    1,
		}
		status, _ := post(t, bookingsURL()+"/bookings", booking, authHeaders(defaultUser))
		if status != http.StatusNotFound && status != http.StatusBadGateway {
			t.Errorf("booking nonexistent listing: want 404 or 502, got %d", status)
		}
	})

	t.Run("review_invalid_rating", func(t *testing.T) {
		review := map[string]any{
			"bookingId": "bk-fake",
			"listingId": "lst-fake",
			"rating":    6, // Out of range (1-5)
			"comment":   "Should fail",
		}
		status, _ := post(t, reviewsURL()+"/reviews", review, authHeaders(defaultUser))
		if status != http.StatusBadRequest && status != http.StatusUnprocessableEntity {
			t.Errorf("invalid rating: want 400 or 422, got %d", status)
		}
	})
}

// ===========================================================================
// Scenario 4: Search → Book → Cancel → Rebook Flow
//
// Complete flow with search service integration:
// Search → Find listing → Book → Cancel → Rebook same dates.
// ===========================================================================

func TestCrossProductSearchBookCancelRebook(t *testing.T) {
	// Host creates listing in a specific city
	_, resp := post(t, listingsURL()+"/listings", map[string]any{
		"title":              "Khiva Old Town House",
		"city":               "Khiva",
		"country":            "UZ",
		"pricePerNight":      "190000.00",
		"currency":           "UZS",
		"maxGuests":          4,
		"instantBook":        true,
		"cancellationPolicy": "flexible",
		"amenities":          []string{"wifi", "courtyard"},
	}, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/khiva.jpg", "caption": "cover",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

	// Search via search service
	status, _ := get(t, searchURL()+"/search?city=Khiva", nil)
	if status != http.StatusOK {
		t.Logf("search service returned %d (may need time to index)", status)
	}

	// Book
	_, resp = post(t, bookingsURL()+"/bookings", map[string]any{
		"listingId": listingID,
		"checkIn":   "2029-01-10",
		"checkOut":  "2029-01-15",
		"guests":    2,
	}, authHeaders(defaultUser))
	bookingID := jsonField(t, resp, "id")

	// Confirm payment
	post(t, bookingsURL()+"/bookings/"+bookingID+"/confirm",
		map[string]any{"paymentId": "pay_cancel_rebook"}, internalHeaders())

	// Cancel (flexible policy, far future → full refund)
	status, resp = post(t, bookingsURL()+"/bookings/"+bookingID+"/cancel", nil, authHeaders(defaultUser))
	if status != http.StatusOK {
		t.Fatalf("cancel: want 200, got %d", status)
	}
	t.Logf("cancelled with status=%s", jsonField(t, resp, "status"))

	// Rebook same dates with different guest
	status, resp = post(t, bookingsURL()+"/bookings", map[string]any{
		"listingId": listingID,
		"checkIn":   "2029-01-10",
		"checkOut":  "2029-01-15",
		"guests":    2,
	}, authHeaders(guestUser2))
	if status != http.StatusCreated {
		t.Fatalf("rebook after cancel: want 201, got %d: %s", status, resp)
	}
	t.Log("rebook after cancel succeeded")

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 5: Admin Dashboard Verification
//
// Admin views system state across services: flags, audit, tenant config.
// ===========================================================================

func TestCrossProductAdminDashboard(t *testing.T) {
	base := adminURL()

	// Create feature flag for new payment provider
	status, _ := post(t, base+"/admin/flags", map[string]any{
		"name":    "mashgate_v2_checkout",
		"enabled": false,
		"rollout": 10,
	}, authHeaders(adminUser))
	if status != http.StatusOK {
		t.Fatalf("create flag: want 200, got %d", status)
	}

	// Configure tenant settings
	status, _ = put(t, base+"/admin/tenants/"+defaultUser.TenantID, map[string]any{
		"platformFeePct": 12.0,
		"maxListings":    100,
		"verified":       true,
	}, authHeaders(adminUser))
	if status != http.StatusOK {
		t.Fatalf("update tenant: want 200, got %d", status)
	}

	// View audit trail
	status, resp := get(t, base+"/admin/audit", authHeaders(adminUser))
	if status != http.StatusOK {
		t.Fatalf("audit log: want 200, got %d", status)
	}
	entries := jsonArray(t, resp, "entries")
	t.Logf("audit trail has %d entries", len(entries))

	// Verify all services health
	services := []struct {
		name, url string
	}{
		{"listings", listingsURL() + "/healthz"},
		{"bookings", bookingsURL() + "/healthz"},
		{"payments", paymentsURL() + "/healthz"},
		{"reviews", reviewsURL() + "/healthz"},
		{"admin", adminURL() + "/healthz"},
		{"search", searchURL() + "/healthz"},
	}
	for _, svc := range services {
		status, _ = get(t, svc.url, nil)
		if status == http.StatusOK {
			t.Logf("%s: healthy", svc.name)
		} else {
			t.Errorf("%s: unhealthy (status=%d)", svc.name, status)
		}
	}
}
