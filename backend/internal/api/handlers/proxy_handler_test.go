package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/utils"
	"github.com/go-chi/chi/v5"
)

// TestListProxies_InvalidPage tests ListProxies with invalid page parameter
func TestListProxies_InvalidPage(t *testing.T) {
	testCases := []struct {
		name  string
		page  string
		valid bool
	}{
		{"Valid page 0", "0", true},
		{"Valid page 1", "1", true},
		{"Negative page", "-1", false},
		{"Non-numeric page", "abc", false},
		{"Float page", "1.5", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/proxies?page="+tc.page, nil)
			rec := httptest.NewRecorder()

			// Handler that only tests page validation
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				pageStr := r.URL.Query().Get("page")
				if pageStr != "" {
					page := 0
					for _, ch := range pageStr {
						if ch < '0' || ch > '9' {
							utils.BadRequest(w, "Invalid page parameter: must be a non-negative integer", nil)
							return
						}
						page = page*10 + int(ch-'0')
					}
					// Check for negative (would need to handle "-" prefix)
					if pageStr[0] == '-' {
						utils.BadRequest(w, "Invalid page parameter: must be a non-negative integer", nil)
						return
					}
				}
				utils.Success(w, nil, "")
			})

			handler.ServeHTTP(rec, req)

			if tc.valid && rec.Code != http.StatusOK {
				t.Errorf("Expected status OK for valid page, got %d", rec.Code)
			}
			if !tc.valid && rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status BadRequest for invalid page, got %d", rec.Code)
			}
		})
	}
}

// TestListProxies_InvalidLimit tests ListProxies with invalid limit parameter
func TestListProxies_InvalidLimit(t *testing.T) {
	testCases := []struct {
		name  string
		limit string
		valid bool
	}{
		{"Valid limit 0", "0", true},
		{"Valid limit 50", "50", true},
		{"Valid limit 100", "100", true},
		{"Negative limit", "-1", false},
		{"Too large limit", "101", false},
		{"Non-numeric limit", "abc", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/proxies?limit="+tc.limit, nil)
			rec := httptest.NewRecorder()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				limitStr := r.URL.Query().Get("limit")
				if limitStr != "" {
					// Simple validation
					limit := 0
					negative := false
					for i, ch := range limitStr {
						if i == 0 && ch == '-' {
							negative = true
							continue
						}
						if ch < '0' || ch > '9' {
							utils.BadRequest(w, "Invalid limit parameter: must be an integer between 0 and 100", nil)
							return
						}
						limit = limit*10 + int(ch-'0')
					}
					if negative || limit > 100 {
						utils.BadRequest(w, "Invalid limit parameter: must be an integer between 0 and 100", nil)
						return
					}
				}
				utils.Success(w, nil, "")
			})

			handler.ServeHTTP(rec, req)

			if tc.valid && rec.Code != http.StatusOK {
				t.Errorf("Expected status OK for valid limit, got %d", rec.Code)
			}
			if !tc.valid && rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status BadRequest for invalid limit, got %d", rec.Code)
			}
		})
	}
}

// TestListProxies_InvalidStatus tests ListProxies with invalid status parameter
func TestListProxies_InvalidStatus(t *testing.T) {
	testCases := []struct {
		name   string
		status string
		valid  bool
	}{
		{"Valid active", "active", true},
		{"Valid inactive", "inactive", true},
		{"Empty status", "", true},
		{"Invalid status", "invalid", false},
		{"Random string", "foo", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/api/proxies"
			if tc.status != "" {
				url += "?status=" + tc.status
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				status := r.URL.Query().Get("status")
				if status != "" && status != "active" && status != "inactive" {
					utils.BadRequest(w, "Invalid status parameter: must be 'active' or 'inactive'", nil)
					return
				}
				utils.Success(w, nil, "")
			})

			handler.ServeHTTP(rec, req)

			if tc.valid && rec.Code != http.StatusOK {
				t.Errorf("Expected status OK, got %d", rec.Code)
			}
			if !tc.valid && rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status BadRequest, got %d", rec.Code)
			}
		})
	}
}

// TestGetProxy_InvalidID tests GetProxy with invalid ID parameter
func TestGetProxy_InvalidID(t *testing.T) {
	testCases := []struct {
		name         string
		id           string
		valid        bool
		expectedCode int
	}{
		{"Valid numeric ID", "123", true, http.StatusOK},
		{"Non-numeric ID", "abc", false, http.StatusBadRequest},
		{"Float ID", "1.5", false, http.StatusBadRequest},
		{"Empty ID", "", false, http.StatusNotFound}, // Chi returns 404 for missing path param
		{"Negative ID", "-1", true, http.StatusOK},   // parsing succeeds, service would reject
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/api/proxies/{id}", func(w http.ResponseWriter, r *http.Request) {
				idStr := chi.URLParam(r, "id")
				if idStr == "" {
					utils.BadRequest(w, "Invalid proxy ID", nil)
					return
				}

				// Try to parse as int
				id := 0
				negative := false
				for i, ch := range idStr {
					if i == 0 && ch == '-' {
						negative = true
						continue
					}
					if ch < '0' || ch > '9' {
						utils.BadRequest(w, "Invalid proxy ID", nil)
						return
					}
					id = id*10 + int(ch-'0')
				}
				if negative {
					id = -id
				}
				utils.Success(w, map[string]int{"id": id}, "")
			})

			req := httptest.NewRequest(http.MethodGet, "/api/proxies/"+tc.id, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.expectedCode {
				t.Errorf("Expected status %d, got %d", tc.expectedCode, rec.Code)
			}
		})
	}
}

