// Package httputil provides shared HTTP helpers used by all Zist services.
package httputil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
)

// WriteJSON serialises v as JSON and writes it with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// WriteError writes a JSON body of the form {"error":"<msg>"} with status.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// Getenv returns the value of the environment variable key,
// or fallback if the variable is unset or empty.
func Getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// GetenvFloat returns the float64 value of key, or fallback.
func GetenvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f == 0 {
		return fallback
	}
	return f
}

// GetenvInt returns the int value of key, or fallback.
func GetenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// OrDefault returns s if non-empty, otherwise def.
func OrDefault(s, def string) string {
	if s != "" {
		return s
	}
	return def
}

// Sprintf is a convenience re-export of fmt.Sprintf for use in templates.
var Sprintf = fmt.Sprintf
