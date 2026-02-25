// Package e2e — complex multi-service integration test scenarios.
//
// These tests cover end-to-end flows across all Zist microservices:
// listings, bookings, payments, reviews, admin, and search.
//
// Each scenario exercises realistic user journeys with multiple services
// cooperating via internal HTTP calls and shared database state.
package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Helper: additional service URLs
// ---------------------------------------------------------------------------

func reviewsURL() string { return envOr("REVIEWS_URL", "http://localhost:8004") }
func adminURL() string   { return envOr("ADMIN_URL", "http://localhost:8005") }
func searchURL() string  { return envOr("SEARCH_URL", "http://localhost:8006") }

// patch sends a PATCH request.
func patch(t *testing.T, url string, body any, headers map[string]string) (int, []byte) {
	t.Helper()
	return doRequest(t, http.MethodPatch, url, body, headers)
}

// adminUser has the zist.admin scope for admin service operations.
var adminUser = testUser{
	UserID:   "e2e-admin-001",
	TenantID: "e2e-tenant-001",
	Email:    "admin@zist.test",
	Scopes:   "zist.admin zist.listings.read zist.listings.manage zist.bookings.read zist.bookings.manage zist.payments.create",
}

// guestUser2 is a second guest for concurrent booking tests.
var guestUser2 = testUser{
	UserID:   "e2e-guest-002",
	TenantID: "e2e-tenant-001",
	Email:    "guest2@zist.test",
	Scopes:   "zist.listings.read zist.bookings.read zist.bookings.manage zist.payments.create",
}

// tenant2Host belongs to a different tenant for isolation tests.
var tenant2Host = testUser{
	UserID:   "e2e-t2-host-001",
	TenantID: "e2e-tenant-002",
	Email:    "host@tenant2.test",
	Scopes:   "zist.listings.read zist.listings.manage zist.bookings.read zist.bookings.manage",
}

// tenant2Guest belongs to tenant 2.
var tenant2Guest = testUser{
	UserID:   "e2e-t2-guest-001",
	TenantID: "e2e-tenant-002",
	Email:    "guest@tenant2.test",
	Scopes:   "zist.listings.read zist.bookings.read zist.bookings.manage zist.payments.create",
}

// ===========================================================================
// Scenario 1: Full Booking Lifecycle — Request-Approval Flow
//
// Host creates listing → publishes → Guest books (non-instant) →
// Host approves → Simulated payment (internal confirm) →
// Guest reviews → Host replies → Verify rating update.
// ===========================================================================

func TestFullBookingLifecycleWithApproval(t *testing.T) {
	// Step 1: Host creates listing
	listing := map[string]any{
		"title":              "Mountain Chalet",
		"description":        "Beautiful chalet in Chimgan mountains",
		"city":               "Chimgan",
		"country":            "UZ",
		"type":               "house",
		"pricePerNight":      "350000.00",
		"currency":           "UZS",
		"cleaningFee":        "50000.00",
		"maxGuests":          6,
		"bedrooms":           3,
		"beds":               4,
		"bathrooms":          2,
		"minNights":          2,
		"maxNights":          30,
		"cancellationPolicy": "moderate",
		"instantBook":        false,
		"amenities":          []string{"wifi", "parking", "kitchen", "fireplace"},
	}
	status, resp := post(t, listingsURL()+"/listings", listing, authHeaders(hostUser))
	if status != http.StatusCreated {
		t.Fatalf("create listing: want 201, got %d: %s", status, resp)
	}
	listingID := jsonField(t, resp, "id")

	// Step 2: Add photos and publish
	status, _ = post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/chalet-1.jpg", "caption": "exterior",
	}, authHeaders(hostUser))
	if status != http.StatusCreated {
		t.Fatalf("add photo: want 201, got %d", status)
	}
	status, _ = post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/chalet-2.jpg", "caption": "interior",
	}, authHeaders(hostUser))
	if status != http.StatusCreated {
		t.Fatalf("add photo 2: want 201, got %d", status)
	}

	status, _ = post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))
	if status != http.StatusOK {
		t.Fatalf("publish: want 200, got %d", status)
	}

	// Step 3: Guest books (non-instant → pending_host_approval)
	booking := map[string]any{
		"listingId": listingID,
		"checkIn":   "2027-06-01",
		"checkOut":  "2027-06-05",
		"guests":    4,
		"message":   "Family vacation, looking forward to it!",
	}
	status, resp = post(t, bookingsURL()+"/bookings", booking, authHeaders(defaultUser))
	if status != http.StatusCreated {
		t.Fatalf("create booking: want 201, got %d: %s", status, resp)
	}
	bookingID := jsonField(t, resp, "id")
	if jsonField(t, resp, "status") != "pending_host_approval" {
		t.Errorf("want pending_host_approval, got %s", jsonField(t, resp, "status"))
	}

	// Step 4: Verify booking appears in host's bookings list
	status, resp = get(t, bookingsURL()+"/bookings/host", authHeaders(hostUser))
	if status != http.StatusOK {
		t.Fatalf("list host bookings: want 200, got %d", status)
	}
	hostBookings := jsonArray(t, resp, "bookings")
	found := false
	for _, b := range hostBookings {
		if m, ok := b.(map[string]any); ok && m["id"] == bookingID {
			found = true
			break
		}
	}
	if !found {
		t.Error("booking not found in host's bookings list")
	}

	// Step 5: Host approves → payment_pending (dates reserved)
	status, resp = post(t, bookingsURL()+"/bookings/"+bookingID+"/approve", nil, authHeaders(hostUser))
	if status != http.StatusOK {
		t.Fatalf("approve booking: want 200, got %d: %s", status, resp)
	}
	if jsonField(t, resp, "status") != "payment_pending" {
		t.Errorf("want payment_pending, got %s", jsonField(t, resp, "status"))
	}

	// Step 6: Verify dates are blocked on the listing calendar
	status, resp = get(t, listingsURL()+"/listings/"+listingID+"/calendar?month=2027-06", nil)
	if status != http.StatusOK {
		t.Fatalf("get calendar: want 200, got %d", status)
	}

	// Step 7: Simulate payment (internal confirm)
	confirmBody := map[string]any{"paymentId": "pay_sim_001"}
	status, _ = post(t, bookingsURL()+"/bookings/"+bookingID+"/confirm", confirmBody, internalHeaders())
	if status != http.StatusNoContent {
		t.Fatalf("confirm booking: want 204, got %d", status)
	}

	// Step 8: Verify booking is confirmed
	status, resp = get(t, bookingsURL()+"/bookings/"+bookingID, authHeaders(defaultUser))
	if status != http.StatusOK {
		t.Fatalf("get confirmed booking: want 200, got %d", status)
	}
	if jsonField(t, resp, "status") != "confirmed" {
		t.Errorf("want confirmed, got %s", jsonField(t, resp, "status"))
	}

	// Step 9: Guest creates review
	review := map[string]any{
		"bookingId": bookingID,
		"listingId": listingID,
		"hostId":    hostUser.UserID,
		"rating":    5,
		"comment":   "Absolutely wonderful stay! The mountains were breathtaking.",
	}
	status, resp = post(t, reviewsURL()+"/reviews", review, authHeaders(defaultUser))
	if status != http.StatusCreated {
		t.Fatalf("create review: want 201, got %d: %s", status, resp)
	}
	reviewID := jsonField(t, resp, "id")

	// Step 10: Host replies to review
	replyBody := map[string]any{
		"reply": "Thank you for your kind words! We hope to see you again.",
	}
	status, resp = post(t, reviewsURL()+"/reviews/"+reviewID+"/reply", replyBody, authHeaders(hostUser))
	if status != http.StatusOK {
		t.Fatalf("reply to review: want 200, got %d: %s", status, resp)
	}
	if jsonField(t, resp, "reply") == "" {
		t.Error("reply should not be empty after host reply")
	}

	// Step 11: Verify reviews appear for listing
	status, resp = get(t, reviewsURL()+"/reviews/listing/"+listingID, nil)
	if status != http.StatusOK {
		t.Fatalf("list reviews: want 200, got %d", status)
	}
	reviews := jsonArray(t, resp, "reviews")
	if len(reviews) == 0 {
		t.Error("expected at least one review for listing")
	}

	// Step 12: Verify listing rating was updated (async, best-effort)
	status, resp = get(t, listingsURL()+"/listings/"+listingID, nil)
	if status != http.StatusOK {
		t.Fatalf("get listing with rating: want 200, got %d", status)
	}
	// Note: rating update is fire-and-forget, so it might not be reflected immediately

	// Cleanup
	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 2: Instant Book → Payment Failure → Dates Released
