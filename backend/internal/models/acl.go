package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ACL Combination Mode constants
const (
	ACLCombinationModeAny      = "any"       // Any auth method grants access
	ACLCombinationModeAll      = "all"       // All auth methods must pass
	ACLCombinationModeIPBypass = "ip_bypass" // IP rules can bypass other auth
)

// ACL IP Rule Type constants
const (
	ACLIPRuleTypeAllow  = "allow"
	ACLIPRuleTypeDeny   = "deny"
	ACLIPRuleTypeBypass = "bypass"
)

// ACL External Provider Type constants
const (
	ACLProviderTypeAuthelia  = "authelia"
	ACLProviderTypeAuthentik = "authentik"
	ACLProviderTypeCustom    = "custom"
)

// JSONStringArray is a custom type for storing JSON string arrays in database
type JSONStringArray []string

// Value implements the driver.Valuer interface for database storage
func (j JSONStringArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for database retrieval
func (j *JSONStringArray) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}

	if len(bytes) == 0 {
		*j = nil
		return nil
	}

	var result []string
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}

// ACLGroup represents a reusable ACL group
type ACLGroup struct {
	ID              int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name            string    `json:"name" gorm:"type:varchar(255);uniqueIndex;not null"`
	Description     *string   `json:"description,omitempty" gorm:"type:text"`
	CombinationMode string    `json:"combination_mode" gorm:"type:varchar(20);not null;default:'any'"`
	CreatedBy       int       `json:"-" gorm:"not null"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Relations
	Creator           *User                 `json:"created_by,omitempty" gorm:"foreignKey:CreatedBy"`
	IPRules           []ACLIPRule           `json:"ip_rules,omitempty" gorm:"foreignKey:ACLGroupID"`
	BasicAuthUsers    []ACLBasicAuthUser    `json:"basic_auth_users,omitempty" gorm:"foreignKey:ACLGroupID"`
	ExternalProviders []ACLExternalProvider `json:"external_providers,omitempty" gorm:"foreignKey:ACLGroupID"`
	WaygatesAuth      *ACLWaygatesAuth      `json:"waygates_auth,omitempty" gorm:"foreignKey:ACLGroupID"`
}

// TableName specifies the table name for GORM
func (ACLGroup) TableName() string {
	return "acl_groups"
}

// ACLIPRule represents an IP whitelist/blacklist/bypass rule
type ACLIPRule struct {
	ID          int       `json:"id" gorm:"primaryKey;autoIncrement"`
	ACLGroupID  int       `json:"acl_group_id" gorm:"not null;index"`
	RuleType    string    `json:"rule_type" gorm:"type:varchar(20);not null"`
	CIDR        string    `json:"cidr" gorm:"type:varchar(50);not null"`
	Description *string   `json:"description,omitempty" gorm:"type:text"`
	Priority    int       `json:"priority" gorm:"not null;default:0"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`

	// Relations
	ACLGroup *ACLGroup `json:"-" gorm:"foreignKey:ACLGroupID"`
}

// TableName specifies the table name for GORM
func (ACLIPRule) TableName() string {
	return "acl_ip_rules"
}

// ACLBasicAuthUser represents basic auth credentials for an ACL group
type ACLBasicAuthUser struct {
	ID           int       `json:"id" gorm:"primaryKey;autoIncrement"`
	ACLGroupID   int       `json:"acl_group_id" gorm:"not null;index;uniqueIndex:uq_acl_basic_auth_users_group_username"`
	Username     string    `json:"username" gorm:"type:varchar(255);not null;uniqueIndex:uq_acl_basic_auth_users_group_username"`
	PasswordHash string    `json:"-" gorm:"type:varchar(255);not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Relations
	ACLGroup *ACLGroup `json:"-" gorm:"foreignKey:ACLGroupID"`
}

// TableName specifies the table name for GORM
func (ACLBasicAuthUser) TableName() string {
	return "acl_basic_auth_users"
}

