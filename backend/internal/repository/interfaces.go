package repository

import (
	"time"

	"github.com/aloks98/waygates/backend/internal/models"
)

// ProxyRepositoryInterface defines the interface for proxy database operations
type ProxyRepositoryInterface interface {
	List(params ProxyListParams) ([]models.Proxy, int64, error)
	GetByID(id int) (*models.Proxy, error)
	GetByIDs(ids []int) ([]models.Proxy, error)
	GetByHostname(hostname string) (*models.Proxy, error)
	ExistingHostnames(hostnames []string) (map[string]bool, error)
	Create(proxy *models.Proxy) error
	Update(proxy *models.Proxy) error
	Delete(id int) error
	UpdateStatus(id int, isActive bool) error
	HostnameExists(hostname string, excludeID int) (bool, error)
	GetStats() (*ProxyStats, error)
}

// UserRepositoryInterface defines the interface for user database operations
type UserRepositoryInterface interface {
	Create(user *models.User) error
	GetByEmail(email string) (*models.User, error)
	GetByUsernameOrEmail(identifier string) (*models.User, error)
	GetByID(id int) (*models.User, error)
	Count() (int64, error)
	Delete(id int) error
	UpdatePassword(id int, passwordHash string) error
}

// SettingsRepositoryInterface defines the interface for settings database operations
type SettingsRepositoryInterface interface {
	Get(key string) (*models.Setting, error)
	GetValue(key, defaultValue string) string
	Set(key, value string) error
	GetAll() (map[string]string, error)
	Delete(key string) error
	GetNotFoundSettings() (*models.NotFoundSettings, error)
	SetNotFoundSettings(settings *models.NotFoundSettings) error
	GetMetricsPublishSettings() (*models.MetricsPublishSettings, error)
	SetMetricsPublishSettings(settings *models.MetricsPublishSettings) error
}

// AuditLogRepositoryInterface defines the interface for audit log database operations
type AuditLogRepositoryInterface interface {
	Create(log *models.AuditLog) error
	GetByID(id int) (*models.AuditLog, error)
	List(params AuditLogListParams) ([]models.AuditLog, int64, error)
	GetStats() (*models.AuditLogStats, error)
	DeleteOlderThan(before time.Time) (int64, error)
	CountByAction(action string) (int64, error)
	CountByUserID(userID int) (int64, error)
}

// L4ProxyRepositoryInterface defines the interface for L4 proxy database operations
type L4ProxyRepositoryInterface interface {
	Create(proxy *models.L4Proxy) error
	GetByID(id int) (*models.L4Proxy, error)
	GetByIDs(ids []int) ([]models.L4Proxy, error)
	List(params L4ProxyListParams) ([]models.L4Proxy, int64, error)
	Update(proxy *models.L4Proxy) error
	Delete(id int) error
	GetByPort(port int, protocol string) (*models.L4Proxy, error)
}

// TrafficSampleRepositoryInterface defines the interface for traffic sample operations.
type TrafficSampleRepositoryInterface interface {
	Create(sample *models.TrafficSample) error
	ListSince(t time.Time) ([]models.TrafficSample, error)
	DeleteOlderThan(t time.Time) (int64, error)
}

// L4ProxyListParams holds parameters for listing L4 proxies
type L4ProxyListParams struct {
	Page     int
	Limit    int
	Search   string // Search in name and description
	Protocol string // Filter by protocol: "tcp" or "udp"
	IsActive *bool  // Filter by active status (nil = no filter)
	Sort     string // Sort field: id, name, listen_port, protocol, is_active, created_at, updated_at
	Order    string // Sort order: asc, desc
}

// Ensure repositories implement interfaces
var (
	_ ProxyRepositoryInterface         = (*ProxyRepository)(nil)
	_ UserRepositoryInterface          = (*UserRepository)(nil)
	_ SettingsRepositoryInterface      = (*SettingsRepository)(nil)
	_ AuditLogRepositoryInterface      = (*AuditLogRepository)(nil)
	_ ACLRepositoryInterface           = (*ACLRepository)(nil)
	_ L4ProxyRepositoryInterface       = (*L4ProxyRepository)(nil)
	_ TrafficSampleRepositoryInterface = (*TrafficSampleRepository)(nil)
)
