package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// CaddyLogsHandler handles Caddy log read requests.
type CaddyLogsHandler struct {
	svc    *service.CaddyLogsService
	logger *zap.Logger
}

// NewCaddyLogsHandler creates a new CaddyLogsHandler.
func NewCaddyLogsHandler(svc *service.CaddyLogsService, logger *zap.Logger) *CaddyLogsHandler {
	return &CaddyLogsHandler{
		svc:    svc,
		logger: logger,
	}
}

// List returns a snapshot of the last N lines from the requested Caddy log source.
//
// Query parameters:
//   - source: "runtime" (default) or "access"
//   - limit:  number of lines to return, default 200, capped at 1000
func (h *CaddyLogsHandler) List(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		source = "runtime"
	}
	if source != "runtime" && source != "access" {
		utils.BadRequest(w, `invalid source: must be "runtime" or "access"`, nil)
		return
	}

	limit := 200
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed < 1 {
			utils.BadRequest(w, "limit must be a positive integer", nil)
			return
		}
		if parsed > 1000 {
			parsed = 1000
		}
		limit = parsed
	}

	msgs, err := h.svc.Snapshot(source, limit)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("failed to read caddy log snapshot",
				zap.String("source", source),
				zap.Int("limit", limit),
				zap.Error(err))
		}
		utils.InternalError(w, "failed to read log snapshot")
		return
	}

	utils.Success(w, msgs, "Caddy logs retrieved successfully")
}

// Stream backfills the last 500 lines from the requested Caddy log source and
// then streams new lines in real time using Server-Sent Events (SSE).
//
// Query parameters:
//   - source: "runtime" (default) or "access"
//
// The SSE stream closes when the client disconnects; the tailer goroutine is
// stopped via r.Context() cancellation — no goroutine leak.
func (h *CaddyLogsHandler) Stream(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		source = "runtime"
	}
	tailer, err := h.svc.Tailer(source)
	if err != nil {
		utils.BadRequest(w, "invalid log source", nil)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	rc := http.NewResponseController(w)
	// This endpoint is long-lived; the server's normal 15-second write deadline
	// must not terminate a healthy event stream.
	_ = rc.SetWriteDeadline(time.Time{})
	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
		return
	}
	if err := rc.Flush(); err != nil {
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	backfill := 500
	if r.URL.Query().Get("backfill") == "0" {
		backfill = 0
	}
	lines := make(chan []byte, 256)

	go func() {
		defer close(lines)
		_ = tailer.Stream(ctx, backfill, lines)
	}()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		case line, ok := <-lines:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", line); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}
}
