// Package mashgate provides shared Mashgate client initialization for Zist services.
//
// Usage:
//
//	mg := mashgate.NewFromEnv()
//	session, err := mg.CreateCheckout(ctx, ...)
package mashgate

import (
	"os"

	mg "github.com/saidmashhud/mashgate/packages/sdk/go"
)

// NewFromEnv creates a Mashgate client from environment variables:
//
//	MASHGATE_URL      — default "http://localhost:9661"
//	MASHGATE_API_KEY  — required in production
func NewFromEnv() *mg.Client {
	baseURL := getenv("MASHGATE_URL", "http://localhost:9661")
	apiKey  := getenv("MASHGATE_API_KEY", "")
	return mg.New(baseURL, apiKey)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
