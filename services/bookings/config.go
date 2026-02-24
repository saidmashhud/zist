package main

import (
	"github.com/saidmashhud/zist/internal/httputil"
)

// Config holds all environment-driven configuration for the bookings service.
type Config struct {
	Port            string
	DatabaseURL     string
	ListingsURL     string
	InternalToken   string
	FeeGuestPct     float64
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() *Config {
	return &Config{
		Port:          httputil.Getenv("BOOKINGS_PORT", "8002"),
		DatabaseURL:   httputil.Getenv("DATABASE_URL", "postgres://dev:dev@db:5432/zist?sslmode=disable"),
		ListingsURL:   httputil.Getenv("LISTINGS_SERVICE_URL", "http://listings:8001"),
		InternalToken: httputil.Getenv("INTERNAL_TOKEN", ""),
		FeeGuestPct:   httputil.GetenvFloat("PLATFORM_FEE_GUEST_PCT", 12.0),
	}
}
