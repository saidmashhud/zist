// Package main implements the Zist payments service.
//
// Integrates with Mashgate SDK to:
//   - POST /checkout         → creates a Mashgate checkout session and returns the redirect URL
//   - POST /webhooks/mashgate → receives Mashgate webhook events (signature-verified)
//
// Runs on port 8003 (PAYMENTS_PORT env).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	mashgate "github.com/saidmashhud/mashgate/packages/sdk/go"
	zistauth "github.com/saidmashhud/zist/internal/auth"
)

type server struct {
	mg            *mashgate.Client
	webhookSecret string
	bookingsURL   string
}

func main() {
	port          := getenv("PAYMENTS_PORT", "8003")
	mashgateURL   := getenv("MASHGATE_URL", "http://localhost:9661")
	mashgateKey   := getenv("MASHGATE_API_KEY", "")
	webhookSecret := getenv("MASHGATE_WEBHOOK_SECRET", "")
	bookingsURL   := getenv("BOOKINGS_URL", "http://bookings:8002")

	mg := mashgate.New(mashgateURL, mashgateKey)
	s := &server{mg: mg, webhookSecret: webhookSecret, bookingsURL: bookingsURL}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(zistauth.Middleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	r.With(zistauth.RequireScope("zist.payments.create")).Post("/checkout", s.createCheckout)
	r.Post("/webhooks/mashgate", s.handleWebhook)

	slog.Info("Payments service starting", "port", port, "mashgate", mashgateURL, "bookings", bookingsURL)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		slog.Error("payments service failed", "err", err)
		os.Exit(1)
	}
}

// createCheckout creates a Mashgate checkout session and returns the hosted checkout URL.
//
// Request body:
//
//	{
//	  "listingId": "...",
//	  "bookingId": "...",
//	  "amount":    "150000.00",
//	  "currency":  "UZS",
//	  "successUrl": "https://zist.app/bookings/{id}/success",
//	  "cancelUrl":  "https://zist.app/bookings/{id}/cancel",
//	  "customerEmail": "guest@example.com"
//	}
//
// Response: { "checkoutUrl": "https://..." }
func (s *server) createCheckout(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Amount == "" || req.Currency == "" {
		writeError(w, http.StatusUnprocessableEntity, "amount and currency are required")
		return
	}

	session, err := s.mg.CreateCheckout(r.Context(), mashgate.CreateCheckoutRequest{
		TotalAmount: mashgate.Money{
			Amount:   req.Amount,
			Currency: req.Currency,
		},
		Items: []mashgate.LineItem{
			{
				Name:     fmt.Sprintf("Zist booking %s", req.BookingID),
				Quantity: 1,
				UnitPrice: mashgate.Money{
					Amount:   req.Amount,
					Currency: req.Currency,
				},
			},
		},
		CustomerEmail: req.CustomerEmail,
		SuccessURL:    req.SuccessURL,
		CancelURL:     req.CancelURL,
		Metadata: map[string]string{
			"bookingId": req.BookingID,
			"listingId": req.ListingID,
		},
	})
	if err != nil {
		slog.Error("Mashgate CreateCheckout failed", "err", err)
		writeError(w, http.StatusBadGateway, "payment gateway error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"sessionId":   session.SessionID,
		"checkoutUrl": session.CheckoutURL,
	})
}

// handleWebhook receives Mashgate webhook events.
//
// Verifies the signature and dispatches events to handlers.
// Always returns 200 to acknowledge receipt.
func (s *server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	timestamp := r.Header.Get("X-Webhook-Timestamp")
	signature := r.Header.Get("X-Webhook-Signature")

	if s.webhookSecret != "" {
		if err := mashgate.VerifySignature(s.webhookSecret, timestamp, string(body), signature); err != nil {
			slog.Warn("webhook signature verification failed", "err", err)
			writeError(w, http.StatusUnauthorized, "invalid webhook signature")
			return
		}
	}

	event, err := mashgate.ParseEvent(body)
	if err != nil {
		slog.Error("failed to parse webhook event", "err", err)
		writeError(w, http.StatusBadRequest, "invalid event payload")
		return
	}

	slog.Info("received webhook event",
		"eventId", event.EventID,
		"eventType", event.EventType,
		"tenantId", event.TenantID,
		"aggregateId", event.AggregateID,
	)

	switch event.EventType {
	case mashgate.EventPaymentCaptured:
		slog.Info("payment captured", "paymentId", event.AggregateID)
		var payload struct {
			Metadata map[string]string `json:"metadata"`
		}
		if err := json.Unmarshal(event.Data, &payload); err == nil {
			if bookingID := payload.Metadata["bookingId"]; bookingID != "" {
				if err := s.confirmBooking(r.Context(), bookingID); err != nil {
					slog.Error("failed to confirm booking", "bookingId", bookingID, "err", err)
				} else {
					slog.Info("booking confirmed", "bookingId", bookingID)
				}
			}
		}

	case mashgate.EventPaymentFailed, mashgate.EventPaymentCaptureFailed:
		slog.Warn("payment failed", "paymentId", event.AggregateID)

	case mashgate.EventCheckoutCompleted:
		slog.Info("checkout completed", "sessionId", event.AggregateID)

	case mashgate.EventCheckoutExpired:
		slog.Warn("checkout expired", "sessionId", event.AggregateID)

	default:
		slog.Debug("unhandled event type", "eventType", event.EventType)
	}

	// Always acknowledge
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// confirmBooking calls the bookings service to mark a booking as confirmed.
func (s *server) confirmBooking(ctx context.Context, bookingID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.bookingsURL+"/bookings/"+bookingID+"/confirm", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("bookings service returned %d", resp.StatusCode)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
