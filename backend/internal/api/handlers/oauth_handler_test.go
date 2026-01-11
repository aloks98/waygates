package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
func (m *oauthMockACLService) CreateGroup(group *models.ACLGroup, createdBy int) error {
	return nil
}
func (m *oauthMockACLService) GetGroup(id int) (*models.ACLGroup, error) { return nil, nil }
func (m *oauthMockACLService) GetGroupByName(name string) (*models.ACLGroup, error) {
	return nil, nil
}
func (m *oauthMockACLService) ListGroups(params service.ListACLGroupsRequest) (*models.ACLGroupListResponse, error) {
	return nil, nil
}
func (m *oauthMockACLService) UpdateGroup(id int, updates *models.ACLGroup) error {
	return nil
}
func (m *oauthMockACLService) DeleteGroup(id int) error { return nil }
func (m *oauthMockACLService) DeleteGroupWithSync(id int, syncFn service.SyncCallback) error {
	return nil
}

// IP Rules
func (m *oauthMockACLService) AddIPRule(groupID int, rule *models.ACLIPRule) error {
	return nil
}
func (m *oauthMockACLService) UpdateIPRule(id int, rule *models.ACLIPRule) error {
	return nil
}
func (m *oauthMockACLService) DeleteIPRule(id int) error { return nil }

// Basic Auth
func (m *oauthMockACLService) AddBasicAuthUser(groupID int, username, password string) error {
	return nil
}
func (m *oauthMockACLService) UpdateBasicAuthPassword(id int, password string) error {
	return nil
}
func (m *oauthMockACLService) DeleteBasicAuthUser(id int) error { return nil }

// External Providers
func (m *oauthMockACLService) AddExternalProvider(groupID int, provider *models.ACLExternalProvider) error {
	return nil
}
func (m *oauthMockACLService) UpdateExternalProvider(id int, provider *models.ACLExternalProvider) error {
	return nil
}
func (m *oauthMockACLService) DeleteExternalProvider(id int) error { return nil }

// Waygates Auth Config
func (m *oauthMockACLService) GetWaygatesAuth(groupID int) (*models.ACLWaygatesAuth, error) {
	return nil, nil
}
func (m *oauthMockACLService) ConfigureWaygatesAuth(groupID int, config *models.ACLWaygatesAuth) error {
	return nil
}

// Proxy Assignment
func (m *oauthMockACLService) AssignToProxy(proxyID, groupID int, pathPattern string, priority int) error {
	return nil
}
func (m *oauthMockACLService) UpdateProxyAssignment(id int, pathPattern string, priority int, enabled bool) error {
	return nil
}
func (m *oauthMockACLService) RemoveFromProxy(proxyID, groupID int) error { return nil }
func (m *oauthMockACLService) GetProxyACL(proxyID int) ([]models.ProxyACLAssignment, error) {
	return nil, nil
}
func (m *oauthMockACLService) GetGroupUsage(groupID int) ([]models.ProxyACLAssignment, error) {
	return nil, nil
}

// Branding
func (m *oauthMockACLService) GetBranding() (*models.ACLBranding, error) { return nil, nil }
func (m *oauthMockACLService) UpdateBranding(branding *models.ACLBranding) error {
	return nil
}

// OAuth Provider Restrictions
func (m *oauthMockACLService) GetOAuthProviderRestrictions(groupID int) ([]models.ACLOAuthProviderRestriction, error) {
	return nil, nil
}
func (m *oauthMockACLService) SetOAuthProviderRestriction(groupID int, provider string, emails, domains []string, enabled bool) error {
	return nil
}
func (m *oauthMockACLService) DeleteOAuthProviderRestriction(groupID int, provider string) error {
	return nil
}

// Access Verification
func (m *oauthMockACLService) VerifyAccess(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
	return nil, nil
}

// Auth Options
func (m *oauthMockACLService) GetAuthOptionsForProxy(hostname string) (*service.AuthOptionsResponse, error) {
	return nil, nil
}

