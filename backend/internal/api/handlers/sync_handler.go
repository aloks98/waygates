package handlers

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// SyncHandler handles sync-related HTTP requests
type SyncHandler struct {
	syncService service.SyncServiceInterface
	logger      *zap.Logger
}

// NewSyncHandler creates a new sync handler
func NewSyncHandler(syncService service.SyncServiceInterface, logger *zap.Logger) *SyncHandler {
	return &SyncHandler{
		syncService: syncService,
		logger:      logger,
	}
}

// GetStatus returns the current sync status
func (h *SyncHandler) GetStatus(w http.ResponseWriter, _ *http.Request) {
	status := h.syncService.GetStatus()
	utils.Success(w, status, "Sync status retrieved successfully")
}

// Trigger manually triggers a full sync
func (h *SyncHandler) Trigger(w http.ResponseWriter, _ *http.Request) {
	if err := h.syncService.FullSync(); err != nil {
		if h.logger != nil {
			h.logger.Error("Manual sync trigger failed", zap.Error(err))
		}
		utils.InternalError(w, "Sync failed: "+err.Error())
		return
	}

	if h.logger != nil {
		h.logger.Info("Manual sync trigger completed successfully")
	}

	status := h.syncService.GetStatus()
	utils.Success(w, status, "Sync completed successfully")
}
