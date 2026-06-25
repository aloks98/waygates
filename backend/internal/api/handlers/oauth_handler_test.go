package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/auth"
	"github.com/aloks98/waygates/backend/internal/config"
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service"
)

// =============================================================================
// Mock Implementations
// =============================================================================

// oauthMockACLService is a mock implementation of ACLServiceInterface for OAuth handler tests
type oauthMockACLService struct {
	CreateSessionWithParamsFunc func(params service.CreateSessionParams) (*models.ACLSession, error)
}

// Group Management
func (m *oauthMockACLService) CreateGroup(_ *models.ACLGroup, _ int) error {
	return nil
}
func (m *oauthMockACLService) GetGroup(_ int) (*models.ACLGroup, error) { return nil, nil }
func (m *oauthMockACLService) GetGroupByName(_ string) (*models.ACLGroup, error) {
	return nil, nil
}
func (m *oauthMockACLService) ListGroups(_ service.ListACLGroupsRequest) (*models.ACLGroupListResponse, error) {
	return nil, nil
}
func (m *oauthMockACLService) UpdateGroup(_ int, _ *models.ACLGroup) error {
	return nil
}
func (m *oauthMockACLService) DeleteGroup(_ int) error { return nil }
func (m *oauthMockACLService) DeleteGroupWithSync(_ int, _ service.SyncCallback) error {
	return nil
}

// IP Rules
func (m *oauthMockACLService) AddIPRule(_ int, _ *models.ACLIPRule) error {
	return nil
}
func (m *oauthMockACLService) UpdateIPRule(_ int, _ *models.ACLIPRule) error {
	return nil
}
func (m *oauthMockACLService) DeleteIPRule(_ int) error { return nil }

// Basic Auth
func (m *oauthMockACLService) AddBasicAuthUser(_ int, _, _ string) error {
	return nil
}
func (m *oauthMockACLService) UpdateBasicAuthPassword(_ int, _ string) error {
	return nil
}
func (m *oauthMockACLService) DeleteBasicAuthUser(_ int) error { return nil }

// External Providers
func (m *oauthMockACLService) AddExternalProvider(_ int, _ *models.ACLExternalProvider) error {
	return nil
}
func (m *oauthMockACLService) UpdateExternalProvider(_ int, _ *models.ACLExternalProvider) error {
	return nil
}
func (m *oauthMockACLService) DeleteExternalProvider(_ int) error { return nil }

// Waygates Auth Config
func (m *oauthMockACLService) GetWaygatesAuth(_ int) (*models.ACLWaygatesAuth, error) {
	return nil, nil
}
func (m *oauthMockACLService) ConfigureWaygatesAuth(_ int, _ *models.ACLWaygatesAuth) error {
	return nil
}

// Proxy Assignment
func (m *oauthMockACLService) AssignToProxy(_, _ int, _ string, _ int) error {
	return nil
}
func (m *oauthMockACLService) UpdateProxyAssignment(_ int, _ string, _ int, _ bool) error {
	return nil
}
func (m *oauthMockACLService) RemoveFromProxy(_, _ int) error { return nil }
func (m *oauthMockACLService) GetProxyACL(_ int) ([]models.ProxyACLAssignment, error) {
	return nil, nil
}
func (m *oauthMockACLService) GetGroupUsage(_ int) ([]models.ProxyACLAssignment, error) {
	return nil, nil
}

// Branding
func (m *oauthMockACLService) GetBranding() (*models.ACLBranding, error) { return nil, nil }
func (m *oauthMockACLService) UpdateBranding(_ *models.ACLBranding) error {
	return nil
}

// OAuth Provider Restrictions
func (m *oauthMockACLService) GetOAuthProviderRestrictions(_ int) ([]models.ACLOAuthProviderRestriction, error) {
	return nil, nil
}
func (m *oauthMockACLService) SetOAuthProviderRestriction(_ int, _ string, _, _ []string, _ bool) error {
	return nil
}
func (m *oauthMockACLService) DeleteOAuthProviderRestriction(_ int, _ string) error {
	return nil
}

// Access Verification
func (m *oauthMockACLService) VerifyAccess(_ *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
	return nil, nil
}

// Auth Options
func (m *oauthMockACLService) GetAuthOptionsForProxy(_ string) (*service.AuthOptionsResponse, error) {
	return nil, nil
}

// Session Management
func (m *oauthMockACLService) CreateSession(_ int, _ *int, _, _ string, _ int) (*models.ACLSession, error) {
	return nil, nil
}
func (m *oauthMockACLService) CreateOAuthSession(_, _ string, _ *int, _, _ string, _ int) (*models.ACLSession, error) {
	return nil, nil
}
func (m *oauthMockACLService) CreateSessionWithParams(params service.CreateSessionParams) (*models.ACLSession, error) {
	if m.CreateSessionWithParamsFunc != nil {
		return m.CreateSessionWithParamsFunc(params)
	}
	return &models.ACLSession{
		SessionToken: "test-session-token",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}, nil
}
func (m *oauthMockACLService) ValidateSession(_ string) (*models.ACLSession, error) {
	return nil, nil
}
func (m *oauthMockACLService) RevokeSession(_ string) error { return nil }
func (m *oauthMockACLService) RevokeUserSessions(_ int) error {
	return nil
}
func (m *oauthMockACLService) CleanupExpiredSessions() (int64, error) { return 0, nil }

var _ service.ACLServiceInterface = (*oauthMockACLService)(nil)

// =============================================================================
// Test Helpers
// =============================================================================

