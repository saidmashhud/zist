package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// notifyClient sends SMS and email notifications via mgNotify.
type notifyClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func newNotifyClient(baseURL, apiKey string) *notifyClient {
	return &notifyClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// NotifyUser sends an event-based notification to a user identified by userID.
// mgNotify resolves the user's preferred contact (phone/email) via mgID.
// Fire-and-forget: errors are logged only.
func (c *notifyClient) NotifyUser(ctx context.Context, userID, eventType, message string) {
	if c.baseURL == "" || userID == "" {
		return
	}
	body, _ := json.Marshal(map[string]string{
		"user_id":    userID,
		"event_type": eventType,
		"message":    message,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/notify/user", c.baseURL), bytes.NewReader(body))
	if err != nil {
		slog.Warn("notify: failed to build request", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		slog.Warn("notify: send failed", "err", err, "event_type", eventType)
		return
	}
	resp.Body.Close()
	slog.Info("notify: notification sent", "status", resp.Status, "event_type", eventType)
}

// SendSMS sends an SMS directly to a phone number. Fire-and-forget.
func (c *notifyClient) SendSMS(ctx context.Context, phone, message string) {
	if c.baseURL == "" || phone == "" {
		return
	}
	body, _ := json.Marshal(map[string]string{
		"to":      phone,
		"message": message,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/notify/sms", c.baseURL), bytes.NewReader(body))
	if err != nil {
		slog.Warn("notify: failed to build SMS request", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		slog.Warn("notify: SMS send failed", "err", err)
		return
	}
	resp.Body.Close()
	slog.Info("notify: SMS sent", "status", resp.Status)
}

// SendEmail sends an email directly. Fire-and-forget.
func (c *notifyClient) SendEmail(ctx context.Context, to, subject, htmlBody string) {
	if c.baseURL == "" || to == "" {
		return
	}
	payload, _ := json.Marshal(map[string]string{
		"to":      to,
		"subject": subject,
		"body":    htmlBody,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/notify/email", c.baseURL), bytes.NewReader(payload))
	if err != nil {
		slog.Warn("notify: failed to build email request", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		slog.Warn("notify: email send failed", "err", err)
		return
	}
	resp.Body.Close()
	slog.Info("notify: email sent", "status", resp.Status)
}
