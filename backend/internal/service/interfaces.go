package service

import (
	"context"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// ProxyServiceInterface defines the interface for proxy operations
type ProxyServiceInterface interface {
	ListProxies(req ListProxiesRequest) (*models.ProxyListResponse, error)
	GetProxyByID(id int) (*models.Proxy, error)
	CreateProxy(proxy *models.Proxy, userID int) error
	UpdateProxy(id int, proxy *models.Proxy) error
	DeleteProxy(id int) error
	EnableProxy(id int) error
	DisableProxy(id int) error
	GetStats() (*repository.ProxyStats, error)
}

// SettingsServiceInterface defines the interface for settings operations
type SettingsServiceInterface interface {
	Get(key string) (string, error)
	GetWithDefault(key, defaultValue string) string
	Set(key, value string) error
	GetAll() (map[string]string, error)
	Delete(key string) error
	GetNotFoundSettings() (*models.NotFoundSettings, error)
	SetNotFoundSettings(settings *models.NotFoundSettings) error
}

// SyncServiceInterface defines the interface for sync operations
type SyncServiceInterface interface {
	GetStatus() SyncStatus
	FullSync() error
	SyncProxyByID(proxyID int) error
}

// ProxySyncer defines the interface for proxy sync operations used by ProxyService
type ProxySyncer interface {
	SyncProxy(proxy *models.Proxy) error
	RemoveProxy(proxyID int, hostname string) error
	EnableProxy(proxyID int, hostname string) error
	DisableProxy(proxyID int, hostname string) error
}

// AuditServiceInterface defines the interface for audit logging operations
type AuditServiceInterface interface {
	// Core logging
	LogEvent(ctx context.Context, event models.AuditEvent) error

	// Configuration
	GetConfig() (*models.AuditConfig, error)
	SetConfig(config *models.AuditConfig) error
	InvalidateConfigCache()

	// Queries
	ListAuditLogs(params repository.AuditLogListParams) (*models.AuditLogListResponse, error)
	GetAuditLogByID(id int) (*models.AuditLog, error)
	GetStats() (*models.AuditLogStats, error)

	// Proxy events
	LogProxyCreate(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error
	LogProxyUpdate(ctx context.Context, userID int, proxy *models.Proxy, changes map[string]interface{}, ip, userAgent string) error
	LogProxyDelete(ctx context.Context, userID int, proxyID int, proxyName, hostname string, ip, userAgent string) error
	LogProxyEnable(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error
	LogProxyDisable(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error

	// Auth events
	LogLogin(ctx context.Context, userID int, username string, ip, userAgent string) error
	LogLoginFailed(ctx context.Context, username, ip, userAgent, reason string) error
	LogLogout(ctx context.Context, userID int, username string, ip, userAgent string) error
	LogRegister(ctx context.Context, userID int, username string, ip, userAgent string) error
	LogPasswordChange(ctx context.Context, userID int, username string, ip, userAgent string) error

	// Settings events
	LogSettingsUpdate(ctx context.Context, userID int, key string, oldVal, newVal string, ip, userAgent string) error

	// Sync events
	LogSyncStarted(ctx context.Context) error
	LogSyncCompleted(ctx context.Context, proxiesCount int) error
	LogSyncFailed(ctx context.Context, errMsg string) error

	// System events
	LogSystemStartup(ctx context.Context) error
	LogCaddyReload(ctx context.Context, success bool, errMsg string) error
}

// ACLServiceInterface defines the interface for ACL operations
type ACLServiceInterface interface {
	// Group Management
	CreateGroup(group *models.ACLGroup, createdBy int) error
	GetGroup(id int) (*models.ACLGroup, error)
	GetGroupByName(name string) (*models.ACLGroup, error)
	ListGroups(params ListACLGroupsRequest) (*models.ACLGroupListResponse, error)
	UpdateGroup(id int, updates *models.ACLGroup) error
	DeleteGroup(id int) error

	// IP Rules
	AddIPRule(groupID int, rule *models.ACLIPRule) error
	UpdateIPRule(id int, rule *models.ACLIPRule) error
	DeleteIPRule(id int) error

	// Basic Auth
	AddBasicAuthUser(groupID int, username, password string) error
	UpdateBasicAuthPassword(id int, password string) error
	DeleteBasicAuthUser(id int) error

	// External Providers
	AddExternalProvider(groupID int, provider *models.ACLExternalProvider) error
	UpdateExternalProvider(id int, provider *models.ACLExternalProvider) error
	DeleteExternalProvider(id int) error

	// Waygates Auth Config
	GetWaygatesAuth(groupID int) (*models.ACLWaygatesAuth, error)
	ConfigureWaygatesAuth(groupID int, config *models.ACLWaygatesAuth) error

	// Proxy Assignment
	AssignToProxy(proxyID, groupID int, pathPattern string, priority int) error
	UpdateProxyAssignment(id int, pathPattern string, priority int, enabled bool) error
	RemoveFromProxy(proxyID, groupID int) error
	GetProxyACL(proxyID int) ([]models.ProxyACLAssignment, error)
	GetGroupUsage(groupID int) ([]models.ProxyACLAssignment, error)

	// Branding
	GetBranding() (*models.ACLBranding, error)
	UpdateBranding(branding *models.ACLBranding) error

	// OAuth Provider Restrictions
	GetOAuthProviderRestrictions(groupID int) ([]models.ACLOAuthProviderRestriction, error)
	SetOAuthProviderRestriction(groupID int, provider string, emails, domains []string, enabled bool) error
	DeleteOAuthProviderRestriction(groupID int, provider string) error

	// Access Verification (for forward_auth)
	VerifyAccess(request *ACLVerifyRequest) (*ACLVerifyResponse, error)

	// Session Management
	CreateSession(userID int, proxyID *int, ip, userAgent string, ttl int) (*models.ACLSession, error)
	CreateOAuthSession(email, provider string, proxyID *int, ip, userAgent string, ttl int) (*models.ACLSession, error)
	CreateSessionWithParams(params CreateSessionParams) (*models.ACLSession, error)
	ValidateSession(token string) (*models.ACLSession, error)
	RevokeSession(token string) error
	RevokeUserSessions(userID int) error
	CleanupExpiredSessions() (int64, error)
}

// Ensure concrete types implement interfaces
var (
	_ ProxyServiceInterface    = (*ProxyService)(nil)
	_ SettingsServiceInterface = (*SettingsService)(nil)
	_ SyncServiceInterface     = (*SyncService)(nil)
	_ ProxySyncer              = (*SyncService)(nil)
	_ AuditServiceInterface    = (*AuditService)(nil)
	_ ACLServiceInterface      = (*ACLService)(nil)
)
