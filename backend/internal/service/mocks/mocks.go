// Package mocks provides mock implementations for testing
package mocks

import (
	"context"
	"time"

	"github.com/aloks98/waygates/backend/internal/caddy"
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service"
)

// MockProxyService is a mock implementation of ProxyServiceInterface
type MockProxyService struct {
	ListProxiesFunc  func(req service.ListProxiesRequest) (*models.ProxyListResponse, error)
	GetProxyByIDFunc func(id int) (*models.Proxy, error)
	CreateProxyFunc  func(proxy *models.Proxy, userID int) error
	UpdateProxyFunc  func(id int, proxy *models.Proxy) error
	DeleteProxyFunc  func(id int) error
	EnableProxyFunc  func(id int) error
	DisableProxyFunc func(id int) error
	GetStatsFunc     func() (*repository.ProxyStats, error)
}

// ListProxies implements ProxyServiceInterface.
func (m *MockProxyService) ListProxies(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
	if m.ListProxiesFunc != nil {
		return m.ListProxiesFunc(req)
	}
	return &models.ProxyListResponse{Items: []models.Proxy{}, Total: 0}, nil
}

// GetProxyByID implements ProxyServiceInterface.
func (m *MockProxyService) GetProxyByID(id int) (*models.Proxy, error) {
	if m.GetProxyByIDFunc != nil {
		return m.GetProxyByIDFunc(id)
	}
	return nil, service.ErrProxyNotFound
}

// CreateProxy implements ProxyServiceInterface.
func (m *MockProxyService) CreateProxy(proxy *models.Proxy, userID int) error {
	if m.CreateProxyFunc != nil {
		return m.CreateProxyFunc(proxy, userID)
	}
	return nil
}

// UpdateProxy implements ProxyServiceInterface.
func (m *MockProxyService) UpdateProxy(id int, proxy *models.Proxy) error {
	if m.UpdateProxyFunc != nil {
		return m.UpdateProxyFunc(id, proxy)
	}
	return nil
}

// DeleteProxy implements ProxyServiceInterface.
func (m *MockProxyService) DeleteProxy(id int) error {
	if m.DeleteProxyFunc != nil {
		return m.DeleteProxyFunc(id)
	}
	return nil
}

// EnableProxy implements ProxyServiceInterface.
func (m *MockProxyService) EnableProxy(id int) error {
	if m.EnableProxyFunc != nil {
		return m.EnableProxyFunc(id)
	}
	return nil
}

// DisableProxy implements ProxyServiceInterface.
func (m *MockProxyService) DisableProxy(id int) error {
	if m.DisableProxyFunc != nil {
		return m.DisableProxyFunc(id)
	}
	return nil
}

// GetStats implements ProxyServiceInterface.
func (m *MockProxyService) GetStats() (*repository.ProxyStats, error) {
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc()
	}
	return &repository.ProxyStats{}, nil
}

// MockSettingsService is a mock implementation of SettingsServiceInterface
type MockSettingsService struct {
	GetFunc                 func(key string) (string, error)
	GetWithDefaultFunc      func(key, defaultValue string) string
	SetFunc                 func(key, value string) error
	GetAllFunc              func() (map[string]string, error)
	DeleteFunc              func(key string) error
	GetNotFoundSettingsFunc func() (*models.NotFoundSettings, error)
	SetNotFoundSettingsFunc func(settings *models.NotFoundSettings) error
}

// Get implements SettingsServiceInterface.
func (m *MockSettingsService) Get(key string) (string, error) {
	if m.GetFunc != nil {
		return m.GetFunc(key)
	}
	return "", nil
}

// GetWithDefault implements SettingsServiceInterface.
func (m *MockSettingsService) GetWithDefault(key, defaultValue string) string {
	if m.GetWithDefaultFunc != nil {
		return m.GetWithDefaultFunc(key, defaultValue)
	}
	return defaultValue
}