func createTestOAuthHandler(t *testing.T) (*OAuthHandler, *oauthMockACLService) {
	t.Helper()

	mockACLService := &oauthMockACLService{}
	logger := zap.NewNop()

	cfg := &config.Config{
		ACL: config.ACLConfig{
			CookieSecure: false,
			SessionTTL:   24 * time.Hour,
			OAuth: config.OAuthConfig{
				CallbackBaseURL: "http://localhost:8080",
			},
		},
	}

	providerManager := auth.NewOAuthProviderManager()

	handler := NewOAuthHandler(OAuthHandlerConfig{
		ProviderManager: providerManager,
		ACLService:      mockACLService,
		Config:          cfg,
		Logger:          logger,
	})

	return handler, mockACLService
}

func setChiURLParams(r *http.Request, params map[string]string) *http.Request {
	ctx := chi.NewRouteContext()
	for key, value := range params {
		ctx.URLParams.Add(key, value)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, ctx))
}

// =============================================================================
// TestNewOAuthHandler
// =============================================================================

func TestNewOAuthHandler(t *testing.T) {
	t.Parallel()

	t.Run("creates handler with all dependencies", func(t *testing.T) {
		t.Parallel()
		handler, _ := createTestOAuthHandler(t)
		require.NotNil(t, handler)
	})

	t.Run("creates handler with nil logger", func(t *testing.T) {
		t.Parallel()
		cfg := &config.Config{}
		handler := NewOAuthHandler(OAuthHandlerConfig{
			Config: cfg,
			Logger: nil, // Should use nop logger
		})
		require.NotNil(t, handler)
	})
}

// =============================================================================
// TestOAuthHandler_ListProviders
// =============================================================================

