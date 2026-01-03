package handlers

import (
	"net/http"

	"github.com/aloks98/waygates/backend/internal/caddy"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// StatusHandler handles status-related HTTP requests
type StatusHandler struct {
	caddyClient *caddy.Client
	userRepo    *repository.UserRepository
}

// NewStatusHandler creates a new status handler
func NewStatusHandler(caddyClient *caddy.Client, userRepo *repository.UserRepository) *StatusHandler {
	return &StatusHandler{
		caddyClient: caddyClient,
		userRepo:    userRepo,
	}
}

// GetStatus returns the status of the application
func (h *StatusHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	// Check Caddy status
	caddyErr := h.caddyClient.HealthCheck()
	caddyStatus := "healthy"
	if caddyErr != nil {
		caddyStatus = "unhealthy"
	}

	// Check if any user exists
	userCount, err := h.userRepo.Count()
	if err != nil {
		utils.InternalError(w, "Failed to check user status")
		return
	}
	userSetupComplete := userCount > 0

	response := map[string]interface{}{
		"caddy_status":        caddyStatus,
		"user_setup_complete": userSetupComplete,
	}

	utils.Success(w, response, "Status retrieved successfully")
}
