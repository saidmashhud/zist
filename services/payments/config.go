package main

import (
	"github.com/saidmashhud/zist/internal/httputil"
)

// Config holds all environment-driven configuration for the payments service.
type Config struct {
	Port          string
	MashgateURL   string
	MashgateKey   string
	WebhookSecret string
	BookingsURL   string
	InternalToken string
	DatabaseURL   string

	// Service JWT auth (optional; if set, JWT is preferred over InternalToken)
	AuthServiceURL  string
	AuthServiceKey  string
	ServiceName     string
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() *Config {
	return &Config{
		Port:          httputil.Getenv("PAYMENTS_PORT", "8003"),
		MashgateURL:   httputil.Getenv("MASHGATE_URL", "http://localhost:9661"),
		MashgateKey:   httputil.Getenv("MASHGATE_API_KEY", ""),
		WebhookSecret: httputil.Getenv("MASHGATE_WEBHOOK_SECRET", ""),
		BookingsURL:   httputil.Getenv("BOOKINGS_URL", "http://bookings:8002"),
		InternalToken: httputil.Getenv("INTERNAL_TOKEN", ""),
		DatabaseURL:   httputil.Getenv("DATABASE_URL", ""),

		AuthServiceURL: httputil.Getenv("AUTH_SERVICE_URL", ""),
		AuthServiceKey: httputil.Getenv("AUTH_SERVICE_KEY", ""),
		ServiceName:    httputil.Getenv("SERVICE_NAME", "zist-payments"),
	}
}
