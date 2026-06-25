package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/aloks98/goauth/middleware"
	"github.com/aloks98/goauth/store"
	"github.com/aloks98/goauth/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/service/mocks"
	"github.com/aloks98/waygates/backend/internal/utils"
	"github.com/aloks98/waygates/backend/internal/validation"
)

// MockSettingsRepository is a mock implementation of SettingsRepositoryInterface
type MockSettingsRepository struct {
	GetValueFunc func(key, defaultValue string) string
}

func (m *MockSettingsRepository) GetValue(key, defaultValue string) string {
	if m.GetValueFunc != nil {
		return m.GetValueFunc(key, defaultValue)
	}
	return defaultValue
}

func (m *MockSettingsRepository) Get(_ string) (*models.Setting, error) {
	return nil, nil
}

func (m *MockSettingsRepository) Set(_, _ string) error {
	return nil
}

func (m *MockSettingsRepository) GetAll() (map[string]string, error) {
	return map[string]string{}, nil
}

func (m *MockSettingsRepository) Delete(_ string) error {
	return nil
}

func (m *MockSettingsRepository) GetNotFoundSettings() (*models.NotFoundSettings, error) {
	return &models.NotFoundSettings{}, nil
}

func (m *MockSettingsRepository) SetNotFoundSettings(_ *models.NotFoundSettings) error {
	return nil
}

func (m *MockSettingsRepository) GetMetricsPublishSettings() (*models.MetricsPublishSettings, error) {
	return &models.MetricsPublishSettings{}, nil
}

func (m *MockSettingsRepository) SetMetricsPublishSettings(_ *models.MetricsPublishSettings) error {
	return nil
}

// defaultClosedSettings returns a MockSettingsRepository with registration closed (the default).
func defaultClosedSettings() *MockSettingsRepository {
	return &MockSettingsRepository{}
}

// openSettings returns a MockSettingsRepository with registration open.
func openSettings() *MockSettingsRepository {
	return &MockSettingsRepository{
		GetValueFunc: func(_, _ string) string { return "true" },
	}
}

// MockUserRepository is a mock implementation of UserRepositoryInterface
type MockUserRepository struct {
	CreateFunc               func(user *models.User) error
	GetByEmailFunc           func(email string) (*models.User, error)
	GetByUsernameOrEmailFunc func(identifier string) (*models.User, error)
	GetByIDFunc              func(id int) (*models.User, error)
	CountFunc                func() (int64, error)
	DeleteFunc               func(id int) error
	UpdatePasswordFunc       func(id int, passwordHash string) error
	ListFunc                 func() ([]models.User, error)
	UpdateFunc               func(user *models.User) error
	UpdateLastLoginFunc      func(id int, t time.Time) error
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

func (m *MockUserRepository) UpdatePassword(id int, passwordHash string) error {
	if m.UpdatePasswordFunc != nil {
		return m.UpdatePasswordFunc(id, passwordHash)
	}
	return nil
}

func (m *MockUserRepository) List() ([]models.User, error) {
	if m.ListFunc != nil {
		return m.ListFunc()
	}
	return []models.User{}, nil
}

func (m *MockUserRepository) Update(user *models.User) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(user)
	}
	return nil
}

