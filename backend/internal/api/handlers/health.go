package handlers

import (
	"net/http"
	"time"

	"github.com/aloks98/homelab-proxy/backend/internal/utils"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	startTime time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
	}
}

// HealthCheck returns the health status of the service
func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime)

	response := map[string]interface{}{
		"status":  "healthy",
		"service": "caddy-manager-backend",
		"version": "1.0.0",
		"uptime":  uptime.String(),
		"time":    time.Now().UTC().Format(time.RFC3339),
	}

	utils.Success(w, response, "")
}
