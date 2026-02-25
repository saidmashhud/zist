package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	zistauth "github.com/saidmashhud/zist/internal/auth"
	"github.com/saidmashhud/zist/services/search/handler"
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
	r.Use(otelhttp.NewMiddleware("zist-search"))
	r.Use(zistauth.Middleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})

	internal := chi.Chain(zistauth.RequireServiceAuth(s.cfg.InternalToken, nil))

	r.Route("/search", func(r chi.Router) {
		r.Get("/", s.h.Search)

		// Internal: update listing location (called by listings service on create/update)
		r.With(internal...).Put("/locations/{id}", s.h.UpdateLocation)
	})

	return r
}