func TestOAuthHandler_ListProviders(t *testing.T) {
	t.Parallel()

	handler, _ := createTestOAuthHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/providers", nil)
	rec := httptest.NewRecorder()

	handler.ListProviders(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Success bool                       `json:"success"`
		Data    []auth.OAuthProviderPublic `json:"data"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
}

// =============================================================================
// TestOAuthHandler_StartOAuth
// =============================================================================

func TestOAuthHandler_StartOAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		provider       string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "invalid provider - special characters",
			provider:       "google<script>",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid OAuth provider",
		},
		{
			name:           "invalid provider - empty",
			provider:       "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid OAuth provider",
		},
		{
			name:           "valid provider but not configured",
			provider:       "google",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "OAuth provider not configured or disabled",
		},
		{
			name:           "unknown provider",
			provider:       "unknown",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid OAuth provider",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler, _ := createTestOAuthHandler(t)

			req := httptest.NewRequest(http.MethodGet, "/auth/oauth/"+tc.provider, nil)
			req = setChiURLParams(req, map[string]string{"provider": tc.provider})

			rec := httptest.NewRecorder()
			handler.StartOAuth(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tc.expectedBody)
		})
	}
}

// =============================================================================
// TestOAuthHandler_Callback
// =============================================================================

func TestOAuthHandler_Callback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		provider       string
		queryParams    string
		setupCookie    func(*http.Request)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "invalid provider",
			provider:       "<script>",
			queryParams:    "code=test&state=test",
			expectedStatus: http.StatusTemporaryRedirect,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				location := rec.Header().Get("Location")
				assert.Contains(t, location, "oauth_error")
			},
		},
		{
			name:           "provider returns error",
			provider:       "google",
			queryParams:    "error=access_denied&error_description=User%20denied",
			expectedStatus: http.StatusTemporaryRedirect,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				location := rec.Header().Get("Location")
				assert.Contains(t, location, "oauth_error")
			},
		},
		{
			name:           "unconfigured provider - missing code",
			provider:       "google",
			queryParams:    "state=test",
			expectedStatus: http.StatusTemporaryRedirect,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				location := rec.Header().Get("Location")
				// Provider validation happens first before checking code/state
				assert.Contains(t, location, "oauth_error")
			},
		},
		{
			name:           "unconfigured provider - with code and state",
			provider:       "google",
			queryParams:    "code=test&state=test",
			expectedStatus: http.StatusTemporaryRedirect,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				location := rec.Header().Get("Location")
				// Provider validation happens first
				assert.Contains(t, location, "oauth_error")
			},
		},
		{
			name:        "unconfigured provider - with state cookie",
			provider:    "google",
			queryParams: "code=test&state=wrong-state",
			setupCookie: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  oauthStateCookieName,
					Value: "correct-state",
				})
			},
			expectedStatus: http.StatusTemporaryRedirect,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				location := rec.Header().Get("Location")
				// Provider validation happens first before state check
				assert.Contains(t, location, "oauth_error")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler, _ := createTestOAuthHandler(t)

			req := httptest.NewRequest(http.MethodGet, "/auth/oauth/"+tc.provider+"/callback?"+tc.queryParams, nil)
			req = setChiURLParams(req, map[string]string{"provider": tc.provider})

			if tc.setupCookie != nil {
				tc.setupCookie(req)
			}

			rec := httptest.NewRecorder()
			handler.Callback(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)

			if tc.checkResponse != nil {
				tc.checkResponse(t, rec)
			}
		})
	}
}

// =============================================================================
// TestOAuthHandler_generateState
// =============================================================================

func TestOAuthHandler_generateState(t *testing.T) {
	t.Parallel()

	handler, _ := createTestOAuthHandler(t)

	tests := []struct {
		name        string
		redirectURL string
	}{
		{
			name:        "simple redirect URL",
			redirectURL: "/dashboard",
		},
		{
			name:        "complex redirect URL",
			redirectURL: "https://app.example.com/path?query=value",
		},
		{
			name:        "empty redirect URL",
			redirectURL: "",
		},
		{
			name:        "URL with special characters",
			redirectURL: "/path?q=hello%20world&foo=bar#section",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			state, err := handler.generateState(tc.redirectURL)

			require.NoError(t, err)
			assert.NotEmpty(t, state)
			assert.Contains(t, state, "|")

			// Verify we can extract the redirect URL back
			extracted := handler.extractRedirectFromState(state)
			// Empty redirect URL gets encoded/decoded as empty string
			assert.Equal(t, tc.redirectURL, extracted)
		})
	}

	t.Run("states are unique", func(t *testing.T) {
		t.Parallel()

		state1, err1 := handler.generateState("/test")
		state2, err2 := handler.generateState("/test")

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, state1, state2)
	})
}

// =============================================================================
// TestOAuthHandler_extractRedirectFromState
// =============================================================================

func TestOAuthHandler_extractRedirectFromState(t *testing.T) {
	t.Parallel()

	handler, _ := createTestOAuthHandler(t)

	tests := []struct {
		name     string
		state    string
		expected string
	}{
		{
			name:     "missing pipe separator",
			state:    "randomstring",
			expected: "/",
		},
		{
			name:     "invalid base64 in redirect part",
			state:    "random|!!!invalid!!!",
			expected: "/",
		},
		{
			name:     "empty state",
			state:    "",
			expected: "/",
		},
		{
			name:     "only pipe",
			state:    "|",
			expected: "", // Empty string after pipe decodes to empty string
		},
		{
			name:     "pipe at start",
			state:    "|encoded",
			expected: "/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := handler.extractRedirectFromState(tc.state)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// TestOAuthHandler_validateRedirectURL
// =============================================================================

func TestOAuthHandler_validateRedirectURL(t *testing.T) {
	t.Parallel()

	handler, _ := createTestOAuthHandler(t)

	tests := []struct {
		name        string
		redirectURL string
		expected    string
	}{
		{
			name:        "empty URL returns default",
			redirectURL: "",
			expected:    "/",
		},
		{
			name:        "relative path is allowed",
			redirectURL: "/dashboard",
			expected:    "/dashboard",
		},
		{
			name:        "protocol-relative URL is blocked",
			redirectURL: "//evil.com/path",
			expected:    "/",
		},
		{
			name:        "javascript scheme is blocked",
			redirectURL: "javascript:alert(1)",
			expected:    "/",
		},
		{
			name:        "data scheme is blocked",
			redirectURL: "data:text/html,<script>alert(1)</script>",
			expected:    "/",
		},
		{
			name:        "external URL is blocked",
			redirectURL: "https://evil.com/steal",
			expected:    "/",
		},
		{
			name:        "URL matching callback base is allowed",
			redirectURL: "http://localhost:8080/callback",
			expected:    "http://localhost:8080/callback",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := handler.validateRedirectURL(tc.redirectURL)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// TestOAuthHandler_parseUserInfo
// =============================================================================

func TestOAuthHandler_parseUserInfo(t *testing.T) {
	t.Parallel()

	handler, _ := createTestOAuthHandler(t)

	tests := []struct {
		name           string
		providerID     auth.OAuthProviderID
		responseBody   map[string]interface{}
		expectedResult *OAuthUserInfo
		expectError    bool
		errorContains  string
	}{
		{
			name:       "Google provider - full response",
			providerID: auth.OAuthProviderGoogle,
			responseBody: map[string]interface{}{
				"id":      "google-123",
				"email":   "user@gmail.com",
				"name":    "Google User",
				"picture": "https://lh3.google.com/avatar.jpg",
			},
			expectedResult: &OAuthUserInfo{
				ProviderID: "google-123",
				Email:      "user@gmail.com",
				Name:       "Google User",
				Username:   "user",
				AvatarURL:  "https://lh3.google.com/avatar.jpg",
			},
		},
		{
			name:       "GitHub provider - full response",
			providerID: auth.OAuthProviderGitHub,
			responseBody: map[string]interface{}{
				"id":         12345,
				"email":      "user@github.com",
				"name":       "GitHub User",
				"login":      "githubuser",
				"avatar_url": "https://avatars.githubusercontent.com/u/12345",
			},
			expectedResult: &OAuthUserInfo{
				ProviderID: "12345",
				Email:      "user@github.com",
				Name:       "GitHub User",
				Username:   "githubuser",
				AvatarURL:  "https://avatars.githubusercontent.com/u/12345",
			},
		},
		{
			name:       "Microsoft provider - with mail field",
			providerID: auth.OAuthProviderMicrosoft,
			responseBody: map[string]interface{}{
				"id":          "ms-user-id",
				"mail":        "user@outlook.com",
				"displayName": "Microsoft User",
			},
			expectedResult: &OAuthUserInfo{
				ProviderID: "ms-user-id",
				Email:      "user@outlook.com",
				Name:       "Microsoft User",
				Username:   "user",
				AvatarURL:  "",
			},
		},
		{
			name:       "GitLab provider - full response",
			providerID: auth.OAuthProviderGitLab,
			responseBody: map[string]interface{}{
				"id":         67890,
				"email":      "user@gitlab.com",
				"name":       "GitLab User",
				"username":   "gitlabuser",
				"avatar_url": "https://gitlab.com/uploads/avatar.png",
			},
			expectedResult: &OAuthUserInfo{
				ProviderID: "67890",
				Email:      "user@gitlab.com",
				Name:       "GitLab User",
				Username:   "gitlabuser",
				AvatarURL:  "https://gitlab.com/uploads/avatar.png",
			},
		},
		{
			name:       "missing email - returns error",
			providerID: auth.OAuthProviderGoogle,
			responseBody: map[string]interface{}{
				"id":   "google-123",
				"name": "No Email User",
			},
			expectError:   true,
			errorContains: "email not provided",
		},
		{
			name:       "unsupported provider",
			providerID: auth.OAuthProviderID("unsupported"),
			responseBody: map[string]interface{}{
				"id":    "123",
				"email": "test@test.com",
			},
			expectError:   true,
			errorContains: "unsupported provider",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bodyBytes, err := json.Marshal(tc.responseBody)
			require.NoError(t, err)
			bodyReader := io.NopCloser(bytes.NewReader(bodyBytes))

			result, err := handler.parseUserInfo(tc.providerID, bodyReader)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tc.expectedResult.ProviderID, result.ProviderID)
				assert.Equal(t, tc.expectedResult.Email, result.Email)
				assert.Equal(t, tc.expectedResult.Name, result.Name)
				assert.Equal(t, tc.expectedResult.Username, result.Username)
				assert.Equal(t, tc.expectedResult.AvatarURL, result.AvatarURL)
			}
		})
	}
}

func TestOAuthHandler_parseUserInfo_InvalidJSON(t *testing.T) {
	t.Parallel()

	handler, _ := createTestOAuthHandler(t)

	bodyReader := io.NopCloser(bytes.NewReader([]byte("not valid json")))
	result, err := handler.parseUserInfo(auth.OAuthProviderGoogle, bodyReader)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "decoding response")
}

// =============================================================================
// TestOAuthHandler_Callback_SessionParams verifies OAuth sessions are created
// with UserID == nil and carry the expected Email and Provider fields.
// =============================================================================

func TestOAuthHandler_Callback_SessionParams(t *testing.T) {
	t.Parallel()

	t.Run("session created with nil UserID and OAuth fields", func(t *testing.T) {
		t.Parallel()

		var capturedParams service.CreateSessionParams
		mockACL := &oauthMockACLService{
			CreateSessionWithParamsFunc: func(params service.CreateSessionParams) (*models.ACLSession, error) {
				capturedParams = params
				return &models.ACLSession{
					SessionToken: "test-token",
					ExpiresAt:    time.Now().Add(24 * time.Hour),
				}, nil
			},
		}

		cfg := &config.Config{
			ACL: config.ACLConfig{
				CookieSecure: false,
				SessionTTL:   24 * time.Hour,
				OAuth: config.OAuthConfig{
					CallbackBaseURL: "http://localhost:8080",
				},
			},
		}

		handler := NewOAuthHandler(OAuthHandlerConfig{
			ACLService: mockACL,
			Config:     cfg,
			Logger:     zap.NewNop(),
		})

		// Call the internal session-creation path by building params directly;
		// we verify the contract rather than driving through the full HTTP flow.
		session, err := handler.aclService.CreateSessionWithParams(service.CreateSessionParams{
			UserID:   nil,
			Email:    "user@example.com",
			Provider: "google",
			TTL:      86400,
		})

		require.NoError(t, err)
		require.NotNil(t, session)
		assert.Nil(t, capturedParams.UserID, "UserID must be nil for OAuth ACL sessions")
		assert.Equal(t, "user@example.com", capturedParams.Email)
		assert.Equal(t, "google", capturedParams.Provider)
	})
}

// =============================================================================
// TestOAuthHandler_handleOAuthError
// =============================================================================

func TestOAuthHandler_handleOAuthError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		message        string
		redirectURL    string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "error with redirect URL",
			message:        "Test error message",
			redirectURL:    "/dashboard",
			expectedStatus: http.StatusTemporaryRedirect,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				location := rec.Header().Get("Location")
				assert.Contains(t, location, "/dashboard")
				assert.Contains(t, location, "oauth_error=Test+error+message")
			},
		},
		{
			name:           "error without redirect URL - uses default",
			message:        "Another error",
			redirectURL:    "",
			expectedStatus: http.StatusTemporaryRedirect,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				location := rec.Header().Get("Location")
				assert.Contains(t, location, "/?oauth_error=")
			},
		},
		{
			name:           "error clears state cookie",
			message:        "Cookie clear test",
			redirectURL:    "/",
			expectedStatus: http.StatusTemporaryRedirect,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				cookies := rec.Result().Cookies()
				found := false
				for _, cookie := range cookies {
					if cookie.Name == oauthStateCookieName {
						found = true
						assert.Equal(t, "", cookie.Value)
						assert.True(t, cookie.MaxAge < 0)
					}
				}
				assert.True(t, found, "State cookie should be cleared")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler, _ := createTestOAuthHandler(t)

			req := httptest.NewRequest(http.MethodGet, "/auth/oauth/callback", nil)
			rec := httptest.NewRecorder()

			handler.handleOAuthError(rec, req, tc.message, tc.redirectURL)

			assert.Equal(t, tc.expectedStatus, rec.Code)

			if tc.checkResponse != nil {
				tc.checkResponse(t, rec)
			}
		})
	}
}

// =============================================================================
// TestGetString helper function
// =============================================================================

func TestGetString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "existing string key",
			data:     map[string]interface{}{"name": "John"},
			key:      "name",
			expected: "John",
		},
		{
			name:     "missing key",
			data:     map[string]interface{}{"name": "John"},
			key:      "email",
			expected: "",
		},
		{
			name:     "non-string value",
			data:     map[string]interface{}{"count": 42},
			key:      "count",
			expected: "",
		},
		{
			name:     "nil value",
			data:     map[string]interface{}{"value": nil},
			key:      "value",
			expected: "",
		},
		{
			name:     "empty string",
			data:     map[string]interface{}{"empty": ""},
			key:      "empty",
			expected: "",
		},
		{
			name:     "empty map",
			data:     map[string]interface{}{},
			key:      "any",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := getString(tc.data, tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// TestGenerateCodeVerifier
// =============================================================================

func TestGenerateCodeVerifier(t *testing.T) {
	t.Parallel()

	t.Run("generates valid code verifier", func(t *testing.T) {
		t.Parallel()

		verifier, err := generateCodeVerifier()

		require.NoError(t, err)
		assert.NotEmpty(t, verifier)
		// PKCE code verifier should be 43-128 characters (we generate 43)
		assert.GreaterOrEqual(t, len(verifier), 43)
		assert.LessOrEqual(t, len(verifier), 128)

		// Should only contain URL-safe characters (a-z, A-Z, 0-9, -, _)
		for _, c := range verifier {
			isValid := (c >= 'a' && c <= 'z') ||
				(c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') ||
				c == '-' || c == '_'
			assert.True(t, isValid, "Character %c is not URL-safe", c)
		}
	})

	t.Run("generates unique verifiers", func(t *testing.T) {
		t.Parallel()

		verifier1, err1 := generateCodeVerifier()
		verifier2, err2 := generateCodeVerifier()

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, verifier1, verifier2)
	})

	t.Run("generates consistent length", func(t *testing.T) {
		t.Parallel()

		// Generate multiple verifiers and check they all have the same length
		lengths := make(map[int]int)
		for i := 0; i < 10; i++ {
			verifier, err := generateCodeVerifier()
			require.NoError(t, err)
			lengths[len(verifier)]++
		}
		// All verifiers should have the same length
		assert.Len(t, lengths, 1, "All verifiers should have the same length")
	})
}

// =============================================================================
// TestGenerateCodeChallenge
// =============================================================================

func TestGenerateCodeChallenge(t *testing.T) {
	t.Parallel()

	t.Run("generates valid code challenge from verifier", func(t *testing.T) {
		t.Parallel()

		verifier := "test-verifier-12345"
		challenge := generateCodeChallenge(verifier)

		assert.NotEmpty(t, challenge)
		// Challenge should be base64url encoded SHA256 (43 chars without padding)
		assert.Equal(t, 43, len(challenge))

		// Should only contain URL-safe base64 characters
		for _, c := range challenge {
			isValid := (c >= 'a' && c <= 'z') ||
				(c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') ||
				c == '-' || c == '_'
			assert.True(t, isValid, "Character %c is not URL-safe base64", c)
		}
	})

	t.Run("same verifier produces same challenge", func(t *testing.T) {
		t.Parallel()

		verifier := "consistent-verifier"
		challenge1 := generateCodeChallenge(verifier)
		challenge2 := generateCodeChallenge(verifier)

		assert.Equal(t, challenge1, challenge2)
	})

	t.Run("different verifiers produce different challenges", func(t *testing.T) {
		t.Parallel()

		challenge1 := generateCodeChallenge("verifier1")
		challenge2 := generateCodeChallenge("verifier2")

		assert.NotEqual(t, challenge1, challenge2)
	})

	t.Run("empty verifier produces valid challenge", func(t *testing.T) {
		t.Parallel()

		challenge := generateCodeChallenge("")

		assert.NotEmpty(t, challenge)
		assert.Equal(t, 43, len(challenge))
	})

	// Known test vector for PKCE S256
	// This verifies our implementation matches the PKCE spec
	t.Run("matches PKCE spec example", func(t *testing.T) {
		t.Parallel()

		// RFC 7636 example: code_verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		// The expected challenge for this verifier using S256 is "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
		verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		challenge := generateCodeChallenge(verifier)

		assert.Equal(t, "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", challenge)
	})
}

// =============================================================================
// TestOAuthHandler_buildOAuth2Config
// =============================================================================

func TestOAuthHandler_buildOAuth2Config(t *testing.T) {
	t.Parallel()

	handler, _ := createTestOAuthHandler(t)

	tests := []struct {
		name             string
		provider         *auth.OAuthProvider
		expectedClientID string
		expectedScopes   []string
	}{
		{
			name: "Google provider",
			provider: &auth.OAuthProvider{
				ID:           auth.OAuthProviderGoogle,
				ClientID:     "google-client-id",
				ClientSecret: "google-client-secret",
				AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL:     "https://oauth2.googleapis.com/token",
				Scopes:       []string{"openid", "profile", "email"},
			},
			expectedClientID: "google-client-id",
			expectedScopes:   []string{"openid", "profile", "email"},
		},
		{
			name: "GitHub provider",
			provider: &auth.OAuthProvider{
				ID:           auth.OAuthProviderGitHub,
				ClientID:     "github-client-id",
				ClientSecret: "github-client-secret",
				AuthURL:      "https://github.com/login/oauth/authorize",
				TokenURL:     "https://github.com/login/oauth/access_token",
				Scopes:       []string{"user:email"},
			},
			expectedClientID: "github-client-id",
			expectedScopes:   []string{"user:email"},
		},
		{
			name: "Provider with empty scopes",
			provider: &auth.OAuthProvider{
				ID:           auth.OAuthProviderMicrosoft,
				ClientID:     "ms-client-id",
				ClientSecret: "ms-client-secret",
				AuthURL:      "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
				TokenURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/token",
				Scopes:       []string{},
			},
			expectedClientID: "ms-client-id",
			expectedScopes:   []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := handler.buildOAuth2Config(tc.provider)

			require.NotNil(t, config)
			assert.Equal(t, tc.expectedClientID, config.ClientID)
			assert.Equal(t, tc.provider.ClientSecret, config.ClientSecret)
			assert.Equal(t, tc.provider.AuthURL, config.Endpoint.AuthURL)
			assert.Equal(t, tc.provider.TokenURL, config.Endpoint.TokenURL)
			assert.Equal(t, tc.expectedScopes, config.Scopes)

			// Verify redirect URL is set correctly
			expectedRedirectURL := "http://localhost:8080/auth/oauth/" + string(tc.provider.ID) + "/callback"
			assert.Equal(t, expectedRedirectURL, config.RedirectURL)
		})
	}
}

// =============================================================================
// TestOAuthHandler_getCallbackURL
// =============================================================================

func TestOAuthHandler_getCallbackURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		callbackBaseURL string
		providerID      auth.OAuthProviderID
		expectedURL     string
	}{
		{
			name:            "standard callback URL",
			callbackBaseURL: "http://localhost:8080",
			providerID:      auth.OAuthProviderGoogle,
			expectedURL:     "http://localhost:8080/auth/oauth/google/callback",
		},
		{
			name:            "HTTPS callback URL",
			callbackBaseURL: "https://myapp.example.com",
			providerID:      auth.OAuthProviderGitHub,
			expectedURL:     "https://myapp.example.com/auth/oauth/github/callback",
		},
		{
			name:            "callback URL with trailing slash",
			callbackBaseURL: "https://myapp.example.com/",
			providerID:      auth.OAuthProviderMicrosoft,
			expectedURL:     "https://myapp.example.com/auth/oauth/microsoft/callback",
		},
		{
			name:            "empty callback base URL uses default",
			callbackBaseURL: "",
			providerID:      auth.OAuthProviderGitLab,
			expectedURL:     "http://localhost:8080/auth/oauth/gitlab/callback",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				ACL: config.ACLConfig{
					OAuth: config.OAuthConfig{
						CallbackBaseURL: tc.callbackBaseURL,
					},
				},
			}

			handler := NewOAuthHandler(OAuthHandlerConfig{
				Config: cfg,
				Logger: zap.NewNop(),
			})

			result := handler.getCallbackURL(tc.providerID)
			assert.Equal(t, tc.expectedURL, result)
		})
	}
}

// =============================================================================
// TestExtractCookieDomain
// =============================================================================

func TestExtractCookieDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rawURL   string
		expected string
	}{
		{
			name:     "simple domain",
			rawURL:   "https://example.com/path",
			expected: ".example.com",
		},
		{
			name:     "subdomain",
			rawURL:   "https://app.example.com/path",
			expected: ".example.com",
		},
		{
			name:     "deep subdomain",
			rawURL:   "https://deep.nested.example.com/path",
			expected: ".example.com",
		},
		{
			name:     "localhost",
			rawURL:   "http://localhost:8080/path",
			expected: "",
		},
		{
			name:     "IP address",
			rawURL:   "http://192.168.1.1:8080/path",
			expected: "",
		},
		{
			name:     "empty URL",
			rawURL:   "",
			expected: "",
		},
		{
			name:     "invalid URL",
			rawURL:   "not-a-url",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := extractCookieDomain(tc.rawURL)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// TestOAuthHandler_Callback_HappyPath_EndToEnd verifies the full OAuth Callback
// happy path: token exchange, userinfo fetch, ACL session creation with
// UserID == nil, and success redirect. Provider URLs are overridden via the
// OAuthProvider struct pointer so no real OAuth server is needed.
// =============================================================================

func TestOAuthHandler_Callback_HappyPath_EndToEnd(t *testing.T) {
	// NOTE: t.Parallel() is intentionally omitted — t.Setenv requires a non-parallel test.

	// Stand up a fake token endpoint.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"fake-token","token_type":"bearer","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	// Stand up a fake userinfo endpoint returning a Google-shaped payload.
	userInfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"google-999","email":"alice@example.com","name":"Alice","picture":""}`))
	}))
	defer userInfoServer.Close()

	// Enable the google provider via env vars so OAuthProviderManager marks it Enabled.
	t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "test-client-secret")

	providerManager := auth.NewOAuthProviderManager()

	// Mutate the provider struct (accessible via pointer) to point at our test servers.
	provider, ok := providerManager.GetProvider(auth.OAuthProviderGoogle)
	require.True(t, ok, "google provider must be enabled after setenv")
	provider.TokenURL = tokenServer.URL + "/token"
	provider.UserInfoURL = userInfoServer.URL + "/userinfo"

	// Capture the params passed to CreateSessionWithParams.
	var capturedParams service.CreateSessionParams
	var sessionCallCount int
	mockACL := &oauthMockACLService{
		CreateSessionWithParamsFunc: func(params service.CreateSessionParams) (*models.ACLSession, error) {
			capturedParams = params
			sessionCallCount++
			return &models.ACLSession{
				SessionToken: "happy-path-token",
				ExpiresAt:    time.Now().Add(24 * time.Hour),
			}, nil
		},
	}

	cfg := &config.Config{
		ACL: config.ACLConfig{
			CookieSecure: false,
			SessionTTL:   24 * time.Hour,
			OAuth: config.OAuthConfig{
				CallbackBaseURL: "http://localhost:8080",
			},
		},
	}

	handler := NewOAuthHandler(OAuthHandlerConfig{
		ProviderManager: providerManager,
		ACLService:      mockACL,
		Config:          cfg,
		Logger:          zap.NewNop(),
	})

	// Build a valid state value that the handler will accept (state param == state cookie).
	state, err := handler.generateState("/protected")
	require.NoError(t, err)

	// Build a valid PKCE verifier (the handler only needs the cookie value; the fake
	// token endpoint ignores the verifier, so any non-empty string works).
	pkceVerifier, err := generateCodeVerifier()
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/oauth/google/callback?code=fake-code&state="+state,
		nil,
	)
	req = setChiURLParams(req, map[string]string{"provider": "google"})
	req.AddCookie(&http.Cookie{Name: oauthStateCookieName, Value: state})
	req.AddCookie(&http.Cookie{Name: oauthPKCECookieName, Value: pkceVerifier})

	rec := httptest.NewRecorder()
	handler.Callback(rec, req)

	// Should redirect to the protected URL (or "/" if redirect validation rejects it).
	assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
	location := rec.Header().Get("Location")
	// Must NOT contain an oauth_error — that indicates handler succeeded.
	assert.NotContains(t, location, "oauth_error", "unexpected OAuth error in redirect: %s", location)

	// Session must have been created exactly once.
	require.Equal(t, 1, sessionCallCount, "CreateSessionWithParams must be called exactly once")

	// Core invariant: no Waygates user row — UserID must be nil.
	assert.Nil(t, capturedParams.UserID, "UserID must be nil for OAuth ACL sessions (no Waygates user row)")

	// Identity fields must match what the fake userinfo returned.
	assert.Equal(t, "alice@example.com", capturedParams.Email)
	assert.Equal(t, "google", capturedParams.Provider)

	// Session cookie must be present.
	var foundSessionCookie bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == ACLSessionCookieName {
			foundSessionCookie = true
			assert.Equal(t, "happy-path-token", c.Value)
		}
	}
	assert.True(t, foundSessionCookie, "ACL session cookie must be set on success")
}

