package repository

import (
	"time"

	"github.com/aloks98/waygates/backend/internal/models"
)

// ProxyRepositoryInterface defines the interface for proxy database operations
type ProxyRepositoryInterface interface {
	List(params ProxyListParams) ([]models.Proxy, int64, error)
	GetByID(id int) (*models.Proxy, error)
	GetByHostname(hostname string) (*models.Proxy, error)
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

// Ensure repositories implement interfaces
var (
	_ ProxyRepositoryInterface    = (*ProxyRepository)(nil)
	_ UserRepositoryInterface     = (*UserRepository)(nil)
	_ SettingsRepositoryInterface = (*SettingsRepository)(nil)
	_ AuditLogRepositoryInterface = (*AuditLogRepository)(nil)
	_ ACLRepositoryInterface      = (*ACLRepository)(nil)
)
