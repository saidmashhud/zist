package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	zistauth "github.com/saidmashhud/zist/internal/auth"
	"github.com/saidmashhud/zist/services/payments/handler"
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
	r.Use(otelhttp.NewMiddleware("zist-payments"))
	r.Use(zistauth.Middleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})

	internal := zistauth.RequireInternalToken(s.cfg.InternalToken)

	r.With(zistauth.RequireScope("zist.payments.create")).Post("/checkout", s.h.CreateCheckout)
	r.With(internal).Post("/refund", s.h.CreateRefund)
	r.Post("/webhooks/mashgate", s.h.HandleWebhook)

	return r
}
