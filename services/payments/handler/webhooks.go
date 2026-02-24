package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	mashgate "github.com/saidmashhud/mashgate/packages/sdk-go"
	"github.com/saidmashhud/zist/internal/httputil"
)

// HandleWebhook receives Mashgate webhook events, verifies the signature,
// deduplicates, and dispatches to the appropriate handler.
// POST /webhooks/mashgate
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	// Canonical HookLine/Mashgate headers.
	timestamp := r.Header.Get("x-gp-timestamp")
	signature := r.Header.Get("x-gp-signature")
	// Backward-compatible fallback for legacy emitters.
	if timestamp == "" {
		timestamp = r.Header.Get("X-Webhook-Timestamp")
	}
	if signature == "" {
		signature = r.Header.Get("X-Webhook-Signature")
	}

	if h.WebhookSecret != "" {
		if err := mashgate.VerifySignature(h.WebhookSecret, timestamp, string(body), signature); err != nil {
			slog.Warn("webhook signature verification failed", "err", err)
			httputil.WriteError(w, http.StatusUnauthorized, "invalid webhook signature")
			return
		}
	}

	event, err := mashgate.ParseEvent(body)
	if err != nil {
		slog.Error("failed to parse webhook event", "err", err)
		httputil.WriteError(w, http.StatusBadRequest, "invalid event payload")
		return
	}

	slog.Info("received webhook event",
		"eventId", event.EventID,
		"eventType", event.EventType,
		"aggregateId", event.AggregateID,
	)

	if h.Dedup.Check(event.EventID) {
		slog.Info("duplicate webhook, skipping", "eventId", event.EventID)
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "dedup": "skipped"})
		return
	}
	if event.TenantID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing tenant_id in webhook event")
		return
	}

	switch event.EventType {
	case mashgate.EventPaymentCaptured:
		h.onPaymentCaptured(r, *event)
	case mashgate.EventPaymentFailed, mashgate.EventPaymentCaptureFailed:
		h.onPaymentFailed(r, *event)
	case mashgate.EventRefundSettled:
		slog.Info("refund settled", "paymentId", event.AggregateID)
	case mashgate.EventRefundFailed:
		slog.Warn("refund failed", "paymentId", event.AggregateID)
	case mashgate.EventCheckoutCompleted:
		slog.Info("checkout completed", "sessionId", event.AggregateID)
	case mashgate.EventCheckoutExpired:
		slog.Warn("checkout expired", "sessionId", event.AggregateID)

	default:
		slog.Debug("unhandled event type", "eventType", event.EventType)
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) onPaymentCaptured(r *http.Request, event mashgate.WebhookEvent) {
	slog.Info("payment captured", "paymentId", event.AggregateID)
	bookingID := extractBookingID(event)
	if bookingID == "" {
		return
	}
	if err := h.Bookings.ConfirmBooking(r.Context(), event.TenantID, bookingID, event.AggregateID); err != nil {
		slog.Error("failed to confirm booking", "bookingId", bookingID, "err", err)
	} else {
		slog.Info("booking confirmed", "bookingId", bookingID)
	}
}

func (h *Handler) onPaymentFailed(r *http.Request, event mashgate.WebhookEvent) {
	slog.Warn("payment failed", "paymentId", event.AggregateID)
	bookingID := extractBookingID(event)
	if bookingID == "" {
		return
	}
	if err := h.Bookings.FailBooking(r.Context(), event.TenantID, bookingID); err != nil {
		slog.Error("failed to mark booking as failed", "bookingId", bookingID, "err", err)
	} else {
		slog.Info("booking marked as failed", "bookingId", bookingID)
	}
}

func extractBookingID(event mashgate.WebhookEvent) string {
	var payload struct {
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return ""
	}
	return payload.Metadata["bookingId"]
}
