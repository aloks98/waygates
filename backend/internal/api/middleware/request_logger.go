package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// RequestLogger returns a middleware that logs each request using structured zap logging
func RequestLogger(logger *zap.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	reqLogger := logger.Named("http")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap the response writer to capture status code and bytes written
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Get request ID if available
			reqID := middleware.GetReqID(r.Context())

			defer func() {
				// Calculate duration
				duration := time.Since(start)

				// Log the request
				reqLogger.Info("request",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("query", r.URL.RawQuery),
					zap.Int("status", ww.Status()),
					zap.Int("bytes", ww.BytesWritten()),
					zap.Duration("duration", duration),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("request_id", reqID),
					zap.String("user_agent", r.UserAgent()),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// RequestLoggerConfig holds configuration options for the request logger.
type RequestLoggerConfig struct {
	// SkipPaths are paths to skip logging (e.g., health checks)
	SkipPaths []string
	// LogLevel is the log level for requests (default: Info)
	LogLevel zap.AtomicLevel
}

// RequestLoggerWithSkip returns a request logger that skips certain paths
func RequestLoggerWithSkip(logger *zap.Logger, skipPaths ...string) func(http.Handler) http.Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	reqLogger := logger.Named("http")

	// Create a map for faster path lookup
	skipMap := make(map[string]bool)
	for _, path := range skipPaths {
		skipMap[path] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap the response writer to capture status code and bytes written
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Get request ID if available
			reqID := middleware.GetReqID(r.Context())

			defer func() {
				// Skip logging for specified paths
				if skipMap[r.URL.Path] {
					return
				}

				// Calculate duration
				duration := time.Since(start)
				status := ww.Status()

				// Choose log level based on status code
				logFunc := reqLogger.Info
				if status >= 500 {
					logFunc = reqLogger.Error
				} else if status >= 400 {
					logFunc = reqLogger.Warn
				}

				// Log the request
				logFunc("request",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("query", r.URL.RawQuery),
					zap.Int("status", status),
					zap.Int("bytes", ww.BytesWritten()),
					zap.Duration("duration", duration),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("request_id", reqID),
					zap.String("user_agent", r.UserAgent()),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
