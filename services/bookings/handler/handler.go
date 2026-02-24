// Package handler contains HTTP handlers for the bookings service.
package handler

import (
	"github.com/saidmashhud/zist/services/bookings/store"
)

// Handler holds shared dependencies for all bookings HTTP handlers.
type Handler struct {
	Store       *store.Store
	Listings    *ListingsClient
	FeeGuestPct float64 // e.g. 12.0 → 12%
}

// New returns a Handler with the given dependencies.
func New(s *store.Store, lc *ListingsClient, feeGuestPct float64) *Handler {
	return &Handler{Store: s, Listings: lc, FeeGuestPct: feeGuestPct}
}
