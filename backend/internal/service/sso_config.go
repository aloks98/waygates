package service

import (
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// SSOConfig is the resolved admin-SSO configuration read from the settings store.
type SSOConfig struct {
	Enabled       bool
	Issuer        string
	ClientID      string
	ClientSecret  string
	AutoProvision bool
	DefaultRole   string
	ButtonLabel   string
	BaseURL       string
}

// LoadSSOConfig reads the sso.* settings into an SSOConfig, applying defaults.
func LoadSSOConfig(s repository.SettingsRepositoryInterface) SSOConfig {
	return SSOConfig{
		Enabled:       s.GetValue(models.SettingSSOEnabled, "false") == "true",
		Issuer:        s.GetValue(models.SettingSSOOIDCIssuer, ""),
		ClientID:      s.GetValue(models.SettingSSOOIDCClientID, ""),
		ClientSecret:  s.GetValue(models.SettingSSOOIDCClientSecret, ""),
		AutoProvision: s.GetValue(models.SettingSSOAutoProvision, "false") == "true",
		DefaultRole:   s.GetValue(models.SettingSSODefaultRole, "viewer"),
		ButtonLabel:   s.GetValue(models.SettingSSOButtonLabel, "Sign in with SSO"),
		BaseURL:       s.GetValue(models.SettingSSOBaseURL, ""),
	}
}
