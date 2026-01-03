package handlers

import (
	"net/http"

	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// SyncHandler handles sync-related HTTP requests
type SyncHandler struct {
	syncService *service.SyncService
}

// NewSyncHandler creates a new sync handler
func NewSyncHandler(syncService *service.SyncService) *SyncHandler {
	return &SyncHandler{
		syncService: syncService,
	}
}

// GetStatus returns the current sync status
func (h *SyncHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.syncService.GetStatus()
	utils.Success(w, status, "Sync status retrieved successfully")
}

// Trigger manually triggers a full sync
func (h *SyncHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	if err := h.syncService.FullSync(); err != nil {
		utils.InternalError(w, "Sync failed: "+err.Error())
		return
	}

	status := h.syncService.GetStatus()
	utils.Success(w, status, "Sync completed successfully")
}
