package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	chimw "github.com/aloks98/goauth/middleware/chi"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// SettingsHandler handles settings-related HTTP requests
type SettingsHandler struct {
	settingsService service.SettingsServiceInterface
	auditService    service.AuditServiceInterface
	logger          *zap.Logger
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(settingsService service.SettingsServiceInterface, auditService service.AuditServiceInterface, logger *zap.Logger) *SettingsHandler {
	return &SettingsHandler{
		settingsService: settingsService,
		auditService:    auditService,
		logger:          logger,
	}
}

// GetAll returns all settings as a key-value map
func (h *SettingsHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settingsService.GetAll()
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to get all settings", zap.Error(err))
		}
		utils.InternalError(w, "Failed to get settings")
		return
	}

	utils.Success(w, settings, "Settings retrieved successfully")
}

// Get returns a single setting by key
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		utils.BadRequest(w, "Setting key is required", nil)
		return
	}

	value, err := h.settingsService.Get(key)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to get setting",
				zap.String("key", key),
				zap.Error(err))
		}
		utils.NotFound(w, "Setting not found")
		return
	}

	utils.Success(w, map[string]string{"key": key, "value": value}, "Setting retrieved successfully")
}

// UpdateSettingRequest represents the request body for updating a setting
type UpdateSettingRequest struct {
	Value string `json:"value"`
}

// Update updates a setting by key
func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userIDStr := chimw.UserID(r)
	userID, _ := strconv.Atoi(userIDStr)

	key := chi.URLParam(r, "key")
	if key == "" {
		utils.BadRequest(w, "Setting key is required", nil)
		return
	}

	// Get old value for audit log
	oldValue, _ := h.settingsService.Get(key)

	var req UpdateSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}

	if err := h.settingsService.Set(key, req.Value); err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to update setting",
				zap.String("key", key),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to update setting")
		return
	}

	// Log audit event
	if h.auditService != nil && userID > 0 {
		_ = h.auditService.LogSettingsUpdate(r.Context(), userID, key, oldValue, req.Value, getClientIP(r), r.UserAgent())
	}

	utils.Success(w, map[string]string{"key": key, "value": req.Value}, "Setting updated successfully")
}

// GetNotFound returns the 404 page configuration
func (h *SettingsHandler) GetNotFound(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settingsService.GetNotFoundSettings()
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to get 404 settings", zap.Error(err))
		}
		utils.InternalError(w, "Failed to get 404 settings")
		return
	}

	utils.Success(w, settings, "404 settings retrieved successfully")
}

// UpdateNotFoundRequest represents the request body for updating 404 settings
type UpdateNotFoundRequest struct {
	Mode        string `json:"mode"`         // "default" or "redirect"
	RedirectURL string `json:"redirect_url"` // URL to redirect when mode is "redirect"
}

// UpdateNotFound updates the 404 page configuration
func (h *SettingsHandler) UpdateNotFound(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userIDStr := chimw.UserID(r)
	userID, _ := strconv.Atoi(userIDStr)

	// Get old settings for audit log
	oldSettings, _ := h.settingsService.GetNotFoundSettings()
	var oldValue string
	if oldSettings != nil {
		oldValue = oldSettings.Mode
		if oldSettings.RedirectURL != "" {
			oldValue += " -> " + oldSettings.RedirectURL
		}
	}

	var req UpdateNotFoundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}

	// Validate mode
	if req.Mode != "default" && req.Mode != "redirect" {
		utils.BadRequest(w, "Mode must be 'default' or 'redirect'", nil)
		return
	}

	// Validate redirect URL if mode is redirect
	if req.Mode == "redirect" && req.RedirectURL == "" {
		utils.BadRequest(w, "Redirect URL is required when mode is 'redirect'", nil)
		return
	}

	settings := &models.NotFoundSettings{
		Mode:        req.Mode,
		RedirectURL: req.RedirectURL,
	}

	if err := h.settingsService.SetNotFoundSettings(settings); err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to update 404 settings",
				zap.String("mode", settings.Mode),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to update 404 settings")
		return
	}

	// Log audit event
	if h.auditService != nil && userID > 0 {
		newValue := req.Mode
		if req.RedirectURL != "" {
			newValue += " -> " + req.RedirectURL
		}
		_ = h.auditService.LogSettingsUpdate(r.Context(), userID, "not_found_settings", oldValue, newValue, getClientIP(r), r.UserAgent())
	}

	utils.Success(w, settings, "404 settings updated successfully")
}
