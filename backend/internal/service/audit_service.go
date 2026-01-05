package service

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// AuditService handles audit logging business logic
type AuditService struct {
	repo            repository.AuditLogRepositoryInterface
	settingsService SettingsServiceInterface
	logger          *zap.Logger
	configCache     *models.AuditConfig
	configMutex     sync.RWMutex
}

// NewAuditService creates a new audit service
func NewAuditService(
	repo repository.AuditLogRepositoryInterface,
	settingsService SettingsServiceInterface,
	logger *zap.Logger,
) *AuditService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AuditService{
		repo:            repo,
		settingsService: settingsService,
		logger:          logger.Named("audit-service"),
	}
}

// LogEvent is the main method for creating audit entries
func (s *AuditService) LogEvent(ctx context.Context, event models.AuditEvent) error {
	// Check if event category is enabled
	if !s.isEventEnabled(event.Action) {
		s.logger.Debug("Skipping audit log (category disabled)",
			zap.String("action", event.Action))
		return nil
	}

	log := &models.AuditLog{
		UserID:    event.UserID,
		Action:    event.Action,
		Status:    event.Status,
		Details:   models.JSONField(event.Details),
		UserAgent: stringPtr(event.UserAgent),
		IPAddress: stringPtr(event.IPAddress),
	}

	if event.ResourceType != "" {
		log.ResourceType = &event.ResourceType
	}
	if event.ResourceID != nil {
		log.ResourceID = event.ResourceID
	}
	if event.ResourceName != "" {
		log.ResourceName = &event.ResourceName
	}
	if event.ErrorMessage != "" {
		log.ErrorMessage = &event.ErrorMessage
	}

	if err := s.repo.Create(log); err != nil {
		s.logger.Error("Failed to create audit log",
			zap.String("action", event.Action),
			zap.Error(err))
		return err
	}

	s.logger.Debug("Audit log created",
		zap.String("action", event.Action),
		zap.String("status", event.Status))

	return nil
}

// isEventEnabled checks if an event category is enabled in the config
func (s *AuditService) isEventEnabled(action string) bool {
	config := s.getConfig()

	switch {
	case strings.HasPrefix(action, "proxy."):
		return config.ProxyEvents
	case strings.HasPrefix(action, "auth."):
		return config.AuthEvents
	case strings.HasPrefix(action, "settings."):
		return config.SettingsEvents
	case strings.HasPrefix(action, "sync."):
		return config.SyncEvents
	case strings.HasPrefix(action, "system."), strings.HasPrefix(action, "caddy."):
		return config.SystemEvents
	default:
		// Log unknown events by default
		return true
	}
}

// getConfig retrieves the current audit configuration (with caching)
func (s *AuditService) getConfig() *models.AuditConfig {
	s.configMutex.RLock()
	if s.configCache != nil {
		defer s.configMutex.RUnlock()
		return s.configCache
	}
	s.configMutex.RUnlock()

	// Load from settings
	config, err := s.GetConfig()
	if err != nil {
		s.logger.Warn("Failed to load audit config, using defaults", zap.Error(err))
		return models.DefaultAuditConfig()
	}

	return config
}

// GetConfig retrieves the audit configuration from settings
func (s *AuditService) GetConfig() (*models.AuditConfig, error) {
	configStr := s.settingsService.GetWithDefault(models.SettingAuditConfig, "")
	if configStr == "" {
		return models.DefaultAuditConfig(), nil
	}

	var config models.AuditConfig
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		s.logger.Error("Failed to parse audit config", zap.Error(err))
		return models.DefaultAuditConfig(), nil
	}

	return &config, nil
}

