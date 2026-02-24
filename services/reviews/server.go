package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	zistauth "github.com/saidmashhud/zist/internal/auth"
	"github.com/saidmashhud/zist/services/reviews/handler"
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
	r.Use(otelhttp.NewMiddleware("zist-reviews"))
	r.Use(zistauth.Middleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})

	authMW := chi.Chain(zistauth.RequireAuth)

	r.Route("/reviews", func(r chi.Router) {
		// Public: list reviews for a listing
		r.Get("/listing/{id}", s.h.ListReviewsByListing)

		// Authenticated: create review, view own reviews, reply
		r.With(authMW...).Post("/", s.h.CreateReview)
		r.With(authMW...).Get("/my", s.h.ListMyReviews)
		r.With(authMW...).Post("/{id}/reply", s.h.ReplyToReview)
	})

	return r
}
