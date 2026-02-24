package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	mashgate "github.com/saidmashhud/mashgate/packages/sdk-go"
	zistauth "github.com/saidmashhud/zist/internal/auth"
	"github.com/saidmashhud/zist/internal/httputil"
)

// CreateCheckout creates a Mashgate checkout session and returns the hosted checkout URL.
// POST /checkout
func (h *Handler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	principal := zistauth.FromContext(r.Context())
	if principal == nil || principal.TenantID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		ListingID     string `json:"listingId"`
		BookingID     string `json:"bookingId"`
		Amount        string `json:"amount"`
		Currency      string `json:"currency"`
		SuccessURL    string `json:"successUrl"`
		CancelURL     string `json:"cancelUrl"`
		CustomerEmail string `json:"customerEmail"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Amount == "" || req.Currency == "" {
		httputil.WriteError(w, http.StatusUnprocessableEntity, "amount and currency are required")
		return
	}

	session, err := h.MG.CreateCheckout(r.Context(), mashgate.CreateCheckoutRequest{
		TotalAmount: mashgate.Money{Amount: req.Amount, Currency: req.Currency},
		Items: []mashgate.LineItem{
			{
				Name:      fmt.Sprintf("Zist booking %s", req.BookingID),
				Quantity:  1,
				UnitPrice: mashgate.Money{Amount: req.Amount, Currency: req.Currency},
			},
		},
		CustomerEmail:  req.CustomerEmail,
		SuccessURL:     req.SuccessURL,
		CancelURL:      req.CancelURL,
		IdempotencyKey: req.BookingID,
		Metadata: map[string]string{
			"bookingId": req.BookingID,
			"listingId": req.ListingID,
		},
	})
	if err != nil {
		slog.Error("Mashgate CreateCheckout failed", "err", err)
		httputil.WriteError(w, http.StatusBadGateway, "payment gateway error")
		return
	}

	if req.BookingID != "" {
		if err := h.Bookings.SetCheckoutID(r.Context(), principal.TenantID, req.BookingID, session.SessionID); err != nil {
			slog.Warn("failed to store checkout_id on booking", "bookingId", req.BookingID, "err", err)
		}
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]string{
		"sessionId":   session.SessionID,
		"checkoutUrl": session.CheckoutURL,
	})
}
