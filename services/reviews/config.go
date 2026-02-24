package main

import "github.com/saidmashhud/zist/internal/httputil"

// Config holds environment-driven configuration for the reviews service.
type Config struct {
	Port          string
	DatabaseURL   string
	ListingsURL   string
	InternalToken string
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() *Config {
	return &Config{
		Port:          httputil.Getenv("REVIEWS_PORT", "8004"),
		DatabaseURL:   httputil.Getenv("DATABASE_URL", "postgres://dev:dev@db:5432/zist?sslmode=disable"),
		ListingsURL:   httputil.Getenv("LISTINGS_SERVICE_URL", "http://listings:8001"),
		InternalToken: httputil.Getenv("INTERNAL_TOKEN", ""),
	}
}
