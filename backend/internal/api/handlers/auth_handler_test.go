package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aloks98/goauth/middleware"
	"github.com/aloks98/goauth/store"
	"github.com/aloks98/goauth/token"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/utils"
	"github.com/aloks98/waygates/backend/internal/validation"
)

// MockUserRepository is a mock implementation of UserRepositoryInterface
type MockUserRepository struct {
	CreateFunc              func(user *models.User) error
	GetByEmailFunc          func(email string) (*models.User, error)
	GetByUsernameOrEmailFunc func(identifier string) (*models.User, error)
	GetByIDFunc             func(id int) (*models.User, error)
	CountFunc               func() (int64, error)
	DeleteFunc              func(id int) error
}

func (m *MockUserRepository) Create(user *models.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(user)
	}
	return nil
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(email)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockUserRepository) GetByUsernameOrEmail(identifier string) (*models.User, error) {
	if m.GetByUsernameOrEmailFunc != nil {
		return m.GetByUsernameOrEmailFunc(identifier)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockUserRepository) GetByID(id int) (*models.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockUserRepository) Count() (int64, error) {
	if m.CountFunc != nil {
		return m.CountFunc()
	}
	return 0, nil
}

func (m *MockUserRepository) Delete(id int) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

// MockAuthProvider is a mock implementation of AuthProvider
type MockAuthProvider struct {
	AssignRoleFunc         func(ctx context.Context, userID, role string) error
	GenerateTokenPairFunc  func(ctx context.Context, userID string, metadata map[string]any) (*token.Pair, error)
	RefreshTokensFunc      func(ctx context.Context, refreshToken string) (*token.Pair, error)
	RevokeAccessTokenFunc  func(ctx context.Context, accessToken string) error
	RevokeRefreshTokenFunc func(ctx context.Context, refreshToken string) error
	GetUserPermissionsFunc func(ctx context.Context, userID string) (*store.UserPermissions, error)
}

func (m *MockAuthProvider) AssignRole(ctx context.Context, userID, role string) error {
	if m.AssignRoleFunc != nil {
		return m.AssignRoleFunc(ctx, userID, role)
	}
	return nil
}

func (m *MockAuthProvider) GenerateTokenPair(ctx context.Context, userID string, metadata map[string]any) (*token.Pair, error) {
	if m.GenerateTokenPairFunc != nil {
		return m.GenerateTokenPairFunc(ctx, userID, metadata)
	}
	return &token.Pair{
		AccessToken:  "mock-access-token",
		RefreshToken: "mock-refresh-token",
	}, nil
}

func (m *MockAuthProvider) RefreshTokens(ctx context.Context, refreshToken string) (*token.Pair, error) {
	if m.RefreshTokensFunc != nil {
		return m.RefreshTokensFunc(ctx, refreshToken)
	}
	return &token.Pair{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
	}, nil
}

func (m *MockAuthProvider) RevokeAccessToken(ctx context.Context, accessToken string) error {
	if m.RevokeAccessTokenFunc != nil {
		return m.RevokeAccessTokenFunc(ctx, accessToken)
	}
	return nil
}

func (m *MockAuthProvider) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	if m.RevokeRefreshTokenFunc != nil {
		return m.RevokeRefreshTokenFunc(ctx, refreshToken)
	}
	return nil
}

func (m *MockAuthProvider) GetUserPermissions(ctx context.Context, userID string) (*store.UserPermissions, error) {
	if m.GetUserPermissionsFunc != nil {
		return m.GetUserPermissionsFunc(ctx, userID)
	}
	return &store.UserPermissions{
		RoleLabel:   "admin",
		Permissions: []string{"read", "write"},
	}, nil
}

// Helper to create a test user with hashed password
func createTestUser(t *testing.T, password string) *models.User {
	user := &models.User{
		ID:       1,
		Name:     "Test User",
		Username: "testuser",
		Email:    "test@example.com",
	}
	if err := user.SetPassword(password, 4); err != nil { // Use low cost for tests
		t.Fatalf("Failed to set password: %v", err)
	}
	return user
}

