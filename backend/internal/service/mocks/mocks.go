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
	GetStatusFunc func() service.SyncStatus
	FullSyncFunc  func() error
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

// MockUserRepository is a mock implementation of UserRepositoryInterface
type MockUserRepository struct {
	CreateFunc              func(user *models.User) error
	GetByEmailFunc          func(email string) (*models.User, error)
	GetByUsernameOrEmailFunc func(identifier string) (*models.User, error)
	GetByIDFunc             func(id int) (*models.User, error)
	CountFunc               func() (int64, error)
	DeleteFunc              func(id int) error
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

// Ensure mocks implement interfaces
var (
	_ service.ProxyServiceInterface      = (*MockProxyService)(nil)
	_ service.SettingsServiceInterface   = (*MockSettingsService)(nil)
	_ service.SyncServiceInterface       = (*MockSyncService)(nil)
	_ repository.UserRepositoryInterface = (*MockUserRepository)(nil)
	_ caddy.ReloaderInterface            = (*MockReloader)(nil)
)