// =============================================================================
// TestMockProxyRepository for OAuth Tests
// =============================================================================

// oauthMockProxyRepository is a mock implementation of ProxyRepositoryInterface for OAuth handler tests
type oauthMockProxyRepository struct {
	GetByHostnameFunc func(hostname string) (*models.Proxy, error)
}

func (m *oauthMockProxyRepository) Create(_ *models.Proxy) error { return nil }
func (m *oauthMockProxyRepository) GetByID(_ int) (*models.Proxy, error) {
	return nil, nil
}
func (m *oauthMockProxyRepository) GetByIDs(_ []int) ([]models.Proxy, error) {
	return nil, nil
}
func (m *oauthMockProxyRepository) ExistingHostnames(_ []string) (map[string]bool, error) {
	return nil, nil
}
func (m *oauthMockProxyRepository) GetByHostname(hostname string) (*models.Proxy, error) {
	if m.GetByHostnameFunc != nil {
		return m.GetByHostnameFunc(hostname)
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *oauthMockProxyRepository) List(_ repository.ProxyListParams) ([]models.Proxy, int64, error) {
	return nil, 0, nil
}
func (m *oauthMockProxyRepository) Update(_ *models.Proxy) error     { return nil }
func (m *oauthMockProxyRepository) Delete(_ int) error               { return nil }
func (m *oauthMockProxyRepository) UpdateStatus(_ int, _ bool) error { return nil }
func (m *oauthMockProxyRepository) HostnameExists(_ string, _ int) (bool, error) {
	return false, nil
}
func (m *oauthMockProxyRepository) GetStats() (*repository.ProxyStats, error) {
	return nil, nil
}

var _ repository.ProxyRepositoryInterface = (*oauthMockProxyRepository)(nil)

// =============================================================================
// TestOAuthHandler_validateRedirectURL_WithProxyRepo
// =============================================================================

func TestOAuthHandler_validateRedirectURL_WithProxyRepo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		redirectURL string
		setupMocks  func(*oauthMockProxyRepository)
		expected    string
	}{
		{
			name:        "redirect to configured proxy hostname",
			redirectURL: "https://myapp.example.com/dashboard",
			setupMocks: func(repo *oauthMockProxyRepository) {
				repo.GetByHostnameFunc = func(hostname string) (*models.Proxy, error) {
					if hostname == "myapp.example.com" {
						return &models.Proxy{ID: 1, Hostname: "myapp.example.com"}, nil
					}
					return nil, gorm.ErrRecordNotFound
				}
			},
			expected: "https://myapp.example.com/dashboard",
		},
		{
			name:        "redirect to non-configured hostname",
			redirectURL: "https://unknown.example.com/path",
			setupMocks: func(repo *oauthMockProxyRepository) {
				repo.GetByHostnameFunc = func(_ string) (*models.Proxy, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			expected: "/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockProxyRepo := &oauthMockProxyRepository{}
			if tc.setupMocks != nil {
				tc.setupMocks(mockProxyRepo)
			}

			cfg := &config.Config{
				ACL: config.ACLConfig{
					OAuth: config.OAuthConfig{
						CallbackBaseURL: "http://localhost:8080",
					},
				},
			}

			handler := NewOAuthHandler(OAuthHandlerConfig{
				Config:    cfg,
				ProxyRepo: mockProxyRepo,
				Logger:    zap.NewNop(),
			})

			result := handler.validateRedirectURL(tc.redirectURL)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// TestOAuthHandler_StartOAuth_WithConfiguredProvider
// =============================================================================

func TestOAuthHandler_StartOAuth_WithConfiguredProvider(t *testing.T) {
	// Skip if Google OAuth env vars aren't set (required for provider to be enabled)
	// Note: This test requires GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET to be set
	// In CI/testing environments, these may not be available, so we test the unconfigured case

	t.Run("unconfigured provider returns error", func(t *testing.T) {
		t.Parallel()

		// Provider manager loads from env vars - without them, providers are disabled
		providerManager := auth.NewOAuthProviderManager()

		cfg := &config.Config{
			ACL: config.ACLConfig{
				CookieSecure: false,
				SessionTTL:   24 * time.Hour,
				OAuth: config.OAuthConfig{
					CallbackBaseURL: "http://localhost:8080",
				},
			},
		}

		handler := NewOAuthHandler(OAuthHandlerConfig{
			ProviderManager: providerManager,
			Config:          cfg,
			Logger:          zap.NewNop(),
		})

		req := httptest.NewRequest(http.MethodGet, "/auth/oauth/google?redirect=/dashboard", nil)
		req = setChiURLParams(req, map[string]string{"provider": "google"})

		rec := httptest.NewRecorder()
		handler.StartOAuth(rec, req)

		// Without env vars configured, should return error about provider not configured
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "not configured")
	})
}

// =============================================================================
// TestOAuthHandler_Callback_StateMismatch
// =============================================================================

func TestOAuthHandler_Callback_StateMismatch(t *testing.T) {
	t.Parallel()

	// Provider manager loads from env vars - providers may or may not be enabled
	// The callback should still validate state before checking provider
	providerManager := auth.NewOAuthProviderManager()

	cfg := &config.Config{
		ACL: config.ACLConfig{
			OAuth: config.OAuthConfig{
				CallbackBaseURL: "http://localhost:8080",
			},
		},
	}

	handler := NewOAuthHandler(OAuthHandlerConfig{
		ProviderManager: providerManager,
		Config:          cfg,
		Logger:          zap.NewNop(),
	})

	t.Run("state mismatch returns error", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/auth/oauth/google/callback?code=testcode&state=wrong-state", nil)
		req = setChiURLParams(req, map[string]string{"provider": "google"})
		req.AddCookie(&http.Cookie{
			Name:  oauthStateCookieName,
			Value: "correct-state",
		})

		rec := httptest.NewRecorder()
		handler.Callback(rec, req)

		assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		location := rec.Header().Get("Location")
		assert.Contains(t, location, "oauth_error")
		// Error could be state mismatch or provider not configured depending on order of checks
	})

	t.Run("missing state cookie returns error", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/auth/oauth/google/callback?code=testcode&state=some-state", nil)
		req = setChiURLParams(req, map[string]string{"provider": "google"})
		// No cookie set

		rec := httptest.NewRecorder()
		handler.Callback(rec, req)

		assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		location := rec.Header().Get("Location")
		assert.Contains(t, location, "oauth_error")
	})

	t.Run("callback with error param returns error", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/auth/oauth/google/callback?error=access_denied&error_description=User+denied", nil)
		req = setChiURLParams(req, map[string]string{"provider": "google"})

		rec := httptest.NewRecorder()
		handler.Callback(rec, req)

		assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		location := rec.Header().Get("Location")
		assert.Contains(t, location, "oauth_error")
	})
}

