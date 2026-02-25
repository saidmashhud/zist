package main

import httputil "github.com/saidmashhud/zist/internal/httputil"

// Config holds configuration for the search service.
type Config struct {
	Port          string
	DatabaseURL   string
	InternalToken string
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() *Config {
	return &Config{
		Port:          httputil.Getenv("SEARCH_PORT", "8006"),
		DatabaseURL:   httputil.Getenv("DATABASE_URL", "postgres://dev:dev@db:5432/zist?sslmode=disable"),
		InternalToken: httputil.Getenv("INTERNAL_TOKEN", ""),
	}
}
