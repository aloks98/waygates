package service

import (
	"context"
	"encoding/json"
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
func (s *AuditService) LogEvent(_ context.Context, event models.AuditEvent) error {
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

// isEventEnabled checks if a specific event is enabled in the config
func (s *AuditService) isEventEnabled(action string) bool {
	config := s.getConfig()

	switch action {
	// Proxy events
	case models.AuditActionProxyCreate:
		return config.ProxyCreate
	case models.AuditActionProxyUpdate:
		return config.ProxyUpdate
	case models.AuditActionProxyDelete:
		return config.ProxyDelete
	case models.AuditActionProxyEnable:
		return config.ProxyEnable
	case models.AuditActionProxyDisable:
		return config.ProxyDisable
	// Auth events
	case models.AuditActionAuthLogin:
		return config.AuthLogin
	case models.AuditActionAuthLogout:
		return config.AuthLogout
	case models.AuditActionAuthRegister:
		return config.AuthRegister
	case models.AuditActionAuthPasswordChange:
		return config.AuthPasswordChange
	case models.AuditActionAuthLoginFailed:
		return config.AuthLoginFailed
	// Settings events
	case models.AuditActionSettingsUpdate:
		return config.SettingsUpdate
	// Sync events
	case models.AuditActionSyncStarted:
		return config.SyncStarted
	case models.AuditActionSyncCompleted:
		return config.SyncCompleted
	case models.AuditActionSyncFailed:
		return config.SyncFailed
	// System events
	case models.AuditActionSystemStartup:
		return config.SystemStartup
	case models.AuditActionCaddyReload:
		return config.CaddyReload
	// ACL Group events
	case models.AuditActionACLGroupCreate:
		return config.ACLGroupCreate
	case models.AuditActionACLGroupUpdate:
		return config.ACLGroupUpdate
	case models.AuditActionACLGroupDelete:
		return config.ACLGroupDelete
	// ACL IP Rule events
	case models.AuditActionACLIPRuleAdd:
		return config.ACLIPRuleAdd
	case models.AuditActionACLIPRuleUpdate:
		return config.ACLIPRuleUpdate
	case models.AuditActionACLIPRuleDelete:
		return config.ACLIPRuleDelete
	// ACL Basic Auth events
	case models.AuditActionACLBasicAuthAdd:
		return config.ACLBasicAuthAdd
	case models.AuditActionACLBasicAuthUpdate:
		return config.ACLBasicAuthUpdate
	case models.AuditActionACLBasicAuthDelete:
		return config.ACLBasicAuthDelete
	// ACL Waygates Auth events
	case models.AuditActionACLWaygatesAuthUpdate:
		return config.ACLWaygatesAuthUpdate
	// ACL Proxy Assignment events
	case models.AuditActionACLAssignmentCreate:
		return config.ACLAssignmentCreate
	case models.AuditActionACLAssignmentUpdate:
		return config.ACLAssignmentUpdate
	case models.AuditActionACLAssignmentDelete:
		return config.ACLAssignmentDelete
	// ACL Branding events
	case models.AuditActionACLBrandingUpdate:
		return config.ACLBrandingUpdate
	// ACL Session events
	case models.AuditActionACLSessionRevoke:
		return config.ACLSessionRevoke
	case models.AuditActionACLOAuthRestrictionSet:
		return config.ACLOAuthRestrictionSet
	case models.AuditActionACLOAuthRestrictionDelete:
		return config.ACLOAuthRestrictionDelete
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

	// Populate the cache so subsequent events don't re-fetch and re-parse the
	// config from settings on every audit write.
	s.configMutex.Lock()
	s.configCache = config
	s.configMutex.Unlock()

	return config
}

// GetConfig retrieves the audit configuration from settings
func (s *AuditService) GetConfig() (*models.AuditConfig, error) {
	configStr := s.settingsService.GetWithDefault(models.SettingAuditConfig, "")
	if configStr == "" {
		return models.DefaultAuditConfig(), nil
	}

	// Start with defaults so any new fields added later are enabled by default
	config := models.DefaultAuditConfig()
	if err := json.Unmarshal([]byte(configStr), config); err != nil {
		s.logger.Error("Failed to parse audit config", zap.Error(err))
		return models.DefaultAuditConfig(), nil
	}

	return config, nil
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

	s.logger.Info("Audit config updated")

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
		// Use the same nested `changes` shape as every other update event so the
		// frontend reads one consistent structure (details.changes[field] =
		// {old,new}) instead of special-casing settings.
		Details: map[string]interface{}{
			"key": key,
			"changes": map[string]interface{}{
				key: map[string]interface{}{"old": oldVal, "new": newVal},
			},
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

// ACL Audit Logging Methods

// LogACLGroupCreate logs an ACL group creation event
func (s *AuditService) LogACLGroupCreate(ctx context.Context, userID int, group *models.ACLGroup, ip, userAgent string) error {
	resourceID := group.ID
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLGroupCreate,
		ResourceType: models.AuditResourceTypeACL,
		ResourceID:   &resourceID,
		ResourceName: group.Name,
		Details: map[string]interface{}{
			"group_id":         group.ID,
			"group_name":       group.Name,
			"combination_mode": group.CombinationMode,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// LogACLGroupUpdate logs an ACL group update event
func (s *AuditService) LogACLGroupUpdate(ctx context.Context, userID int, group *models.ACLGroup, changes map[string]interface{}, ip, userAgent string) error {
	resourceID := group.ID
	details := map[string]interface{}{
		"group_id":   group.ID,
		"group_name": group.Name,
	}
	if changes != nil {
		details["changes"] = changes
	}

	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLGroupUpdate,
		ResourceType: models.AuditResourceTypeACL,
		ResourceID:   &resourceID,
		ResourceName: group.Name,
		Details:      details,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Status:       models.AuditStatusSuccess,
	})
}

// LogACLGroupDelete logs an ACL group deletion event
func (s *AuditService) LogACLGroupDelete(ctx context.Context, userID int, groupID int, groupName, ip, userAgent string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLGroupDelete,
		ResourceType: models.AuditResourceTypeACL,
		ResourceID:   &groupID,
		ResourceName: groupName,
		Details: map[string]interface{}{
			"group_id":   groupID,
			"group_name": groupName,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// LogACLIPRuleAdd logs an IP rule addition event
func (s *AuditService) LogACLIPRuleAdd(ctx context.Context, userID int, groupID int, groupName string, rule *models.ACLIPRule, ip, userAgent string) error {
	ruleID := rule.ID
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLIPRuleAdd,
		ResourceType: models.AuditResourceTypeACL,
		ResourceID:   &ruleID,
		ResourceName: groupName,
		Details: map[string]interface{}{
			"group_id":   groupID,
			"group_name": groupName,
			"rule_id":    rule.ID,
			"rule_type":  rule.RuleType,
			"cidr":       rule.CIDR,
			"priority":   rule.Priority,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// LogACLIPRuleUpdate logs an IP rule update event
func (s *AuditService) LogACLIPRuleUpdate(ctx context.Context, userID int, rule *models.ACLIPRule, changes map[string]interface{}, ip, userAgent string) error {
	ruleID := rule.ID
	details := map[string]interface{}{
		"rule_id":   rule.ID,
		"rule_type": rule.RuleType,
		"cidr":      rule.CIDR,
	}
	if changes != nil {
		details["changes"] = changes
	}

	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLIPRuleUpdate,
		ResourceType: models.AuditResourceTypeACL,
		ResourceID:   &ruleID,
		Details:      details,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Status:       models.AuditStatusSuccess,
	})
}

// LogACLIPRuleDelete logs an IP rule deletion event
func (s *AuditService) LogACLIPRuleDelete(ctx context.Context, userID int, ruleID int, groupName, cidr, ruleType, ip, userAgent string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLIPRuleDelete,
		ResourceType: models.AuditResourceTypeACL,
		ResourceID:   &ruleID,
		ResourceName: groupName,
		Details: map[string]interface{}{
			"rule_id":    ruleID,
			"group_name": groupName,
			"cidr":       cidr,
			"rule_type":  ruleType,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// LogACLBasicAuthAdd logs a basic auth user addition event
func (s *AuditService) LogACLBasicAuthAdd(ctx context.Context, userID int, groupID int, groupName, username, ip, userAgent string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLBasicAuthAdd,
		ResourceType: models.AuditResourceTypeACL,
		ResourceName: groupName,
		Details: map[string]interface{}{
			"group_id":   groupID,
			"group_name": groupName,
			"username":   username,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// LogACLBasicAuthUpdate logs a basic auth password change event
func (s *AuditService) LogACLBasicAuthUpdate(ctx context.Context, userID int, authUserID int, groupName, username, ip, userAgent string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLBasicAuthUpdate,
		ResourceType: models.AuditResourceTypeACL,
		ResourceID:   &authUserID,
		ResourceName: groupName,
		Details: map[string]interface{}{
			"auth_user_id": authUserID,
			"group_name":   groupName,
			"username":     username,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// LogACLBasicAuthDelete logs a basic auth user deletion event
func (s *AuditService) LogACLBasicAuthDelete(ctx context.Context, userID int, authUserID int, groupName, username, ip, userAgent string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLBasicAuthDelete,
		ResourceType: models.AuditResourceTypeACL,
		ResourceID:   &authUserID,
		ResourceName: groupName,
		Details: map[string]interface{}{
			"auth_user_id": authUserID,
			"group_name":   groupName,
			"username":     username,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// LogACLWaygatesAuthUpdate logs a Waygates auth configuration change event
func (s *AuditService) LogACLWaygatesAuthUpdate(ctx context.Context, userID int, groupID int, groupName string, changes map[string]interface{}, ip, userAgent string) error {
	details := map[string]interface{}{
		"group_id":   groupID,
		"group_name": groupName,
	}

	if len(changes) > 0 {
		details["changes"] = changes
	}

	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLWaygatesAuthUpdate,
		ResourceType: models.AuditResourceTypeACL,
		ResourceID:   &groupID,
		ResourceName: groupName,
		Details:      details,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Status:       models.AuditStatusSuccess,
	})
}

// LogACLAssignmentCreate logs an ACL proxy assignment creation event
func (s *AuditService) LogACLAssignmentCreate(ctx context.Context, userID int, proxyID int, proxyName string, groupID int, groupName, pathPattern, ip, userAgent string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLAssignmentCreate,
		ResourceType: models.AuditResourceTypeACL,
		ResourceID:   &proxyID,
		ResourceName: proxyName,
		Details: map[string]interface{}{
			"proxy_id":     proxyID,
			"proxy_name":   proxyName,
			"group_id":     groupID,
			"group_name":   groupName,
			"path_pattern": pathPattern,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// LogACLAssignmentUpdate logs an ACL proxy assignment update event
func (s *AuditService) LogACLAssignmentUpdate(ctx context.Context, userID int, assignment *models.ProxyACLAssignment, changes map[string]interface{}, ip, userAgent string) error {
	assignmentID := assignment.ID
	details := map[string]interface{}{
		"assignment_id": assignment.ID,
		"proxy_id":      assignment.ProxyID,
		"group_id":      assignment.ACLGroupID,
		"path_pattern":  assignment.PathPattern,
	}
	if changes != nil {
		details["changes"] = changes
	}

	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLAssignmentUpdate,
		ResourceType: models.AuditResourceTypeACL,
		ResourceID:   &assignmentID,
		Details:      details,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Status:       models.AuditStatusSuccess,
	})
}

// LogACLAssignmentDelete logs an ACL proxy assignment deletion event
func (s *AuditService) LogACLAssignmentDelete(ctx context.Context, userID int, proxyID int, proxyName string, groupID int, groupName, ip, userAgent string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLAssignmentDelete,
		ResourceType: models.AuditResourceTypeACL,
		ResourceID:   &proxyID,
		ResourceName: proxyName,
		Details: map[string]interface{}{
			"proxy_id":   proxyID,
			"proxy_name": proxyName,
			"group_id":   groupID,
			"group_name": groupName,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// LogACLBrandingUpdate logs a branding configuration update event
func (s *AuditService) LogACLBrandingUpdate(ctx context.Context, userID int, changes map[string]interface{}, ip, userAgent string) error {
	details := map[string]interface{}{}
	if changes != nil {
		details["changes"] = changes
	}

	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLBrandingUpdate,
		ResourceType: models.AuditResourceTypeACL,
		ResourceName: "branding",
		Details:      details,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Status:       models.AuditStatusSuccess,
	})
}

// LogACLSessionRevoke logs a session revocation event
func (s *AuditService) LogACLSessionRevoke(ctx context.Context, userID int, sessionID int, sessionEmail, ip, userAgent string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLSessionRevoke,
		ResourceType: models.AuditResourceTypeACL,
		ResourceID:   &sessionID,
		ResourceName: sessionEmail,
		Details: map[string]interface{}{
			"session_id":    sessionID,
			"session_email": sessionEmail,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// LogACLOAuthRestrictionSet logs an OAuth provider restriction set event
func (s *AuditService) LogACLOAuthRestrictionSet(ctx context.Context, userID int, groupID int, groupName, provider string, oldRestriction *models.ACLOAuthProviderRestriction, newEnabled bool, newAllowedEmails, newAllowedDomains []string, ip, userAgent string) error {
	details := map[string]interface{}{
		"group_id":   groupID,
		"group_name": groupName,
		"provider":   provider,
	}

	// Build changes showing old and new values
	changes := make(map[string]interface{})

	oldEnabled := false
	var oldEmails, oldDomains []string
	if oldRestriction != nil {
		oldEnabled = oldRestriction.Enabled
		oldEmails = oldRestriction.AllowedEmails
		oldDomains = oldRestriction.AllowedDomains
	}

	if oldEnabled != newEnabled {
		changes["enabled"] = map[string]interface{}{
			"old": oldEnabled,
			"new": newEnabled,
		}
	}

	oldEmailsStr := joinStringSlice(oldEmails)
	newEmailsStr := joinStringSlice(newAllowedEmails)
	if oldEmailsStr != newEmailsStr {
		changes["allowed_emails"] = map[string]interface{}{
			"old": oldEmails,
			"new": newAllowedEmails,
		}
	}

	oldDomainsStr := joinStringSlice(oldDomains)
	newDomainsStr := joinStringSlice(newAllowedDomains)
	if oldDomainsStr != newDomainsStr {
		changes["allowed_domains"] = map[string]interface{}{
			"old": oldDomains,
			"new": newAllowedDomains,
		}
	}

	if len(changes) > 0 {
		details["changes"] = changes
	}

	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLOAuthRestrictionSet,
		ResourceType: models.AuditResourceTypeACL,
		ResourceID:   &groupID,
		ResourceName: groupName,
		Details:      details,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Status:       models.AuditStatusSuccess,
	})
}

// joinStringSlice joins a slice of strings for comparison
func joinStringSlice(s []string) string {
	if s == nil {
		return ""
	}
	result := ""
	for i, v := range s {
		if i > 0 {
			result += ","
		}
		result += v
	}
	return result
}

// LogACLOAuthRestrictionDelete logs an OAuth provider restriction delete event
func (s *AuditService) LogACLOAuthRestrictionDelete(ctx context.Context, userID int, groupID int, groupName, provider, ip, userAgent string) error {
	return s.LogEvent(ctx, models.AuditEvent{
		UserID:       &userID,
		Action:       models.AuditActionACLOAuthRestrictionDelete,
		ResourceType: models.AuditResourceTypeACL,
		ResourceID:   &groupID,
		ResourceName: groupName,
		Details: map[string]interface{}{
			"group_id":   groupID,
			"group_name": groupName,
			"provider":   provider,
		},
		IPAddress: ip,
		UserAgent: userAgent,
		Status:    models.AuditStatusSuccess,
	})
}

// stringPtr returns a pointer to a string, or nil if empty
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
