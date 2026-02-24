// Package handler contains HTTP handlers for the payments service.
package handler

import (
	mashgate "github.com/saidmashhud/mashgate/packages/sdk-go"
)

// DedupChecker abstracts the dedup store (in-memory or PostgreSQL-backed).
type DedupChecker interface {
	Check(eventID string) bool
}

// Handler holds shared dependencies for all payments HTTP handlers.
type Handler struct {
	MG            *mashgate.Client
	WebhookSecret string
	Bookings      *BookingsClient
	Dedup         DedupChecker
}

// New returns a Handler with the given dependencies.
func New(mg *mashgate.Client, webhookSecret string, bc *BookingsClient, dc DedupChecker) *Handler {
	return &Handler{
		MG:            mg,
		WebhookSecret: webhookSecret,
		Bookings:      bc,
		Dedup:         dc,
	}
}
