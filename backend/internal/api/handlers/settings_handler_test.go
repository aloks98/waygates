package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/service/mocks"
)

// =============================================================================
// NewSettingsHandler Tests
// =============================================================================

func TestNewSettingsHandler(t *testing.T) {
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
	if handler.settingsService != mockService {
		t.Error("Expected settings service to be set")
	}
}

// =============================================================================
// GetAll Tests
// =============================================================================

func TestSettingsHandler_Unit_GetAll_Success(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		GetAllFunc: func() (map[string]string, error) {
			return map[string]string{
				"not_found.mode":         "default",
				"not_found.redirect_url": "",
				"app.theme":              "dark",
			}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()

	handler.GetAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response["success"].(bool) {
		t.Error("Expected success to be true")
	}

	data := response["data"].(map[string]interface{})
	if data["not_found.mode"] != "default" {
		t.Errorf("Expected mode 'default', got '%v'", data["not_found.mode"])
	}
}

func TestSettingsHandler_GetAll_ServiceError(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		GetAllFunc: func() (map[string]string, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewSettingsHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()

	handler.GetAll(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestSettingsHandler_GetAll_EmptySettings(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		GetAllFunc: func() (map[string]string, error) {
			return map[string]string{}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()

	handler.GetAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// =============================================================================
// Get Tests
// =============================================================================

func TestSettingsHandler_Unit_Get_Success(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		GetFunc: func(key string) (string, error) {
			if key == "not_found.mode" {
				return "default", nil
			}
			return "", errors.New("not found")
		},
	}
	handler := NewSettingsHandler(mockService, nil)

	r := chi.NewRouter()
	r.Get("/api/settings/{key}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/not_found.mode", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data := response["data"].(map[string]interface{})
	if data["key"] != "not_found.mode" {
		t.Errorf("Expected key 'not_found.mode', got '%v'", data["key"])
	}
	if data["value"] != "default" {
		t.Errorf("Expected value 'default', got '%v'", data["value"])
	}
}

func TestSettingsHandler_Unit_Get_NotFound(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		GetFunc: func(key string) (string, error) {
			return "", errors.New("setting not found")
		},
	}
	handler := NewSettingsHandler(mockService, nil)

	r := chi.NewRouter()
	r.Get("/api/settings/{key}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/nonexistent", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestSettingsHandler_Get_EmptyKey(t *testing.T) {
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil)

	// Test with empty key by calling handler directly without chi context
	req := httptest.NewRequest(http.MethodGet, "/api/settings/", nil)
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// =============================================================================
// Update Tests
// =============================================================================

func TestSettingsHandler_Unit_Update_Success(t *testing.T) {
	var capturedKey, capturedValue string
	mockService := &mocks.MockSettingsService{
		SetFunc: func(key, value string) error {
			capturedKey = key
			capturedValue = value
			return nil
		},
	}
	handler := NewSettingsHandler(mockService, nil)

	r := chi.NewRouter()
	r.Put("/api/settings/{key}", handler.Update)

	body, _ := json.Marshal(map[string]string{"value": "dark"})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/app.theme", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if capturedKey != "app.theme" {
		t.Errorf("Expected key 'app.theme', got '%s'", capturedKey)
	}
	if capturedValue != "dark" {
		t.Errorf("Expected value 'dark', got '%s'", capturedValue)
	}
}

func TestSettingsHandler_Update_EmptyKey(t *testing.T) {
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil)

	body, _ := json.Marshal(map[string]string{"value": "test"})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSettingsHandler_Update_InvalidJSON(t *testing.T) {
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil)

	r := chi.NewRouter()
	r.Put("/api/settings/{key}", handler.Update)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/app.theme", bytes.NewBufferString("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSettingsHandler_Update_ServiceError(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		SetFunc: func(key, value string) error {
			return errors.New("database error")
		},
	}
	handler := NewSettingsHandler(mockService, nil)

	r := chi.NewRouter()
	r.Put("/api/settings/{key}", handler.Update)

	body, _ := json.Marshal(map[string]string{"value": "test"})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/app.theme", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestSettingsHandler_Update_EmptyValue(t *testing.T) {
	var capturedValue string
	mockService := &mocks.MockSettingsService{
		SetFunc: func(key, value string) error {
			capturedValue = value
			return nil
		},
	}
	handler := NewSettingsHandler(mockService, nil)

	r := chi.NewRouter()
	r.Put("/api/settings/{key}", handler.Update)

	body, _ := json.Marshal(map[string]string{"value": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/app.theme", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if capturedValue != "" {
		t.Errorf("Expected empty value, got '%s'", capturedValue)
	}
}

// =============================================================================
// GetNotFound Tests
// =============================================================================

func TestSettingsHandler_Unit_GetNotFound_Success(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{
				Mode:        "default",
				RedirectURL: "",
			}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/404", nil)
	rec := httptest.NewRecorder()

	handler.GetNotFound(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data := response["data"].(map[string]interface{})
	if data["mode"] != "default" {
		t.Errorf("Expected mode 'default', got '%v'", data["mode"])
	}
}

func TestSettingsHandler_GetNotFound_RedirectMode(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{
				Mode:        "redirect",
				RedirectURL: "https://example.com/404",
			}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/404", nil)
	rec := httptest.NewRecorder()

	handler.GetNotFound(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data := response["data"].(map[string]interface{})
	if data["mode"] != "redirect" {
		t.Errorf("Expected mode 'redirect', got '%v'", data["mode"])
	}
	if data["redirect_url"] != "https://example.com/404" {
		t.Errorf("Expected redirect_url 'https://example.com/404', got '%v'", data["redirect_url"])
	}
}

func TestSettingsHandler_GetNotFound_ServiceError(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewSettingsHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/404", nil)
	rec := httptest.NewRecorder()

	handler.GetNotFound(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// =============================================================================
// UpdateNotFound Tests
// =============================================================================

func TestSettingsHandler_UpdateNotFound_DefaultMode(t *testing.T) {
	var capturedSettings *models.NotFoundSettings
	mockService := &mocks.MockSettingsService{
		SetNotFoundSettingsFunc: func(settings *models.NotFoundSettings) error {
			capturedSettings = settings
			return nil
		},
	}
	handler := NewSettingsHandler(mockService, nil)

	body, _ := json.Marshal(map[string]string{"mode": "default", "redirect_url": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateNotFound(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if capturedSettings.Mode != "default" {
		t.Errorf("Expected mode 'default', got '%s'", capturedSettings.Mode)
	}
}

func TestSettingsHandler_UpdateNotFound_RedirectMode(t *testing.T) {
	var capturedSettings *models.NotFoundSettings
	mockService := &mocks.MockSettingsService{
		SetNotFoundSettingsFunc: func(settings *models.NotFoundSettings) error {
			capturedSettings = settings
			return nil
		},
	}
	handler := NewSettingsHandler(mockService, nil)

	body, _ := json.Marshal(map[string]string{
		"mode":         "redirect",
		"redirect_url": "https://example.com/404",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateNotFound(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if capturedSettings.Mode != "redirect" {
		t.Errorf("Expected mode 'redirect', got '%s'", capturedSettings.Mode)
	}
	if capturedSettings.RedirectURL != "https://example.com/404" {
		t.Errorf("Expected redirect_url 'https://example.com/404', got '%s'", capturedSettings.RedirectURL)
	}
}

func TestSettingsHandler_UpdateNotFound_InvalidJSON(t *testing.T) {
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBufferString("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateNotFound(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSettingsHandler_Unit_UpdateNotFound_InvalidMode(t *testing.T) {
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil)

	body, _ := json.Marshal(map[string]string{"mode": "invalid", "redirect_url": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateNotFound(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSettingsHandler_Unit_UpdateNotFound_RedirectWithoutURL(t *testing.T) {
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil)

	body, _ := json.Marshal(map[string]string{"mode": "redirect", "redirect_url": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateNotFound(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSettingsHandler_UpdateNotFound_ServiceError(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		SetNotFoundSettingsFunc: func(settings *models.NotFoundSettings) error {
			return errors.New("database error")
		},
	}
	handler := NewSettingsHandler(mockService, nil)

	body, _ := json.Marshal(map[string]string{"mode": "default", "redirect_url": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateNotFound(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestSettingsHandler_UpdateNotFound_EmptyMode(t *testing.T) {
	mockService := &mocks.MockSettingsService{}
	handler := NewSettingsHandler(mockService, nil)

	body, _ := json.Marshal(map[string]string{"mode": "", "redirect_url": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateNotFound(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// =============================================================================
// Response Format Tests
// =============================================================================

func TestSettingsHandler_ResponseFormat(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		GetAllFunc: func() (map[string]string, error) {
			return map[string]string{"key": "value"}, nil
		},
	}
	handler := NewSettingsHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()

	handler.GetAll(rec, req)

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, ok := response["success"]; !ok {
		t.Error("Expected 'success' field in response")
	}
	if _, ok := response["message"]; !ok {
		t.Error("Expected 'message' field in response")
	}
	if _, ok := response["data"]; !ok {
		t.Error("Expected 'data' field in response")
	}
}