func (m *MockUserRepository) UpdateLastLogin(id int, t time.Time) error {
	if m.UpdateLastLoginFunc != nil {
		return m.UpdateLastLoginFunc(id, t)
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

// Helper to create a test user with hashed password.
// Active is set to true so existing login tests are not blocked by the
// inactive-account check introduced in Task 4.
func createTestUser(t *testing.T, password string) *models.User {
	t.Helper()
	user := &models.User{
		ID:       1,
		Name:     "Test User",
		Username: "testuser",
		Email:    "test@example.com",
		Active:   true,
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

	handler := NewAuthHandler(authProvider, userRepo, defaultClosedSettings(), nil, 10, nil)

	if handler == nil {
		t.Fatal("Expected non-nil handler")
	} else if handler.bcryptCost != 10 {
		t.Errorf("Expected bcryptCost 10, got %d", handler.bcryptCost)
	}
}

// TestRegister tests user registration
func TestRegister(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		settingsRepo   *MockSettingsRepository
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
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
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
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
				callCount := 0
				userRepo.GetByUsernameOrEmailFunc = func(_ string) (*models.User, error) {
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
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(_ string) (*models.User, error) {
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
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(_ string) (*models.User, error) {
					return nil, gorm.ErrRecordNotFound
				}
				userRepo.CreateFunc = func(_ *models.User) error {
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
			settingsRepo: defaultClosedSettings(),
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(_ string) (*models.User, error) {
					return nil, gorm.ErrRecordNotFound
				}
				userRepo.CreateFunc = func(user *models.User) error {
					user.ID = 1
					return nil
				}
				userRepo.CountFunc = func() (int64, error) {
					return 0, nil // Empty table: bootstrap path reaches AssignRole
				}
				authProvider.AssignRoleFunc = func(_ context.Context, _, _ string) error {
					return errors.New("rbac error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "success - first user gets admin role (bootstrap, even when closed)",
			requestBody: validation.RegisterRequest{
				Name:     "Test User",
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			settingsRepo: defaultClosedSettings(), // registration closed
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(_ string) (*models.User, error) {
					return nil, gorm.ErrRecordNotFound
				}
				userRepo.CreateFunc = func(user *models.User) error {
					user.ID = 1
					return nil
				}
				userRepo.CountFunc = func() (int64, error) {
					return 0, nil // Empty table — bootstrap path
				}
				authProvider.AssignRoleFunc = func(_ context.Context, _, role string) error {
					if role != "admin" {
						return errors.New("expected admin role for first user")
					}
					return nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "success - second user gets viewer role when open",
			requestBody: validation.RegisterRequest{
				Name:     "Test User",
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			settingsRepo: openSettings(),
			setupMocks: func(userRepo *MockUserRepository, authProvider *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(_ string) (*models.User, error) {
					return nil, gorm.ErrRecordNotFound
				}
				userRepo.CreateFunc = func(user *models.User) error {
					user.ID = 2
					return nil
				}
				userRepo.CountFunc = func() (int64, error) {
					return 1, nil // Existing user, table non-empty
				}
				authProvider.AssignRoleFunc = func(_ context.Context, _, role string) error {
					if role != "viewer" {
						return errors.New("expected viewer role when open")
					}
					return nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "registration closed - non-empty table returns 403",
			requestBody: validation.RegisterRequest{
				Name:     "Test User",
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			settingsRepo: defaultClosedSettings(),
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(_ string) (*models.User, error) {
					return nil, gorm.ErrRecordNotFound
				}
				userRepo.CountFunc = func() (int64, error) {
					return 3, nil // Non-empty table, registration closed
				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			authProvider := &MockAuthProvider{}

			if tc.setupMocks != nil {
				tc.setupMocks(userRepo, authProvider)
			}

			settingsRepo := tc.settingsRepo
			if settingsRepo == nil {
				settingsRepo = defaultClosedSettings()
			}

			handler := NewAuthHandler(authProvider, userRepo, settingsRepo, nil, 4, nil) // Low cost for tests

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

// TestCreateUserAndAssignRole_ConcurrentExactlyOneAdmin ensures concurrent
// registrations produce exactly one admin. The gate (count check + role
// decision) is serialized by firstUserRegistrationMu, so only the goroutine
// that sees count==0 gets the admin role; all others are rejected when
// registration is closed (the default), or get viewer when it is open.
// We drive concurrent requests through Register to exercise the full path.
func TestCreateUserAndAssignRole_ConcurrentExactlyOneAdmin(t *testing.T) {
	const n = 50

	var mu sync.Mutex
	created := 0
	var roles []string

	userRepo := &MockUserRepository{
		GetByUsernameOrEmailFunc: func(_ string) (*models.User, error) {
			return nil, gorm.ErrRecordNotFound
		},
		CreateFunc: func(u *models.User) error {
			mu.Lock()
			created++
			u.ID = created
			mu.Unlock()
			runtime.Gosched() // widen the create->count window to expose the race
			return nil
		},
		CountFunc: func() (int64, error) {
			mu.Lock()
			defer mu.Unlock()
			return int64(created), nil
		},
	}
	authProvider := &MockAuthProvider{
		AssignRoleFunc: func(_ context.Context, _, role string) error {
			mu.Lock()
			roles = append(roles, role)
			mu.Unlock()
			return nil
		},
	}

	// registration closed: only the bootstrap (count==0) path assigns a role
	handler := NewAuthHandler(authProvider, userRepo, defaultClosedSettings(), nil, 4, nil)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body, _ := json.Marshal(validation.RegisterRequest{
				Name:     "User",
				Username: "user",
				Email:    "user@example.com",
				Password: "password123",
			})
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			handler.Register(rec, req)
		}()
	}
	wg.Wait()

	adminCount := 0
	for _, r := range roles {
		if r == "admin" {
			adminCount++
		}
	}
	// Exactly one request must have succeeded with the admin bootstrap role;
	// all others were blocked by the closed-registration gate.
	assert.Equal(t, 1, adminCount, "exactly one registrant should become admin")
	assert.Len(t, roles, 1, "only the bootstrap registration should assign a role when closed")
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
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(_ string) (*models.User, error) {
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
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(_ string) (*models.User, error) {
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
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(_ string) (*models.User, error) {
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
				userRepo.GetByUsernameOrEmailFunc = func(_ string) (*models.User, error) {
					return testUser, nil
				}
				authProvider.GenerateTokenPairFunc = func(_ context.Context, _ string, _ map[string]any) (*token.Pair, error) {
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
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
				userRepo.GetByUsernameOrEmailFunc = func(_ string) (*models.User, error) {
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

			handler := NewAuthHandler(authProvider, userRepo, defaultClosedSettings(), nil, 4, nil)

			handler.Login(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestLogin_InactiveUser verifies that a disabled account is rejected even when
// the password is correct, and that the response is 403 Forbidden.
func TestLogin_InactiveUser(t *testing.T) {
	t.Parallel()

	testUser := createTestUser(t, "password123")
	testUser.Active = false // account is disabled

	logLoginFailedCalled := false
	mockAudit := &mocks.MockAuditService{
		LogLoginFailedFunc: func(_ context.Context, _ string, _, _, reason string) error {
			logLoginFailedCalled = true
			assert.Equal(t, "disabled", reason)
			return nil
		},
	}

	userRepo := &MockUserRepository{
		GetByUsernameOrEmailFunc: func(_ string) (*models.User, error) {
			return testUser, nil
		},
	}

	handler := NewAuthHandler(&MockAuthProvider{}, userRepo, defaultClosedSettings(), mockAudit, 4, nil)

	body, _ := json.Marshal(LoginRequest{Identifier: "testuser", Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code, "disabled account must be rejected with 403")
	assert.True(t, logLoginFailedCalled, "LogLoginFailed must be called with reason 'disabled'")

	var resp utils.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "account is disabled", resp.Error.Message)
}

// TestLogin_UpdatesLastLoginAndReturnsMustChangePassword verifies that a
// successful login records the last-login timestamp and echoes
// must_change_password from the user record.
func TestLogin_UpdatesLastLoginAndReturnsMustChangePassword(t *testing.T) {
	t.Parallel()

	testUser := createTestUser(t, "password123")
	testUser.Active = true
	testUser.MustChangePassword = true

	updateLastLoginCalled := false
	var updateLastLoginID int

	userRepo := &MockUserRepository{
		GetByUsernameOrEmailFunc: func(_ string) (*models.User, error) {
			return testUser, nil
		},
		UpdateLastLoginFunc: func(id int, _ time.Time) error {
			updateLastLoginCalled = true
			updateLastLoginID = id
			return nil
		},
	}

	handler := NewAuthHandler(&MockAuthProvider{}, userRepo, defaultClosedSettings(), nil, 4, nil)

	body, _ := json.Marshal(LoginRequest{Identifier: "testuser", Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, updateLastLoginCalled, "UpdateLastLogin must be called on successful login")
	assert.Equal(t, testUser.ID, updateLastLoginID, "UpdateLastLogin must be called with the correct user ID")

	var resp utils.SuccessResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)

	// Unmarshal data as a map to inspect must_change_password
	dataBytes, err := json.Marshal(resp.Data)
	require.NoError(t, err)
	var loginResp map[string]any
	require.NoError(t, json.Unmarshal(dataBytes, &loginResp))
	mustChange, ok := loginResp["must_change_password"].(bool)
	require.True(t, ok, "must_change_password must be present in login response")
	assert.True(t, mustChange, "must_change_password should match the user's flag (true)")
}

// TestChangePassword_ClearsMustChangePasswordFlag verifies that after a
// successful password change the Update repo method is called with
// MustChangePassword set to false.
func TestChangePassword_ClearsMustChangePasswordFlag(t *testing.T) {
	t.Parallel()

	testUser := &models.User{
		ID:                 1,
		Name:               "Test User",
		Username:           "testuser",
		Email:              "test@example.com",
		MustChangePassword: true,
	}
	if err := testUser.SetPassword("oldpassword123", 4); err != nil {
		t.Fatalf("Failed to set password: %v", err)
	}

	var updateCalledWith *models.User

	userRepo := &MockUserRepository{
		GetByIDFunc: func(_ int) (*models.User, error) {
			return testUser, nil
		},
		UpdatePasswordFunc: func(_ int, _ string) error {
			return nil
		},
		UpdateFunc: func(user *models.User) error {
			updateCalledWith = user
			return nil
		},
	}

	handler := NewAuthHandler(&MockAuthProvider{}, userRepo, defaultClosedSettings(), nil, 4, nil)

	body, _ := json.Marshal(ChangePasswordRequest{
		CurrentPassword: "oldpassword123",
		NewPassword:     "newpassword123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ChangePassword(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, updateCalledWith, "Update must be called to clear the must_change_password flag")
	assert.False(t, updateCalledWith.MustChangePassword, "MustChangePassword must be false after a successful password change")
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
				authProvider.RefreshTokensFunc = func(_ context.Context, _ string) (*token.Pair, error) {
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

			handler := NewAuthHandler(authProvider, &MockUserRepository{}, defaultClosedSettings(), nil, 4, nil)

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
			handler := NewAuthHandler(&MockAuthProvider{}, &MockUserRepository{}, defaultClosedSettings(), nil, 4, nil)

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
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
				userRepo.GetByIDFunc = func(_ int) (*models.User, error) {
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
				userRepo.GetByIDFunc = func(_ int) (*models.User, error) {
					return &models.User{
						ID:       1,
						Name:     "Test User",
						Username: "testuser",
						Email:    "test@example.com",
					}, nil
				}
				authProvider.GetUserPermissionsFunc = func(_ context.Context, _ string) (*store.UserPermissions, error) {
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
				userRepo.GetByIDFunc = func(_ int) (*models.User, error) {
					return &models.User{
						ID:       1,
						Name:     "Test User",
						Username: "testuser",
						Email:    "test@example.com",
					}, nil
				}
				authProvider.GetUserPermissionsFunc = func(_ context.Context, _ string) (*store.UserPermissions, error) {
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

			handler := NewAuthHandler(authProvider, userRepo, defaultClosedSettings(), nil, 4, nil)

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

// TestChangePassword tests password change functionality
func TestChangePassword(t *testing.T) {
	t.Parallel()

	// Helper to create test user with known password
	makeTestUser := func(t *testing.T, password string) *models.User {
		t.Helper()
		user := &models.User{
			ID:       1,
			Name:     "Test User",
			Username: "testuser",
			Email:    "test@example.com",
		}
		if err := user.SetPassword(password, 4); err != nil {
			t.Fatalf("Failed to set password: %v", err)
		}
		return user
	}

	tests := []struct {
		name           string
		requestBody    interface{}
		setupContext   func(*http.Request) *http.Request
		setupMocks     func(*MockUserRepository, *MockAuthProvider)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success - password changed",
			requestBody: ChangePasswordRequest{
				CurrentPassword: "oldpassword123",
				NewPassword:     "newpassword123",
			},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "1")
				return r.WithContext(ctx)
			},
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
				testUser := makeTestUser(t, "oldpassword123")
				userRepo.GetByIDFunc = func(_ int) (*models.User, error) {
					return testUser, nil
				}
				userRepo.UpdatePasswordFunc = func(_ int, _ string) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.SuccessResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.True(t, response.Success)
				assert.Equal(t, "Password changed successfully", response.Message)
			},
		},
		{
			name:        "error - not authenticated (no user ID in context)",
			requestBody: ChangePasswordRequest{CurrentPassword: "old", NewPassword: "newpassword123"},
			setupContext: func(r *http.Request) *http.Request {
				return r // No context modification - no user ID
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "Not authenticated", response.Error.Message)
			},
		},
		{
			name:        "error - invalid request body (malformed JSON)",
			requestBody: `{invalid json}`,
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "1")
				return r.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "Invalid request body", response.Error.Message)
			},
		},
		{
			name: "error - missing current password",
			requestBody: ChangePasswordRequest{
				CurrentPassword: "",
				NewPassword:     "newpassword123",
			},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "1")
				return r.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "Current password and new password are required", response.Error.Message)
			},
		},
		{
			name: "error - missing new password",
			requestBody: ChangePasswordRequest{
				CurrentPassword: "oldpassword123",
				NewPassword:     "",
			},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "1")
				return r.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "Current password and new password are required", response.Error.Message)
			},
		},
		{
			name: "error - both passwords missing",
			requestBody: ChangePasswordRequest{
				CurrentPassword: "",
				NewPassword:     "",
			},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "1")
				return r.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "Current password and new password are required", response.Error.Message)
			},
		},
		{
			name: "error - new password too short (less than 8 characters)",
			requestBody: ChangePasswordRequest{
				CurrentPassword: "oldpassword123",
				NewPassword:     "short",
			},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "1")
				return r.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "New password must be at least 8 characters", response.Error.Message)
			},
		},
		{
			name: "error - new password exactly 7 characters (boundary)",
			requestBody: ChangePasswordRequest{
				CurrentPassword: "oldpassword123",
				NewPassword:     "1234567",
			},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "1")
				return r.WithContext(ctx)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "New password must be at least 8 characters", response.Error.Message)
			},
		},
		{
			name: "success - new password exactly 8 characters (boundary)",
			requestBody: ChangePasswordRequest{
				CurrentPassword: "oldpassword123",
				NewPassword:     "12345678",
			},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "1")
				return r.WithContext(ctx)
			},
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
				testUser := makeTestUser(t, "oldpassword123")
				userRepo.GetByIDFunc = func(_ int) (*models.User, error) {
					return testUser, nil
				}
				userRepo.UpdatePasswordFunc = func(_ int, _ string) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "error - user not found",
			requestBody: ChangePasswordRequest{
				CurrentPassword: "oldpassword123",
				NewPassword:     "newpassword123",
			},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "999")
				return r.WithContext(ctx)
			},
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
				userRepo.GetByIDFunc = func(_ int) (*models.User, error) {
					return nil, gorm.ErrRecordNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "User not found", response.Error.Message)
			},
		},
		{
			name: "error - wrong current password",
			requestBody: ChangePasswordRequest{
				CurrentPassword: "wrongpassword",
				NewPassword:     "newpassword123",
			},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "1")
				return r.WithContext(ctx)
			},
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
				testUser := makeTestUser(t, "correctpassword123")
				userRepo.GetByIDFunc = func(_ int) (*models.User, error) {
					return testUser, nil
				}
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "Current password is incorrect", response.Error.Message)
			},
		},
		{
			name: "error - database error when updating password",
			requestBody: ChangePasswordRequest{
				CurrentPassword: "oldpassword123",
				NewPassword:     "newpassword123",
			},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "1")
				return r.WithContext(ctx)
			},
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
				testUser := makeTestUser(t, "oldpassword123")
				userRepo.GetByIDFunc = func(_ int) (*models.User, error) {
					return testUser, nil
				}
				userRepo.UpdatePasswordFunc = func(_ int, _ string) error {
					return errors.New("database connection error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "Failed to update password", response.Error.Message)
			},
		},
		{
			name: "error - database error when fetching user",
			requestBody: ChangePasswordRequest{
				CurrentPassword: "oldpassword123",
				NewPassword:     "newpassword123",
			},
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), middleware.UserIDKey, "1")
				return r.WithContext(ctx)
			},
			setupMocks: func(userRepo *MockUserRepository, _ *MockAuthProvider) {
				userRepo.GetByIDFunc = func(_ int) (*models.User, error) {
					return nil, errors.New("database connection error")
				}
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "User not found", response.Error.Message)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRepo := &MockUserRepository{}
			authProvider := &MockAuthProvider{}

			if tc.setupMocks != nil {
				tc.setupMocks(userRepo, authProvider)
			}

			handler := NewAuthHandler(authProvider, userRepo, defaultClosedSettings(), nil, 4, nil) // Low bcrypt cost for tests

			var body []byte
			switch v := tc.requestBody.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			if tc.setupContext != nil {
				req = tc.setupContext(req)
			}
			rec := httptest.NewRecorder()

			handler.ChangePassword(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if tc.checkResponse != nil {
				tc.checkResponse(t, rec)
			}
		})
	}
}

