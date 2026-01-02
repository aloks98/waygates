package middleware

import (
	"net/http"

	"github.com/aloks98/homelab-proxy/backend/internal/utils"
)

// BodyLimit returns a middleware that limits the request body size
func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only limit if there's a body
			if r.Body != nil && r.ContentLength > 0 {
				if r.ContentLength > maxBytes {
					utils.BadRequest(w, "Request body too large", nil)
					return
				}
			}

			// Wrap the body with a limited reader as a safety measure
			// This handles cases where Content-Length is not set or is incorrect
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

			next.ServeHTTP(w, r)
		})
	}
}

// DefaultBodyLimit is 1MB
const DefaultBodyLimit = 1 * 1024 * 1024

// LargeBodyLimit is 10MB (for file uploads if needed)
const LargeBodyLimit = 10 * 1024 * 1024
