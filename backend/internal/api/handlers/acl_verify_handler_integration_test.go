package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/service/mocks"
)

// =============================================================================
// Test Setup
// =============================================================================

// setupACLVerifyTestRouter creates a router with ACL verify handler for testing
func setupACLVerifyTestRouter(
	mockACLService *MockACLService,
	mockUserRepo *mocks.MockUserRepository,
	mockAuditService *mocks.MockAuditService,
) *chi.Mux {
	handler := NewACLVerifyHandler(mockACLService, mockUserRepo, mockAuditService, nil)
	r := chi.NewRouter()

	r.Get("/api/auth/acl/verify", handler.Verify)
	r.Post("/api/auth/acl/login", handler.Login)
	r.Post("/api/auth/acl/logout", handler.Logout)
	r.Get("/api/auth/acl/session", handler.GetSession)

	return r
}

// createTestACLUser creates a test user with hashed password for ACL tests
func createTestACLUser(password string) *models.User {
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	_ = user.SetPassword(password, 4) // Use low cost for tests
	return user
}

// =============================================================================
// Verify Endpoint Tests
// =============================================================================

func TestACLVerifyHandler_Verify_NoACLConfigured(t *testing.T) {
	mockService := &MockACLService{
		VerifyAccessFunc: func(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
			return &service.ACLVerifyResponse{
				Allowed: true,
				Headers: map[string]string{},
			}, nil
		},
	}

	r := setupACLVerifyTestRouter(mockService, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/verify", nil)
	req.Header.Set("X-Forwarded-Host", "app.example.com")
	req.Header.Set("X-Forwarded-Uri", "/dashboard")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLVerifyHandler_Verify_IPBypassMatch(t *testing.T) {
	mockService := &MockACLService{
		VerifyAccessFunc: func(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
			// Simulate IP bypass - no further auth required
			return &service.ACLVerifyResponse{
				Allowed:      true,
				RequiresAuth: false,
				Headers:      map[string]string{},
			}, nil
		},
	}

	r := setupACLVerifyTestRouter(mockService, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/verify", nil)
	req.Header.Set("X-Forwarded-Host", "app.example.com")
	req.Header.Set("X-Forwarded-Uri", "/api/resource")
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLVerifyHandler_Verify_IPDeny(t *testing.T) {
	mockService := &MockACLService{
		VerifyAccessFunc: func(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
			return &service.ACLVerifyResponse{
				Allowed:      false,
				RequiresAuth: false,
				Reason:       "IP denied by ACL rules",
			}, nil
		},
	}

	r := setupACLVerifyTestRouter(mockService, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/verify", nil)
	req.Header.Set("X-Forwarded-Host", "app.example.com")
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestACLVerifyHandler_Verify_NoSessionAuthRequired(t *testing.T) {
	mockService := &MockACLService{
		VerifyAccessFunc: func(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
			return &service.ACLVerifyResponse{
				Allowed:      false,
				RequiresAuth: true,
				RedirectURL:  "/auth/login?redirect=app.example.com/dashboard",
			}, nil
		},
	}

	r := setupACLVerifyTestRouter(mockService, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/verify", nil)
	req.Header.Set("X-Forwarded-Host", "app.example.com")
	req.Header.Set("X-Forwarded-Uri", "/dashboard")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Auth-Redirect"))
}

func TestACLVerifyHandler_Verify_ValidSession(t *testing.T) {
	mockService := &MockACLService{
		VerifyAccessFunc: func(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
			return &service.ACLVerifyResponse{
				Allowed: true,
				User: &models.User{
					ID:       1,
					Username: "testuser",
					Email:    "test@example.com",
				},
				Headers: map[string]string{
					"X-Auth-User":       "testuser",
					"X-Auth-User-ID":    "1",
					"X-Auth-User-Email": "test@example.com",
				},
			}, nil
		},
	}

	r := setupACLVerifyTestRouter(mockService, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/verify", nil)
	req.Header.Set("X-Forwarded-Host", "app.example.com")
	req.AddCookie(&http.Cookie{
		Name:  ACLSessionCookieName,
		Value: "valid-session-token",
	})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "testuser", w.Header().Get("X-Auth-User"))
	assert.Equal(t, "1", w.Header().Get("X-Auth-User-ID"))
	assert.Equal(t, "test@example.com", w.Header().Get("X-Auth-User-Email"))
}

func TestACLVerifyHandler_Verify_ValidBasicAuth(t *testing.T) {
	mockService := &MockACLService{
		VerifyAccessFunc: func(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
			// Check that basic auth credentials are passed through
			if request.BasicAuth != nil &&
				request.BasicAuth.Username == "apiuser" &&
				request.BasicAuth.Password == "apipassword" {
				return &service.ACLVerifyResponse{
					Allowed: true,
					Headers: map[string]string{
						"X-Auth-User": "apiuser",
					},
				}, nil
			}
			return &service.ACLVerifyResponse{
				Allowed:      false,
				RequiresAuth: true,
			}, nil
		},
	}

	r := setupACLVerifyTestRouter(mockService, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/verify", nil)
	req.Header.Set("X-Forwarded-Host", "app.example.com")

	// Set basic auth header
	credentials := base64.StdEncoding.EncodeToString([]byte("apiuser:apipassword"))
	req.Header.Set("Authorization", "Basic "+credentials)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "apiuser", w.Header().Get("X-Auth-User"))
}

func TestACLVerifyHandler_Verify_InvalidBasicAuth(t *testing.T) {
	mockService := &MockACLService{
		VerifyAccessFunc: func(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
			// Basic auth fails
			return &service.ACLVerifyResponse{
				Allowed:      false,
				RequiresAuth: true,
			}, nil
		},
	}

	r := setupACLVerifyTestRouter(mockService, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/verify", nil)
	req.Header.Set("X-Forwarded-Host", "app.example.com")

	// Set invalid basic auth header
	credentials := base64.StdEncoding.EncodeToString([]byte("wronguser:wrongpassword"))
	req.Header.Set("Authorization", "Basic "+credentials)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestACLVerifyHandler_Verify_ExpiredSession(t *testing.T) {
	mockService := &MockACLService{
		VerifyAccessFunc: func(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
			// Session expired
			return &service.ACLVerifyResponse{
				Allowed:      false,
				RequiresAuth: true,
				RedirectURL:  "/auth/login",
			}, nil
		},
	}

	r := setupACLVerifyTestRouter(mockService, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/verify", nil)
	req.Header.Set("X-Forwarded-Host", "app.example.com")
	req.AddCookie(&http.Cookie{
		Name:  ACLSessionCookieName,
		Value: "expired-session-token",
	})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestACLVerifyHandler_Verify_InternalError(t *testing.T) {
	mockService := &MockACLService{
		VerifyAccessFunc: func(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
			return nil, errors.New("database connection error")
		},
	}

	r := setupACLVerifyTestRouter(mockService, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/verify", nil)
	req.Header.Set("X-Forwarded-Host", "app.example.com")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestACLVerifyHandler_Verify_XForwardedForMultipleIPs(t *testing.T) {
	var capturedIP string
	mockService := &MockACLService{
		VerifyAccessFunc: func(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
			capturedIP = request.RemoteIP
			return &service.ACLVerifyResponse{Allowed: true}, nil
		},
	}

	r := setupACLVerifyTestRouter(mockService, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/verify", nil)
	req.Header.Set("X-Forwarded-Host", "app.example.com")
	// Multiple IPs in X-Forwarded-For (original client, then proxies)
	req.Header.Set("X-Forwarded-For", "192.168.1.100, 10.0.0.1, 172.16.0.1")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Should extract the first IP (original client)
	assert.Equal(t, "192.168.1.100", capturedIP)
}

func TestACLVerifyHandler_Verify_MalformedBasicAuth(t *testing.T) {
	var capturedBasicAuth *service.BasicAuthCredentials
	mockService := &MockACLService{
		VerifyAccessFunc: func(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
			capturedBasicAuth = request.BasicAuth
			return &service.ACLVerifyResponse{Allowed: true}, nil
		},
	}

	r := setupACLVerifyTestRouter(mockService, nil, nil)

	testCases := []struct {
		name   string
		header string
	}{
		{"not base64", "Basic not-valid-base64!!!"},
		{"missing colon", "Basic " + base64.StdEncoding.EncodeToString([]byte("usernameonly"))},
		{"empty", "Basic "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			capturedBasicAuth = nil
			req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/verify", nil)
			req.Header.Set("X-Forwarded-Host", "app.example.com")
			req.Header.Set("Authorization", tc.header)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			// Should not crash, basic auth should be nil
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Nil(t, capturedBasicAuth)
		})
	}
}

// =============================================================================
// Login Endpoint Tests
// =============================================================================

func TestACLVerifyHandler_Login_Success(t *testing.T) {
	testPassword := "testpassword123"
	testUser := createTestACLUser(testPassword)

	mockUserRepo := &mocks.MockUserRepository{
		GetByUsernameOrEmailFunc: func(identifier string) (*models.User, error) {
			if identifier == "test@example.com" {
				return testUser, nil
			}
			return nil, errors.New("user not found")
		},
	}

	mockACLService := &MockACLService{
		CreateSessionFunc: func(userID int, proxyID *int, ip, userAgent string, ttl int) (*models.ACLSession, error) {
			uid := userID
			return &models.ACLSession{
				ID:           1,
				SessionToken: "new-session-token",
				UserID:       &uid,
				ExpiresAt:    time.Now().Add(24 * time.Hour),
			}, nil
		},
	}

	r := setupACLVerifyTestRouter(mockACLService, mockUserRepo, nil)
	body := `{"email": "test@example.com", "password": "testpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/acl/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check session cookie is set
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == ACLSessionCookieName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)
	assert.Equal(t, "new-session-token", sessionCookie.Value)
	assert.True(t, sessionCookie.HttpOnly)
	assert.True(t, sessionCookie.Secure)
}

func TestACLVerifyHandler_Login_InvalidCredentials_WrongPassword(t *testing.T) {
	testUser := createTestACLUser("correctpassword")

	mockUserRepo := &mocks.MockUserRepository{
		GetByUsernameOrEmailFunc: func(identifier string) (*models.User, error) {
			return testUser, nil
		},
	}

	mockAuditService := &mocks.MockAuditService{}

	r := setupACLVerifyTestRouter(&MockACLService{}, mockUserRepo, mockAuditService)
	body := `{"email": "test@example.com", "password": "wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/acl/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestACLVerifyHandler_Login_InvalidCredentials_UserNotFound(t *testing.T) {
	mockUserRepo := &mocks.MockUserRepository{
		GetByUsernameOrEmailFunc: func(identifier string) (*models.User, error) {
			return nil, errors.New("user not found")
		},
	}

	mockAuditService := &mocks.MockAuditService{}

	r := setupACLVerifyTestRouter(&MockACLService{}, mockUserRepo, mockAuditService)
	body := `{"email": "nonexistent@example.com", "password": "anypassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/acl/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestACLVerifyHandler_Login_MissingEmail(t *testing.T) {
	r := setupACLVerifyTestRouter(&MockACLService{}, nil, nil)
	body := `{"password": "testpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/acl/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLVerifyHandler_Login_MissingPassword(t *testing.T) {
	r := setupACLVerifyTestRouter(&MockACLService{}, nil, nil)
	body := `{"email": "test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/acl/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLVerifyHandler_Login_InvalidJSON(t *testing.T) {
	r := setupACLVerifyTestRouter(&MockACLService{}, nil, nil)
	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/acl/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLVerifyHandler_Login_WithRedirect(t *testing.T) {
	testPassword := "testpassword123"
	testUser := createTestACLUser(testPassword)

	mockUserRepo := &mocks.MockUserRepository{
		GetByUsernameOrEmailFunc: func(identifier string) (*models.User, error) {
			return testUser, nil
		},
	}

	mockACLService := &MockACLService{
		CreateSessionFunc: func(userID int, proxyID *int, ip, userAgent string, ttl int) (*models.ACLSession, error) {
			uid := userID
			return &models.ACLSession{
				ID:           1,
				SessionToken: "new-session-token",
				UserID:       &uid,
				ExpiresAt:    time.Now().Add(24 * time.Hour),
			}, nil
		},
	}

	r := setupACLVerifyTestRouter(mockACLService, mockUserRepo, nil)
	body := `{"email": "test@example.com", "password": "testpassword123", "redirect": "/dashboard"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/acl/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "/dashboard", data["redirect_url"])
}

func TestACLVerifyHandler_Login_SessionCreationError(t *testing.T) {
	testPassword := "testpassword123"
	testUser := createTestACLUser(testPassword)

	mockUserRepo := &mocks.MockUserRepository{
		GetByUsernameOrEmailFunc: func(identifier string) (*models.User, error) {
			return testUser, nil
		},
	}

	mockACLService := &MockACLService{
		CreateSessionFunc: func(userID int, proxyID *int, ip, userAgent string, ttl int) (*models.ACLSession, error) {
			return nil, errors.New("failed to create session")
		},
	}

	r := setupACLVerifyTestRouter(mockACLService, mockUserRepo, nil)
	body := `{"email": "test@example.com", "password": "testpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/acl/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// =============================================================================
// Logout Endpoint Tests
// =============================================================================

func TestACLVerifyHandler_Logout_ValidSession(t *testing.T) {
	revokedToken := ""
	mockACLService := &MockACLService{
		RevokeSessionFunc: func(token string) error {
			revokedToken = token
			return nil
		},
	}

	r := setupACLVerifyTestRouter(mockACLService, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/acl/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  ACLSessionCookieName,
		Value: "valid-session-token",
	})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "valid-session-token", revokedToken)

	// Check session cookie is cleared
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == ACLSessionCookieName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)
	assert.Equal(t, "", sessionCookie.Value)
	assert.True(t, sessionCookie.MaxAge < 0) // Cookie should be expired
}

func TestACLVerifyHandler_Logout_NoSession(t *testing.T) {
	r := setupACLVerifyTestRouter(&MockACLService{}, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/acl/logout", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Should succeed even with no session (idempotent)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLVerifyHandler_Logout_RevokeError(t *testing.T) {
	mockACLService := &MockACLService{
		RevokeSessionFunc: func(token string) error {
			return errors.New("database error")
		},
	}

	r := setupACLVerifyTestRouter(mockACLService, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/acl/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  ACLSessionCookieName,
		Value: "valid-session-token",
	})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Should still succeed (cookie cleared regardless of error)
	assert.Equal(t, http.StatusOK, w.Code)

	// Cookie should still be cleared
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == ACLSessionCookieName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)
	assert.True(t, sessionCookie.MaxAge < 0)
}

// =============================================================================
// Session Endpoint Tests
// =============================================================================

func TestACLVerifyHandler_GetSession_ValidSession(t *testing.T) {
	mockACLService := &MockACLService{
		ValidateSessionFunc: func(token string) (*models.ACLSession, error) {
			userID := 1
			return &models.ACLSession{
				ID:           1,
				SessionToken: token,
				UserID:       &userID,
				ExpiresAt:    time.Now().Add(12 * time.Hour),
				User: &models.User{
					ID:       1,
					Username: "testuser",
					Email:    "test@example.com",
				},
			}, nil
		},
	}

	r := setupACLVerifyTestRouter(mockACLService, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/session", nil)
	req.AddCookie(&http.Cookie{
		Name:  ACLSessionCookieName,
		Value: "valid-session-token",
	})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, true, data["authenticated"])
	assert.Equal(t, float64(1), data["user_id"])
	assert.Equal(t, "testuser", data["username"])
	assert.Equal(t, "test@example.com", data["email"])
}

func TestACLVerifyHandler_GetSession_NoSession(t *testing.T) {
	r := setupACLVerifyTestRouter(&MockACLService{}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/session", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, false, data["authenticated"])
}

func TestACLVerifyHandler_GetSession_ExpiredSession(t *testing.T) {
	mockACLService := &MockACLService{
		ValidateSessionFunc: func(token string) (*models.ACLSession, error) {
			return nil, service.ErrSessionExpired
		},
	}

	r := setupACLVerifyTestRouter(mockACLService, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/session", nil)
	req.AddCookie(&http.Cookie{
		Name:  ACLSessionCookieName,
		Value: "expired-session-token",
	})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, false, data["authenticated"])
	assert.Equal(t, "Session expired", response["message"])

	// Cookie should be cleared
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == ACLSessionCookieName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)
	assert.True(t, sessionCookie.MaxAge < 0)
}

func TestACLVerifyHandler_GetSession_InvalidSession(t *testing.T) {
	mockACLService := &MockACLService{
		ValidateSessionFunc: func(token string) (*models.ACLSession, error) {
			return nil, service.ErrSessionNotFound
		},
	}

	r := setupACLVerifyTestRouter(mockACLService, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/session", nil)
	req.AddCookie(&http.Cookie{
		Name:  ACLSessionCookieName,
		Value: "invalid-session-token",
	})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, false, data["authenticated"])

	// Cookie should be cleared
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == ACLSessionCookieName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)
	assert.True(t, sessionCookie.MaxAge < 0)
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestParseBasicAuth(t *testing.T) {
	testCases := []struct {
		name           string
		header         string
		expectNil      bool
		expectUsername string
		expectPassword string
	}{
		{
			name:           "valid credentials",
			header:         "Basic " + base64.StdEncoding.EncodeToString([]byte("user:password")),
			expectNil:      false,
			expectUsername: "user",
			expectPassword: "password",
		},
		{
			name:           "valid credentials with colon in password",
			header:         "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass:word:with:colons")),
			expectNil:      false,
			expectUsername: "user",
			expectPassword: "pass:word:with:colons",
		},
		{
			name:      "missing Basic prefix",
			header:    base64.StdEncoding.EncodeToString([]byte("user:password")),
			expectNil: true,
		},
		{
			name:      "invalid base64",
			header:    "Basic not-valid-base64!!!",
			expectNil: true,
		},
		{
			name:      "no colon separator",
			header:    "Basic " + base64.StdEncoding.EncodeToString([]byte("usernameonly")),
			expectNil: true,
		},
		{
			name:      "empty after Basic",
			header:    "Basic ",
			expectNil: true,
		},
		{
			name:      "Bearer token (not Basic)",
			header:    "Bearer some-jwt-token",
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseBasicAuth(tc.header)

			if tc.expectNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tc.expectUsername, result.Username)
				assert.Equal(t, tc.expectPassword, result.Password)
			}
		})
	}
}

// =============================================================================
// Integration with Headers
// =============================================================================

func TestACLVerifyHandler_Verify_HeaderParsing(t *testing.T) {
	var capturedRequest *service.ACLVerifyRequest
	mockService := &MockACLService{
		VerifyAccessFunc: func(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
			capturedRequest = request
			return &service.ACLVerifyResponse{Allowed: true}, nil
		},
	}

	r := setupACLVerifyTestRouter(mockService, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/verify", nil)
	req.Header.Set("X-Forwarded-Host", "app.example.com")
	req.Header.Set("X-Forwarded-Uri", "/api/v1/users")
	req.Header.Set("X-Forwarded-Method", "POST")
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedRequest)
	assert.Equal(t, "app.example.com", capturedRequest.Host)
	assert.Equal(t, "/api/v1/users", capturedRequest.Path)
	assert.Equal(t, "POST", capturedRequest.Method)
	assert.Equal(t, "10.0.0.1", capturedRequest.RemoteIP)
}

func TestACLVerifyHandler_Verify_FallbackToRequestValues(t *testing.T) {
	var capturedRequest *service.ACLVerifyRequest
	mockService := &MockACLService{
		VerifyAccessFunc: func(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
			capturedRequest = request
			return &service.ACLVerifyResponse{Allowed: true}, nil
		},
	}

	r := setupACLVerifyTestRouter(mockService, nil, nil)
	// No X-Forwarded-* headers - should fall back to request values
	req := httptest.NewRequest(http.MethodGet, "/api/auth/acl/verify", nil)
	req.Host = "direct.example.com"
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedRequest)
	assert.Equal(t, "direct.example.com", capturedRequest.Host)
	assert.Equal(t, "/api/auth/acl/verify", capturedRequest.Path)
	assert.Equal(t, "GET", capturedRequest.Method)
}