// TestChangePassword_WithAuditLogging tests that password changes are properly audit logged
func TestChangePassword_WithAuditLogging(t *testing.T) {
	t.Parallel()

	testUser := &models.User{
		ID:       1,
		Name:     "Test User",
		Username: "testuser",
		Email:    "test@example.com",
	}
	if err := testUser.SetPassword("oldpassword123", 4); err != nil {
		t.Fatalf("Failed to set password: %v", err)
	}

	auditLogCalled := false
	mockAuditService := &mocks.MockAuditService{
		LogPasswordChangeFunc: func(_ context.Context, userID int, username string, _, _ string) error {
			auditLogCalled = true
			assert.Equal(t, 1, userID)
			assert.Equal(t, "testuser", username)
			return nil
		},
	}

	userRepo := &MockUserRepository{
		GetByIDFunc: func(_ int) (*models.User, error) {
			return testUser, nil
		},
		UpdatePasswordFunc: func(_ int, _ string) error {
			return nil
		},
	}

	handler := NewAuthHandler(&MockAuthProvider{}, userRepo, defaultClosedSettings(), mockAuditService, 4, nil)

	reqBody := ChangePasswordRequest{
		CurrentPassword: "oldpassword123",
		NewPassword:     "newpassword123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Test-Agent")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ChangePassword(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, auditLogCalled, "Audit log should be called on successful password change")
}

