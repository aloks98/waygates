package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/aloks98/waygates/backend/internal/caddy"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// StatusHandler handles status-related HTTP requests
type StatusHandler struct {
	reloader *caddy.Reloader
	userRepo *repository.UserRepository
}

// NewStatusHandler creates a new status handler
func NewStatusHandler(reloader *caddy.Reloader, userRepo *repository.UserRepository) *StatusHandler {
	return &StatusHandler{
		reloader: reloader,
		userRepo: userRepo,
	}
}

// GetStatus returns the status of the application
func (h *StatusHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	// Check Caddy status by testing connection
	caddyStatus := "healthy"
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.reloader.TestConnection(ctx); err != nil {
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
