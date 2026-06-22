package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// ConfigPreviewHandler handles generated Caddy config preview requests.
type ConfigPreviewHandler struct {
	sync   *service.SyncService
	logger *zap.Logger
}

// NewConfigPreviewHandler creates a new ConfigPreviewHandler.
func NewConfigPreviewHandler(sync *service.SyncService, logger *zap.Logger) *ConfigPreviewHandler {
	return &ConfigPreviewHandler{
		sync:   sync,
		logger: logger,
	}
}

// GetFull returns the full generated Caddy JSON config without writing or reloading.
func (h *ConfigPreviewHandler) GetFull(w http.ResponseWriter, _ *http.Request) {
	cfg, err := h.sync.GenerateConfigJSON()
	if err != nil {
		h.logger.Error("generate config failed", zap.Error(err))
		utils.InternalError(w, "failed to generate Caddy config")
		return
	}
	utils.Success(w, cfg, "Caddy config generated successfully")
}

// GetForProxy returns the generated Caddy JSON config for a single proxy
// (including its ACL handlers), without writing or reloading.
//
// Route parameter: {id} — the proxy ID.
func (h *ConfigPreviewHandler) GetForProxy(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.BadRequest(w, "invalid proxy id", nil)
		return
	}
	cfg, err := h.sync.GenerateProxyConfigJSON(id)
	if errors.Is(err, service.ErrProxyNotFound) {
		utils.NotFound(w, "proxy not found")
		return
	}
	if err != nil {
		h.logger.Error("generate proxy config failed", zap.Error(err))
		utils.InternalError(w, "failed to generate proxy config")
		return
	}
	utils.Success(w, cfg, "Proxy config generated successfully")
}