// SetPassword hashes the password and sets it on the user model
func (u *ACLBasicAuthUser) SetPassword(password string, cost int) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword verifies if the provided password matches the hash
func (u *ACLBasicAuthUser) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// ACLExternalProvider represents an external auth provider config
type ACLExternalProvider struct {
	ID              int             `json:"id" gorm:"primaryKey;autoIncrement"`
	ACLGroupID      int             `json:"acl_group_id" gorm:"not null;index"`
	ProviderType    string          `json:"provider_type" gorm:"type:varchar(50);not null"`
	Name            string          `json:"name" gorm:"type:varchar(255);not null"`
	VerifyURL       string          `json:"verify_url" gorm:"type:varchar(500);not null"`
	AuthRedirectURL *string         `json:"auth_redirect_url,omitempty" gorm:"type:varchar(500)"`
	HeadersToCopy   JSONStringArray `json:"headers_to_copy,omitempty" gorm:"type:text"`
	CreatedAt       time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time       `json:"updated_at" gorm:"autoUpdateTime"`

	// Relations
	ACLGroup *ACLGroup `json:"-" gorm:"foreignKey:ACLGroupID"`
}

// TableName specifies the table name for GORM
func (ACLExternalProvider) TableName() string {
	return "acl_external_providers"
}

// ACLWaygatesAuth represents Waygates login config for an ACL group
type ACLWaygatesAuth struct {
	ID                   int             `json:"id" gorm:"primaryKey;autoIncrement"`
	ACLGroupID           int             `json:"acl_group_id" gorm:"uniqueIndex;not null"`
	Enabled              bool            `json:"enabled" gorm:"not null;default:true"`
	AllowedUsers         JSONStringArray `json:"allowed_users,omitempty" gorm:"type:text"`
	AllowedRoles         JSONStringArray `json:"allowed_roles,omitempty" gorm:"type:text"`
	AllowedEmailPatterns JSONStringArray `json:"allowed_email_patterns,omitempty" gorm:"type:text"`
	Require2FA           bool            `json:"require_2fa" gorm:"default:false"`
	SessionTTL           int             `json:"session_ttl" gorm:"default:86400"`
	CreatedAt            time.Time       `json:"created_at" gorm:"autoCreateTime"`

	// Relations
	ACLGroup *ACLGroup `json:"-" gorm:"foreignKey:ACLGroupID"`
}

// TableName specifies the table name for GORM
func (ACLWaygatesAuth) TableName() string {
	return "acl_waygates_auth"
}

// ProxyACLAssignment represents a proxy-to-ACL-group mapping
type ProxyACLAssignment struct {
	ID          int       `json:"id" gorm:"primaryKey;autoIncrement"`
	ProxyID     int       `json:"proxy_id" gorm:"not null;index;uniqueIndex:uq_proxy_acl_assignments_proxy_path"`
	ACLGroupID  int       `json:"acl_group_id" gorm:"not null;index"`
	PathPattern string    `json:"path_pattern" gorm:"type:varchar(500);not null;default:'/*';uniqueIndex:uq_proxy_acl_assignments_proxy_path"`
	Priority    int       `json:"priority" gorm:"not null;default:0"`
	Enabled     bool      `json:"enabled" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`

	// Relations
	Proxy    *Proxy    `json:"proxy,omitempty" gorm:"foreignKey:ProxyID"`
	ACLGroup *ACLGroup `json:"acl_group,omitempty" gorm:"foreignKey:ACLGroupID"`
}

// TableName specifies the table name for GORM
func (ProxyACLAssignment) TableName() string {
	return "proxy_acl_assignments"
}