// TestChangePassword_AuditLoggingNotCalledOnFailure tests that audit logging is not called when password change fails
func TestChangePassword_AuditLoggingNotCalledOnFailure(t *testing.T) {
	t.Parallel()

	testUser := &models.User{
		ID:       1,
		Name:     "Test User",
		Username: "testuser",
		Email:    "test@example.com",
	}
	if err := testUser.SetPassword("oldpassword123", 4); err != nil {
		t.Fatalf("Failed to set password: %v", err)
	}

	auditLogCalled := false
	mockAuditService := &mocks.MockAuditService{
		LogPasswordChangeFunc: func(_ context.Context, _ int, _ string, _, _ string) error {
			auditLogCalled = true
			return nil
		},
	}

	userRepo := &MockUserRepository{
		GetByIDFunc: func(_ int) (*models.User, error) {
			return testUser, nil
		},
	}

	handler := NewAuthHandler(&MockAuthProvider{}, userRepo, defaultClosedSettings(), mockAuditService, 4, nil)

	// Try with wrong password
	reqBody := ChangePasswordRequest{
		CurrentPassword: "wrongpassword",
		NewPassword:     "newpassword123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ChangePassword(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, auditLogCalled, "Audit log should NOT be called when password change fails")
}

// TestChangePassword_PasswordHashActuallyUpdated verifies the new password hash is different from the old one
func TestChangePassword_PasswordHashActuallyUpdated(t *testing.T) {
	t.Parallel()

	testUser := &models.User{
		ID:       1,
		Name:     "Test User",
		Username: "testuser",
		Email:    "test@example.com",
	}
	if err := testUser.SetPassword("oldpassword123", 4); err != nil {
		t.Fatalf("Failed to set password: %v", err)
	}
	originalHash := testUser.PasswordHash

	var updatedHash string
	userRepo := &MockUserRepository{
		GetByIDFunc: func(_ int) (*models.User, error) {
			return testUser, nil
		},
		UpdatePasswordFunc: func(_ int, passwordHash string) error {
			updatedHash = passwordHash
			return nil
		},
	}

	handler := NewAuthHandler(&MockAuthProvider{}, userRepo, defaultClosedSettings(), nil, 4, nil)

	reqBody := ChangePasswordRequest{
		CurrentPassword: "oldpassword123",
		NewPassword:     "newpassword123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ChangePassword(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, updatedHash, "Password hash should be updated")
	assert.NotEqual(t, originalHash, updatedHash, "New password hash should be different from original")
}

// TestChangePassword_CorrectUserIDUsed verifies the correct user ID is passed to repository
func TestChangePassword_CorrectUserIDUsed(t *testing.T) {
	t.Parallel()

	testUser := &models.User{
		ID:       42,
		Name:     "Test User",
		Username: "testuser",
		Email:    "test@example.com",
	}
	if err := testUser.SetPassword("oldpassword123", 4); err != nil {
		t.Fatalf("Failed to set password: %v", err)
	}

	var getByIDCalledWith int
	var updatePasswordCalledWith int

	userRepo := &MockUserRepository{
		GetByIDFunc: func(id int) (*models.User, error) {
			getByIDCalledWith = id
			return testUser, nil
		},
		UpdatePasswordFunc: func(id int, _ string) error {
			updatePasswordCalledWith = id
			return nil
		},
	}

	handler := NewAuthHandler(&MockAuthProvider{}, userRepo, defaultClosedSettings(), nil, 4, nil)

	reqBody := ChangePasswordRequest{
		CurrentPassword: "oldpassword123",
		NewPassword:     "newpassword123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "42")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ChangePassword(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 42, getByIDCalledWith, "GetByID should be called with user ID 42")
	assert.Equal(t, 42, updatePasswordCalledWith, "UpdatePassword should be called with user ID 42")
}

// TestRegistrationStatus tests the public registration status endpoint.
func TestRegistrationStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		settingsRepo *MockSettingsRepository
		wantOpen     bool
	}{
		{
			name:         "registration closed (default)",
			settingsRepo: defaultClosedSettings(),
			wantOpen:     false,
		},
		{
			name:         "registration open",
			settingsRepo: openSettings(),
			wantOpen:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := NewAuthHandler(&MockAuthProvider{}, &MockUserRepository{}, tc.settingsRepo, nil, 4, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/auth/registration-status", nil)
			rec := httptest.NewRecorder()

			handler.RegistrationStatus(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)

			var resp utils.SuccessResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			assert.True(t, resp.Success)

			dataBytes, err := json.Marshal(resp.Data)
			require.NoError(t, err)
			var data map[string]bool
			require.NoError(t, json.Unmarshal(dataBytes, &data))
			assert.Equal(t, tc.wantOpen, data["open"])
		})
	}
}
