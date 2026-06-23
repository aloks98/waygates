package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/service/mocks"
)

// =============================================================================
// NewSettingsHandler Tests
// =============================================================================

func TestNewSettingsHandler(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil, nil)

	require.NotNil(t, handler, "handler should be created")
	assert.Equal(t, mockService, handler.settingsService, "settings service should be set")
}

// =============================================================================
// GetAll Tests
// =============================================================================

func TestSettingsHandler_Unit_GetAll_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetAllFunc: func() (map[string]string, error) {
			return map[string]string{
				"not_found.mode":         "default",
				"not_found.redirect_url": "",
				"app.theme":              "dark",
			}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()

	handler.GetAll(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Success bool              `json:"success"`
		Data    map[string]string `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.True(t, response.Success, "expected success to be true")
	assert.Equal(t, "default", response.Data["not_found.mode"])
}

func TestSettingsHandler_GetAll_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetAllFunc: func() (map[string]string, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()

	handler.GetAll(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestSettingsHandler_GetAll_EmptySettings(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetAllFunc: func() (map[string]string, error) {
			return map[string]string{}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()

	handler.GetAll(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

// =============================================================================
// Get Tests
// =============================================================================

func TestSettingsHandler_Unit_Get_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetFunc: func(key string) (string, error) {
			if key == "not_found.mode" {
				return "default", nil
			}
			return "", errors.New("not found")
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/settings/{key}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/not_found.mode", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Data struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.Equal(t, "not_found.mode", response.Data.Key)
	assert.Equal(t, "default", response.Data.Value)
}

func TestSettingsHandler_Unit_Get_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetFunc: func(_ string) (string, error) {
			return "", errors.New("setting not found")
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/settings/{key}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/nonexistent", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSettingsHandler_Get_EmptyKey(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil, nil)

	// Test with empty key by calling handler directly without chi context
	req := httptest.NewRequest(http.MethodGet, "/api/settings/", nil)
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// Update Tests
// =============================================================================

func TestSettingsHandler_Unit_Update_Success(t *testing.T) {
	t.Parallel()
	var capturedKey, capturedValue string
	mockService := &mocks.MockSettingsService{
		SetFunc: func(key, value string) error {
			capturedKey = key
			capturedValue = value
			return nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/settings/{key}", handler.Update)

	body, _ := json.Marshal(map[string]string{"value": "dark"})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/app.theme", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	assert.Equal(t, "app.theme", capturedKey)
	assert.Equal(t, "dark", capturedValue)
}

func TestSettingsHandler_Update_EmptyKey(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]string{"value": "test"})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSettingsHandler_Update_InvalidJSON(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/settings/{key}", handler.Update)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/app.theme", bytes.NewBufferString("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSettingsHandler_Update_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		SetFunc: func(_, _ string) error {
			return errors.New("database error")
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/settings/{key}", handler.Update)

	body, _ := json.Marshal(map[string]string{"value": "test"})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/app.theme", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestSettingsHandler_Update_EmptyValue(t *testing.T) {
	t.Parallel()
	var capturedValue string
	mockService := &mocks.MockSettingsService{
		SetFunc: func(_, value string) error {
			capturedValue = value
			return nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/settings/{key}", handler.Update)

	body, _ := json.Marshal(map[string]string{"value": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/app.theme", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	assert.Equal(t, "", capturedValue)
}

// =============================================================================
// GetNotFound Tests
// =============================================================================

func TestSettingsHandler_Unit_GetNotFound_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{
				Mode:        "default",
				RedirectURL: "",
			}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/404", nil)
	rec := httptest.NewRecorder()

	handler.GetNotFound(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Data struct {
			Mode        string `json:"mode"`
			RedirectURL string `json:"redirect_url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.Equal(t, "default", response.Data.Mode)
}

func TestSettingsHandler_GetNotFound_RedirectMode(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{
				Mode:        "redirect",
				RedirectURL: "https://example.com/404",
			}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/404", nil)
	rec := httptest.NewRecorder()

	handler.GetNotFound(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Data struct {
			Mode        string `json:"mode"`
			RedirectURL string `json:"redirect_url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.Equal(t, "redirect", response.Data.Mode)
	assert.Equal(t, "https://example.com/404", response.Data.RedirectURL)
}

func TestSettingsHandler_GetNotFound_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/404", nil)
	rec := httptest.NewRecorder()

	handler.GetNotFound(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// UpdateNotFound Tests
// =============================================================================

func TestSettingsHandler_UpdateNotFound_DefaultMode(t *testing.T) {
	t.Parallel()
	var capturedSettings *models.NotFoundSettings
	mockService := &mocks.MockSettingsService{
		SetNotFoundSettingsFunc: func(settings *models.NotFoundSettings) error {
			capturedSettings = settings
			return nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]string{"mode": "default", "redirect_url": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateNotFound(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	require.NotNil(t, capturedSettings)
	assert.Equal(t, "default", capturedSettings.Mode)
}

func TestSettingsHandler_UpdateNotFound_RedirectMode(t *testing.T) {
	t.Parallel()
	var capturedSettings *models.NotFoundSettings
	mockService := &mocks.MockSettingsService{
		SetNotFoundSettingsFunc: func(settings *models.NotFoundSettings) error {
			capturedSettings = settings
			return nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]string{
		"mode":         "redirect",
		"redirect_url": "https://example.com/404",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateNotFound(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	require.NotNil(t, capturedSettings)
	assert.Equal(t, "redirect", capturedSettings.Mode)
	assert.Equal(t, "https://example.com/404", capturedSettings.RedirectURL)
}

func TestSettingsHandler_UpdateNotFound_InvalidJSON(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBufferString("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateNotFound(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSettingsHandler_Unit_UpdateNotFound_InvalidMode(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]string{"mode": "invalid", "redirect_url": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateNotFound(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSettingsHandler_Unit_UpdateNotFound_RedirectWithoutURL(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]string{"mode": "redirect", "redirect_url": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateNotFound(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSettingsHandler_UpdateNotFound_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		SetNotFoundSettingsFunc: func(_ *models.NotFoundSettings) error {
			return errors.New("database error")
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]string{"mode": "default", "redirect_url": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateNotFound(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestSettingsHandler_UpdateNotFound_EmptyMode(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]string{"mode": "", "redirect_url": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateNotFound(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// Response Format Tests
// =============================================================================

func TestSettingsHandler_ResponseFormat(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetAllFunc: func() (map[string]string, error) {
			return map[string]string{"key": "value"}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()

	handler.GetAll(rec, req)

	var response struct {
		Success bool                   `json:"success"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.True(t, response.Success, "expected 'success' field")
	assert.NotEmpty(t, response.Message, "expected 'message' field")
	assert.NotNil(t, response.Data, "expected 'data' field")
}

// =============================================================================
// GetMetricsPublish Tests
// =============================================================================

func TestSettingsHandler_GetMetricsPublish_OmitsHash(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetMetricsPublishSettingsFunc: func() (*models.MetricsPublishSettings, error) {
			return &models.MetricsPublishSettings{
				Enabled:       true,
				Host:          "metrics.example.com",
				Path:          "/metrics",
				BasicAuthUser: "scraper",
				BasicAuthHash: "$2a$14$verysecretbcrypthash",
				AllowedCIDRs:  []string{"10.0.0.0/8"},
			}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/metrics-publish", nil)
	rec := httptest.NewRecorder()

	handler.GetMetricsPublish(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.True(t, response.Success)

	// Hash must NOT be in the response.
	_, hashPresent := response.Data["basic_auth_hash"]
	assert.False(t, hashPresent, "bcrypt hash must never be returned in the GET response")

	// has_basic_auth must be true when a hash is stored.
	hasAuth, _ := response.Data["has_basic_auth"].(bool)
	assert.True(t, hasAuth, "has_basic_auth must be true when a hash is stored")
}

func TestSettingsHandler_GetMetricsPublish_HasBasicAuthFalseWhenNoHash(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetMetricsPublishSettingsFunc: func() (*models.MetricsPublishSettings, error) {
			return &models.MetricsPublishSettings{
				Enabled:       false,
				Host:          "",
				Path:          "/metrics",
				BasicAuthUser: "",
				BasicAuthHash: "",
				AllowedCIDRs:  nil,
			}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/metrics-publish", nil)
	rec := httptest.NewRecorder()

	handler.GetMetricsPublish(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	hasAuth, _ := response.Data["has_basic_auth"].(bool)
	assert.False(t, hasAuth, "has_basic_auth must be false when no hash is stored")
}

func TestSettingsHandler_GetMetricsPublish_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetMetricsPublishSettingsFunc: func() (*models.MetricsPublishSettings, error) {
			return nil, errors.New("db error")
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/metrics-publish", nil)
	rec := httptest.NewRecorder()

	handler.GetMetricsPublish(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// UpdateMetricsPublish Tests
// =============================================================================

func TestSettingsHandler_UpdateMetricsPublish_RejectsEnableWithoutHost(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetMetricsPublishSettingsFunc: func() (*models.MetricsPublishSettings, error) {
			return &models.MetricsPublishSettings{
				Enabled:       false,
				BasicAuthHash: "$2a$14$existinghash",
			}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	body := bytes.NewBufferString(`{
		"enabled": true,
		"host": "",
		"path": "/metrics",
		"basic_auth_user": "admin",
		"basic_auth_password": ""
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/settings/metrics-publish", body)
	rec := httptest.NewRecorder()

	handler.UpdateMetricsPublish(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSettingsHandler_UpdateMetricsPublish_RejectsEnableWithoutUser(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetMetricsPublishSettingsFunc: func() (*models.MetricsPublishSettings, error) {
			return &models.MetricsPublishSettings{
				Enabled:       false,
				BasicAuthHash: "$2a$14$existinghash",
			}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	body := bytes.NewBufferString(`{
		"enabled": true,
		"host": "metrics.example.com",
		"path": "/metrics",
		"basic_auth_user": "",
		"basic_auth_password": ""
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/settings/metrics-publish", body)
	rec := httptest.NewRecorder()

	handler.UpdateMetricsPublish(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSettingsHandler_UpdateMetricsPublish_RejectsEnableWithNoHashAndNoPassword(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetMetricsPublishSettingsFunc: func() (*models.MetricsPublishSettings, error) {
			return &models.MetricsPublishSettings{
				Enabled:       false,
				BasicAuthHash: "", // no existing hash
			}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	body := bytes.NewBufferString(`{
		"enabled": true,
		"host": "metrics.example.com",
		"path": "/metrics",
		"basic_auth_user": "admin",
		"basic_auth_password": ""
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/settings/metrics-publish", body)
	rec := httptest.NewRecorder()

	handler.UpdateMetricsPublish(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSettingsHandler_UpdateMetricsPublish_RejectsInvalidCIDR(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockSettingsService{
		GetMetricsPublishSettingsFunc: func() (*models.MetricsPublishSettings, error) {
			return &models.MetricsPublishSettings{
				Enabled:       false,
				BasicAuthHash: "$2a$14$existinghash",
			}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	body := bytes.NewBufferString(`{
		"enabled": true,
		"host": "metrics.example.com",
		"path": "/metrics",
		"basic_auth_user": "admin",
		"basic_auth_password": "",
		"allowed_cidrs": ["not-a-cidr"]
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/settings/metrics-publish", body)
	rec := httptest.NewRecorder()

	handler.UpdateMetricsPublish(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSettingsHandler_UpdateMetricsPublish_KeepsExistingHashWhenPasswordBlank(t *testing.T) {
	t.Parallel()
	const existingHash = "$2a$14$existinghashvalue"
	var storedSettings *models.MetricsPublishSettings

	mockService := &mocks.MockSettingsService{
		GetMetricsPublishSettingsFunc: func() (*models.MetricsPublishSettings, error) {
			return &models.MetricsPublishSettings{
				Enabled:       false,
				BasicAuthHash: existingHash,
			}, nil
		},
		SetMetricsPublishSettingsFunc: func(s *models.MetricsPublishSettings) error {
			storedSettings = s
			return nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	body := bytes.NewBufferString(`{
		"enabled": true,
		"host": "metrics.example.com",
		"path": "/metrics",
		"basic_auth_user": "admin",
		"basic_auth_password": ""
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/settings/metrics-publish", body)
	rec := httptest.NewRecorder()

	handler.UpdateMetricsPublish(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, storedSettings)
	assert.Equal(t, existingHash, storedSettings.BasicAuthHash, "existing hash must be preserved when no new password is supplied")
}

func TestSettingsHandler_UpdateMetricsPublish_HashesNewPassword(t *testing.T) {
	t.Parallel()
	var storedSettings *models.MetricsPublishSettings

	mockService := &mocks.MockSettingsService{
		GetMetricsPublishSettingsFunc: func() (*models.MetricsPublishSettings, error) {
			return &models.MetricsPublishSettings{
				Enabled:       false,
				BasicAuthHash: "",
			}, nil
		},
		SetMetricsPublishSettingsFunc: func(s *models.MetricsPublishSettings) error {
			storedSettings = s
			return nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	body := bytes.NewBufferString(`{
		"enabled": true,
		"host": "metrics.example.com",
		"path": "/metrics",
		"basic_auth_user": "admin",
		"basic_auth_password": "supersecret"
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/settings/metrics-publish", body)
	rec := httptest.NewRecorder()

	handler.UpdateMetricsPublish(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, storedSettings)
	assert.NotEmpty(t, storedSettings.BasicAuthHash, "a bcrypt hash must be stored")
	assert.NotEqual(t, "supersecret", storedSettings.BasicAuthHash, "plaintext must not be stored")
	assert.True(t, len(storedSettings.BasicAuthHash) > 20, "stored value should look like a bcrypt hash")
}

// =============================================================================
// Sensitive-key denylist leak tests (regression guard)
// =============================================================================

// TestSettingsHandler_GetAll_OmitsSensitiveKeys asserts that GetAll never returns
// the bcrypt hash key even when it is present in the underlying store.
func TestSettingsHandler_GetAll_OmitsSensitiveKeys(t *testing.T) {
	t.Parallel()

	const fakeHash = "$2a$14$supersecretbcrypthashvalue"

	mockService := &mocks.MockSettingsService{
		GetAllFunc: func() (map[string]string, error) {
			return map[string]string{
				"not_found.mode":          "default",
				"metrics.basic_auth_hash": fakeHash,
				"metrics.publish_enabled": "true",
			}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()

	handler.GetAll(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// The raw body must not contain the hash value or the key name.
	body := rec.Body.String()
	assert.NotContains(t, body, fakeHash, "bcrypt hash value must not appear in GetAll response")
	assert.NotContains(t, body, "basic_auth_hash", "bcrypt hash key must not appear in GetAll response")

	var response struct {
		Success bool              `json:"success"`
		Data    map[string]string `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.True(t, response.Success)
	_, present := response.Data["metrics.basic_auth_hash"]
	assert.False(t, present, "metrics.basic_auth_hash must be absent from the GetAll data map")

	// Non-sensitive keys must still be returned.
	assert.Equal(t, "default", response.Data["not_found.mode"])
}

// TestSettingsHandler_Get_BlocksSensitiveKey asserts that fetching the hash key
// directly returns 404 and does not expose the hash value.
func TestSettingsHandler_Get_BlocksSensitiveKey(t *testing.T) {
	t.Parallel()

	const fakeHash = "$2a$14$supersecretbcrypthashvalue"

	// The service Get func should never be called for a sensitive key, but we
	// return the hash anyway to prove the handler short-circuits before hitting it.
	mockService := &mocks.MockSettingsService{
		GetFunc: func(key string) (string, error) {
			if key == "metrics.basic_auth_hash" {
				return fakeHash, nil
			}
			return "", errors.New("not found")
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/settings/{key}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/metrics.basic_auth_hash", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code, "sensitive key must return 404")

	body := rec.Body.String()
	assert.NotContains(t, body, fakeHash, "bcrypt hash value must not appear in Get response")
}

// TestSettingsHandler_Update_BlocksSensitiveKey asserts that the generic Update
// handler rejects writes to sensitive keys before touching the service or the
// audit log — preventing both hash overwrite and audit-log leakage.
func TestSettingsHandler_Update_BlocksSensitiveKey(t *testing.T) {
	t.Parallel()

	serviceWriteCalled := false
	auditWriteCalled := false

	mockService := &mocks.MockSettingsService{
		// Get should never be called for a sensitive key; if it is, fail fast.
		GetFunc: func(key string) (string, error) {
			t.Errorf("settings service Get must not be called for sensitive key %q", key)
			return "", nil
		},
		// Set must not be called; the handler must short-circuit before it.
		SetFunc: func(key, _ string) error {
			serviceWriteCalled = true
			t.Errorf("settings service Set must not be called for sensitive key %q", key)
			return nil
		},
	}

	mockAudit := &mocks.MockAuditService{
		LogSettingsUpdateFunc: func(_ context.Context, _ int, key string, _, _, _, _ string) error {
			auditWriteCalled = true
			t.Errorf("audit LogSettingsUpdate must not be called for sensitive key %q", key)
			return nil
		},
	}

	handler := NewSettingsHandler(mockService, mockAudit, nil)

	r := chi.NewRouter()
	r.Put("/api/settings/{key}", handler.Update)

	body, _ := json.Marshal(map[string]string{"value": "attacker-chosen-value"})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/metrics.basic_auth_hash", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code, "sensitive key must return 404")
	assert.False(t, serviceWriteCalled, "settings service must not have been called")
	assert.False(t, auditWriteCalled, "audit service must not have been called")

	// The response body must not reveal the key name or any stored value.
	body2 := rec.Body.String()
	assert.NotContains(t, body2, "metrics.basic_auth_hash", "key name must not appear in the 404 response")
}

func TestSettingsHandler_UpdateMetricsPublish_DisabledNoValidationRequired(t *testing.T) {
	t.Parallel()
	var storedSettings *models.MetricsPublishSettings

	mockService := &mocks.MockSettingsService{
		GetMetricsPublishSettingsFunc: func() (*models.MetricsPublishSettings, error) {
			return &models.MetricsPublishSettings{Enabled: true}, nil
		},
		SetMetricsPublishSettingsFunc: func(s *models.MetricsPublishSettings) error {
			storedSettings = s
			return nil
		},
	}
	handler := NewSettingsHandler(mockService, nil, nil)

	// Disabling with no host or user should be allowed.
	body := bytes.NewBufferString(`{
		"enabled": false,
		"host": "",
		"basic_auth_user": ""
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/settings/metrics-publish", body)
	rec := httptest.NewRecorder()

	handler.UpdateMetricsPublish(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, storedSettings)
	assert.False(t, storedSettings.Enabled)
}
