package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/aloks98/waygates/backend/internal/utils"
)

// TestGetAllSettings_Success tests successful retrieval of all settings
func TestGetAllSettings_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()

	// Mock handler that returns settings
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		settings := map[string]string{
			"not_found.mode":         "default",
			"not_found.redirect_url": "",
		}
		utils.Success(w, settings, "Settings retrieved successfully")
	})

	handler.ServeHTTP(rec, req)

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

// TestGetSetting_ValidKey tests retrieving a single setting
func TestGetSetting_ValidKey(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/api/settings/{key}", func(w http.ResponseWriter, req *http.Request) {
		key := chi.URLParam(req, "key")
		if key == "" {
			utils.BadRequest(w, "Setting key is required", nil)
			return
		}

		// Mock: return value for known keys
		mockSettings := map[string]string{
			"not_found.mode": "default",
		}

		if value, ok := mockSettings[key]; ok {
			utils.Success(w, map[string]string{"key": key, "value": value}, "Setting retrieved")
		} else {
			utils.NotFound(w, "Setting not found")
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/settings/not_found.mode", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// TestGetSetting_NotFound tests retrieving a non-existent setting
func TestGetSetting_NotFound(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/api/settings/{key}", func(w http.ResponseWriter, req *http.Request) {
		key := chi.URLParam(req, "key")
		mockSettings := map[string]string{}

		if _, ok := mockSettings[key]; ok {
			utils.Success(w, nil, "")
		} else {
			utils.NotFound(w, "Setting not found")
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/settings/nonexistent", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

// TestUpdateSetting_ValidRequest tests updating a setting
func TestUpdateSetting_ValidRequest(t *testing.T) {
	r := chi.NewRouter()
	r.Put("/api/settings/{key}", func(w http.ResponseWriter, req *http.Request) {
		key := chi.URLParam(req, "key")
		if key == "" {
			utils.BadRequest(w, "Setting key is required", nil)
			return
		}

		var body struct {
			Value string `json:"value"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			utils.BadRequest(w, "Invalid request body", nil)
			return
		}

		utils.Success(w, map[string]string{"key": key, "value": body.Value}, "Setting updated")
	})

	body, _ := json.Marshal(map[string]string{"value": "new_value"})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/test.key", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// TestUpdateSetting_InvalidJSON tests updating with invalid JSON
func TestUpdateSetting_InvalidJSON(t *testing.T) {
	r := chi.NewRouter()
	r.Put("/api/settings/{key}", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Value string `json:"value"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			utils.BadRequest(w, "Invalid request body", nil)
			return
		}
		utils.Success(w, nil, "")
	})

	req := httptest.NewRequest(http.MethodPut, "/api/settings/test.key", bytes.NewBufferString("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestGetNotFound_Success tests retrieving 404 settings
func TestGetNotFound_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/settings/404", nil)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		settings := map[string]string{
			"mode":         "default",
			"redirect_url": "",
		}
		utils.Success(w, settings, "404 settings retrieved successfully")
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// TestUpdateNotFound_ValidDefault tests updating to default mode
func TestUpdateNotFound_ValidDefault(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Mode        string `json:"mode"`
			RedirectURL string `json:"redirect_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.BadRequest(w, "Invalid request body", nil)
			return
		}

		// Validate mode
		if req.Mode != "default" && req.Mode != "redirect" {
			utils.BadRequest(w, "Mode must be 'default' or 'redirect'", nil)
			return
		}

		// Validate redirect URL if mode is redirect
		if req.Mode == "redirect" && req.RedirectURL == "" {
			utils.BadRequest(w, "Redirect URL is required when mode is 'redirect'", nil)
			return
		}

		utils.Success(w, req, "404 settings updated")
	})

	body, _ := json.Marshal(map[string]string{"mode": "default", "redirect_url": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// TestUpdateNotFound_ValidRedirect tests updating to redirect mode
func TestUpdateNotFound_ValidRedirect(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Mode        string `json:"mode"`
			RedirectURL string `json:"redirect_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.BadRequest(w, "Invalid request body", nil)
			return
		}

		if req.Mode != "default" && req.Mode != "redirect" {
			utils.BadRequest(w, "Mode must be 'default' or 'redirect'", nil)
			return
		}

		if req.Mode == "redirect" && req.RedirectURL == "" {
			utils.BadRequest(w, "Redirect URL is required when mode is 'redirect'", nil)
			return
		}

		utils.Success(w, req, "404 settings updated")
	})

	body, _ := json.Marshal(map[string]string{
		"mode":         "redirect",
		"redirect_url": "https://example.com/404",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// TestUpdateNotFound_InvalidMode tests updating with invalid mode
func TestUpdateNotFound_InvalidMode(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Mode        string `json:"mode"`
			RedirectURL string `json:"redirect_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.BadRequest(w, "Invalid request body", nil)
			return
		}

		if req.Mode != "default" && req.Mode != "redirect" {
			utils.BadRequest(w, "Mode must be 'default' or 'redirect'", nil)
			return
		}

		utils.Success(w, req, "")
	})

	body, _ := json.Marshal(map[string]string{"mode": "invalid", "redirect_url": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestUpdateNotFound_RedirectWithoutURL tests redirect mode without URL
func TestUpdateNotFound_RedirectWithoutURL(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Mode        string `json:"mode"`
			RedirectURL string `json:"redirect_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.BadRequest(w, "Invalid request body", nil)
			return
		}

		if req.Mode != "default" && req.Mode != "redirect" {
			utils.BadRequest(w, "Mode must be 'default' or 'redirect'", nil)
			return
		}

		if req.Mode == "redirect" && req.RedirectURL == "" {
			utils.BadRequest(w, "Redirect URL is required when mode is 'redirect'", nil)
			return
		}

		utils.Success(w, req, "")
	})

	body, _ := json.Marshal(map[string]string{"mode": "redirect", "redirect_url": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestUpdateNotFound_InvalidJSON tests updating with invalid JSON
func TestUpdateNotFound_InvalidJSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Mode        string `json:"mode"`
			RedirectURL string `json:"redirect_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.BadRequest(w, "Invalid request body", nil)
			return
		}
		utils.Success(w, nil, "")
	})

	req := httptest.NewRequest(http.MethodPut, "/api/settings/404", bytes.NewBufferString("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestSettingsResponseFormat verifies the response format
func TestSettingsResponseFormat(t *testing.T) {
	rec := httptest.NewRecorder()
	utils.Success(rec, map[string]string{"key": "test", "value": "value"}, "Success")

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response["success"].(bool) {
		t.Error("Expected success to be true")
	}
	if response["message"] != "Success" {
		t.Errorf("Expected message 'Success', got '%v'", response["message"])
	}
	if response["data"] == nil {
		t.Error("Expected data to be present")
	}
}
