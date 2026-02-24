package main

import httputil "github.com/saidmashhud/zist/internal/httputil"

// Config holds all configuration for the listings service, loaded from environment variables.
type Config struct {
	Port                string
	DatabaseURL         string
	InternalToken       string
	PlatformFeeGuestPct float64
	MgLogsURL           string // mgLogs analytics endpoint (optional)
	MgFlagsURL          string // mgFlags feature flags endpoint (optional)
	MashgateAPIKey      string // shared API key for mgLogs + mgFlags
}

// LoadConfig reads configuration from environment variables with sensible defaults.
func LoadConfig() *Config {
	return &Config{
		Port:                httputil.Getenv("LISTINGS_PORT", "8001"),
		DatabaseURL:         httputil.Getenv("DATABASE_URL", "postgres://dev:dev@db:5432/zist?sslmode=disable"),
		InternalToken:       httputil.Getenv("INTERNAL_TOKEN", ""),
		PlatformFeeGuestPct: httputil.GetenvFloat("PLATFORM_FEE_GUEST_PCT", 12.0),
		MgLogsURL:           httputil.Getenv("MGLOGS_URL", ""),
		MgFlagsURL:          httputil.Getenv("MGFLAGS_URL", ""),
		MashgateAPIKey:      httputil.Getenv("MASHGATE_API_KEY", ""),
	}
}
