package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/config"
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/service"
)

// settingsStub implements repository.SettingsRepositoryInterface for the handler.
type settingsStub struct{ vals map[string]string }

func (s *settingsStub) Get(_ string) (*models.Setting, error) { return nil, nil }
func (s *settingsStub) GetValue(key, def string) string {
	if v, ok := s.vals[key]; ok {
		return v
	}
	return def
}
func (s *settingsStub) Set(key, value string) error        { s.vals[key] = value; return nil }
func (s *settingsStub) GetAll() (map[string]string, error) { return s.vals, nil }
func (s *settingsStub) Delete(_ string) error              { return nil }
func (s *settingsStub) GetNotFoundSettings() (*models.NotFoundSettings, error) {
	return &models.NotFoundSettings{}, nil
}
func (s *settingsStub) SetNotFoundSettings(_ *models.NotFoundSettings) error { return nil }
func (s *settingsStub) GetMetricsPublishSettings() (*models.MetricsPublishSettings, error) {
	return &models.MetricsPublishSettings{}, nil
}
func (s *settingsStub) SetMetricsPublishSettings(_ *models.MetricsPublishSettings) error {
	return nil
}

func TestSSOHandler_Status_Disabled(t *testing.T) {
	st := &settingsStub{vals: map[string]string{}}
	svc := service.NewSSOService(service.SSODeps{Settings: st, Logger: zap.NewNop()})
	h := NewSSOHandler(svc, st, &config.Config{}, zap.NewNop())

	rr := httptest.NewRecorder()
	h.Status(rr, httptest.NewRequest(http.MethodGet, "/api/auth/sso/status", nil))
	require.Equal(t, http.StatusOK, rr.Code)

	var body struct {
		Data struct {
			Enabled bool   `json:"enabled"`
			Label   string `json:"label"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.False(t, body.Data.Enabled)
}

func TestSSOHandler_Lookup_Password_WhenDisabled(t *testing.T) {
	st := &settingsStub{vals: map[string]string{"sso.enabled": "false"}}
	svc := service.NewSSOService(service.SSODeps{Settings: st, Logger: zap.NewNop()})
	h := NewSSOHandler(svc, st, &config.Config{}, zap.NewNop())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sso/lookup",
		strings.NewReader(`{"email":"a@x.com"}`))
	h.Lookup(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var body struct {
		Data struct {
			Method string `json:"method"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, "password", body.Data.Method)
}

// settingsFailStub is a settings stub whose Set always returns an error.
type settingsFailStub struct{ vals map[string]string }

func (s *settingsFailStub) Get(_ string) (*models.Setting, error) { return nil, nil }
func (s *settingsFailStub) GetValue(key, def string) string {
	if v, ok := s.vals[key]; ok {
		return v
	}
	return def
}
func (s *settingsFailStub) Set(_, _ string) error { return errors.New("db write error") }
func (s *settingsFailStub) GetAll() (map[string]string, error) {
	return s.vals, nil
}
func (s *settingsFailStub) Delete(_ string) error { return nil }
func (s *settingsFailStub) GetNotFoundSettings() (*models.NotFoundSettings, error) {
	return &models.NotFoundSettings{}, nil
}
func (s *settingsFailStub) SetNotFoundSettings(_ *models.NotFoundSettings) error { return nil }
func (s *settingsFailStub) GetMetricsPublishSettings() (*models.MetricsPublishSettings, error) {
	return &models.MetricsPublishSettings{}, nil
}
func (s *settingsFailStub) SetMetricsPublishSettings(_ *models.MetricsPublishSettings) error {
	return nil
}

func TestSSOHandler_UpdateConfig_Returns500WhenSettingsSetFails(t *testing.T) {
	st := &settingsFailStub{vals: map[string]string{}}
	svc := service.NewSSOService(service.SSODeps{Settings: st, Logger: zap.NewNop()})
	h := NewSSOHandler(svc, st, &config.Config{}, zap.NewNop())

	body := `{"enabled":true,"issuer":"https://idp.example.com","client_id":"cid"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/admin/sso/config", strings.NewReader(body))
	h.UpdateConfig(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
