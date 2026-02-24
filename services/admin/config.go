package main

import "github.com/saidmashhud/zist/internal/httputil"

// Config holds environment-driven configuration for the admin service.
type Config struct {
	Port          string
	DatabaseURL   string
	InternalToken string
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() *Config {
	return &Config{
		Port:          httputil.Getenv("ADMIN_PORT", "8005"),
		DatabaseURL:   httputil.Getenv("DATABASE_URL", "postgres://dev:dev@db:5432/zist?sslmode=disable"),
		InternalToken: httputil.Getenv("INTERNAL_TOKEN", ""),
	}
}