// SetConfig updates the audit configuration
func (s *AuditService) SetConfig(config *models.AuditConfig) error {
	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}

	if err := s.settingsService.Set(models.SettingAuditConfig, string(configBytes)); err != nil {
		return err
	}

	// Update cache
	s.configMutex.Lock()
	s.configCache = config
	s.configMutex.Unlock()

	s.logger.Info("Audit config updated",
		zap.Bool("proxy_events", config.ProxyEvents),
		zap.Bool("auth_events", config.AuthEvents),
		zap.Bool("settings_events", config.SettingsEvents),
		zap.Bool("sync_events", config.SyncEvents),
		zap.Bool("system_events", config.SystemEvents))

	return nil
}

// InvalidateConfigCache invalidates the config cache (call after settings update)
func (s *AuditService) InvalidateConfigCache() {
	s.configMutex.Lock()
	s.configCache = nil
	s.configMutex.Unlock()
}

// ListAuditLogs returns a paginated list of audit logs
func (s *AuditService) ListAuditLogs(params repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
	logs, total, err := s.repo.List(params)
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / params.Limit
	if int(total)%params.Limit > 0 {
		totalPages++
	}

	return &models.AuditLogListResponse{
		Items:      logs,
		Total:      total,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}, nil
}

// GetAuditLogByID retrieves a single audit log by ID
func (s *AuditService) GetAuditLogByID(id int) (*models.AuditLog, error) {
	return s.repo.GetByID(id)
}

// GetStats retrieves audit log statistics
func (s *AuditService) GetStats() (*models.AuditLogStats, error) {
	return s.repo.GetStats()
}

// Helper methods for common audit logging patterns

// LogProxyCreate logs a proxy creation event
func (s *AuditService) LogProxyCreate(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error {
	resourceID := proxy.ID
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionProxyCreate,
		ResourceType: models.AuditResourceTypeProxy,
		ResourceID:   &resourceID,
		ResourceName: proxy.Name,
		Details: map[string]interface{}{
			"hostname":    proxy.Hostname,
			"type":        proxy.Type,
			"ssl_enabled": proxy.SSLEnabled,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// LogProxyUpdate logs a proxy update event
func (s *AuditService) LogProxyUpdate(ctx context.Context, userID int, proxy *models.Proxy, changes map[string]interface{}, ip, userAgent string) error {
	resourceID := proxy.ID
	details := map[string]interface{}{
		"hostname": proxy.Hostname,
		"type":     proxy.Type,
	}
	if changes != nil {
		details["changes"] = changes
	}

	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionProxyUpdate,
		ResourceType: models.AuditResourceTypeProxy,
		ResourceID:   &resourceID,
		ResourceName: proxy.Name,
		Details:      details,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Status:       models.AuditStatusSuccess,
	})
}

// LogProxyDelete logs a proxy deletion event
func (s *AuditService) LogProxyDelete(ctx context.Context, userID int, proxyID int, proxyName, hostname string, ip, userAgent string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionProxyDelete,
		ResourceType: models.AuditResourceTypeProxy,
		ResourceID:   &proxyID,
		ResourceName: proxyName,
		Details: map[string]interface{}{
			"hostname": hostname,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// LogProxyEnable logs a proxy enable event
func (s *AuditService) LogProxyEnable(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error {
	resourceID := proxy.ID
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionProxyEnable,
		ResourceType: models.AuditResourceTypeProxy,
		ResourceID:   &resourceID,
		ResourceName: proxy.Name,
		Details: map[string]interface{}{
			"hostname": proxy.Hostname,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// LogProxyDisable logs a proxy disable event
func (s *AuditService) LogProxyDisable(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error {
	resourceID := proxy.ID
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionProxyDisable,
		ResourceType: models.AuditResourceTypeProxy,
		ResourceID:   &resourceID,
		ResourceName: proxy.Name,
		Details: map[string]interface{}{
			"hostname": proxy.Hostname,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// LogLogin logs a successful login event
func (s *AuditService) LogLogin(ctx context.Context, userID int, username string, ip, userAgent string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionAuthLogin,
		ResourceType: models.AuditResourceTypeUser,
		ResourceID:   &userID,
		ResourceName: username,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Status:       models.AuditStatusSuccess,
	})
}

// LogLoginFailed logs a failed login attempt
func (s *AuditService) LogLoginFailed(ctx context.Context, username, ip, userAgent, reason string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       nil, // No user ID for failed login
		Action:       models.AuditActionAuthLoginFailed,
		ResourceType: models.AuditResourceTypeUser,
		ResourceName: username,
		Details: map[string]interface{}{
			"reason": reason,
		},
		IPAddress:    ip,
		UserAgent:    userAgent,
		Status:       models.AuditStatusFailure,
		ErrorMessage: reason,
	})
}

// LogLogout logs a logout event
func (s *AuditService) LogLogout(ctx context.Context, userID int, username string, ip, userAgent string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionAuthLogout,
		ResourceType: models.AuditResourceTypeUser,
		ResourceID:   &userID,
		ResourceName: username,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Status:       models.AuditStatusSuccess,
	})
}

// LogRegister logs a user registration event
func (s *AuditService) LogRegister(ctx context.Context, userID int, username string, ip, userAgent string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionAuthRegister,
		ResourceType: models.AuditResourceTypeUser,
		ResourceID:   &userID,
		ResourceName: username,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Status:       models.AuditStatusSuccess,
	})
}

// LogPasswordChange logs a password change event
func (s *AuditService) LogPasswordChange(ctx context.Context, userID int, username string, ip, userAgent string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionAuthPasswordChange,
		ResourceType: models.AuditResourceTypeUser,
		ResourceID:   &userID,
		ResourceName: username,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Status:       models.AuditStatusSuccess,
	})
}

