package handlers

import (
	"bytes"
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
		GetFunc: func(key string) (string, error) {
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
		SetFunc: func(key, value string) error {
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
		SetFunc: func(key, value string) error {
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
		SetNotFoundSettingsFunc: func(settings *models.NotFoundSettings) error {
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
