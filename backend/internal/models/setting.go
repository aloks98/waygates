package models

import (
	"time"
)

// Setting represents a key-value configuration setting
type Setting struct {
	Key       string    `json:"key" gorm:"primaryKey;type:varchar(255)"`
	Value     string    `json:"value" gorm:"type:text;not null"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (Setting) TableName() string {
	return "settings"
}

// NotFoundSettings represents the 404 page configuration
type NotFoundSettings struct {
	Mode        string `json:"mode"`         // "default" or "redirect"
	RedirectURL string `json:"redirect_url"` // URL to redirect when mode is "redirect"
}

// Settings keys
const (
	SettingNotFoundMode        = "not_found.mode"
	SettingNotFoundRedirectURL = "not_found.redirect_url"
	SettingOAuthProviders      = "oauth_providers" // JSON map of provider_id -> enabled
)

// MetricsPublishSettings represents the opt-in protected external metrics endpoint config.
type MetricsPublishSettings struct {
	Enabled       bool     `json:"enabled"`
	Host          string   `json:"host"`
	Path          string   `json:"path"`
	BasicAuthUser string   `json:"basic_auth_user"`
	BasicAuthHash string   `json:"-"`             // bcrypt hash — NEVER serialized; persistence is key-value only
	AllowedCIDRs  []string `json:"allowed_cidrs"` // stored as comma-separated string
}

// Metrics publish settings keys.
const (
	SettingMetricsPublishEnabled = "metrics.publish_enabled"
	SettingMetricsPublishHost    = "metrics.publish_host"
	SettingMetricsPublishPath    = "metrics.publish_path"
	SettingMetricsBasicAuthUser  = "metrics.basic_auth_user"
	SettingMetricsBasicAuthHash  = "metrics.basic_auth_hash"
	SettingMetricsAllowedCIDRs   = "metrics.allowed_cidrs"
)

// sensitiveSettingKeys is the set of setting keys whose values must never be
// returned through the generic GET /settings or GET /settings/{key} endpoints.
// Add future secret keys here.
var sensitiveSettingKeys = map[string]struct{}{
	SettingMetricsBasicAuthHash: {},
}

// IsSensitiveSettingKey reports whether key holds a secret value (e.g. a bcrypt
// hash) that must not be exposed via the generic settings endpoints.
func IsSensitiveSettingKey(key string) bool {
	_, ok := sensitiveSettingKeys[key]
	return ok
}

// OAuthProviderSettings represents the enabled state of OAuth providers
type OAuthProviderSettings map[string]bool

// OAuthProviderInfo represents public info about an OAuth provider
type OAuthProviderInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Available bool   `json:"available"` // Env vars are configured
	Enabled   bool   `json:"enabled"`   // Admin has enabled it
}
