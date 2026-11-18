package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
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

// PreviewDraft uses the production config builder for an unsaved proxy.
func (h *ConfigPreviewHandler) PreviewDraft(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Proxy       *models.Proxy               `json:"proxy"`
		Assignments []models.ProxyACLAssignment `json:"assignments"`
		L4Proxy     *models.L4Proxy             `json:"l4_proxy"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.BadRequest(w, "invalid proxy preview", nil)
		return
	}
	if (request.Proxy == nil) == (request.L4Proxy == nil) {
		utils.BadRequest(w, "provide one HTTP or TCP / UDP proxy", nil)
		return
	}
	var cfg json.RawMessage
	var err error
	if request.L4Proxy != nil {
		cfg, err = h.sync.GenerateDraftL4ConfigJSON(*request.L4Proxy)
	} else {
		cfg, err = h.sync.GenerateDraftProxyConfigJSON(*request.Proxy, request.Assignments)
	}
	if err != nil {
		utils.BadRequest(w, "unable to generate config: check the proxy and access rules", nil)
		return
	}
	utils.Success(w, cfg, "Proxy config generated successfully")
}
