package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	zistauth "github.com/saidmashhud/zist/internal/auth"
	"github.com/saidmashhud/zist/services/admin/handler"
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
	r.Use(otelhttp.NewMiddleware("zist-admin"))
	r.Use(zistauth.Middleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})

	// All admin routes require authentication (scope enforcement is in handlers).
	adminMW := chi.Chain(zistauth.RequireAuth)

	r.Route("/admin", func(r chi.Router) {
		r.With(adminMW...).Get("/flags", s.h.ListFlags)
		r.With(adminMW...).Post("/flags", s.h.UpsertFlag)

		r.With(adminMW...).Get("/audit", s.h.ListAudit)

		r.With(adminMW...).Get("/tenants/{id}", s.h.GetTenantConfig)
		r.With(adminMW...).Put("/tenants/{id}", s.h.UpsertTenantConfig)
	})

	return r
}