// ACLBranding represents the login page branding configuration
type ACLBranding struct {
	ID              int       `json:"id" gorm:"primaryKey;autoIncrement"`
	LogoURL         *string   `json:"logo_url,omitempty" gorm:"type:varchar(500)"`
	PrimaryColor    string    `json:"primary_color" gorm:"type:varchar(20);default:'#3b82f6'"`
	BackgroundColor string    `json:"background_color" gorm:"type:varchar(20);default:'#ffffff'"`
	Title           string    `json:"title" gorm:"type:varchar(255);default:'Login Required'"`
	Subtitle        *string   `json:"subtitle,omitempty" gorm:"type:text"`
	FooterText      *string   `json:"footer_text,omitempty" gorm:"type:text"`
	CustomCSS       *string   `json:"custom_css,omitempty" gorm:"type:text"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (ACLBranding) TableName() string {
	return "acl_branding"
}

// ACLSession represents a session token for Waygates auth
type ACLSession struct {
	ID           int       `json:"id" gorm:"primaryKey;autoIncrement"`
	SessionToken string    `json:"-" gorm:"type:varchar(255);uniqueIndex;not null"`
	UserID       int       `json:"user_id" gorm:"not null;index"`
	ProxyID      *int      `json:"proxy_id,omitempty" gorm:"index"`
	IPAddress    *string   `json:"ip_address,omitempty" gorm:"type:varchar(50)"`
	UserAgent    *string   `json:"user_agent,omitempty" gorm:"type:text"`
	ExpiresAt    time.Time `json:"expires_at" gorm:"not null;index"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`

	// Relations
	User  *User  `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Proxy *Proxy `json:"proxy,omitempty" gorm:"foreignKey:ProxyID"`
}

// TableName specifies the table name for GORM
func (ACLSession) TableName() string {
	return "acl_sessions"
}

// IsExpired checks if the session has expired
func (s *ACLSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// ACLGroupListResponse is the response for listing ACL groups
type ACLGroupListResponse struct {
	Items      []ACLGroup `json:"items"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	TotalPages int        `json:"total_pages"`
}

// ACLSessionListResponse is the response for listing ACL sessions
type ACLSessionListResponse struct {
	Items      []ACLSession `json:"items"`
	Total      int64        `json:"total"`
	Page       int          `json:"page"`
	Limit      int          `json:"limit"`
	TotalPages int          `json:"total_pages"`
}

// ACL validation errors
var (
	ErrACLGroupNameRequired   = &ValidationError{Message: "ACL group name is required"}
	ErrACLGroupNameTooLong    = &ValidationError{Message: "ACL group name must be at most 255 characters"}
	ErrInvalidCombinationMode = &ValidationError{Message: "invalid combination mode"}
	ErrInvalidIPRuleType      = &ValidationError{Message: "invalid IP rule type"}
	ErrInvalidCIDR            = &ValidationError{Message: "invalid CIDR format"}
	ErrBasicAuthUsernameEmpty = &ValidationError{Message: "basic auth username cannot be empty"}
	ErrBasicAuthPasswordEmpty = &ValidationError{Message: "basic auth password cannot be empty"}
	ErrInvalidProviderType    = &ValidationError{Message: "invalid external provider type"}
	ErrProviderNameRequired   = &ValidationError{Message: "external provider name is required"}
	ErrProviderVerifyURLEmpty = &ValidationError{Message: "external provider verify URL is required"}
	ErrInvalidPathPattern     = &ValidationError{Message: "invalid path pattern"}
	ErrSessionTokenEmpty      = &ValidationError{Message: "session token cannot be empty"}
)

// Validate performs basic validation on the ACL group
func (g *ACLGroup) Validate() error {
	if g.Name == "" {
		return ErrACLGroupNameRequired
	}

	if len(g.Name) > 255 {
		return ErrACLGroupNameTooLong
	}

	if g.CombinationMode != ACLCombinationModeAny &&
		g.CombinationMode != ACLCombinationModeAll &&
		g.CombinationMode != ACLCombinationModeIPBypass {
		return ErrInvalidCombinationMode
	}

	return nil
}

// Validate performs basic validation on the ACL IP rule
func (r *ACLIPRule) Validate() error {
	if r.RuleType != ACLIPRuleTypeAllow &&
		r.RuleType != ACLIPRuleTypeDeny &&
		r.RuleType != ACLIPRuleTypeBypass {
		return ErrInvalidIPRuleType
	}

	if r.CIDR == "" {
		return ErrInvalidCIDR
	}

	return nil
}

// Validate performs basic validation on the ACL external provider
func (p *ACLExternalProvider) Validate() error {
	if p.ProviderType != ACLProviderTypeAuthelia &&
		p.ProviderType != ACLProviderTypeAuthentik &&
		p.ProviderType != ACLProviderTypeCustom {
		return ErrInvalidProviderType
	}

	if p.Name == "" {
		return ErrProviderNameRequired
	}

	if p.VerifyURL == "" {
		return ErrProviderVerifyURLEmpty
	}

	return nil
}