// TestNewAuthHandler tests handler creation
func TestNewAuthHandler(t *testing.T) {
	userRepo := &MockUserRepository{}
	authProvider := &MockAuthProvider{}

	handler := NewAuthHandler(authProvider, userRepo, 10)

	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
	if handler.bcryptCost != 10 {
		t.Errorf("Expected bcryptCost 10, got %d", handler.bcryptCost)
	}
}

// TestRegister tests user registration
func TestRegister(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockUserRepository, *MockAuthProvider)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "invalid json",
			requestBody:    `{invalid}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - empty email",
			requestBody: validation.RegisterRequest{
				Name:     "Test",
				Username: "testuser",
				Email:    "",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - weak password",
			requestBody: validation.RegisterRequest{
				Name:     "Test",
				Username: "testuser",
				Email:    "test@example.com",
				Password: "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "email already exists",
			requestBody: validation.RegisterRequest{
				Name:     "Test User",
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					if identifier == "test@example.com" {
						return &models.User{ID: 1, Email: "test@example.com"}, nil
					}
					return nil, gorm.ErrRecordNotFound
				}
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "username already exists",
			requestBody: validation.RegisterRequest{
				Name:     "Test User",
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				callCount := 0
				userRepo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					callCount++
					if callCount == 1 {
						return nil, gorm.ErrRecordNotFound // Email not found
					}
					return &models.User{ID: 1, Username: "testuser"}, nil // Username exists
				}
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "database error checking email",
			requestBody: validation.RegisterRequest{
				Name:     "Test User",
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "create user fails",
			requestBody: validation.RegisterRequest{
				Name:     "Test User",
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					return nil, gorm.ErrRecordNotFound
				}
				userRepo.CreateFunc = func(user *models.User) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "assign role fails - rollback succeeds",
			requestBody: validation.RegisterRequest{
				Name:     "Test User",
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					return nil, gorm.ErrRecordNotFound
				}
				userRepo.CreateFunc = func(user *models.User) error {
					user.ID = 1
					return nil
				}
				userRepo.CountFunc = func() (int64, error) {
					return 1, nil
				}
				authProvider.AssignRoleFunc = func(ctx context.Context, userID, role string) error {
					return errors.New("rbac error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "success - first user gets admin role",
			requestBody: validation.RegisterRequest{
				Name:     "Test User",
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					return nil, gorm.ErrRecordNotFound
				}
				userRepo.CreateFunc = func(user *models.User) error {
					user.ID = 1
					return nil
				}
				userRepo.CountFunc = func() (int64, error) {
					return 1, nil // First user
				}
				authProvider.AssignRoleFunc = func(ctx context.Context, userID, role string) error {
					if role != "admin" {
						return errors.New("expected admin role for first user")
					}
					return nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "success - second user gets operator role",
			requestBody: validation.RegisterRequest{
				Name:     "Test User",
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					return nil, gorm.ErrRecordNotFound
				}
				userRepo.CreateFunc = func(user *models.User) error {
					user.ID = 2
					return nil
				}
				userRepo.CountFunc = func() (int64, error) {
					return 2, nil // Second user
				}
			},
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			authProvider := &MockAuthProvider{}

			if tc.setupMocks != nil {
				tc.setupMocks(userRepo, authProvider)
			}

			handler := NewAuthHandler(authProvider, userRepo, 4) // Low cost for tests

			var body []byte
			switch v := tc.requestBody.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.Register(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if tc.checkResponse != nil {
				tc.checkResponse(t, rec)
			}
		})
	}
}

// TestLogin tests user login
func TestLogin(t *testing.T) {
	testUser := createTestUser(t, "password123")

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockUserRepository, *MockAuthProvider)
		expectedStatus int
	}{
		{
			name:           "invalid json",
			requestBody:    `{invalid}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing identifier",
			requestBody: LoginRequest{
				Identifier: "",
				Password:   "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			requestBody: LoginRequest{
				Identifier: "testuser",
				Password:   "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "user not found",
			requestBody: LoginRequest{
				Identifier: "nonexistent",
				Password:   "password123",
			},
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "database error",
			requestBody: LoginRequest{
				Identifier: "testuser",
				Password:   "password123",
			},
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "wrong password",
			requestBody: LoginRequest{
				Identifier: "testuser",
				Password:   "wrongpassword",
			},
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					return testUser, nil
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "token generation fails",
			requestBody: LoginRequest{
				Identifier: "testuser",
				Password:   "password123",
			},
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					return testUser, nil
				}
				authProvider.GenerateTokenPairFunc = func(ctx context.Context, userID string, metadata map[string]any) (*token.Pair, error) {
					return nil, errors.New("token error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "success",
			requestBody: LoginRequest{
				Identifier: "testuser",
				Password:   "password123",
			},
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(identifier string) (*models.User, error) {
					return testUser, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			authProvider := &MockAuthProvider{}

			if tc.setupMocks != nil {
				tc.setupMocks(userRepo, authProvider)
			}

			handler := NewAuthHandler(authProvider, userRepo, 4)

			var body []byte
			switch v := tc.requestBody.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.Login(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestRefreshToken tests token refresh
func TestRefreshToken(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockAuthProvider)
		expectedStatus int
	}{
		{
			name:           "invalid json",
			requestBody:    `{invalid}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing refresh token",
			requestBody: RefreshTokenRequest{
				RefreshToken: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid refresh token",
			requestBody: RefreshTokenRequest{
				RefreshToken: "invalid-token",
			},
			setupMocks: func(authProvider *MockAuthProvider) {
				authProvider.RefreshTokensFunc = func(ctx context.Context, refreshToken string) (*token.Pair, error) {
					return nil, errors.New("invalid token")
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "success",
			requestBody: RefreshTokenRequest{
				RefreshToken: "valid-refresh-token",
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			authProvider := &MockAuthProvider{}

			if tc.setupMocks != nil {
				tc.setupMocks(authProvider)
			}

			handler := NewAuthHandler(authProvider, &MockUserRepository{}, 4)

			var body []byte
			switch v := tc.requestBody.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.RefreshToken(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestLogout tests user logout
func TestLogout(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name:           "no token",
			authHeader:     "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "bearer token only - edge case",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusOK, // "Bearer " is 7 chars, so condition len > 7 fails and token = "Bearer " (not empty)
		},
		{
			name:           "valid token without refresh",
			authHeader:     "Bearer some-access-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid token with refresh token",
			authHeader:     "Bearer some-access-token",
			requestBody:    LogoutRequest{RefreshToken: "some-refresh-token"},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewAuthHandler(&MockAuthProvider{}, &MockUserRepository{}, 4)

			var body []byte
			if tc.requestBody != nil {
				body, _ = json.Marshal(tc.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", bytes.NewBuffer(body))
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()

			handler.Logout(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestGetMe tests getting current user info
func TestGetMe(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*http.Request) *http.Request
		setupMocks     func(*MockUserRepository, *MockAuthProvider)
		expectedStatus int
	}{
		{
			name: "not authenticated - no user ID in context",
			setupContext: func(r *http.Request) *http.Request {
				return r // No context modification
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "user not found",
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "1")
				return r.WithContext(ctx)
			},
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByIDFunc = func(id int) (*models.User, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "success with permissions",
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "1")
				return r.WithContext(ctx)
			},
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByIDFunc = func(id int) (*models.User, error) {
					return &models.User{
						ID:       1,
						Name:     "Test User",
						Username: "testuser",
						Email:    "test@example.com",
					}, nil
				}
				authProvider.GetUserPermissionsFunc = func(ctx context.Context, userID string) (*store.UserPermissions, error) {
					return &store.UserPermissions{
						RoleLabel:   "admin",
						Permissions: []string{"read", "write"},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "success without permissions (error getting permissions)",
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "1")
				return r.WithContext(ctx)
			},
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByIDFunc = func(id int) (*models.User, error) {
					return &models.User{
						ID:       1,
						Name:     "Test User",
						Username: "testuser",
						Email:    "test@example.com",
					}, nil
				}
				authProvider.GetUserPermissionsFunc = func(ctx context.Context, userID string) (*store.UserPermissions, error) {
					return nil, errors.New("permissions error")
				}
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			authProvider := &MockAuthProvider{}

			if tc.setupMocks != nil {
				tc.setupMocks(userRepo, authProvider)
			}

			handler := NewAuthHandler(authProvider, userRepo, 4)

			req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
			if tc.setupContext != nil {
				req = tc.setupContext(req)
			}
			rec := httptest.NewRecorder()

			handler.GetMe(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

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
