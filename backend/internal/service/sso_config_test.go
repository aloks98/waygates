package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aloks98/waygates/backend/internal/models"
)

// fakeSettings is a minimal SettingsRepositoryInterface for config tests.
type fakeSettings struct{ vals map[string]string }

func (f *fakeSettings) Get(_ string) (*models.Setting, error) { return nil, nil }
func (f *fakeSettings) GetValue(key, def string) string {
	if v, ok := f.vals[key]; ok {
		return v
	}
	return def
}
func (f *fakeSettings) Set(key, value string) error        { f.vals[key] = value; return nil }
func (f *fakeSettings) GetAll() (map[string]string, error) { return f.vals, nil }
func (f *fakeSettings) Delete(_ string) error              { return nil }
func (f *fakeSettings) GetNotFoundSettings() (*models.NotFoundSettings, error) {
	return nil, nil
}
func (f *fakeSettings) SetNotFoundSettings(_ *models.NotFoundSettings) error { return nil }
func (f *fakeSettings) GetMetricsPublishSettings() (*models.MetricsPublishSettings, error) {
	return nil, nil
}
func (f *fakeSettings) SetMetricsPublishSettings(_ *models.MetricsPublishSettings) error {
	return nil
}

func TestLoadSSOConfig_Defaults(t *testing.T) {
	cfg := LoadSSOConfig(&fakeSettings{vals: map[string]string{}})
	assert.False(t, cfg.Enabled)
	assert.False(t, cfg.AutoProvision)
	assert.Equal(t, "viewer", cfg.DefaultRole)
	assert.Equal(t, "Sign in with SSO", cfg.ButtonLabel)
}

func TestLoadSSOConfig_Values(t *testing.T) {
	cfg := LoadSSOConfig(&fakeSettings{vals: map[string]string{
		"sso.enabled":            "true",
		"sso.oidc_issuer":        "https://idp.example.com",
		"sso.oidc_client_id":     "cid",
		"sso.oidc_client_secret": "secret",
		"sso.auto_provision":     "true",
		"sso.default_role":       "operator",
		"sso.button_label":       "Company SSO",
		"sso.base_url":           "https://wg.example.com",
	}})
	assert.True(t, cfg.Enabled)
	assert.True(t, cfg.AutoProvision)
	assert.Equal(t, "https://idp.example.com", cfg.Issuer)
	assert.Equal(t, "operator", cfg.DefaultRole)
	assert.Equal(t, "Company SSO", cfg.ButtonLabel)
}
