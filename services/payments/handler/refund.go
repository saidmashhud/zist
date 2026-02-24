package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	mashgate "github.com/saidmashhud/mashgate/packages/sdk-go"
	"github.com/saidmashhud/zist/internal/httputil"
)

// CreateRefund initiates a Mashgate refund for a captured payment.
// POST /refund  (internal token required)
func (h *Handler) CreateRefund(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PaymentID string `json:"paymentId"`
		Amount    string `json:"amount"`
		Currency  string `json:"currency"`
		BookingID string `json:"bookingId"`
		Reason    string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.PaymentID == "" || req.Amount == "" || req.Currency == "" {
		httputil.WriteError(w, http.StatusUnprocessableEntity, "paymentId, amount, and currency are required")
		return
	}

	reason := req.Reason
	if reason == "" {
		reason = "cancellation"
	}

	payment, err := h.MG.RefundPayment(r.Context(), req.PaymentID, mashgate.RefundRequest{
		Amount:         mashgate.Money{Amount: req.Amount, Currency: req.Currency},
		Reason:         reason,
		IdempotencyKey: req.BookingID,
	})
	if err != nil {
		slog.Error("Mashgate RefundPayment failed", "paymentId", req.PaymentID, "err", err)
		httputil.WriteError(w, http.StatusBadGateway, "refund request failed")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]string{
		"paymentId": payment.PaymentID,
		"status":    payment.Status,
	})
}
