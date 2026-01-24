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

func TestSettingsHandler_GetAll_Success(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		GetAllFunc: func() (map[string]string, error) {
			return map[string]string{
				"key1": "value1",
				"key2": "value2",
			}, nil
		},
	}

	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()

	handler.GetAll(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["success"] != true {
		t.Error("Expected success to be true")
	}
}

func TestSettingsHandler_GetAll_Error(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		GetAllFunc: func() (map[string]string, error) {
			return nil, errors.New("database error")
		},
	}

	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()

	handler.GetAll(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestSettingsHandler_Get_Success(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		GetFunc: func(_ string) (string, error) {
			return "test_value", nil
		},
	}

	handler := NewSettingsHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/settings/{key}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/test_key", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSettingsHandler_Get_NotFound(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		GetFunc: func(_ string) (string, error) {
			return "", errors.New("not found")
		},
	}

	handler := NewSettingsHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/settings/{key}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/missing_key", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestSettingsHandler_Update_Success(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		SetFunc: func(_, _ string) error {
			return nil
		},
	}

	handler := NewSettingsHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/settings/{key}", handler.Update)

	body := `{"value": "new_value"}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings/test_key", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSettingsHandler_Update_InvalidBody(t *testing.T) {
	handler := NewSettingsHandler(&mocks.MockSettingsService{}, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/settings/{key}", handler.Update)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/test_key", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestSettingsHandler_Update_Error(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		SetFunc: func(_, _ string) error {
			return errors.New("database error")
		},
	}

	handler := NewSettingsHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/settings/{key}", handler.Update)

	body := `{"value": "new_value"}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings/test_key", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestSettingsHandler_GetNotFound_Success(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
			return &models.NotFoundSettings{
				Mode:        "redirect",
				RedirectURL: "https://example.com",
			}, nil
		},
	}

	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/404", nil)
	w := httptest.NewRecorder()

	handler.GetNotFound(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSettingsHandler_GetNotFound_Error(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		GetNotFoundSettingsFunc: func() (*models.NotFoundSettings, error) {
			return nil, errors.New("database error")
		},
	}

	handler := NewSettingsHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/404", nil)
	w := httptest.NewRecorder()

	handler.GetNotFound(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestSettingsHandler_UpdateNotFound_Success(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		SetNotFoundSettingsFunc: func(_ *models.NotFoundSettings) error {
			return nil
		},
	}

	handler := NewSettingsHandler(mockService, nil, nil)

	body := `{"mode": "redirect", "redirect_url": "https://example.com"}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateNotFound(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSettingsHandler_UpdateNotFound_InvalidMode(t *testing.T) {
	handler := NewSettingsHandler(&mocks.MockSettingsService{}, nil, nil)

	body := `{"mode": "invalid", "redirect_url": ""}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateNotFound(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestSettingsHandler_UpdateNotFound_RedirectWithoutURL(t *testing.T) {
	handler := NewSettingsHandler(&mocks.MockSettingsService{}, nil, nil)

	body := `{"mode": "redirect", "redirect_url": ""}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateNotFound(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestSettingsHandler_UpdateNotFound_Error(t *testing.T) {
	mockService := &mocks.MockSettingsService{
		SetNotFoundSettingsFunc: func(_ *models.NotFoundSettings) error {
			return errors.New("database error")
		},
	}

	handler := NewSettingsHandler(mockService, nil, nil)

	body := `{"mode": "default", "redirect_url": ""}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateNotFound(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
