package handlers

import (
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/database"
	"github.com/aloks98/waygates/backend/internal/utils"
	"github.com/aloks98/waygates/backend/internal/version"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	startTime time.Time
	db        *gorm.DB
}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
	}
}

// NewHealthHandlerWithDB creates a new health handler with database reference
func NewHealthHandlerWithDB(db *gorm.DB) *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
		db:        db,
	}
}

// HealthCheck returns the health status of the service
func (h *HealthHandler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	uptime := time.Since(h.startTime)

	// Check database health
	dbStatus := "healthy"
	if h.db != nil {
		if err := database.PingDB(h.db); err != nil {
			dbStatus = "unhealthy"
		}
	} else {
		if err := database.Ping(); err != nil {
			dbStatus = "unhealthy"
		}
	}

	// Overall status is unhealthy if any component is unhealthy
	overallStatus := "healthy"
	if dbStatus == "unhealthy" {
		overallStatus = "degraded"
	}

	response := map[string]interface{}{
		"status":  overallStatus,
		"service": "caddy-manager-backend",
		"version": version.String(),
		"uptime":  uptime.String(),
		"time":    time.Now().UTC().Format(time.RFC3339),
		"components": map[string]string{
			"database": dbStatus,
		},
	}

	utils.Success(w, response, "")
}
