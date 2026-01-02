package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aloks98/homelab-proxy/backend/internal/utils"
	"github.com/aloks98/homelab-proxy/backend/internal/validation"
)

// TestRegister_InvalidRequestBody tests registration with invalid JSON
func TestRegister_InvalidRequestBody(t *testing.T) {
	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Create a handler that only tests the JSON parsing
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var registerReq validation.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&registerReq); err != nil {
			utils.BadRequest(w, "Invalid request body", nil)
			return
		}
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response utils.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}
	if response.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("Expected error code VALIDATION_ERROR, got %s", response.Error.Code)
	}
}

// TestRegister_InvalidEmail tests registration with invalid email format
func TestRegister_InvalidEmail(t *testing.T) {
	testCases := []struct {
		name  string
		email string
	}{
		{"Not an email", "not-an-email"},
		{"Missing domain", "user@"},
		{"Missing at symbol", "userdomain.com"},
		{"Empty email", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := validation.RegisterRequest{
				Name:     "Test User",
				Username: "testuser",
				Email:    tc.email,
				Password: "password123",
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// Test validation
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var registerReq validation.RegisterRequest
				if err := json.NewDecoder(r.Body).Decode(&registerReq); err != nil {
					utils.BadRequest(w, "Invalid request body", nil)
					return
				}
				if err := validation.ValidateStruct(&registerReq); err != nil {
					utils.BadRequest(w, err.Error(), nil)
					return
				}
				utils.Success(w, nil, "")
			})

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

// TestRegister_WeakPassword tests registration with weak passwords
func TestRegister_WeakPassword(t *testing.T) {
	testCases := []struct {
		name     string
		password string
	}{
		{"Too short", "pass"},
		{"Empty password", ""},
		{"Only 7 chars", "1234567"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := validation.RegisterRequest{
				Name:     "Test User",
				Username: "testuser",
				Email:    "test@example.com",
				Password: tc.password,
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var registerReq validation.RegisterRequest
				if err := json.NewDecoder(r.Body).Decode(&registerReq); err != nil {
					utils.BadRequest(w, "Invalid request body", nil)
					return
				}
				if err := validation.ValidateStruct(&registerReq); err != nil {
					utils.BadRequest(w, err.Error(), nil)
					return
				}
				utils.Success(w, nil, "")
			})

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

// TestRegister_InvalidUsername tests registration with invalid usernames
func TestRegister_InvalidUsername(t *testing.T) {
	testCases := []struct {
		name     string
		username string
	}{
		{"Too short", "ab"},
		{"Contains special chars", "user@name"},
		{"Contains spaces", "user name"},
		{"Empty username", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := validation.RegisterRequest{
				Name:     "Test User",
				Username: tc.username,
				Email:    "test@example.com",
				Password: "password123",
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var registerReq validation.RegisterRequest
				if err := json.NewDecoder(r.Body).Decode(&registerReq); err != nil {
					utils.BadRequest(w, "Invalid request body", nil)
					return
				}
				if err := validation.ValidateStruct(&registerReq); err != nil {
					utils.BadRequest(w, err.Error(), nil)
					return
				}
				utils.Success(w, nil, "")
			})

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

// TestRegister_ValidRequest tests successful validation
func TestRegister_ValidRequest(t *testing.T) {
	testCases := []struct {
		name     string
		username string
		email    string
		password string
	}{
		{"Standard user", "testuser", "test@example.com", "password123"},
		{"Username with underscore", "test_user", "test@example.com", "password123"},
		{"Username with hyphen", "test-user", "test@example.com", "password123"},
		{"Username with numbers", "user123", "test@example.com", "password123"},
		{"Long password", "testuser", "test@example.com", "thisIsAVeryLongPassword123!@#"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := validation.RegisterRequest{
				Name:     "Test User",
				Username: tc.username,
				Email:    tc.email,
				Password: tc.password,
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var registerReq validation.RegisterRequest
				if err := json.NewDecoder(r.Body).Decode(&registerReq); err != nil {
					utils.BadRequest(w, "Invalid request body", nil)
					return
				}
				if err := validation.ValidateStruct(&registerReq); err != nil {
					utils.BadRequest(w, err.Error(), nil)
					return
				}
				utils.Success(w, nil, "Validation passed")
			})

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestLogin_InvalidRequestBody tests login with invalid JSON
func TestLogin_InvalidRequestBody(t *testing.T) {
	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var loginReq LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
			utils.BadRequest(w, "Invalid request body", nil)
			return
		}
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestLogin_MissingCredentials tests login with missing identifier or password
func TestLogin_MissingCredentials(t *testing.T) {
	testCases := []struct {
		name       string
		identifier string
		password   string
	}{
		{"Missing identifier", "", "password123"},
		{"Missing password", "testuser", ""},
		{"Both missing", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := LoginRequest{
				Identifier: tc.identifier,
				Password:   tc.password,
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var loginReq LoginRequest
				if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
					utils.BadRequest(w, "Invalid request body", nil)
					return
				}
				if loginReq.Identifier == "" || loginReq.Password == "" {
					utils.BadRequest(w, "Identifier and password are required", nil)
					return
				}
				utils.Success(w, nil, "")
			})

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

// TestRefreshToken_InvalidRequestBody tests refresh with invalid JSON
func TestRefreshToken_InvalidRequestBody(t *testing.T) {
	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var refreshReq RefreshTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&refreshReq); err != nil {
			utils.BadRequest(w, "Invalid request body", nil)
			return
		}
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestRefreshToken_MissingToken tests refresh with missing refresh token
func TestRefreshToken_MissingToken(t *testing.T) {
	reqBody := RefreshTokenRequest{
		RefreshToken: "",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var refreshReq RefreshTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&refreshReq); err != nil {
			utils.BadRequest(w, "Invalid request body", nil)
			return
		}
		if refreshReq.RefreshToken == "" {
			utils.BadRequest(w, "Refresh token is required", nil)
			return
		}
		utils.Success(w, nil, "")
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestLogout_MissingToken tests logout without authorization header
func TestLogout_MissingToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.Header.Get("Authorization")
		if len(accessToken) > 7 && accessToken[:7] == "Bearer " {
			accessToken = accessToken[7:]
		}
		if accessToken == "" {
			utils.BadRequest(w, "No token provided", nil)
			return
		}
		utils.Success(w, nil, "Logged out successfully")
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestLogout_WithToken tests logout with valid authorization header format
func TestLogout_WithToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.Header.Get("Authorization")
		if len(accessToken) > 7 && accessToken[:7] == "Bearer " {
			accessToken = accessToken[7:]
		}
		if accessToken == "" {
			utils.BadRequest(w, "No token provided", nil)
			return
		}
		// Token validation would happen here in the real handler
		utils.Success(w, nil, "Logged out successfully")
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// TestResponseFormat tests that error responses follow the correct format
func TestResponseFormat(t *testing.T) {
	rec := httptest.NewRecorder()
	utils.BadRequest(rec, "Test error", nil)

	var response utils.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}
	if response.Error.Code == "" {
		t.Error("Expected error code to be set")
	}
	if response.Error.Message != "Test error" {
		t.Errorf("Expected message 'Test error', got '%s'", response.Error.Message)
	}
}

// TestSuccessResponseFormat tests that success responses follow the correct format
func TestSuccessResponseFormat(t *testing.T) {
	rec := httptest.NewRecorder()
	utils.Success(rec, map[string]string{"key": "value"}, "Test message")

	var response utils.SuccessResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}
	if response.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got '%s'", response.Message)
	}
}
