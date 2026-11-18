package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aloks98/waygates/backend/internal/api/middleware"
	"github.com/aloks98/waygates/backend/internal/service"
)

type streamRecorder struct {
	*httptest.ResponseRecorder
	deadline    time.Time
	deadlineSet bool
	cancel      context.CancelFunc
}

func (r *streamRecorder) SetWriteDeadline(deadline time.Time) error {
	r.deadlineSet = true
	r.deadline = deadline
	return nil
}
func (r *streamRecorder) Flush() { r.ResponseRecorder.Flush(); r.cancel() }

func TestLogStreamClearsDeadlineAndFlushesBeforeLogs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	recorder := &streamRecorder{ResponseRecorder: httptest.NewRecorder(), cancel: cancel}
	handler := NewCaddyLogsHandler(service.NewCaddyLogsService(t.TempDir()), nil)
	middleware.RequestLogger(nil)(http.HandlerFunc(handler.Stream)).ServeHTTP(recorder, httptest.NewRequest("GET", "/api/caddy-logs/stream?source=access", nil).WithContext(ctx))
	if !recorder.deadlineSet || !recorder.deadline.IsZero() {
		t.Fatal("stream must clear the normal server write deadline")
	}
	if recorder.Header().Get("Content-Type") != "text/event-stream" || !recorder.Flushed {
		t.Fatal("stream must flush SSE headers before any log arrives")
	}
	if !strings.Contains(recorder.Body.String(), ": connected\n\n") {
		t.Fatal("expected initial connection heartbeat")
	}
}

func TestLogStreamRejectsUnknownSource(t *testing.T) {
	handler := NewCaddyLogsHandler(service.NewCaddyLogsService(t.TempDir()), nil)
	recorder := httptest.NewRecorder()
	handler.Stream(recorder, httptest.NewRequest("GET", "/api/caddy-logs/stream?source=unknown", nil))
	if recorder.Code != 400 {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}