// Set implements SettingsServiceInterface.
func (m *MockSettingsService) Set(key, value string) error {
	if m.SetFunc != nil {
		return m.SetFunc(key, value)
	}
	return nil
}

// GetAll implements SettingsServiceInterface.
func (m *MockSettingsService) GetAll() (map[string]string, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return map[string]string{}, nil
}

// Delete implements SettingsServiceInterface.
func (m *MockSettingsService) Delete(key string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(key)
	}
	return nil
}

// GetNotFoundSettings implements SettingsServiceInterface.
func (m *MockSettingsService) GetNotFoundSettings() (*models.NotFoundSettings, error) {
	if m.GetNotFoundSettingsFunc != nil {
		return m.GetNotFoundSettingsFunc()
	}
	return &models.NotFoundSettings{Mode: "default"}, nil
}

// SetNotFoundSettings implements SettingsServiceInterface.
func (m *MockSettingsService) SetNotFoundSettings(settings *models.NotFoundSettings) error {
	if m.SetNotFoundSettingsFunc != nil {
		return m.SetNotFoundSettingsFunc(settings)
	}
	return nil
}

// MockSyncService is a mock implementation of SyncServiceInterface
type MockSyncService struct {
	GetStatusFunc     func() service.SyncStatus
	FullSyncFunc      func() error
	SyncProxyByIDFunc func(proxyID int) error
}

// GetStatus implements SyncServiceInterface.
func (m *MockSyncService) GetStatus() service.SyncStatus {
	if m.GetStatusFunc != nil {
		return m.GetStatusFunc()
	}
	return service.SyncStatus{
		IsSyncing:       false,
		LastSyncTime:    time.Now(),
		LastSyncSuccess: true,
		LastError:       "",
		SyncCount:       0,
	}
}

// FullSync implements SyncServiceInterface.
func (m *MockSyncService) FullSync() error {
	if m.FullSyncFunc != nil {
		return m.FullSyncFunc()
	}
	return nil
}

// SyncProxyByID implements SyncServiceInterface.
func (m *MockSyncService) SyncProxyByID(proxyID int) error {
	if m.SyncProxyByIDFunc != nil {
		return m.SyncProxyByIDFunc(proxyID)
	}
	return nil
}

// MockUserRepository is a mock implementation of UserRepositoryInterface
type MockUserRepository struct {
	CreateFunc               func(user *models.User) error
	GetByEmailFunc           func(email string) (*models.User, error)
	GetByUsernameOrEmailFunc func(identifier string) (*models.User, error)
	GetByIDFunc              func(id int) (*models.User, error)
	CountFunc                func() (int64, error)
	DeleteFunc               func(id int) error
	UpdatePasswordFunc       func(id int, passwordHash string) error
}

// Create implements UserRepositoryInterface.
func (m *MockUserRepository) Create(user *models.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(user)
	}
	return nil
}

// GetByEmail implements UserRepositoryInterface.
func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(email)
	}
	return nil, nil
}

// GetByUsernameOrEmail implements UserRepositoryInterface.
func (m *MockUserRepository) GetByUsernameOrEmail(identifier string) (*models.User, error) {
	if m.GetByUsernameOrEmailFunc != nil {
		return m.GetByUsernameOrEmailFunc(identifier)
	}
	return nil, nil
}

// GetByID implements UserRepositoryInterface.
func (m *MockUserRepository) GetByID(id int) (*models.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return nil, nil
}

// Count implements UserRepositoryInterface.
func (m *MockUserRepository) Count() (int64, error) {
	if m.CountFunc != nil {
		return m.CountFunc()
	}
	return 0, nil
}

// Delete implements UserRepositoryInterface.
func (m *MockUserRepository) Delete(id int) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

// UpdatePassword implements UserRepositoryInterface.
func (m *MockUserRepository) UpdatePassword(id int, passwordHash string) error {
	if m.UpdatePasswordFunc != nil {
		return m.UpdatePasswordFunc(id, passwordHash)
	}
	return nil
}