// Session Management
func (m *oauthMockACLService) CreateSession(userID int, proxyID *int, ip, userAgent string, ttl int) (*models.ACLSession, error) {
	return nil, nil
}
func (m *oauthMockACLService) CreateOAuthSession(email, provider string, proxyID *int, ip, userAgent string, ttl int) (*models.ACLSession, error) {
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
func (m *oauthMockACLService) ValidateSession(token string) (*models.ACLSession, error) {
	return nil, nil
}
func (m *oauthMockACLService) RevokeSession(token string) error { return nil }
func (m *oauthMockACLService) RevokeUserSessions(userID int) error {
	return nil
}
func (m *oauthMockACLService) CleanupExpiredSessions() (int64, error) { return 0, nil }

var _ service.ACLServiceInterface = (*oauthMockACLService)(nil)

// oauthMockUserRepository is a mock implementation of UserRepositoryInterface
type oauthMockUserRepository struct {
	GetByUsernameOrEmailFunc func(identifier string) (*models.User, error)
	CreateFunc               func(user *models.User) error
	GetByIDFunc              func(id int) (*models.User, error)
	GetByEmailFunc           func(email string) (*models.User, error)
	CountFunc                func() (int64, error)
	DeleteFunc               func(id int) error
	UpdatePasswordFunc       func(id int, passwordHash string) error
}

func (m *oauthMockUserRepository) GetByUsernameOrEmail(identifier string) (*models.User, error) {
	if m.GetByUsernameOrEmailFunc != nil {
		return m.GetByUsernameOrEmailFunc(identifier)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *oauthMockUserRepository) Create(user *models.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(user)
	}
	return nil
}

func (m *oauthMockUserRepository) GetByID(id int) (*models.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *oauthMockUserRepository) GetByEmail(email string) (*models.User, error) {
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(email)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *oauthMockUserRepository) Count() (int64, error) {
	if m.CountFunc != nil {
		return m.CountFunc()
	}
	return 0, nil
}

func (m *oauthMockUserRepository) Delete(id int) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *oauthMockUserRepository) UpdatePassword(id int, passwordHash string) error {
	if m.UpdatePasswordFunc != nil {
		return m.UpdatePasswordFunc(id, passwordHash)
	}
	return nil
}

// =============================================================================
// Test Helpers
// =============================================================================

func createTestOAuthHandler(t *testing.T) (*OAuthHandler, *oauthMockACLService, *oauthMockUserRepository) {
	t.Helper()

	mockACLService := &oauthMockACLService{}
	mockUserRepo := &oauthMockUserRepository{}
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
		UserRepo:        mockUserRepo,
		Config:          cfg,
		Logger:          logger,
	})

	return handler, mockACLService, mockUserRepo
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
		handler, _, _ := createTestOAuthHandler(t)
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

	handler, _, _ := createTestOAuthHandler(t)

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

			handler, _, _ := createTestOAuthHandler(t)

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

			handler, _, _ := createTestOAuthHandler(t)

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

	handler, _, _ := createTestOAuthHandler(t)

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

	handler, _, _ := createTestOAuthHandler(t)

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

	handler, _, _ := createTestOAuthHandler(t)

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

	handler, _, _ := createTestOAuthHandler(t)

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

	handler, _, _ := createTestOAuthHandler(t)

	bodyReader := io.NopCloser(bytes.NewReader([]byte("not valid json")))
	result, err := handler.parseUserInfo(auth.OAuthProviderGoogle, bodyReader)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "decoding response")
}

// =============================================================================
// TestOAuthHandler_findOrCreateUser
// =============================================================================

