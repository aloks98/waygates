package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestResolveCORSOptions_Wildcard ensures a wildcard origin opens all origins
// with credentials disabled (spec-compliant), rather than silently denying
// every cross-origin request.
func TestResolveCORSOptions_Wildcard(t *testing.T) {
	origins, allowCredentials := resolveCORSOptions([]string{"*"})

	assert.Equal(t, []string{"*"}, origins)
	assert.False(t, allowCredentials, "wildcard origin must not be combined with credentials")
}

// TestResolveCORSOptions_Explicit ensures explicit origins are passed through
// with credentials enabled.
func TestResolveCORSOptions_Explicit(t *testing.T) {
	origins, allowCredentials := resolveCORSOptions([]string{"https://app.example.com"})

	assert.Equal(t, []string{"https://app.example.com"}, origins)
	assert.True(t, allowCredentials)
}

// TestAuthRateLimiter_RejectsBeyondThreshold ensures the auth rate limiter
// rejects requests from a single IP once the per-window threshold is exceeded.
func TestAuthRateLimiter_RejectsBeyondThreshold(t *testing.T) {
	h := authRateLimiter()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	var lastCode int
	for i := 0; i < authRateLimitRequests+1; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		req.RemoteAddr = "203.0.113.9:1234"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		lastCode = rec.Code
	}

	assert.Equal(t, http.StatusTooManyRequests, lastCode,
		"requests beyond the limit should be rejected with 429")
}
