package service

import (
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
}

// ProxySyncer defines the interface for proxy sync operations used by ProxyService
type ProxySyncer interface {
	SyncProxy(proxy *models.Proxy) error
	RemoveProxy(proxyID int, hostname string) error
	EnableProxy(proxyID int, hostname string) error
	DisableProxy(proxyID int, hostname string) error
}

// Ensure concrete types implement interfaces
var (
	_ ProxyServiceInterface    = (*ProxyService)(nil)
	_ SettingsServiceInterface = (*SettingsService)(nil)
	_ SyncServiceInterface     = (*SyncService)(nil)
	_ ProxySyncer              = (*SyncService)(nil)
)