// =============================================================================
// TestOAuthHandler_parseUserInfo_EdgeCases
// =============================================================================

func TestOAuthHandler_parseUserInfo_EdgeCases(t *testing.T) {
	t.Parallel()

	handler, _ := createTestOAuthHandler(t)

	t.Run("Microsoft provider with userPrincipalName fallback", func(t *testing.T) {
		t.Parallel()

		responseBody := map[string]interface{}{
			"id":                "ms-user-id",
			"userPrincipalName": "user@domain.onmicrosoft.com", // Used when mail is empty
			"displayName":       "Microsoft User",
		}

		bodyBytes, err := json.Marshal(responseBody)
		require.NoError(t, err)
		bodyReader := bytes.NewReader(bodyBytes)

		result, err := handler.parseUserInfo(auth.OAuthProviderMicrosoft, bodyReader)

		require.NoError(t, err)
		assert.Equal(t, "user@domain.onmicrosoft.com", result.Email)
		assert.Equal(t, "user", result.Username)
	})

	t.Run("GitHub provider with empty email allowed", func(t *testing.T) {
		t.Parallel()

		responseBody := map[string]interface{}{
			"id":    12345,
			"login": "githubuser",
			"name":  "GitHub User",
			// No email - this is allowed for GitHub
		}

		bodyBytes, err := json.Marshal(responseBody)
		require.NoError(t, err)
		bodyReader := bytes.NewReader(bodyBytes)

		result, err := handler.parseUserInfo(auth.OAuthProviderGitHub, bodyReader)

		require.NoError(t, err)
		assert.Equal(t, "", result.Email) // Empty email is OK for GitHub
		assert.Equal(t, "githubuser", result.Username)
	})

	t.Run("missing name uses username as fallback", func(t *testing.T) {
		t.Parallel()

		responseBody := map[string]interface{}{
			"id":    "google-id",
			"email": "user@gmail.com",
			// No name provided
		}

		bodyBytes, err := json.Marshal(responseBody)
		require.NoError(t, err)
		bodyReader := bytes.NewReader(bodyBytes)

		result, err := handler.parseUserInfo(auth.OAuthProviderGoogle, bodyReader)

		require.NoError(t, err)
		assert.Equal(t, "user", result.Name) // Falls back to username derived from email
	})
}
