package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	zistauth "github.com/saidmashhud/zist/internal/auth"
	"github.com/saidmashhud/zist/services/bookings/handler"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type server struct {
	cfg *Config
	h   *handler.Handler
}

func (s *server) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(otelhttp.NewMiddleware("zist-bookings"))
	r.Use(zistauth.Middleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})

	internal := chi.Chain(zistauth.RequireInternalToken(s.cfg.InternalToken))
	guestAuth := chi.Chain(zistauth.RequireAuth, zistauth.RequireScope("zist.bookings.manage"))
	readAuth := chi.Chain(zistauth.RequireAuth, zistauth.RequireScope("zist.bookings.read"))
	hostAuth := chi.Chain(zistauth.RequireAuth, zistauth.RequireScope("zist.listings.manage"))

	r.Route("/bookings", func(r chi.Router) {
		// Static route before /{id}.
		r.With(hostAuth...).Get("/host", s.h.ListHostBookings)

		r.With(readAuth...).Get("/", s.h.ListBookings)
		r.With(guestAuth...).Post("/", s.h.CreateBooking)

		r.With(readAuth...).Get("/{id}", s.h.GetBooking)
		r.With(zistauth.RequireAuth).Post("/{id}/cancel", s.h.CancelBooking)

		r.With(hostAuth...).Post("/{id}/approve", s.h.ApproveBooking)
		r.With(hostAuth...).Post("/{id}/reject", s.h.RejectBooking)

		r.With(internal...).Post("/{id}/confirm", s.h.ConfirmBooking)
		r.With(internal...).Post("/{id}/fail", s.h.FailBooking)
		r.With(internal...).Put("/{id}/checkout", s.h.SetCheckoutID)
	})

	return r
}
