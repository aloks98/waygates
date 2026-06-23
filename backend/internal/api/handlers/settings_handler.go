package handlers

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	chimw "github.com/aloks98/goauth/middleware/chi"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

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

// GetAll returns all settings as a key-value map.
// Sensitive keys (e.g. bcrypt hashes) are stripped before the response is sent.
func (h *SettingsHandler) GetAll(w http.ResponseWriter, _ *http.Request) {
	settings, err := h.settingsService.GetAll()
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to get all settings", zap.Error(err))
		}
		utils.InternalError(w, "Failed to get settings")
		return
	}

	// Remove any sensitive keys so they are never returned to callers.
	for key := range settings {
		if models.IsSensitiveSettingKey(key) {
			delete(settings, key)
		}
	}

	utils.Success(w, settings, "Settings retrieved successfully")
}

// Get returns a single setting by key.
// Sensitive keys (e.g. bcrypt hashes) return 404 so their existence is not confirmed.
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		utils.BadRequest(w, "Setting key is required", nil)
		return
	}

	// Treat sensitive keys as non-existent to avoid leaking secrets.
	if models.IsSensitiveSettingKey(key) {
		utils.NotFound(w, "Setting not found")
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

	// Treat sensitive keys as non-existent to prevent overwriting secrets via
	// the generic endpoint and to avoid the old value appearing in audit logs.
	if models.IsSensitiveSettingKey(key) {
		utils.NotFound(w, "setting not found")
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
func (h *SettingsHandler) GetNotFound(w http.ResponseWriter, _ *http.Request) {
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

// MetricsPublishDTO is the response DTO for the metrics publish endpoint.
// The bcrypt hash is intentionally NEVER included.
type MetricsPublishDTO struct {
	Enabled       bool     `json:"enabled"`
	Host          string   `json:"host"`
	Path          string   `json:"path"`
	BasicAuthUser string   `json:"basic_auth_user"`
	HasBasicAuth  bool     `json:"has_basic_auth"`
	AllowedCIDRs  []string `json:"allowed_cidrs"`
}

// GetMetricsPublish returns the metrics publish endpoint configuration.
// The bcrypt hash is never included in the response.
func (h *SettingsHandler) GetMetricsPublish(w http.ResponseWriter, _ *http.Request) {
	settings, err := h.settingsService.GetMetricsPublishSettings()
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to get metrics publish settings", zap.Error(err))
		}
		utils.InternalError(w, "Failed to get metrics publish settings")
		return
	}

	dto := MetricsPublishDTO{
		Enabled:       settings.Enabled,
		Host:          settings.Host,
		Path:          settings.Path,
		BasicAuthUser: settings.BasicAuthUser,
		HasBasicAuth:  settings.BasicAuthHash != "",
		AllowedCIDRs:  settings.AllowedCIDRs,
	}
	if dto.AllowedCIDRs == nil {
		dto.AllowedCIDRs = []string{}
	}

	utils.Success(w, dto, "Metrics publish settings retrieved successfully")
}

// UpdateMetricsPublishRequest is the request body for updating the metrics publish settings.
type UpdateMetricsPublishRequest struct {
	Enabled           bool     `json:"enabled"`
	Host              string   `json:"host"`
	Path              string   `json:"path"`
	BasicAuthUser     string   `json:"basic_auth_user"`
	BasicAuthPassword string   `json:"basic_auth_password"` // plaintext; bcrypt-hashed before storage
	AllowedCIDRs      []string `json:"allowed_cidrs"`
}

// UpdateMetricsPublish updates the metrics publish endpoint configuration.
func (h *SettingsHandler) UpdateMetricsPublish(w http.ResponseWriter, r *http.Request) {
	userIDStr := chimw.UserID(r)
	userID, _ := strconv.Atoi(userIDStr)

	// Fetch existing settings for audit and password-keep logic.
	existing, err := h.settingsService.GetMetricsPublishSettings()
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to get existing metrics publish settings", zap.Error(err))
		}
		utils.InternalError(w, "Failed to get existing metrics publish settings")
		return
	}

	var req UpdateMetricsPublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}

	// Validate all CIDRs up front.
	for _, cidr := range req.AllowedCIDRs {
		if _, _, parseErr := net.ParseCIDR(cidr); parseErr != nil {
			utils.BadRequest(w, fmt.Sprintf("Invalid CIDR %q: %v", cidr, parseErr), nil)
			return
		}
	}

	// Determine the hash to store.
	hashToStore := existing.BasicAuthHash
	if req.BasicAuthPassword != "" {
		// A new plaintext password was supplied — hash it now.
		hashed, hashErr := bcrypt.GenerateFromPassword([]byte(req.BasicAuthPassword), service.BcryptCost)
		if hashErr != nil {
			if h.logger != nil {
				h.logger.Error("Failed to hash metrics basic auth password", zap.Error(hashErr))
			}
			utils.InternalError(w, "Failed to hash password")
			return
		}
		hashToStore = string(hashed)
	}

	// Security: validate when enabling.
	if req.Enabled {
		if strings.TrimSpace(req.Host) == "" {
			utils.BadRequest(w, "Host is required when enabling metrics publish", nil)
			return
		}
		if strings.TrimSpace(req.BasicAuthUser) == "" {
			utils.BadRequest(w, "Basic auth user is required when enabling metrics publish", nil)
			return
		}
		if hashToStore == "" {
			utils.BadRequest(w, "A password is required when enabling metrics publish (no existing password found)", nil)
			return
		}
	}

	path := req.Path
	if path == "" {
		path = "/metrics"
	}

	newSettings := &models.MetricsPublishSettings{
		Enabled:       req.Enabled,
		Host:          strings.TrimSpace(req.Host),
		Path:          path,
		BasicAuthUser: strings.TrimSpace(req.BasicAuthUser),
		BasicAuthHash: hashToStore,
		AllowedCIDRs:  req.AllowedCIDRs,
	}

	if err := h.settingsService.SetMetricsPublishSettings(newSettings); err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to update metrics publish settings",
				zap.Bool("enabled", req.Enabled),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to update metrics publish settings")
		return
	}

	// Audit log.
	if h.auditService != nil && userID > 0 {
		oldVal := fmt.Sprintf("enabled=%v host=%s", existing.Enabled, existing.Host)
		newVal := fmt.Sprintf("enabled=%v host=%s", req.Enabled, req.Host)
		_ = h.auditService.LogSettingsUpdate(r.Context(), userID, "metrics_publish_settings", oldVal, newVal, getClientIP(r), r.UserAgent())
	}

	dto := MetricsPublishDTO{
		Enabled:       newSettings.Enabled,
		Host:          newSettings.Host,
		Path:          newSettings.Path,
		BasicAuthUser: newSettings.BasicAuthUser,
		HasBasicAuth:  newSettings.BasicAuthHash != "",
		AllowedCIDRs:  newSettings.AllowedCIDRs,
	}
	if dto.AllowedCIDRs == nil {
		dto.AllowedCIDRs = []string{}
	}

	utils.Success(w, dto, "Metrics publish settings updated successfully")
}
