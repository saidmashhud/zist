package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	zistauth "github.com/saidmashhud/zist/internal/auth"
	"github.com/saidmashhud/zist/services/listings/handler"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// server wires together the HTTP router and dependencies.
type server struct {
	cfg *Config
	h   *handler.Handler
}

// routes builds and returns the chi router.
func (s *server) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(otelhttp.NewMiddleware("zist-listings"))
	r.Use(zistauth.Middleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})

	hostWrite := chi.Chain(zistauth.RequireAuth, zistauth.RequireScope("zist.listings.manage"))
	internal := chi.Chain(zistauth.RequireInternalToken(s.cfg.InternalToken))

	r.Route("/listings", func(r chi.Router) {
		// Public
		r.Get("/search", s.h.SearchListings)
		r.With(zistauth.RequireAuth).Get("/mine", s.h.ListMyListings)
		r.Get("/", s.h.ListListings)
		r.Get("/{id}", s.h.GetListing)
		r.Get("/{id}/calendar", s.h.GetCalendar)
		r.Get("/{id}/price-preview", s.h.PricePreview)
		r.Get("/{id}/photos", s.h.ListPhotos)
		r.Get("/{id}/availability/check", s.h.CheckAvailability)

		// Host-only
		r.With(hostWrite...).Post("/", s.h.CreateListing)
		r.With(hostWrite...).Put("/{id}", s.h.UpdateListing)
		r.With(hostWrite...).Patch("/{id}", s.h.UpdateListing)
		r.With(hostWrite...).Delete("/{id}", s.h.DeleteListing)
		r.With(hostWrite...).Post("/{id}/publish", s.h.PublishListing)
		r.With(hostWrite...).Post("/{id}/unpublish", s.h.UnpublishListing)
		r.With(hostWrite...).Post("/{id}/photos", s.h.AddPhoto)
		r.With(hostWrite...).Patch("/{id}/photos/reorder", s.h.ReorderPhotos)
		r.With(hostWrite...).Delete("/{id}/photos/{photoId}", s.h.DeletePhoto)
		r.With(hostWrite...).Post("/{id}/availability/block", s.h.BlockDates)
		r.With(hostWrite...).Delete("/{id}/availability/block", s.h.UnblockDates)
		r.With(hostWrite...).Patch("/{id}/availability/price", s.h.SetPriceOverride)

		// Internal (called by bookings service)
		r.With(internal...).Post("/{id}/availability/book", s.h.MarkDatesBooked)
		r.With(internal...).Delete("/{id}/availability/book", s.h.UnmarkDatesBooked)
	})

	return r
}
