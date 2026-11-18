package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestTimeoutKeepsStreamingClientContext(t *testing.T) {
	for _, tc := range []struct {
		path, method string
		wantDeadline bool
	}{
		{"/api/proxies", "GET", true},
		{"/api/caddy-logs", "GET", true},
		{"/api/caddy-logs/stream", "POST", true},
		{"/api/caddy-logs/stream?source=access", "GET", false},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			handler := RequestTimeout(time.Minute)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				if _, hasDeadline := r.Context().Deadline(); hasDeadline != tc.wantDeadline {
					t.Fatalf("deadline = %v, want %v", hasDeadline, tc.wantDeadline)
				}
				cancel()
				if r.Context().Err() != context.Canceled {
					t.Fatal("client cancellation must propagate")
				}
			}))
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(tc.method, tc.path, nil).WithContext(ctx))
		})
	}
}