// LogSettingsUpdate logs a settings update event
func (s *AuditService) LogSettingsUpdate(ctx context.Context, userID int, key string, oldVal, newVal string, ip, userAgent string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionSettingsUpdate,
		ResourceType: models.AuditResourceTypeSettings,
		ResourceName: key,
		Details: map[string]interface{}{
			"key":       key,
			"old_value": oldVal,
			"new_value": newVal,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// LogSyncStarted logs a sync started event
func (s *AuditService) LogSyncStarted(ctx context.Context) error {
	return s.LogEvent(ctx, models.AuditEvent{
		Action:       models.AuditActionSyncStarted,
		ResourceType: models.AuditResourceTypeSystem,
		Status:       models.AuditStatusSuccess,
	})
}

// LogSyncCompleted logs a sync completed event
func (s *AuditService) LogSyncCompleted(ctx context.Context, proxiesCount int) error {
	return s.LogEvent(ctx, models.AuditEvent{
		Action:       models.AuditActionSyncCompleted,
		ResourceType: models.AuditResourceTypeSystem,
		Details: map[string]interface{}{
			"proxies_synced": proxiesCount,
		},
		Status: models.AuditStatusSuccess,
	})
}

// LogSyncFailed logs a sync failed event
func (s *AuditService) LogSyncFailed(ctx context.Context, errMsg string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		Action:       models.AuditActionSyncFailed,
		ResourceType: models.AuditResourceTypeSystem,
		Status:       models.AuditStatusFailure,
		ErrorMessage: errMsg,
	})
}

// LogSystemStartup logs a system startup event
func (s *AuditService) LogSystemStartup(ctx context.Context) error {
	return s.LogEvent(ctx, models.AuditEvent{
		Action:       models.AuditActionSystemStartup,
		ResourceType: models.AuditResourceTypeSystem,
		Status:       models.AuditStatusSuccess,
	})
}

// LogCaddyReload logs a Caddy reload event
func (s *AuditService) LogCaddyReload(ctx context.Context, success bool, errMsg string) error {
	status := models.AuditStatusSuccess
	if !success {
		status = models.AuditStatusFailure
	}
	return s.LogEvent(ctx, models.AuditEvent{
		Action:       models.AuditActionCaddyReload,
		ResourceType: models.AuditResourceTypeSystem,
		Status:       status,
		ErrorMessage: errMsg,
	})
}

// stringPtr returns a pointer to a string, or nil if empty
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