func TestOAuthHandler_findOrCreateUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		providerID    auth.OAuthProviderID
		userInfo      *OAuthUserInfo
		setupMocks    func(*oauthMockUserRepository)
		expectError   bool
		errorContains string
		checkUser     func(*testing.T, *models.User)
	}{
		{
			name:       "finds existing user by email",
			providerID: auth.OAuthProviderGoogle,
			userInfo: &OAuthUserInfo{
				Email:    "existing@example.com",
				Name:     "Existing User",
				Username: "existing",
			},
			setupMocks: func(repo *oauthMockUserRepository) {
				repo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					return &models.User{
						ID:       1,
						Email:    "existing@example.com",
						Username: "existing",
						Name:     "Existing User",
					}, nil
				}
			},
			checkUser: func(t *testing.T, user *models.User) {
				assert.Equal(t, 1, user.ID)
				assert.Equal(t, "existing@example.com", user.Email)
			},
		},
		{
			name:       "creates new user when not found",
			providerID: auth.OAuthProviderGitHub,
			userInfo: &OAuthUserInfo{
				Email:    "newuser@example.com",
				Name:     "New User",
				Username: "newuser",
			},
			setupMocks: func(repo *oauthMockUserRepository) {
				repo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					return nil, gorm.ErrRecordNotFound
				}
				repo.CreateFunc = func(user *models.User) error {
					user.ID = 2
					return nil
				}
			},
			checkUser: func(t *testing.T, user *models.User) {
				assert.Equal(t, 2, user.ID)
				assert.Equal(t, "newuser@example.com", user.Email)
				assert.Equal(t, "New User", user.Name)
				assert.Equal(t, "newuser", user.Username)
			},
		},
		{
			name:       "generates unique username when already taken",
			providerID: auth.OAuthProviderGoogle,
			userInfo: &OAuthUserInfo{
				Email:    "unique@example.com",
				Name:     "Unique User",
				Username: "taken",
			},
			setupMocks: func(repo *oauthMockUserRepository) {
				callCount := 0
				repo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					callCount++
					// First call for email lookup - not found
					if identifier == "unique@example.com" && callCount == 1 {
						return nil, gorm.ErrRecordNotFound
					}
					// Second call for username "taken" - exists
					if identifier == "taken" {
						return &models.User{ID: 99, Username: "taken"}, nil
					}
					// Any other username check returns not found
					return nil, gorm.ErrRecordNotFound
				}
				repo.CreateFunc = func(user *models.User) error {
					user.ID = 3
					return nil
				}
			},
			checkUser: func(t *testing.T, user *models.User) {
				assert.Equal(t, 3, user.ID)
				assert.NotEqual(t, "taken", user.Username, "Username should be different from taken")
				assert.True(t, strings.HasPrefix(user.Username, "taken"), "Username should start with 'taken'")
			},
		},
		{
			name:       "database error on lookup",
			providerID: auth.OAuthProviderGoogle,
			userInfo: &OAuthUserInfo{
				Email:    "error@example.com",
				Name:     "Error User",
				Username: "erroruser",
			},
			setupMocks: func(repo *oauthMockUserRepository) {
				repo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					return nil, gorm.ErrInvalidDB
				}
			},
			expectError:   true,
			errorContains: "looking up user",
		},
		{
			name:       "database error on create",
			providerID: auth.OAuthProviderGoogle,
			userInfo: &OAuthUserInfo{
				Email:    "createerror@example.com",
				Name:     "Create Error",
				Username: "createerror",
			},
			setupMocks: func(repo *oauthMockUserRepository) {
				repo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					return nil, gorm.ErrRecordNotFound
				}
				repo.CreateFunc = func(user *models.User) error {
					return gorm.ErrInvalidDB
				}
			},
			expectError:   true,
			errorContains: "creating user",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler, _, mockUserRepo := createTestOAuthHandler(t)

			if tc.setupMocks != nil {
				tc.setupMocks(mockUserRepo)
			}

			user, err := handler.findOrCreateUser(context.Background(), tc.providerID, tc.userInfo)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				if tc.checkUser != nil {
					tc.checkUser(t, user)
				}
			}
		})
	}
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

			handler, _, _ := createTestOAuthHandler(t)

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