// MockReloader is a mock implementation of ReloaderInterface
type MockReloader struct {
	ValidateFunc       func(ctx context.Context) error
	ReloadFunc         func(ctx context.Context) (*caddy.ReloadResult, error)
	ForceReloadFunc    func(ctx context.Context) (*caddy.ReloadResult, error)
	AdaptAndReloadFunc func(ctx context.Context) (string, error)
	TestConnectionFunc func(ctx context.Context) error
}

// Validate implements ReloaderInterface.
func (m *MockReloader) Validate(ctx context.Context) error {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(ctx)
	}
	return nil
}

// Reload implements ReloaderInterface.
func (m *MockReloader) Reload(ctx context.Context) (*caddy.ReloadResult, error) {
	if m.ReloadFunc != nil {
		return m.ReloadFunc(ctx)
	}
	return &caddy.ReloadResult{Success: true}, nil
}

// ForceReload implements ReloaderInterface.
func (m *MockReloader) ForceReload(ctx context.Context) (*caddy.ReloadResult, error) {
	if m.ForceReloadFunc != nil {
		return m.ForceReloadFunc(ctx)
	}
	return &caddy.ReloadResult{Success: true}, nil
}

// AdaptAndReload implements ReloaderInterface.
func (m *MockReloader) AdaptAndReload(ctx context.Context) (string, error) {
	if m.AdaptAndReloadFunc != nil {
		return m.AdaptAndReloadFunc(ctx)
	}
	return "", nil
}

// TestConnection implements ReloaderInterface.
func (m *MockReloader) TestConnection(ctx context.Context) error {
	if m.TestConnectionFunc != nil {
		return m.TestConnectionFunc(ctx)
	}
	return nil
}

