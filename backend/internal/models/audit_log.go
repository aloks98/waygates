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

	// ACL Group events
	ACLGroupCreate bool `json:"acl_group_create"`
	ACLGroupUpdate bool `json:"acl_group_update"`
	ACLGroupDelete bool `json:"acl_group_delete"`

	// ACL IP Rule events
	ACLIPRuleAdd    bool `json:"acl_ip_rule_add"`
	ACLIPRuleUpdate bool `json:"acl_ip_rule_update"`
	ACLIPRuleDelete bool `json:"acl_ip_rule_delete"`

	// ACL Basic Auth events
	ACLBasicAuthAdd    bool `json:"acl_basic_auth_add"`
	ACLBasicAuthUpdate bool `json:"acl_basic_auth_update"`
	ACLBasicAuthDelete bool `json:"acl_basic_auth_delete"`

	// ACL Waygates Auth events
	ACLWaygatesAuthUpdate bool `json:"acl_waygates_auth_update"`

	// ACL Proxy Assignment events
	ACLAssignmentCreate bool `json:"acl_assignment_create"`
	ACLAssignmentUpdate bool `json:"acl_assignment_update"`
	ACLAssignmentDelete bool `json:"acl_assignment_delete"`

	// ACL Branding events
	ACLBrandingUpdate bool `json:"acl_branding_update"`

	// ACL Session events
	ACLSessionRevoke bool `json:"acl_session_revoke"`
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
		// ACL events
		ACLGroupCreate:        true,
		ACLGroupUpdate:        true,
		ACLGroupDelete:        true,
		ACLIPRuleAdd:          true,
		ACLIPRuleUpdate:       true,
		ACLIPRuleDelete:       true,
		ACLBasicAuthAdd:       true,
		ACLBasicAuthUpdate:    true,
		ACLBasicAuthDelete:    true,
		ACLWaygatesAuthUpdate: true,
		ACLAssignmentCreate:   true,
		ACLAssignmentUpdate:   true,
		ACLAssignmentDelete:   true,
		ACLBrandingUpdate:     true,
		ACLSessionRevoke:      true,
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

	// ACL Group Events
	AuditActionACLGroupCreate = "acl_group.create"
	AuditActionACLGroupUpdate = "acl_group.update"
	AuditActionACLGroupDelete = "acl_group.delete"

	// ACL IP Rule Events
	AuditActionACLIPRuleAdd    = "acl_ip_rule.add"
	AuditActionACLIPRuleUpdate = "acl_ip_rule.update"
	AuditActionACLIPRuleDelete = "acl_ip_rule.delete"

	// ACL Basic Auth Events
	AuditActionACLBasicAuthAdd    = "acl_basic_auth.add"
	AuditActionACLBasicAuthUpdate = "acl_basic_auth.update"
	AuditActionACLBasicAuthDelete = "acl_basic_auth.delete"

	// ACL Waygates Auth Events
	AuditActionACLWaygatesAuthUpdate = "acl_waygates_auth.update"

	// ACL Proxy Assignment Events
	AuditActionACLAssignmentCreate = "acl_assignment.create"
	AuditActionACLAssignmentUpdate = "acl_assignment.update"
	AuditActionACLAssignmentDelete = "acl_assignment.delete"

	// ACL Branding Events
	AuditActionACLBrandingUpdate = "acl_branding.update"

	// ACL Session Events
	AuditActionACLSessionRevoke = "acl_session.revoke"
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
	AuditResourceTypeACL      = "acl"
)

// SettingAuditConfig is the settings key for audit configuration
const SettingAuditConfig = "audit_config"

// AuditEventDefinition represents a single audit event type
type AuditEventDefinition struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// AuditEventGroup represents a group of related audit events
type AuditEventGroup struct {
	Key         string                 `json:"key"`
	Label       string                 `json:"label"`
	Description string                 `json:"description"`
	Events      []AuditEventDefinition `json:"events"`
}

// GetAuditEventGroups returns all audit event groups for configuration UI
func GetAuditEventGroups() []AuditEventGroup {
	return []AuditEventGroup{
		{
			Key:         "proxy",
			Label:       "Proxy Events",
			Description: "Proxy configuration changes",
			Events: []AuditEventDefinition{
				{Key: "proxy_create", Label: "Create"},
				{Key: "proxy_update", Label: "Update"},
				{Key: "proxy_delete", Label: "Delete"},
				{Key: "proxy_enable", Label: "Enable"},
				{Key: "proxy_disable", Label: "Disable"},
			},
		},
		{
			Key:         "auth",
			Label:       "Authentication Events",
			Description: "User authentication activities",
			Events: []AuditEventDefinition{
				{Key: "auth_login", Label: "Login"},
				{Key: "auth_logout", Label: "Logout"},
				{Key: "auth_register", Label: "Register"},
				{Key: "auth_password_change", Label: "Password Change"},
				{Key: "auth_login_failed", Label: "Failed Login"},
			},
		},
		{
			Key:         "settings",
			Label:       "Settings Events",
			Description: "Configuration changes",
			Events: []AuditEventDefinition{
				{Key: "settings_update", Label: "Settings Update"},
			},
		},
		{
			Key:         "sync",
			Label:       "Sync Events",
			Description: "Caddy synchronization operations",
			Events: []AuditEventDefinition{
				{Key: "sync_started", Label: "Started"},
				{Key: "sync_completed", Label: "Completed"},
				{Key: "sync_failed", Label: "Failed"},
			},
		},
		{
			Key:         "system",
			Label:       "System Events",
			Description: "System and Caddy operations",
			Events: []AuditEventDefinition{
				{Key: "system_startup", Label: "System Startup"},
				{Key: "caddy_reload", Label: "Caddy Reload"},
			},
		},
		{
			Key:         "acl",
			Label:       "ACL Events",
			Description: "Access control list configuration changes",
			Events: []AuditEventDefinition{
				{Key: "acl_group_create", Label: "ACL Group Created"},
				{Key: "acl_group_update", Label: "ACL Group Updated"},
				{Key: "acl_group_delete", Label: "ACL Group Deleted"},
				{Key: "acl_ip_rule_add", Label: "IP Rule Added"},
				{Key: "acl_ip_rule_update", Label: "IP Rule Updated"},
				{Key: "acl_ip_rule_delete", Label: "IP Rule Deleted"},
				{Key: "acl_basic_auth_add", Label: "Basic Auth User Added"},
				{Key: "acl_basic_auth_update", Label: "Basic Auth Updated"},
				{Key: "acl_basic_auth_delete", Label: "Basic Auth User Deleted"},
				{Key: "acl_waygates_auth_update", Label: "Waygates Auth Updated"},
				{Key: "acl_assignment_create", Label: "ACL Assigned to Proxy"},
				{Key: "acl_assignment_update", Label: "ACL Assignment Updated"},
				{Key: "acl_assignment_delete", Label: "ACL Removed from Proxy"},
				{Key: "acl_branding_update", Label: "Branding Updated"},
				{Key: "acl_session_revoke", Label: "Session Revoked"},
			},
		},
	}
}
