package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/MrKriegler/go-insurance/pkg/problem"
)

// SimpleAPIKey provides basic API key authentication for demo purposes.
// In production, use JWT/OAuth2 with proper token validation.
func SimpleAPIKey(apiKey string) func(http.Handler) http.Handler {
	apiKeyBytes := []byte(apiKey)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health checks and swagger
			if strings.HasPrefix(r.URL.Path, "/health") ||
				strings.HasPrefix(r.URL.Path, "/readyz") ||
				strings.HasPrefix(r.URL.Path, "/swagger") {
				next.ServeHTTP(w, r)
				return
			}

			// Check X-API-Key header
			key := r.Header.Get("X-API-Key")
			if key == "" {
				// Also check Authorization: Bearer <key>
				auth := r.Header.Get("Authorization")
				if strings.HasPrefix(auth, "Bearer ") {
					key = strings.TrimPrefix(auth, "Bearer ")
				}
			}

			// Constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(key), apiKeyBytes) != 1 {
				problem.Write(w, http.StatusUnauthorized, "Unauthorized", "Invalid or missing API key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