// MockAuditService is a mock implementation of AuditServiceInterface
type MockAuditService struct {
	LogEventFunc              func(ctx context.Context, event models.AuditEvent) error
	GetConfigFunc             func() (*models.AuditConfig, error)
	SetConfigFunc             func(config *models.AuditConfig) error
	InvalidateConfigCacheFunc func()
	ListAuditLogsFunc         func(params repository.AuditLogListParams) (*models.AuditLogListResponse, error)
	GetAuditLogByIDFunc       func(id int) (*models.AuditLog, error)
	GetStatsFunc              func() (*models.AuditLogStats, error)
	LogProxyCreateFunc        func(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error
	LogProxyUpdateFunc        func(ctx context.Context, userID int, proxy *models.Proxy, changes map[string]interface{}, ip, userAgent string) error
	LogProxyDeleteFunc        func(ctx context.Context, userID int, proxyID int, proxyName, hostname string, ip, userAgent string) error
	LogProxyEnableFunc        func(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error
	LogProxyDisableFunc       func(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error
	LogLoginFunc              func(ctx context.Context, userID int, username string, ip, userAgent string) error
	LogLoginFailedFunc        func(ctx context.Context, username, ip, userAgent, reason string) error
	LogLogoutFunc             func(ctx context.Context, userID int, username string, ip, userAgent string) error
	LogRegisterFunc           func(ctx context.Context, userID int, username string, ip, userAgent string) error
	LogPasswordChangeFunc     func(ctx context.Context, userID int, username string, ip, userAgent string) error
	LogSettingsUpdateFunc     func(ctx context.Context, userID int, key string, oldVal, newVal string, ip, userAgent string) error
	LogSyncStartedFunc        func(ctx context.Context) error
	LogSyncCompletedFunc      func(ctx context.Context, proxiesCount int) error
	LogSyncFailedFunc         func(ctx context.Context, errMsg string) error
	LogSystemStartupFunc      func(ctx context.Context) error
	LogCaddyReloadFunc        func(ctx context.Context, success bool, errMsg string) error
	// ACL audit funcs
	LogACLGroupCreateFunc        func(ctx context.Context, userID int, group *models.ACLGroup, ip, userAgent string) error
	LogACLGroupUpdateFunc        func(ctx context.Context, userID int, group *models.ACLGroup, changes map[string]interface{}, ip, userAgent string) error
	LogACLGroupDeleteFunc        func(ctx context.Context, userID int, groupID int, groupName, ip, userAgent string) error
	LogACLIPRuleAddFunc          func(ctx context.Context, userID int, groupID int, groupName string, rule *models.ACLIPRule, ip, userAgent string) error
	LogACLIPRuleUpdateFunc       func(ctx context.Context, userID int, rule *models.ACLIPRule, changes map[string]interface{}, ip, userAgent string) error
	LogACLIPRuleDeleteFunc       func(ctx context.Context, userID int, ruleID int, groupName, cidr, ip, userAgent string) error
	LogACLBasicAuthAddFunc       func(ctx context.Context, userID int, groupID int, groupName, username, ip, userAgent string) error
	LogACLBasicAuthUpdateFunc    func(ctx context.Context, userID int, authUserID int, groupName, username, ip, userAgent string) error
	LogACLBasicAuthDeleteFunc    func(ctx context.Context, userID int, authUserID int, groupName, username, ip, userAgent string) error
	LogACLWaygatesAuthUpdateFunc func(ctx context.Context, userID int, groupID int, groupName string, changes map[string]interface{}, ip, userAgent string) error
	LogACLAssignmentCreateFunc   func(ctx context.Context, userID int, proxyID int, proxyName string, groupID int, groupName, pathPattern, ip, userAgent string) error
	LogACLAssignmentUpdateFunc   func(ctx context.Context, userID int, assignment *models.ProxyACLAssignment, changes map[string]interface{}, ip, userAgent string) error
	LogACLAssignmentDeleteFunc   func(ctx context.Context, userID int, proxyID int, proxyName string, groupID int, groupName, ip, userAgent string) error
	LogACLBrandingUpdateFunc     func(ctx context.Context, userID int, changes map[string]interface{}, ip, userAgent string) error
	LogACLSessionRevokeFunc      func(ctx context.Context, userID int, sessionID int, sessionEmail, ip, userAgent string) error
}

// LogEvent implements AuditServiceInterface.
func (m *MockAuditService) LogEvent(ctx context.Context, event models.AuditEvent) error {
	if m.LogEventFunc != nil {
		return m.LogEventFunc(ctx, event)
	}
	return nil
}

// GetConfig implements AuditServiceInterface.
func (m *MockAuditService) GetConfig() (*models.AuditConfig, error) {
	if m.GetConfigFunc != nil {
		return m.GetConfigFunc()
	}
	return models.DefaultAuditConfig(), nil
}

// SetConfig implements AuditServiceInterface.
func (m *MockAuditService) SetConfig(config *models.AuditConfig) error {
	if m.SetConfigFunc != nil {
		return m.SetConfigFunc(config)
	}
	return nil
}

// InvalidateConfigCache implements AuditServiceInterface.
func (m *MockAuditService) InvalidateConfigCache() {
	if m.InvalidateConfigCacheFunc != nil {
		m.InvalidateConfigCacheFunc()
	}
}

// ListAuditLogs implements AuditServiceInterface.
func (m *MockAuditService) ListAuditLogs(params repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
	if m.ListAuditLogsFunc != nil {
		return m.ListAuditLogsFunc(params)
	}
	return &models.AuditLogListResponse{Items: []models.AuditLog{}, Total: 0}, nil
}

// GetAuditLogByID implements AuditServiceInterface.
func (m *MockAuditService) GetAuditLogByID(id int) (*models.AuditLog, error) {
	if m.GetAuditLogByIDFunc != nil {
		return m.GetAuditLogByIDFunc(id)
	}
	return nil, nil
}

// GetStats implements AuditServiceInterface.
func (m *MockAuditService) GetStats() (*models.AuditLogStats, error) {
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc()
	}
	return &models.AuditLogStats{}, nil
}

// LogProxyCreate implements AuditServiceInterface.
func (m *MockAuditService) LogProxyCreate(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error {
	if m.LogProxyCreateFunc != nil {
		return m.LogProxyCreateFunc(ctx, userID, proxy, ip, userAgent)
	}
	return nil
}

// LogProxyUpdate implements AuditServiceInterface.
func (m *MockAuditService) LogProxyUpdate(ctx context.Context, userID int, proxy *models.Proxy, changes map[string]interface{}, ip, userAgent string) error {
	if m.LogProxyUpdateFunc != nil {
		return m.LogProxyUpdateFunc(ctx, userID, proxy, changes, ip, userAgent)
	}
	return nil
}

// LogProxyDelete implements AuditServiceInterface.
func (m *MockAuditService) LogProxyDelete(ctx context.Context, userID int, proxyID int, proxyName, hostname string, ip, userAgent string) error {
	if m.LogProxyDeleteFunc != nil {
		return m.LogProxyDeleteFunc(ctx, userID, proxyID, proxyName, hostname, ip, userAgent)
	}
	return nil
}

// LogProxyEnable implements AuditServiceInterface.
func (m *MockAuditService) LogProxyEnable(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error {
	if m.LogProxyEnableFunc != nil {
		return m.LogProxyEnableFunc(ctx, userID, proxy, ip, userAgent)
	}
	return nil
}

// LogProxyDisable implements AuditServiceInterface.
func (m *MockAuditService) LogProxyDisable(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error {
	if m.LogProxyDisableFunc != nil {
		return m.LogProxyDisableFunc(ctx, userID, proxy, ip, userAgent)
	}
	return nil
}

// LogLogin implements AuditServiceInterface.
func (m *MockAuditService) LogLogin(ctx context.Context, userID int, username string, ip, userAgent string) error {
	if m.LogLoginFunc != nil {
		return m.LogLoginFunc(ctx, userID, username, ip, userAgent)
	}
	return nil
}

// LogLoginFailed implements AuditServiceInterface.
func (m *MockAuditService) LogLoginFailed(ctx context.Context, username, ip, userAgent, reason string) error {
	if m.LogLoginFailedFunc != nil {
		return m.LogLoginFailedFunc(ctx, username, ip, userAgent, reason)
	}
	return nil
}

// LogLogout implements AuditServiceInterface.
func (m *MockAuditService) LogLogout(ctx context.Context, userID int, username string, ip, userAgent string) error {
	if m.LogLogoutFunc != nil {
		return m.LogLogoutFunc(ctx, userID, username, ip, userAgent)
	}
	return nil
}

// LogRegister implements AuditServiceInterface.
func (m *MockAuditService) LogRegister(ctx context.Context, userID int, username string, ip, userAgent string) error {
	if m.LogRegisterFunc != nil {
		return m.LogRegisterFunc(ctx, userID, username, ip, userAgent)
	}
	return nil
}

// LogPasswordChange implements AuditServiceInterface.
func (m *MockAuditService) LogPasswordChange(ctx context.Context, userID int, username string, ip, userAgent string) error {
	if m.LogPasswordChangeFunc != nil {
		return m.LogPasswordChangeFunc(ctx, userID, username, ip, userAgent)
	}
	return nil
}

// LogSettingsUpdate implements AuditServiceInterface.
func (m *MockAuditService) LogSettingsUpdate(ctx context.Context, userID int, key string, oldVal, newVal string, ip, userAgent string) error {
	if m.LogSettingsUpdateFunc != nil {
		return m.LogSettingsUpdateFunc(ctx, userID, key, oldVal, newVal, ip, userAgent)
	}
	return nil
}

// LogSyncStarted implements AuditServiceInterface.
func (m *MockAuditService) LogSyncStarted(ctx context.Context) error {
	if m.LogSyncStartedFunc != nil {
		return m.LogSyncStartedFunc(ctx)
	}
	return nil
}

// LogSyncCompleted implements AuditServiceInterface.
func (m *MockAuditService) LogSyncCompleted(ctx context.Context, proxiesCount int) error {
	if m.LogSyncCompletedFunc != nil {
		return m.LogSyncCompletedFunc(ctx, proxiesCount)
	}
	return nil
}

// LogSyncFailed implements AuditServiceInterface.
func (m *MockAuditService) LogSyncFailed(ctx context.Context, errMsg string) error {
	if m.LogSyncFailedFunc != nil {
		return m.LogSyncFailedFunc(ctx, errMsg)
	}
	return nil
}

// LogSystemStartup implements AuditServiceInterface.
func (m *MockAuditService) LogSystemStartup(ctx context.Context) error {
	if m.LogSystemStartupFunc != nil {
		return m.LogSystemStartupFunc(ctx)
	}
	return nil
}

// LogCaddyReload implements AuditServiceInterface.
func (m *MockAuditService) LogCaddyReload(ctx context.Context, success bool, errMsg string) error {
	if m.LogCaddyReloadFunc != nil {
		return m.LogCaddyReloadFunc(ctx, success, errMsg)
	}
	return nil
}

// LogACLGroupCreate implements AuditServiceInterface.
func (m *MockAuditService) LogACLGroupCreate(ctx context.Context, userID int, group *models.ACLGroup, ip, userAgent string) error {
	if m.LogACLGroupCreateFunc != nil {
		return m.LogACLGroupCreateFunc(ctx, userID, group, ip, userAgent)
	}
	return nil
}

// LogACLGroupUpdate implements AuditServiceInterface.
func (m *MockAuditService) LogACLGroupUpdate(ctx context.Context, userID int, group *models.ACLGroup, changes map[string]interface{}, ip, userAgent string) error {
	if m.LogACLGroupUpdateFunc != nil {
		return m.LogACLGroupUpdateFunc(ctx, userID, group, changes, ip, userAgent)
	}
	return nil
}

// LogACLGroupDelete implements AuditServiceInterface.
func (m *MockAuditService) LogACLGroupDelete(ctx context.Context, userID int, groupID int, groupName, ip, userAgent string) error {
	if m.LogACLGroupDeleteFunc != nil {
		return m.LogACLGroupDeleteFunc(ctx, userID, groupID, groupName, ip, userAgent)
	}
	return nil
}

// LogACLIPRuleAdd implements AuditServiceInterface.
func (m *MockAuditService) LogACLIPRuleAdd(ctx context.Context, userID int, groupID int, groupName string, rule *models.ACLIPRule, ip, userAgent string) error {
	if m.LogACLIPRuleAddFunc != nil {
		return m.LogACLIPRuleAddFunc(ctx, userID, groupID, groupName, rule, ip, userAgent)
	}
	return nil
}

// LogACLIPRuleUpdate implements AuditServiceInterface.
func (m *MockAuditService) LogACLIPRuleUpdate(ctx context.Context, userID int, rule *models.ACLIPRule, changes map[string]interface{}, ip, userAgent string) error {
	if m.LogACLIPRuleUpdateFunc != nil {
		return m.LogACLIPRuleUpdateFunc(ctx, userID, rule, changes, ip, userAgent)
	}
	return nil
}

// LogACLIPRuleDelete implements AuditServiceInterface.
func (m *MockAuditService) LogACLIPRuleDelete(ctx context.Context, userID int, ruleID int, groupName, cidr, ip, userAgent string) error {
	if m.LogACLIPRuleDeleteFunc != nil {
		return m.LogACLIPRuleDeleteFunc(ctx, userID, ruleID, groupName, cidr, ip, userAgent)
	}
	return nil
}

// LogACLBasicAuthAdd implements AuditServiceInterface.
func (m *MockAuditService) LogACLBasicAuthAdd(ctx context.Context, userID int, groupID int, groupName, username, ip, userAgent string) error {
	if m.LogACLBasicAuthAddFunc != nil {
		return m.LogACLBasicAuthAddFunc(ctx, userID, groupID, groupName, username, ip, userAgent)
	}
	return nil
}

// LogACLBasicAuthUpdate implements AuditServiceInterface.
func (m *MockAuditService) LogACLBasicAuthUpdate(ctx context.Context, userID int, authUserID int, groupName, username, ip, userAgent string) error {
	if m.LogACLBasicAuthUpdateFunc != nil {
		return m.LogACLBasicAuthUpdateFunc(ctx, userID, authUserID, groupName, username, ip, userAgent)
	}
	return nil
}

// LogACLBasicAuthDelete implements AuditServiceInterface.
func (m *MockAuditService) LogACLBasicAuthDelete(ctx context.Context, userID int, authUserID int, groupName, username, ip, userAgent string) error {
	if m.LogACLBasicAuthDeleteFunc != nil {
		return m.LogACLBasicAuthDeleteFunc(ctx, userID, authUserID, groupName, username, ip, userAgent)
	}
	return nil
}

// LogACLWaygatesAuthUpdate implements AuditServiceInterface.
func (m *MockAuditService) LogACLWaygatesAuthUpdate(ctx context.Context, userID int, groupID int, groupName string, changes map[string]interface{}, ip, userAgent string) error {
	if m.LogACLWaygatesAuthUpdateFunc != nil {
		return m.LogACLWaygatesAuthUpdateFunc(ctx, userID, groupID, groupName, changes, ip, userAgent)
	}
	return nil
}

// LogACLAssignmentCreate implements AuditServiceInterface.
func (m *MockAuditService) LogACLAssignmentCreate(ctx context.Context, userID int, proxyID int, proxyName string, groupID int, groupName, pathPattern, ip, userAgent string) error {
	if m.LogACLAssignmentCreateFunc != nil {
		return m.LogACLAssignmentCreateFunc(ctx, userID, proxyID, proxyName, groupID, groupName, pathPattern, ip, userAgent)
	}
	return nil
}

// LogACLAssignmentUpdate implements AuditServiceInterface.
func (m *MockAuditService) LogACLAssignmentUpdate(ctx context.Context, userID int, assignment *models.ProxyACLAssignment, changes map[string]interface{}, ip, userAgent string) error {
	if m.LogACLAssignmentUpdateFunc != nil {
		return m.LogACLAssignmentUpdateFunc(ctx, userID, assignment, changes, ip, userAgent)
	}
	return nil
}

// LogACLAssignmentDelete implements AuditServiceInterface.
func (m *MockAuditService) LogACLAssignmentDelete(ctx context.Context, userID int, proxyID int, proxyName string, groupID int, groupName, ip, userAgent string) error {
	if m.LogACLAssignmentDeleteFunc != nil {
		return m.LogACLAssignmentDeleteFunc(ctx, userID, proxyID, proxyName, groupID, groupName, ip, userAgent)
	}
	return nil
}

// LogACLBrandingUpdate implements AuditServiceInterface.
func (m *MockAuditService) LogACLBrandingUpdate(ctx context.Context, userID int, changes map[string]interface{}, ip, userAgent string) error {
	if m.LogACLBrandingUpdateFunc != nil {
		return m.LogACLBrandingUpdateFunc(ctx, userID, changes, ip, userAgent)
	}
	return nil
}

// LogACLSessionRevoke implements AuditServiceInterface.
func (m *MockAuditService) LogACLSessionRevoke(ctx context.Context, userID int, sessionID int, sessionEmail, ip, userAgent string) error {
	if m.LogACLSessionRevokeFunc != nil {
		return m.LogACLSessionRevokeFunc(ctx, userID, sessionID, sessionEmail, ip, userAgent)
	}
	return nil
}

// Ensure mocks implement interfaces
var (
	_ service.ProxyServiceInterface      = (*MockProxyService)(nil)
	_ service.SettingsServiceInterface   = (*MockSettingsService)(nil)
	_ service.SyncServiceInterface       = (*MockSyncService)(nil)
	_ service.AuditServiceInterface      = (*MockAuditService)(nil)
	_ repository.UserRepositoryInterface = (*MockUserRepository)(nil)
	_ caddy.ReloaderInterface            = (*MockReloader)(nil)
)