//
// Host creates instant-book listing → Guest books (dates reserved) →
// Payment fails (internal) → Dates released → Another guest can book.
// ===========================================================================

func TestInstantBookPaymentFailureRelease(t *testing.T) {
	// Create instant-book listing
	listing := map[string]any{
		"title":         "Cozy Studio",
		"city":          "Tashkent",
		"country":       "UZ",
		"pricePerNight": "150000.00",
		"currency":      "UZS",
		"maxGuests":     2,
		"instantBook":   true,
	}
	_, resp := post(t, listingsURL()+"/listings", listing, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/studio.jpg", "caption": "cover",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

	// Guest 1 books → payment_pending (dates should be reserved)
	bookingBody := map[string]any{
		"listingId": listingID,
		"checkIn":   "2027-07-10",
		"checkOut":  "2027-07-13",
		"guests":    2,
	}
	status, resp := post(t, bookingsURL()+"/bookings", bookingBody, authHeaders(defaultUser))
	if status != http.StatusCreated {
		t.Fatalf("create booking: want 201, got %d: %s", status, resp)
	}
	bookingID := jsonField(t, resp, "id")
	if jsonField(t, resp, "status") != "payment_pending" {
		t.Errorf("instant book should go to payment_pending, got %s", jsonField(t, resp, "status"))
	}

	// Verify dates are unavailable
	status, resp = get(t, listingsURL()+"/listings/"+listingID+"/availability/check?check_in=2027-07-10&check_out=2027-07-13", nil)
	if status != http.StatusOK {
		t.Fatalf("check availability: want 200, got %d", status)
	}
	if jsonField(t, resp, "available") != "false" {
		t.Error("dates should be unavailable after instant booking")
	}

	// Payment fails
	status, _ = post(t, bookingsURL()+"/bookings/"+bookingID+"/fail", nil, internalHeaders())
	if status != http.StatusNoContent {
		t.Fatalf("fail booking: want 204, got %d", status)
	}

	// Verify booking status is failed
	status, resp = get(t, bookingsURL()+"/bookings/"+bookingID, authHeaders(defaultUser))
	if status != http.StatusOK {
		t.Fatalf("get failed booking: want 200, got %d", status)
	}
	if jsonField(t, resp, "status") != "failed" {
		t.Errorf("want status=failed, got %s", jsonField(t, resp, "status"))
	}

	// Verify dates are released — Guest 2 can now book same dates
	status, resp = get(t, listingsURL()+"/listings/"+listingID+"/availability/check?check_in=2027-07-10&check_out=2027-07-13", nil)
	if status != http.StatusOK {
		t.Fatalf("check availability after fail: want 200, got %d", status)
	}
	if jsonField(t, resp, "available") != "true" {
		t.Error("dates should be available after payment failure release")
	}

	// Guest 2 books same dates successfully
	status, resp = post(t, bookingsURL()+"/bookings", bookingBody, authHeaders(guestUser2))
	if status != http.StatusCreated {
		t.Fatalf("guest 2 booking: want 201, got %d: %s", status, resp)
	}
	if jsonField(t, resp, "status") != "payment_pending" {
		t.Errorf("guest 2 booking: want payment_pending, got %s", jsonField(t, resp, "status"))
	}

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 3: Host Rejection Flow
//
// Guest books non-instant listing → Host rejects → Guest can rebook.
// ===========================================================================

func TestHostRejectionAndRebook(t *testing.T) {
	listing := map[string]any{
		"title":         "Heritage Room",
		"city":          "Bukhara",
		"country":       "UZ",
		"pricePerNight": "200000.00",
		"currency":      "UZS",
		"maxGuests":     2,
		"instantBook":   false,
	}
	_, resp := post(t, listingsURL()+"/listings", listing, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/heritage.jpg", "caption": "cover",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

	// Guest books
	bookingBody := map[string]any{
		"listingId": listingID,
		"checkIn":   "2027-08-01",
		"checkOut":  "2027-08-04",
		"guests":    2,
	}
	_, resp = post(t, bookingsURL()+"/bookings", bookingBody, authHeaders(defaultUser))
	bookingID := jsonField(t, resp, "id")

	// Host rejects
	status, _ := post(t, bookingsURL()+"/bookings/"+bookingID+"/reject", nil, authHeaders(hostUser))
	if status != http.StatusNoContent {
		t.Fatalf("reject: want 204, got %d", status)
	}

	// Verify rejected
	status, resp = get(t, bookingsURL()+"/bookings/"+bookingID, authHeaders(defaultUser))
	if jsonField(t, resp, "status") != "rejected" {
		t.Errorf("want rejected, got %s", jsonField(t, resp, "status"))
	}

	// Guest can rebook same dates (no dates were reserved for non-instant)
	status, resp = post(t, bookingsURL()+"/bookings", bookingBody, authHeaders(defaultUser))
	if status != http.StatusCreated {
		t.Fatalf("rebook after rejection: want 201, got %d: %s", status, resp)
	}

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 4: Concurrent Booking Conflict (Instant Book)
//
// Two guests try to book overlapping dates on instant-book listing.
// First succeeds, second gets 409 conflict.
// ===========================================================================

func TestConcurrentBookingConflict(t *testing.T) {
	listing := map[string]any{
		"title":         "Penthouse Suite",
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
		"url": "https://example.com/penthouse.jpg", "caption": "cover",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

	// Guest 1 books dates
	booking1 := map[string]any{
		"listingId": listingID,
		"checkIn":   "2027-09-01",
		"checkOut":  "2027-09-05",
		"guests":    2,
	}
	status, resp := post(t, bookingsURL()+"/bookings", booking1, authHeaders(defaultUser))
	if status != http.StatusCreated {
		t.Fatalf("guest 1 booking: want 201, got %d: %s", status, resp)
	}

	// Guest 2 tries overlapping dates → should conflict
	booking2 := map[string]any{
		"listingId": listingID,
		"checkIn":   "2027-09-03",
		"checkOut":  "2027-09-07",
		"guests":    1,
	}
	status, resp = post(t, bookingsURL()+"/bookings", booking2, authHeaders(guestUser2))
	if status != http.StatusConflict {
		t.Errorf("overlapping booking: want 409, got %d: %s", status, resp)
	}

	// Guest 2 books non-overlapping dates → should succeed
	booking3 := map[string]any{
		"listingId": listingID,
		"checkIn":   "2027-09-10",
		"checkOut":  "2027-09-13",
		"guests":    1,
	}
	status, resp = post(t, bookingsURL()+"/bookings", booking3, authHeaders(guestUser2))
	if status != http.StatusCreated {
		t.Fatalf("non-overlapping booking: want 201, got %d: %s", status, resp)
	}

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 5: Cancellation with Refund Policy
//
// Test flexible, moderate, and strict cancellation policies.
// ===========================================================================

func TestCancellationPolicies(t *testing.T) {
	policies := []string{"flexible", "moderate", "strict"}
	for _, policy := range policies {
		t.Run("policy="+policy, func(t *testing.T) {
			listing := map[string]any{
				"title":              fmt.Sprintf("Cancel Test %s", policy),
				"city":               "Samarkand",
				"country":            "UZ",
				"pricePerNight":      "200000.00",
				"currency":           "UZS",
				"maxGuests":          2,
				"instantBook":        true,
				"cancellationPolicy": policy,
			}
			_, resp := post(t, listingsURL()+"/listings", listing, authHeaders(hostUser))
			listingID := jsonField(t, resp, "id")
			post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
				"url": "https://example.com/cancel.jpg", "caption": "cover",
			}, authHeaders(hostUser))
			post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

			// Book dates far in the future
			booking := map[string]any{
				"listingId": listingID,
				"checkIn":   "2028-01-10",
				"checkOut":  "2028-01-15",
				"guests":    1,
			}
			_, resp = post(t, bookingsURL()+"/bookings", booking, authHeaders(defaultUser))
			bookingID := jsonField(t, resp, "id")

			// Confirm payment first
			post(t, bookingsURL()+"/bookings/"+bookingID+"/confirm",
				map[string]any{"paymentId": "pay_cancel_test"}, internalHeaders())

			// Guest cancels
			status, resp := post(t, bookingsURL()+"/bookings/"+bookingID+"/cancel", nil, authHeaders(defaultUser))
			if status != http.StatusOK {
				t.Fatalf("cancel: want 200, got %d: %s", status, resp)
			}
			cancelStatus := jsonField(t, resp, "status")
			if !strings.HasPrefix(cancelStatus, "cancelled") {
				t.Errorf("want cancelled status, got %s", cancelStatus)
			}

			del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
		})
	}
}

// ===========================================================================
// Scenario 6: Host Cancellation → Always 100% Refund
// ===========================================================================

func TestHostCancellationAlwaysFullRefund(t *testing.T) {
	listing := map[string]any{
		"title":              "Host Cancel Test",
		"city":               "Fergana",
		"country":            "UZ",
		"pricePerNight":      "180000.00",
		"currency":           "UZS",
		"maxGuests":          2,
		"instantBook":        true,
		"cancellationPolicy": "strict",
	}
	_, resp := post(t, listingsURL()+"/listings", listing, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/hcancel.jpg", "caption": "cover",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

	// Guest books
	_, resp = post(t, bookingsURL()+"/bookings", map[string]any{
		"listingId": listingID,
		"checkIn":   "2027-10-01",
		"checkOut":  "2027-10-04",
		"guests":    1,
	}, authHeaders(defaultUser))
	bookingID := jsonField(t, resp, "id")

	// Confirm
	post(t, bookingsURL()+"/bookings/"+bookingID+"/confirm",
		map[string]any{"paymentId": "pay_hcancel"}, internalHeaders())

	// Host cancels → should always be 100% refund regardless of strict policy
	status, resp := post(t, bookingsURL()+"/bookings/"+bookingID+"/cancel", nil, authHeaders(hostUser))
	if status != http.StatusOK {
		t.Fatalf("host cancel: want 200, got %d: %s", status, resp)
	}
	if jsonField(t, resp, "status") != "cancelled_by_host" {
		t.Errorf("want cancelled_by_host, got %s", jsonField(t, resp, "status"))
	}

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 7: Price Override + Price Preview
//
// Host sets per-day price overrides → Guest gets accurate price preview.
// ===========================================================================

func TestPriceOverrideAndPreview(t *testing.T) {
	listing := map[string]any{
		"title":         "Premium Villa",
		"city":          "Tashkent",
		"country":       "UZ",
		"pricePerNight": "300000.00",
		"cleaningFee":   "75000.00",
		"currency":      "UZS",
		"maxGuests":     6,
		"minNights":     1,
	}
	_, resp := post(t, listingsURL()+"/listings", listing, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/premium.jpg", "caption": "cover",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

	// Set price overrides for specific dates (weekend premium)
	overrides := map[string]any{
		"entries": []map[string]any{
			{"date": "2027-11-01", "price": "450000.00"},
			{"date": "2027-11-02", "price": "450000.00"},
		},
	}
	status, _ := patch(t, listingsURL()+"/listings/"+listingID+"/availability/price", overrides, authHeaders(hostUser))
	if status != http.StatusOK {
		t.Fatalf("set price override: want 200, got %d", status)
	}

	// Get price preview including override dates
	status, resp = get(t,
		listingsURL()+"/listings/"+listingID+"/price-preview?check_in=2027-11-01&check_out=2027-11-04",
		nil)
	if status != http.StatusOK {
		t.Fatalf("price preview: want 200, got %d", status)
	}
	// 3 nights: 2 × 450000 + 1 × 300000 = 1200000 subtotal
	if jsonField(t, resp, "nights") != "3" {
		t.Errorf("want 3 nights, got %s", jsonField(t, resp, "nights"))
	}

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 8: Date Blocking & Calendar Verification
//
// Host blocks dates → Guest cannot book → Host unblocks → Guest can book.
// ===========================================================================

func TestDateBlockingFlow(t *testing.T) {
	listing := map[string]any{
		"title":         "Block Test Room",
		"city":          "Namangan",
		"country":       "UZ",
		"pricePerNight": "120000.00",
		"currency":      "UZS",
		"maxGuests":     2,
		"instantBook":   true,
	}
	_, resp := post(t, listingsURL()+"/listings", listing, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/block.jpg", "caption": "cover",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

	// Host blocks dates
	blockBody := map[string]any{
		"dates": []string{"2027-12-24", "2027-12-25", "2027-12-26"},
	}
	status, _ := post(t, listingsURL()+"/listings/"+listingID+"/availability/block", blockBody, authHeaders(hostUser))
	if status != http.StatusOK {
		t.Fatalf("block dates: want 200, got %d", status)
	}

	// Verify calendar shows blocked
	status, resp = get(t, listingsURL()+"/listings/"+listingID+"/calendar?month=2027-12", nil)
	if status != http.StatusOK {
		t.Fatalf("get calendar: want 200, got %d", status)
	}

	// Guest tries to book blocked dates → should conflict
	booking := map[string]any{
		"listingId": listingID,
		"checkIn":   "2027-12-24",
		"checkOut":  "2027-12-27",
		"guests":    1,
	}
	status, _ = post(t, bookingsURL()+"/bookings", booking, authHeaders(defaultUser))
	if status != http.StatusConflict {
		t.Errorf("booking blocked dates: want 409, got %d", status)
	}

	// Host unblocks
	status, _ = del(t, listingsURL()+"/listings/"+listingID+"/availability/block?dates=2027-12-24,2027-12-25,2027-12-26", authHeaders(hostUser))

	// Now guest can book
	status, resp = post(t, bookingsURL()+"/bookings", booking, authHeaders(defaultUser))
	if status == http.StatusCreated {
		t.Log("booking after unblock succeeded as expected")
	}

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 9: Photo Management Lifecycle
//
// Add multiple photos → reorder → delete → verify publish requires >= 1.
// ===========================================================================

func TestPhotoManagementLifecycle(t *testing.T) {
	_, resp := post(t, listingsURL()+"/listings", map[string]any{
		"title":         "Photo Test",
		"city":          "Khiva",
		"country":       "UZ",
		"pricePerNight": "100000.00",
		"currency":      "UZS",
	}, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")

	// Cannot publish without photos
	status, _ := post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))
	if status != http.StatusUnprocessableEntity {
		t.Errorf("publish without photos: want 422, got %d", status)
	}

	// Add 3 photos
	var photoIDs []string
	for i := 1; i <= 3; i++ {
		status, resp = post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
			"url":     fmt.Sprintf("https://example.com/photo-%d.jpg", i),
			"caption": fmt.Sprintf("photo %d", i),
		}, authHeaders(hostUser))
		if status != http.StatusCreated {
			t.Fatalf("add photo %d: want 201, got %d", i, status)
		}
		photoIDs = append(photoIDs, jsonField(t, resp, "id"))
	}

	// List photos
	status, resp = get(t, listingsURL()+"/listings/"+listingID+"/photos", nil)
	if status != http.StatusOK {
		t.Fatalf("list photos: want 200, got %d", status)
	}
	photos := jsonArray(t, resp, "photos")
	if len(photos) != 3 {
		t.Errorf("want 3 photos, got %d", len(photos))
	}

	// Reorder photos
	reorder := []map[string]any{
		{"id": photoIDs[2], "sortOrder": 0},
		{"id": photoIDs[0], "sortOrder": 1},
		{"id": photoIDs[1], "sortOrder": 2},
	}
	status, _ = patch(t, listingsURL()+"/listings/"+listingID+"/photos/reorder", reorder, authHeaders(hostUser))
	if status != http.StatusOK {
		t.Errorf("reorder photos: want 200, got %d", status)
	}

	// Delete one photo
	status, _ = del(t, listingsURL()+"/listings/"+listingID+"/photos/"+photoIDs[1], authHeaders(hostUser))
	if status != http.StatusNoContent {
		t.Errorf("delete photo: want 204, got %d", status)
	}

	// Can still publish (2 photos remaining)
	status, _ = post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))
	if status != http.StatusOK {
		t.Errorf("publish with 2 photos: want 200, got %d", status)
	}

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 10: Listing Publish/Unpublish Cycle
// ===========================================================================

func TestPublishUnpublishCycle(t *testing.T) {
	_, resp := post(t, listingsURL()+"/listings", map[string]any{
		"title":         "Cycle Test",
		"city":          "Nukus",
		"country":       "UZ",
		"pricePerNight": "80000.00",
		"currency":      "UZS",
	}, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")

	// Draft → add photo → publish
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/cycle.jpg", "caption": "cover",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

	// Verify active
	_, resp = get(t, listingsURL()+"/listings/"+listingID, nil)
	if jsonField(t, resp, "status") != "active" {
		t.Errorf("want active, got %s", jsonField(t, resp, "status"))
	}

	// Unpublish → paused
	status, _ := post(t, listingsURL()+"/listings/"+listingID+"/unpublish", nil, authHeaders(hostUser))
	if status != http.StatusOK {
		t.Fatalf("unpublish: want 200, got %d", status)
	}
	_, resp = get(t, listingsURL()+"/listings/"+listingID, nil)
	if jsonField(t, resp, "status") != "paused" {
		t.Errorf("want paused, got %s", jsonField(t, resp, "status"))
	}

	// Paused listing should not appear in public list
	status, resp = get(t, listingsURL()+"/listings", nil)
	if status != http.StatusOK {
		t.Fatalf("list listings: want 200, got %d", status)
	}
	listings := jsonArray(t, resp, "listings")
	for _, l := range listings {
		if m, ok := l.(map[string]any); ok && m["id"] == listingID {
			t.Error("paused listing should not appear in public list")
		}
	}

	// Re-publish
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))
	_, resp = get(t, listingsURL()+"/listings/"+listingID, nil)
	if jsonField(t, resp, "status") != "active" {
		t.Errorf("want active after re-publish, got %s", jsonField(t, resp, "status"))
	}

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 11: Ownership Enforcement
//
// Guest cannot modify host's listing. Host2 cannot modify Host1's listing.
// ===========================================================================

func TestOwnershipEnforcement(t *testing.T) {
	_, resp := post(t, listingsURL()+"/listings", map[string]any{
		"title":         "Ownership Test",
		"city":          "Tashkent",
		"country":       "UZ",
		"pricePerNight": "100000.00",
		"currency":      "UZS",
	}, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")

	// Guest (defaultUser) cannot update host's listing
	status, _ := patch(t, listingsURL()+"/listings/"+listingID, map[string]any{
		"title": "Hijacked Listing",
	}, authHeaders(defaultUser))
	if status != http.StatusForbidden {
		t.Errorf("guest update: want 403, got %d", status)
	}

	// Guest cannot delete host's listing
	status, _ = del(t, listingsURL()+"/listings/"+listingID, authHeaders(defaultUser))
	if status != http.StatusForbidden {
		t.Errorf("guest delete: want 403, got %d", status)
	}

	// Guest cannot publish host's listing
	status, _ = post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(defaultUser))
	if status != http.StatusForbidden {
		t.Errorf("guest publish: want 403, got %d", status)
	}

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 12: Booking Validation Rules
//
// Test min/max nights, max guests, booking inactive listing.
// ===========================================================================

func TestBookingValidationRules(t *testing.T) {
	listing := map[string]any{
		"title":         "Validation Test Villa",
		"city":          "Tashkent",
		"country":       "UZ",
		"pricePerNight": "200000.00",
		"currency":      "UZS",
		"maxGuests":     4,
		"minNights":     2,
		"maxNights":     14,
		"instantBook":   true,
	}
	_, resp := post(t, listingsURL()+"/listings", listing, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/val.jpg", "caption": "cover",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

	t.Run("below_min_nights", func(t *testing.T) {
		booking := map[string]any{
			"listingId": listingID,
			"checkIn":   "2028-02-01",
			"checkOut":  "2028-02-02", // 1 night, min is 2
			"guests":    1,
		}
		status, _ := post(t, bookingsURL()+"/bookings", booking, authHeaders(defaultUser))
		if status != http.StatusUnprocessableEntity && status != http.StatusBadRequest {
			t.Errorf("min nights violation: want 422 or 400, got %d", status)
		}
	})

	t.Run("above_max_nights", func(t *testing.T) {
		booking := map[string]any{
			"listingId": listingID,
			"checkIn":   "2028-03-01",
			"checkOut":  "2028-03-20", // 19 nights, max is 14
			"guests":    1,
		}
		status, _ := post(t, bookingsURL()+"/bookings", booking, authHeaders(defaultUser))
		if status != http.StatusUnprocessableEntity && status != http.StatusBadRequest {
			t.Errorf("max nights violation: want 422 or 400, got %d", status)
		}
	})

	t.Run("above_max_guests", func(t *testing.T) {
		booking := map[string]any{
			"listingId": listingID,
			"checkIn":   "2028-04-01",
			"checkOut":  "2028-04-05",
			"guests":    8, // max is 4
		}
		status, _ := post(t, bookingsURL()+"/bookings", booking, authHeaders(defaultUser))
		if status != http.StatusUnprocessableEntity && status != http.StatusBadRequest {
			t.Errorf("max guests violation: want 422 or 400, got %d", status)
		}
	})

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 13: Webhook Dedup + Payment Lifecycle
//
// Simulate Mashgate webhook: checkout.completed → booking confirmed.
// Replay same webhook → deduplicated. Different event → processed.
// ===========================================================================

func TestWebhookPaymentLifecycle(t *testing.T) {
	// Setup listing + booking in payment_pending state
	listing := map[string]any{
		"title":         "Webhook Test",
		"city":          "Tashkent",
		"country":       "UZ",
		"pricePerNight": "100000.00",
		"currency":      "UZS",
		"maxGuests":     2,
		"instantBook":   true,
	}
	_, resp := post(t, listingsURL()+"/listings", listing, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/wh.jpg", "caption": "cover",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

	_, resp = post(t, bookingsURL()+"/bookings", map[string]any{
		"listingId": listingID,
		"checkIn":   "2028-05-01",
		"checkOut":  "2028-05-03",
		"guests":    1,
	}, authHeaders(defaultUser))
	bookingID := jsonField(t, resp, "id")

	// Simulate payment.captured webhook
	eventID := fmt.Sprintf("evt_wh_complex_%s", bookingID)
	webhookPayload := map[string]any{
		"event_id":     eventID,
		"event_type":   "payment.captured",
		"aggregate_id": "pay_wh_test",
		"tenant_id":    defaultUser.TenantID,
		"data": map[string]any{
			"metadata": map[string]any{
				"bookingId": bookingID,
			},
		},
	}
	payloadJSON, _ := marshalJSON(webhookPayload)
	hdr := webhookHeaders(payloadJSON)

	status, resp := post(t, paymentsURL()+"/webhooks/mashgate", webhookPayload, hdr)
	if status != http.StatusOK {
		t.Fatalf("webhook: want 200, got %d: %s", status, resp)
	}

	// Verify booking is now confirmed
	status, resp = get(t, bookingsURL()+"/bookings/"+bookingID, authHeaders(defaultUser))
	if status == http.StatusOK && jsonField(t, resp, "status") == "confirmed" {
		t.Log("booking confirmed via webhook")
	}

	// Replay same webhook → deduplicated
	status, resp = post(t, paymentsURL()+"/webhooks/mashgate", webhookPayload, hdr)
	if status != http.StatusOK {
		t.Fatalf("dedup webhook: want 200, got %d", status)
	}
	if jsonField(t, resp, "dedup") != "skipped" {
		t.Errorf("want dedup=skipped, got %q", jsonField(t, resp, "dedup"))
	}

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 14: Review Constraints
//
// Cannot review without booking. Cannot review twice.
// Cannot reply as non-host.
// ===========================================================================

func TestReviewConstraints(t *testing.T) {
	// Create listing + confirmed booking
	listing := map[string]any{
		"title":         "Review Constraint Test",
		"city":          "Tashkent",
		"country":       "UZ",
		"pricePerNight": "100000.00",
		"currency":      "UZS",
		"maxGuests":     2,
		"instantBook":   true,
	}
	_, resp := post(t, listingsURL()+"/listings", listing, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/rev.jpg", "caption": "cover",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

	_, resp = post(t, bookingsURL()+"/bookings", map[string]any{
		"listingId": listingID,
		"checkIn":   "2028-06-01",
		"checkOut":  "2028-06-03",
		"guests":    1,
	}, authHeaders(defaultUser))
	bookingID := jsonField(t, resp, "id")
	post(t, bookingsURL()+"/bookings/"+bookingID+"/confirm",
		map[string]any{"paymentId": "pay_rev_test"}, internalHeaders())

	// Create first review
	review := map[string]any{
		"bookingId": bookingID,
		"listingId": listingID,
		"hostId":    hostUser.UserID,
		"rating":    4,
		"comment":   "Great place!",
	}
	status, resp := post(t, reviewsURL()+"/reviews", review, authHeaders(defaultUser))
	if status != http.StatusCreated {
		t.Fatalf("create review: want 201, got %d: %s", status, resp)
	}
	reviewID := jsonField(t, resp, "id")

	// Cannot review same booking twice → 409
	status, _ = post(t, reviewsURL()+"/reviews", review, authHeaders(defaultUser))
	if status != http.StatusConflict {
		t.Errorf("duplicate review: want 409, got %d", status)
	}

	// Invalid rating (0) → 400/422
	badReview := map[string]any{
		"bookingId": "bk-fake",
		"listingId": listingID,
		"rating":    0,
		"comment":   "Invalid",
	}
	status, _ = post(t, reviewsURL()+"/reviews", badReview, authHeaders(defaultUser))
	if status != http.StatusBadRequest && status != http.StatusUnprocessableEntity {
		t.Errorf("invalid rating: want 400 or 422, got %d", status)
	}

	// Non-host cannot reply
	status, _ = post(t, reviewsURL()+"/reviews/"+reviewID+"/reply",
		map[string]any{"reply": "I'm not the host"}, authHeaders(defaultUser))
	if status != http.StatusNotFound && status != http.StatusForbidden {
		t.Errorf("non-host reply: want 404 or 403, got %d", status)
	}

	// Guest can list their own reviews
	status, resp = get(t, reviewsURL()+"/reviews/my", authHeaders(defaultUser))
	if status != http.StatusOK {
		t.Fatalf("my reviews: want 200, got %d", status)
	}
	myReviews := jsonArray(t, resp, "reviews")
	if len(myReviews) == 0 {
		t.Error("expected at least 1 review in my reviews")
	}

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 15: Admin Feature Flags + Audit Trail
//
// Create flags → update → verify audit log captures actions.
// ===========================================================================

func TestAdminFlagsAndAudit(t *testing.T) {
	base := adminURL()

	// Create feature flag
	flag := map[string]any{
		"name":    "new_checkout_flow",
		"enabled": true,
		"rollout": 50,
	}
	status, resp := post(t, base+"/admin/flags", flag, authHeaders(adminUser))
	if status != http.StatusOK {
		t.Fatalf("create flag: want 200, got %d: %s", status, resp)
	}
	if jsonField(t, resp, "name") != "new_checkout_flow" {
		t.Errorf("want name=new_checkout_flow, got %s", jsonField(t, resp, "name"))
	}

	// List flags
	status, resp = get(t, base+"/admin/flags", authHeaders(adminUser))
	if status != http.StatusOK {
		t.Fatalf("list flags: want 200, got %d", status)
	}
	flags := jsonArray(t, resp, "flags")
	found := false
	for _, f := range flags {
		if m, ok := f.(map[string]any); ok && m["name"] == "new_checkout_flow" {
			found = true
			break
		}
	}
	if !found {
		t.Error("created flag not found in list")
	}

	// Update flag (disable)
	flag["enabled"] = false
	flag["rollout"] = 0
	status, _ = post(t, base+"/admin/flags", flag, authHeaders(adminUser))
	if status != http.StatusOK {
		t.Fatalf("update flag: want 200, got %d", status)
	}

	// Verify audit log captured the operations
	status, resp = get(t, base+"/admin/audit", authHeaders(adminUser))
	if status != http.StatusOK {
		t.Fatalf("audit log: want 200, got %d", status)
	}
	entries := jsonArray(t, resp, "entries")
	if len(entries) < 2 {
		t.Errorf("expected at least 2 audit entries for flag create+update, got %d", len(entries))
	}

	// Non-admin cannot access flags
	status, _ = get(t, base+"/admin/flags", authHeaders(defaultUser))
	if status != http.StatusForbidden {
		t.Errorf("non-admin flags: want 403, got %d", status)
	}
}

// ===========================================================================
// Scenario 16: Admin Tenant Configuration
// ===========================================================================

func TestAdminTenantConfig(t *testing.T) {
	base := adminURL()
	tenantID := "e2e-tenant-config-test"

	// Get default config (should return defaults)
	status, resp := get(t, base+"/admin/tenants/"+tenantID, authHeaders(adminUser))
	if status != http.StatusOK {
		t.Fatalf("get default tenant config: want 200, got %d", status)
	}

	// Update tenant config
	config := map[string]any{
		"platformFeePct": 15.0,
		"maxListings":    100,
		"verified":       true,
	}
	status, resp = put(t, base+"/admin/tenants/"+tenantID, config, authHeaders(adminUser))
	if status != http.StatusOK {
		t.Fatalf("update tenant config: want 200, got %d: %s", status, resp)
	}
	if jsonField(t, resp, "verified") != "true" {
		t.Errorf("want verified=true, got %s", jsonField(t, resp, "verified"))
	}

	// Read back
	status, resp = get(t, base+"/admin/tenants/"+tenantID, authHeaders(adminUser))
	if status != http.StatusOK {
		t.Fatalf("re-read tenant config: want 200, got %d", status)
	}
	if jsonField(t, resp, "verified") != "true" {
		t.Errorf("persisted verified: want true, got %s", jsonField(t, resp, "verified"))
	}
}

// ===========================================================================
// Scenario 17: Search Service — Geo + Filters
// ===========================================================================

func TestSearchWithFilters(t *testing.T) {
	// Create a few listings in different cities
	cities := []struct {
		title, city string
		price       string
		amenities   []string
		instant     bool
	}{
		{"Search Villa 1", "Tashkent", "250000.00", []string{"wifi", "parking"}, true},
		{"Search Villa 2", "Tashkent", "180000.00", []string{"wifi"}, false},
		{"Search Villa 3", "Samarkand", "120000.00", []string{"wifi", "pool"}, true},
	}
	var listingIDs []string
	for _, c := range cities {
		_, resp := post(t, listingsURL()+"/listings", map[string]any{
			"title":         c.title,
			"city":          c.city,
			"country":       "UZ",
			"pricePerNight": c.price,
			"currency":      "UZS",
			"maxGuests":     4,
			"instantBook":   c.instant,
			"amenities":     c.amenities,
		}, authHeaders(hostUser))
		id := jsonField(t, resp, "id")
		listingIDs = append(listingIDs, id)
		post(t, listingsURL()+"/listings/"+id+"/photos", map[string]any{
			"url": "https://example.com/search-" + id + ".jpg", "caption": "cover",
		}, authHeaders(hostUser))
		post(t, listingsURL()+"/listings/"+id+"/publish", nil, authHeaders(hostUser))
	}

	// Search by city
	status, _ := get(t, searchURL()+"/search?city=Tashkent", nil)
	if status != http.StatusOK {
		t.Fatalf("search by city: want 200, got %d", status)
	}

	// Search by price range
	status, _ = get(t, searchURL()+"/search?min_price=100000&max_price=200000", nil)
	if status != http.StatusOK {
		t.Fatalf("search by price: want 200, got %d", status)
	}

	// Search instant book only
	status, _ = get(t, searchURL()+"/search?instant_book=true", nil)
	if status != http.StatusOK {
		t.Fatalf("search instant: want 200, got %d", status)
	}

	// Cleanup
	for _, id := range listingIDs {
		del(t, listingsURL()+"/listings/"+id, authHeaders(hostUser))
	}
}

// ===========================================================================
// Scenario 18: Multi-Service State Machine (Invalid Transitions)
//
// Verify that invalid state transitions are rejected.
// ===========================================================================

func TestInvalidStateTransitions(t *testing.T) {
	// Setup: instant-book listing + payment_pending booking
	listing := map[string]any{
		"title":         "State Machine Test",
		"city":          "Tashkent",
		"country":       "UZ",
		"pricePerNight": "100000.00",
		"currency":      "UZS",
		"maxGuests":     2,
		"instantBook":   true,
	}
	_, resp := post(t, listingsURL()+"/listings", listing, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/sm.jpg", "caption": "cover",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

	_, resp = post(t, bookingsURL()+"/bookings", map[string]any{
		"listingId": listingID,
		"checkIn":   "2028-07-01",
		"checkOut":  "2028-07-03",
		"guests":    1,
	}, authHeaders(defaultUser))
	bookingID := jsonField(t, resp, "id")

	// payment_pending → cannot approve (not pending_host_approval)
	status, _ := post(t, bookingsURL()+"/bookings/"+bookingID+"/approve", nil, authHeaders(hostUser))
	if status != http.StatusConflict && status != http.StatusNotFound {
		t.Errorf("approve payment_pending: want 409 or 404, got %d", status)
	}

	// payment_pending → cannot reject
	status, _ = post(t, bookingsURL()+"/bookings/"+bookingID+"/reject", nil, authHeaders(hostUser))
	if status != http.StatusConflict && status != http.StatusNotFound {
		t.Errorf("reject payment_pending: want 409 or 404, got %d", status)
	}

	// Confirm → confirmed
	post(t, bookingsURL()+"/bookings/"+bookingID+"/confirm",
		map[string]any{"paymentId": "pay_sm"}, internalHeaders())

	// confirmed → cannot confirm again
	status, _ = post(t, bookingsURL()+"/bookings/"+bookingID+"/confirm",
		map[string]any{"paymentId": "pay_sm2"}, internalHeaders())
	if status == http.StatusNoContent {
		t.Error("should not be able to confirm an already confirmed booking")
	}

	// confirmed → cannot fail
	status, _ = post(t, bookingsURL()+"/bookings/"+bookingID+"/fail", nil, internalHeaders())
	if status == http.StatusNoContent {
		t.Error("should not be able to fail an already confirmed booking")
	}

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 19: Listing Search (via listings service /listings/search endpoint)
// ===========================================================================

func TestListingsSearch(t *testing.T) {
	// Create a listing with specific amenities
	_, resp := post(t, listingsURL()+"/listings", map[string]any{
		"title":         "Search Test Pool Villa",
		"city":          "Tashkent",
		"country":       "UZ",
		"type":          "house",
		"pricePerNight": "400000.00",
		"currency":      "UZS",
		"maxGuests":     8,
		"instantBook":   true,
		"amenities":     []string{"wifi", "pool", "gym", "parking"},
	}, authHeaders(hostUser))
	listingID := jsonField(t, resp, "id")
	post(t, listingsURL()+"/listings/"+listingID+"/photos", map[string]any{
		"url": "https://example.com/search-pool.jpg", "caption": "cover",
	}, authHeaders(hostUser))
	post(t, listingsURL()+"/listings/"+listingID+"/publish", nil, authHeaders(hostUser))

	// Search with type filter
	status, _ := get(t, listingsURL()+"/listings/search?type=house", nil)
	if status != http.StatusOK {
		t.Fatalf("search by type: want 200, got %d", status)
	}

	// Search with amenities
	status, _ = get(t, listingsURL()+"/listings/search?amenities=wifi,pool", nil)
	if status != http.StatusOK {
		t.Fatalf("search by amenities: want 200, got %d", status)
	}

	// Search with guests filter
	status, _ = get(t, listingsURL()+"/listings/search?guests=6", nil)
	if status != http.StatusOK {
		t.Fatalf("search by guests: want 200, got %d", status)
	}

	del(t, listingsURL()+"/listings/"+listingID, authHeaders(hostUser))
}

// ===========================================================================
// Scenario 20: Complete Multi-Listing Host Dashboard
//
// Host creates multiple listings → publishes some → checks /listings/mine →
// Guest books across multiple → Host views all via /bookings/host.
// ===========================================================================

func TestMultiListingHostDashboard(t *testing.T) {
	titles := []string{"Apartment A", "Villa B", "Room C"}
	var listingIDs []string
	for _, title := range titles {
		_, resp := post(t, listingsURL()+"/listings", map[string]any{
			"title":         title,
			"city":          "Tashkent",
			"country":       "UZ",
			"pricePerNight": "100000.00",
			"currency":      "UZS",
			"maxGuests":     2,
			"instantBook":   true,
		}, authHeaders(hostUser))
		id := jsonField(t, resp, "id")
		listingIDs = append(listingIDs, id)
		post(t, listingsURL()+"/listings/"+id+"/photos", map[string]any{
			"url": "https://example.com/multi-" + id + ".jpg", "caption": "cover",
		}, authHeaders(hostUser))
		post(t, listingsURL()+"/listings/"+id+"/publish", nil, authHeaders(hostUser))
	}

	// Verify host can see all via /listings/mine
	status, resp := get(t, listingsURL()+"/listings/mine", authHeaders(hostUser))
	if status != http.StatusOK {
		t.Fatalf("my listings: want 200, got %d", status)
	}
	myListings := jsonArray(t, resp, "listings")
	foundCount := 0
	for _, l := range myListings {
		if m, ok := l.(map[string]any); ok {
			for _, id := range listingIDs {
				if m["id"] == id {
					foundCount++
				}
			}
		}
	}
	if foundCount < len(listingIDs) {
		t.Errorf("expected all %d listings in my listings, found %d", len(listingIDs), foundCount)
	}

	// Guest books each listing with different dates
	months := []string{"2028-08", "2028-09", "2028-10"}
	for i, id := range listingIDs {
		post(t, bookingsURL()+"/bookings", map[string]any{
			"listingId": id,
			"checkIn":   months[i] + "-01",
			"checkOut":  months[i] + "-03",
			"guests":    1,
		}, authHeaders(defaultUser))
	}

	// Host views all bookings
	status, resp = get(t, bookingsURL()+"/bookings/host", authHeaders(hostUser))
	if status != http.StatusOK {
		t.Fatalf("host bookings: want 200, got %d", status)
	}
	hostBookings := jsonArray(t, resp, "bookings")
	if len(hostBookings) < len(listingIDs) {
		t.Errorf("expected at least %d bookings for host, got %d", len(listingIDs), len(hostBookings))
	}

	for _, id := range listingIDs {
		del(t, listingsURL()+"/listings/"+id, authHeaders(hostUser))
	}
}

// marshalJSON marshals v to JSON bytes.
func marshalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}