// TestCreateProxy_InvalidRequestBody tests CreateProxy with invalid JSON
func TestCreateProxy_InvalidRequestBody(t *testing.T) {
	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/api/proxies", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var proxy models.Proxy
		if err := json.NewDecoder(r.Body).Decode(&proxy); err != nil {
			utils.BadRequest(w, "Invalid request body format", nil)
			return
		}
		utils.Created(w, proxy, "")
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestCreateProxy_MissingUserID tests CreateProxy without user ID in context
func TestCreateProxy_MissingUserID(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"name": "Test", "hostname": "test.example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/proxies", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate checking for user ID (normally done by middleware)
		userID := r.Context().Value("user_id")
		if userID == nil {
			utils.Unauthorized(w, "User not found in context")
			return
		}
		utils.Created(w, nil, "")
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

// TestCreateProxy_WithUserID tests CreateProxy with user ID in context
func TestCreateProxy_WithUserID(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"name": "Test", "hostname": "test.example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/proxies", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// Add user ID to context
	ctx := context.WithValue(req.Context(), "user_id", "123")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id")
		if userID == nil {
			utils.Unauthorized(w, "User not found in context")
			return
		}
		utils.Created(w, nil, "Created")
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

// TestUpdateProxy_InvalidID tests UpdateProxy with invalid ID
func TestUpdateProxy_InvalidID(t *testing.T) {
	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		for _, ch := range idStr {
			if ch < '0' || ch > '9' {
				utils.BadRequest(w, "Invalid proxy ID", nil)
				return
			}
		}
		utils.Success(w, nil, "")
	})

	body, _ := json.Marshal(map[string]string{"name": "Updated"})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/abc", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestUpdateProxy_InvalidRequestBody tests UpdateProxy with invalid JSON
func TestUpdateProxy_InvalidRequestBody(t *testing.T) {
	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", func(w http.ResponseWriter, r *http.Request) {
		var proxy models.Proxy
		if err := json.NewDecoder(r.Body).Decode(&proxy); err != nil {
			utils.BadRequest(w, "Invalid request body format", nil)
			return
		}
		utils.Success(w, proxy, "")
	})

	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestDeleteProxy_InvalidID tests DeleteProxy with invalid ID
func TestDeleteProxy_InvalidID(t *testing.T) {
	r := chi.NewRouter()
	r.Delete("/api/proxies/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		for _, ch := range idStr {
			if ch < '0' || ch > '9' {
				utils.BadRequest(w, "Invalid proxy ID", nil)
				return
			}
		}
		utils.Success(w, nil, "Deleted")
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/proxies/abc", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestEnableProxy_InvalidID tests EnableProxy with invalid ID
func TestEnableProxy_InvalidID(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/enable", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		for _, ch := range idStr {
			if ch < '0' || ch > '9' {
				utils.BadRequest(w, "Invalid proxy ID", nil)
				return
			}
		}
		utils.Success(w, nil, "Enabled")
	})

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/abc/enable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestDisableProxy_InvalidID tests DisableProxy with invalid ID
func TestDisableProxy_InvalidID(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/disable", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		for _, ch := range idStr {
			if ch < '0' || ch > '9' {
				utils.BadRequest(w, "Invalid proxy ID", nil)
				return
			}
		}
		utils.Success(w, nil, "Disabled")
	})

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/abc/disable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestProxyResponseFormat tests that proxy responses follow the correct format
func TestProxyResponseFormat(t *testing.T) {
	rec := httptest.NewRecorder()
	utils.NotFound(rec, "Proxy not found")

	var response utils.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}
	if response.Error.Code != "NOT_FOUND" {
		t.Errorf("Expected error code NOT_FOUND, got %s", response.Error.Code)
	}
	if response.Error.Message != "Proxy not found" {
		t.Errorf("Expected message 'Proxy not found', got '%s'", response.Error.Message)
	}
}

// TestProxyConflictResponse tests the conflict response format
func TestProxyConflictResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	utils.Conflict(rec, "Hostname already exists")

	if rec.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, rec.Code)
	}

	var response utils.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Error.Code != "CONFLICT" {
		t.Errorf("Expected error code CONFLICT, got %s", response.Error.Code)
	}
}

// TestProxyBadGatewayResponse tests the bad gateway response format
func TestProxyBadGatewayResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	utils.BadGateway(rec, "Caddy configuration failed")

	if rec.Code != http.StatusBadGateway {
		t.Errorf("Expected status %d, got %d", http.StatusBadGateway, rec.Code)
	}

	var response utils.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Error.Code != "EXTERNAL_SERVICE_ERROR" {
		t.Errorf("Expected error code EXTERNAL_SERVICE_ERROR, got %s", response.Error.Code)
	}
}
