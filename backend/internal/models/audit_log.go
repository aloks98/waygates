package models

import (
	"time"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       *int      `json:"user_id" gorm:"index"`
	Action       string    `json:"action" gorm:"type:varchar(100);not null;index"`
	ResourceType *string   `json:"resource_type" gorm:"type:varchar(50);index"`
	ResourceID   *int      `json:"resource_id" gorm:"index"`
	ResourceName *string   `json:"resource_name" gorm:"type:varchar(255)"`
	Details      JSONField `json:"details" gorm:"type:text"`
	IPAddress    *string   `json:"ip_address" gorm:"type:varchar(45)"`
	UserAgent    *string   `json:"user_agent" gorm:"type:text"`
	Status       string    `json:"status" gorm:"type:varchar(20);not null;default:'success';index"`
	ErrorMessage *string   `json:"error_message" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime;index"`

	// Relations
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for GORM
func (AuditLog) TableName() string {
	return "audit_logs"
}

// AuditLogListResponse is the response for listing audit logs
type AuditLogListResponse struct {
	Items      []AuditLog `json:"items"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	TotalPages int        `json:"total_pages"`
}

// AuditLogStats holds aggregate statistics for audit logs
type AuditLogStats struct {
	TotalLogs      int64            `json:"total_logs"`
	ByAction       map[string]int64 `json:"by_action"`
	ByStatus       map[string]int64 `json:"by_status"`
	ByResourceType map[string]int64 `json:"by_resource_type"`
	RecentActivity []AuditLog       `json:"recent_activity"`
}

// AuditConfig represents the audit logging configuration with fine-grained event control
type AuditConfig struct {
	// Proxy events
	ProxyCreate  bool `json:"proxy_create"`
	ProxyUpdate  bool `json:"proxy_update"`
	ProxyDelete  bool `json:"proxy_delete"`
	ProxyEnable  bool `json:"proxy_enable"`
	ProxyDisable bool `json:"proxy_disable"`

	// Auth events
	AuthLogin          bool `json:"auth_login"`
	AuthLogout         bool `json:"auth_logout"`
	AuthRegister       bool `json:"auth_register"`
	AuthPasswordChange bool `json:"auth_password_change"`
	AuthLoginFailed    bool `json:"auth_login_failed"`

	// Settings events
	SettingsUpdate bool `json:"settings_update"`

	// Sync events
	SyncStarted   bool `json:"sync_started"`
	SyncCompleted bool `json:"sync_completed"`
	SyncFailed    bool `json:"sync_failed"`

	// System events
	SystemStartup bool `json:"system_startup"`
	CaddyReload   bool `json:"caddy_reload"`
}

// DefaultAuditConfig returns the default audit configuration with all events enabled
func DefaultAuditConfig() *AuditConfig {
	return &AuditConfig{
		ProxyCreate:        true,
		ProxyUpdate:        true,
		ProxyDelete:        true,
		ProxyEnable:        true,
		ProxyDisable:       true,
		AuthLogin:          true,
		AuthLogout:         true,
		AuthRegister:       true,
		AuthPasswordChange: true,
		AuthLoginFailed:    true,
		SettingsUpdate:     true,
		SyncStarted:        true,
		SyncCompleted:      true,
		SyncFailed:         true,
		SystemStartup:      true,
		CaddyReload:        true,
	}
}

// AuditEvent contains all data for an audit log entry
type AuditEvent struct {
	UserID       *int
	Action       string
	ResourceType string
	ResourceID   *int
	ResourceName string
	Details      map[string]interface{}
	IPAddress    string
	UserAgent    string
	Status       string // "success" or "failure"
	ErrorMessage string
}

// Audit action constants
const (
	// Proxy actions
	AuditActionProxyCreate  = "proxy.create"
	AuditActionProxyUpdate  = "proxy.update"
	AuditActionProxyDelete  = "proxy.delete"
	AuditActionProxyEnable  = "proxy.enable"
	AuditActionProxyDisable = "proxy.disable"

	// Auth actions
	AuditActionAuthLogin          = "auth.login"
	AuditActionAuthLogout         = "auth.logout"
	AuditActionAuthRegister       = "auth.register"
	AuditActionAuthPasswordChange = "auth.password_change"
	AuditActionAuthLoginFailed    = "auth.login_failed"

	// Settings actions
	AuditActionSettingsUpdate = "settings.update"

	// Sync actions
	AuditActionSyncStarted   = "sync.started"
	AuditActionSyncCompleted = "sync.completed"
	AuditActionSyncFailed    = "sync.failed"

	// System actions
	AuditActionSystemStartup = "system.startup"
	AuditActionCaddyReload   = "caddy.reload"
)

// Audit status constants
const (
	AuditStatusSuccess = "success"
	AuditStatusFailure = "failure"
)

// Resource type constants
const (
	AuditResourceTypeProxy    = "proxy"
	AuditResourceTypeUser     = "user"
	AuditResourceTypeSettings = "settings"
	AuditResourceTypeSystem   = "system"
)

// SettingAuditConfig is the settings key for audit configuration
const SettingAuditConfig = "audit_config"
