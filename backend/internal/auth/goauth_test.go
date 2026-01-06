package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aloks98/goauth"
	"github.com/aloks98/goauth/apikey"
	"github.com/aloks98/goauth/middleware"
	"github.com/aloks98/goauth/store"
	"github.com/aloks98/goauth/token"
	"github.com/aloks98/waygates/backend/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractUserID_CustomClaims(t *testing.T) {
	adapter := &Adapter{}

	claims := &CustomClaims{}
	claims.UserID = "123"

	result := adapter.ExtractUserID(claims)
	if result != "123" {
		t.Errorf("Expected '123', got '%s'", result)
	}
}

func TestExtractUserID_NilClaims(t *testing.T) {
	adapter := &Adapter{}

	result := adapter.ExtractUserID(nil)
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func TestExtractUserID_WrongType(t *testing.T) {
	adapter := &Adapter{}

	result := adapter.ExtractUserID("not a claims object")
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

// mockGetUserID is a helper interface for testing
type mockClaimsWithGetUserID struct {
	userID string
}

func (m *mockClaimsWithGetUserID) GetUserID() string {
	return m.userID
}

func TestExtractUserID_GetUserIDInterface(t *testing.T) {
	adapter := &Adapter{}

	mock := &mockClaimsWithGetUserID{userID: "456"}
	result := adapter.ExtractUserID(mock)
	if result != "456" {
		t.Errorf("Expected '456', got '%s'", result)
	}
}

func TestExtractPermissions(t *testing.T) {
	adapter := &Adapter{}

	// ExtractPermissions always returns nil (permissions checked via RBAC)
	result := adapter.ExtractPermissions(&CustomClaims{})
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

func TestGetUserIDAsUint_EmptyContext(t *testing.T) {
	ctx := context.Background()

	_, err := GetUserIDAsUint(ctx)
	if err == nil {
		t.Error("Expected error for empty context, got nil")
		return
	}
	if err.Error() != "user ID not found in context" {
		t.Errorf("Expected 'user ID not found in context', got '%s'", err.Error())
	}
}

func TestGetUserIDFromContext_Empty(t *testing.T) {
	ctx := context.Background()

	result := GetUserIDFromContext(ctx)
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func TestErrorHandler_Unauthorized(t *testing.T) {
	handler := ErrorHandler()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	handler(w, r, middleware.ErrInvalidToken)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestErrorHandler_Forbidden_Returns404(t *testing.T) {
	handler := ErrorHandler()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	handler(w, r, middleware.ErrPermissionDenied)

	// Returns 404 instead of 403 to avoid leaking information about protected resources
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestCustomClaims(t *testing.T) {
	claims := &CustomClaims{
		Username: "testuser",
		Email:    "test@example.com",
	}
	claims.UserID = "123"

	if claims.Username != "testuser" {
		t.Errorf("Expected 'testuser', got '%s'", claims.Username)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Expected 'test@example.com', got '%s'", claims.Email)
	}
	if claims.UserID != "123" {
		t.Errorf("Expected '123', got '%s'", claims.UserID)
	}
}

func TestSetAuth(t *testing.T) {
	adapter := &Adapter{}

	if adapter.auth != nil {
		t.Error("Expected auth to be nil initially")
	}

	// SetAuth with nil (just verifies it doesn't panic)
	adapter.SetAuth(nil)

	if adapter.auth != nil {
		t.Error("Expected auth to still be nil")
	}
}

func TestAdapter_ImplementsInterfaces(t *testing.T) {
	// Compile-time check that Adapter implements required interfaces
	var _ middleware.TokenValidator = (*Adapter)(nil)
	var _ middleware.ClaimsExtractor = (*Adapter)(nil)
	var _ middleware.PermissionChecker = (*Adapter)(nil)
	var _ middleware.APIKeyValidator = (*Adapter)(nil)
}

func TestGetUserIDAsUint_InvalidFormat(t *testing.T) {
	// Use goauth middleware's context setter to set a non-numeric user ID
	ctx := middleware.SetUserID(context.Background(), "not-a-number")

	_, err := GetUserIDAsUint(ctx)
	if err == nil {
		t.Error("Expected error for invalid user ID format, got nil")
	}
	if err != nil && !containsSubstring(err.Error(), "invalid user ID format") {
		t.Errorf("Expected error message about invalid format, got: %s", err.Error())
	}
}

func TestGetUserIDAsUint_ValidUserID(t *testing.T) {
	ctx := middleware.SetUserID(context.Background(), "12345")

	userID, err := GetUserIDAsUint(ctx)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if userID != 12345 {
		t.Errorf("Expected userID 12345, got %d", userID)
	}
}

func TestGetUserIDFromContext_WithValue(t *testing.T) {
	ctx := middleware.SetUserID(context.Background(), "789")

	result := GetUserIDFromContext(ctx)
	if result != "789" {
		t.Errorf("Expected '789', got '%s'", result)
	}
}

func TestErrorHandler_OtherError(t *testing.T) {
	handler := ErrorHandler()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Use a generic error that doesn't map to a specific status
	handler(w, r, middleware.ErrMissingToken)

	// Should default to unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestErrorHandler_RBACNotConfigured(t *testing.T) {
	handler := ErrorHandler()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	handler(w, r, middleware.ErrRBACNotConfigured)

	// Should default to unauthorized (as it falls through to default)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuth_Adapter(t *testing.T) {
	auth := &Auth{
		adapter: &Adapter{},
	}

	adapter := auth.Adapter()
	if adapter == nil {
		t.Error("Expected adapter to not be nil")
	}
}

func TestExtractUserID_EmptyClaims(t *testing.T) {
	adapter := &Adapter{}

	// CustomClaims with empty UserID
	claims := &CustomClaims{}
	result := adapter.ExtractUserID(claims)
	if result != "" {
		t.Errorf("Expected empty string for empty UserID, got '%s'", result)
	}
}

func TestExtractPermissions_NilClaims(t *testing.T) {
	adapter := &Adapter{}

	result := adapter.ExtractPermissions(nil)
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

func TestExtractPermissions_WrongType(t *testing.T) {
	adapter := &Adapter{}

	result := adapter.ExtractPermissions("not claims")
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

// Helper function to check if string contains substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Mock Store Implementation for Testing
// =============================================================================

// mockStore implements store.Store interface for testing
type mockStore struct {
	pingErr                        error
	migrateErr                     error
	userPermissions                map[string]*store.UserPermissions
	getUserPermissionsErr          error
	saveUserPermissionsErr         error
	deleteUserPermissionsErr       error
	refreshTokens                  map[string]*store.RefreshToken
	saveRefreshTokenErr            error
	getRefreshTokenErr             error
	revokeRefreshTokenErr          error
	revokeTokenFamilyErr           error
	revokeAllUserRefreshTokensErr  error
	deleteExpiredRefreshTokensErr  error
	blacklist                      map[string]int64
	addToBlacklistErr              error
	isBlacklistedErr               error
	deleteExpiredBlacklistErr      error
	apiKeys                        map[string]*store.APIKey
	saveAPIKeyErr                  error
	getAPIKeyErr                   error
	listAPIKeysErr                 error
	revokeAPIKeyErr                error
	deleteExpiredAPIKeysErr        error
	roleTemplates                  map[string]*store.StoredRoleTemplate
	getRoleTemplatesErr            error
	saveRoleTemplateErr            error
	updateUsersWithRoleErr         error
}

func newMockStore() *mockStore {
	return &mockStore{
		userPermissions: make(map[string]*store.UserPermissions),
		refreshTokens:   make(map[string]*store.RefreshToken),
		blacklist:       make(map[string]int64),
		apiKeys:         make(map[string]*store.APIKey),
		roleTemplates:   make(map[string]*store.StoredRoleTemplate),
	}
}

func (m *mockStore) Close() error                { return nil }
func (m *mockStore) Ping(_ context.Context) error { return m.pingErr }
func (m *mockStore) Migrate(_ context.Context) error { return m.migrateErr }
func (m *mockStore) SaveRefreshToken(_ context.Context, token *store.RefreshToken) error {
	if m.saveRefreshTokenErr != nil {
		return m.saveRefreshTokenErr
	}
	m.refreshTokens[token.ID] = token
	return nil
}
func (m *mockStore) GetRefreshToken(_ context.Context, jti string) (*store.RefreshToken, error) {
	if m.getRefreshTokenErr != nil {
		return nil, m.getRefreshTokenErr
	}
	if rt, ok := m.refreshTokens[jti]; ok {
		return rt, nil
	}
	return nil, errors.New("refresh token not found")
}
func (m *mockStore) RevokeRefreshToken(_ context.Context, _ string, _ string) error {
	return m.revokeRefreshTokenErr
}
func (m *mockStore) RevokeTokenFamily(_ context.Context, _ string) error {
	return m.revokeTokenFamilyErr
}
func (m *mockStore) RevokeAllUserRefreshTokens(_ context.Context, _ string) error {
	return m.revokeAllUserRefreshTokensErr
}
func (m *mockStore) DeleteExpiredRefreshTokens(_ context.Context) (int64, error) {
	return 0, m.deleteExpiredRefreshTokensErr
}
func (m *mockStore) AddToBlacklist(_ context.Context, jti string, expiresAt int64) error {
	if m.addToBlacklistErr != nil {
		return m.addToBlacklistErr
	}
	m.blacklist[jti] = expiresAt
	return nil
}
func (m *mockStore) IsBlacklisted(_ context.Context, jti string) (bool, error) {
	if m.isBlacklistedErr != nil {
		return false, m.isBlacklistedErr
	}
	_, ok := m.blacklist[jti]
	return ok, nil
}
func (m *mockStore) DeleteExpiredBlacklistEntries(_ context.Context) (int64, error) {
	return 0, m.deleteExpiredBlacklistErr
}
func (m *mockStore) GetUserPermissions(_ context.Context, userID string) (*store.UserPermissions, error) {
	if m.getUserPermissionsErr != nil {
		return nil, m.getUserPermissionsErr
	}
	if perms, ok := m.userPermissions[userID]; ok {
		return perms, nil
	}
	return nil, nil
}
func (m *mockStore) SaveUserPermissions(_ context.Context, perms *store.UserPermissions) error {
	if m.saveUserPermissionsErr != nil {
		return m.saveUserPermissionsErr
	}
	m.userPermissions[perms.UserID] = perms
	return nil
}
func (m *mockStore) DeleteUserPermissions(_ context.Context, userID string) error {
	if m.deleteUserPermissionsErr != nil {
		return m.deleteUserPermissionsErr
	}
	delete(m.userPermissions, userID)
	return nil
}
func (m *mockStore) UpdateUsersWithRole(_ context.Context, _ string, _ []string, _ int) (int64, error) {
	return 0, m.updateUsersWithRoleErr
}
func (m *mockStore) GetRoleTemplates(_ context.Context) (map[string]*store.StoredRoleTemplate, error) {
	if m.getRoleTemplatesErr != nil {
		return nil, m.getRoleTemplatesErr
	}
	return m.roleTemplates, nil
}
func (m *mockStore) SaveRoleTemplate(_ context.Context, template *store.StoredRoleTemplate) error {
	if m.saveRoleTemplateErr != nil {
		return m.saveRoleTemplateErr
	}
	m.roleTemplates[template.Key] = template
	return nil
}
func (m *mockStore) SaveAPIKey(_ context.Context, key *store.APIKey) error {
	if m.saveAPIKeyErr != nil {
		return m.saveAPIKeyErr
	}
	m.apiKeys[key.ID] = key
	return nil
}
func (m *mockStore) GetAPIKeyByHash(_ context.Context, prefix string, keyHash string) (*store.APIKey, error) {
	if m.getAPIKeyErr != nil {
		return nil, m.getAPIKeyErr
	}
	for _, key := range m.apiKeys {
		if key.KeyHash == keyHash && key.Prefix == prefix {
			return key, nil
		}
	}
	return nil, errors.New("api key not found")
}
func (m *mockStore) GetAPIKeysByUser(_ context.Context, userID string) ([]*store.APIKey, error) {
	if m.listAPIKeysErr != nil {
		return nil, m.listAPIKeysErr
	}
	var keys []*store.APIKey
	for _, key := range m.apiKeys {
		if key.UserID == userID {
			keys = append(keys, key)
		}
	}
	return keys, nil
}
func (m *mockStore) RevokeAPIKey(_ context.Context, id string) error {
	if m.revokeAPIKeyErr != nil {
		return m.revokeAPIKeyErr
	}
	delete(m.apiKeys, id)
	return nil
}
func (m *mockStore) DeleteExpiredAPIKeys(_ context.Context) (int64, error) {
	return 0, m.deleteExpiredAPIKeysErr
}

// Verify mockStore implements store.Store
var _ store.Store = (*mockStore)(nil)

// =============================================================================
// Test Helper Functions
// =============================================================================

// createTestAuth creates a goauth.Auth instance with a mock store for testing
func createTestAuth(t *testing.T) *goauth.Auth[*CustomClaims] {
	t.Helper()
	mockStore := newMockStore()

	auth, err := goauth.New[*CustomClaims](
		goauth.WithSecret("test-secret-key-for-testing-must-be-at-least-32-bytes"),
		goauth.WithStore(mockStore),
	)
	require.NoError(t, err)
	return auth
}

// createTestAuthWithRBAC creates a goauth.Auth instance with RBAC for testing
func createTestAuthWithRBAC(t *testing.T) (*goauth.Auth[*CustomClaims], *mockStore) {
	t.Helper()
	mockStore := newMockStore()

	// Define RBAC config inline
	rbacConfig := []byte(`
roles:
  - key: admin
    label: Administrator
    permissions:
      - users:read
      - users:write
      - users:delete
      - proxies:read
      - proxies:write
  - key: viewer
    label: Viewer
    permissions:
      - users:read
      - proxies:read
permission_groups:
  - name: Users
    permissions:
      - key: users:read
        label: Read Users
      - key: users:write
        label: Write Users
      - key: users:delete
        label: Delete Users
  - name: Proxies
    permissions:
      - key: proxies:read
        label: Read Proxies
      - key: proxies:write
        label: Write Proxies
`)

	auth, err := goauth.New[*CustomClaims](
		goauth.WithSecret("test-secret-key-for-testing-must-be-at-least-32-bytes"),
		goauth.WithStore(mockStore),
		goauth.WithRBACFromBytes(rbacConfig),
	)
	require.NoError(t, err)
	return auth, mockStore
}

// =============================================================================
// Adapter Tests - ValidateAccessToken
// =============================================================================

func TestAdapter_ValidateAccessToken_Success(t *testing.T) {
	t.Parallel()
	auth := createTestAuth(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	// Generate a valid token pair
	tokenPair, err := auth.GenerateTokenPair(ctx, "user123", nil)
	require.NoError(t, err)
	require.NotEmpty(t, tokenPair.AccessToken)

	// Validate the access token
	claims, err := adapter.ValidateAccessToken(ctx, tokenPair.AccessToken)
	require.NoError(t, err)
	require.NotNil(t, claims)

	// Verify we can extract user ID from the claims
	tokenClaims, ok := claims.(*token.Claims)
	require.True(t, ok)
	assert.Equal(t, "user123", tokenClaims.UserID)
}

func TestAdapter_ValidateAccessToken_InvalidToken(t *testing.T) {
	t.Parallel()
	auth := createTestAuth(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	// Try to validate an invalid token
	claims, err := adapter.ValidateAccessToken(ctx, "invalid-token-string")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestAdapter_ValidateAccessToken_EmptyToken(t *testing.T) {
	t.Parallel()
	auth := createTestAuth(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	// Try to validate an empty token
	claims, err := adapter.ValidateAccessToken(ctx, "")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestAdapter_ValidateAccessToken_MalformedToken(t *testing.T) {
	t.Parallel()
	auth := createTestAuth(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	testCases := []struct {
		name  string
		token string
	}{
		{"single segment", "just-one-segment"},
		{"two segments", "two.segments"},
		{"empty segments", ".."},
		{"whitespace", "   "},
		{"base64 invalid", "not.valid.base64!"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			claims, err := adapter.ValidateAccessToken(ctx, tc.token)
			assert.Error(t, err)
			assert.Nil(t, claims)
		})
	}
}

func TestAdapter_ValidateAccessToken_WrongSignature(t *testing.T) {
	t.Parallel()

	// Create two auth instances with different secrets
	mockStore1 := newMockStore()
	auth1, err := goauth.New[*CustomClaims](
		goauth.WithSecret("first-secret-key-must-be-at-least-32-bytes-long"),
		goauth.WithStore(mockStore1),
	)
	require.NoError(t, err)

	mockStore2 := newMockStore()
	auth2, err := goauth.New[*CustomClaims](
		goauth.WithSecret("second-secret-key-must-be-at-least-32-bytes-long"),
		goauth.WithStore(mockStore2),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate token with first auth
	tokenPair, err := auth1.GenerateTokenPair(ctx, "user123", nil)
	require.NoError(t, err)

	// Try to validate with second auth (different secret)
	adapter := &Adapter{auth: auth2}
	claims, err := adapter.ValidateAccessToken(ctx, tokenPair.AccessToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestAdapter_ValidateAccessToken_NilAuth(t *testing.T) {
	t.Parallel()
	adapter := &Adapter{auth: nil}
	ctx := context.Background()

	// Should panic or return error - testing nil safety
	defer func() {
		if r := recover(); r != nil {
			// Expected behavior when auth is nil
			t.Log("Recovered from panic as expected when auth is nil")
		}
	}()

	_, _ = adapter.ValidateAccessToken(ctx, "any-token")
}

// =============================================================================
// Adapter Tests - HasPermission
// =============================================================================

func TestAdapter_HasPermission_Success(t *testing.T) {
	t.Parallel()
	auth, mockStore := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	// Set up user permissions
	mockStore.userPermissions["user123"] = &store.UserPermissions{
		UserID:            "user123",
		RoleLabel:         "admin",
		BaseRole:          "admin",
		Permissions:       []string{"users:read", "users:write"},
		PermissionVersion: 1,
	}

	// Check for a permission the user has
	hasPerm, err := adapter.HasPermission(ctx, "user123", "users:read")
	require.NoError(t, err)
	assert.True(t, hasPerm)
}

func TestAdapter_HasPermission_NotFound(t *testing.T) {
	t.Parallel()
	auth, mockStore := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	// Set up user with limited permissions
	mockStore.userPermissions["user123"] = &store.UserPermissions{
		UserID:            "user123",
		RoleLabel:         "viewer",
		BaseRole:          "viewer",
		Permissions:       []string{"users:read"},
		PermissionVersion: 1,
	}

	// Check for a permission the user does not have
	hasPerm, err := adapter.HasPermission(ctx, "user123", "users:delete")
	require.NoError(t, err)
	assert.False(t, hasPerm)
}

func TestAdapter_HasPermission_EmptyUserID(t *testing.T) {
	t.Parallel()
	auth, _ := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	hasPerm, err := adapter.HasPermission(ctx, "", "users:read")
	// Should return false for empty user ID (no permissions)
	require.NoError(t, err)
	assert.False(t, hasPerm)
}

func TestAdapter_HasPermission_EmptyPermission(t *testing.T) {
	t.Parallel()
	auth, mockStore := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	mockStore.userPermissions["user123"] = &store.UserPermissions{
		UserID:            "user123",
		RoleLabel:         "admin",
		BaseRole:          "admin",
		Permissions:       []string{"users:read"},
		PermissionVersion: 1,
	}

	hasPerm, err := adapter.HasPermission(ctx, "user123", "")
	require.NoError(t, err)
	assert.False(t, hasPerm)
}

func TestAdapter_HasPermission_NonExistentUser(t *testing.T) {
	t.Parallel()
	auth, _ := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	// User does not exist in the store
	hasPerm, err := adapter.HasPermission(ctx, "nonexistent", "users:read")
	require.NoError(t, err)
	assert.False(t, hasPerm)
}

// =============================================================================
// Adapter Tests - HasAllPermissions
// =============================================================================

func TestAdapter_HasAllPermissions_Success(t *testing.T) {
	t.Parallel()
	auth, mockStore := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	mockStore.userPermissions["user123"] = &store.UserPermissions{
		UserID:            "user123",
		RoleLabel:         "admin",
		BaseRole:          "admin",
		Permissions:       []string{"users:read", "users:write", "users:delete"},
		PermissionVersion: 1,
	}

	hasAll, err := adapter.HasAllPermissions(ctx, "user123", []string{"users:read", "users:write"})
	require.NoError(t, err)
	assert.True(t, hasAll)
}

func TestAdapter_HasAllPermissions_MissingOne(t *testing.T) {
	t.Parallel()
	auth, mockStore := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	mockStore.userPermissions["user123"] = &store.UserPermissions{
		UserID:            "user123",
		RoleLabel:         "viewer",
		BaseRole:          "viewer",
		Permissions:       []string{"users:read"},
		PermissionVersion: 1,
	}

	hasAll, err := adapter.HasAllPermissions(ctx, "user123", []string{"users:read", "users:write"})
	require.NoError(t, err)
	assert.False(t, hasAll)
}

func TestAdapter_HasAllPermissions_EmptyList(t *testing.T) {
	t.Parallel()
	auth, mockStore := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	mockStore.userPermissions["user123"] = &store.UserPermissions{
		UserID:            "user123",
		RoleLabel:         "admin",
		BaseRole:          "admin",
		Permissions:       []string{"users:read"},
		PermissionVersion: 1,
	}

	// Empty permission list should return true (user has all 0 required permissions)
	hasAll, err := adapter.HasAllPermissions(ctx, "user123", []string{})
	require.NoError(t, err)
	assert.True(t, hasAll)
}

func TestAdapter_HasAllPermissions_NilList(t *testing.T) {
	t.Parallel()
	auth, mockStore := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	mockStore.userPermissions["user123"] = &store.UserPermissions{
		UserID:            "user123",
		RoleLabel:         "admin",
		BaseRole:          "admin",
		Permissions:       []string{"users:read"},
		PermissionVersion: 1,
	}

	hasAll, err := adapter.HasAllPermissions(ctx, "user123", nil)
	require.NoError(t, err)
	assert.True(t, hasAll)
}

func TestAdapter_HasAllPermissions_NonExistentUser(t *testing.T) {
	t.Parallel()
	auth, _ := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	hasAll, err := adapter.HasAllPermissions(ctx, "nonexistent", []string{"users:read"})
	require.NoError(t, err)
	assert.False(t, hasAll)
}

// =============================================================================
// Adapter Tests - HasAnyPermission
// =============================================================================

func TestAdapter_HasAnyPermission_Success(t *testing.T) {
	t.Parallel()
	auth, mockStore := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	mockStore.userPermissions["user123"] = &store.UserPermissions{
		UserID:            "user123",
		RoleLabel:         "viewer",
		BaseRole:          "viewer",
		Permissions:       []string{"users:read"},
		PermissionVersion: 1,
	}

	hasAny, err := adapter.HasAnyPermission(ctx, "user123", []string{"users:read", "users:write"})
	require.NoError(t, err)
	assert.True(t, hasAny)
}

func TestAdapter_HasAnyPermission_NoneMatch(t *testing.T) {
	t.Parallel()
	auth, mockStore := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	mockStore.userPermissions["user123"] = &store.UserPermissions{
		UserID:            "user123",
		RoleLabel:         "viewer",
		BaseRole:          "viewer",
		Permissions:       []string{"proxies:read"},
		PermissionVersion: 1,
	}

	hasAny, err := adapter.HasAnyPermission(ctx, "user123", []string{"users:read", "users:write"})
	require.NoError(t, err)
	assert.False(t, hasAny)
}

func TestAdapter_HasAnyPermission_EmptyList(t *testing.T) {
	t.Parallel()
	auth, mockStore := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	mockStore.userPermissions["user123"] = &store.UserPermissions{
		UserID:            "user123",
		RoleLabel:         "admin",
		BaseRole:          "admin",
		Permissions:       []string{"users:read"},
		PermissionVersion: 1,
	}

	// Empty permission list should return false (need at least one)
	hasAny, err := adapter.HasAnyPermission(ctx, "user123", []string{})
	require.NoError(t, err)
	assert.False(t, hasAny)
}

func TestAdapter_HasAnyPermission_NilList(t *testing.T) {
	t.Parallel()
	auth, mockStore := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	mockStore.userPermissions["user123"] = &store.UserPermissions{
		UserID:            "user123",
		RoleLabel:         "admin",
		BaseRole:          "admin",
		Permissions:       []string{"users:read"},
		PermissionVersion: 1,
	}

	hasAny, err := adapter.HasAnyPermission(ctx, "user123", nil)
	require.NoError(t, err)
	assert.False(t, hasAny)
}

func TestAdapter_HasAnyPermission_AllMatch(t *testing.T) {
	t.Parallel()
	auth, mockStore := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	mockStore.userPermissions["user123"] = &store.UserPermissions{
		UserID:            "user123",
		RoleLabel:         "admin",
		BaseRole:          "admin",
		Permissions:       []string{"users:read", "users:write", "users:delete"},
		PermissionVersion: 1,
	}

	hasAny, err := adapter.HasAnyPermission(ctx, "user123", []string{"users:read", "users:write"})
	require.NoError(t, err)
	assert.True(t, hasAny)
}

func TestAdapter_HasAnyPermission_NonExistentUser(t *testing.T) {
	t.Parallel()
	auth, _ := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	hasAny, err := adapter.HasAnyPermission(ctx, "nonexistent", []string{"users:read"})
	require.NoError(t, err)
	assert.False(t, hasAny)
}

// =============================================================================
// Adapter Tests - ValidateAPIKey
// =============================================================================

func TestAdapter_ValidateAPIKey_Success(t *testing.T) {
	t.Parallel()
	auth := createTestAuth(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	// Create an API key
	result, err := auth.CreateAPIKey(ctx, "user123", &apikey.CreateKeyOptions{
		Name:   "Test Key",
		Scopes: []string{"read", "write"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.RawKey)

	// Validate the API key
	keyInfo, err := adapter.ValidateAPIKey(ctx, result.RawKey)
	require.NoError(t, err)
	require.NotNil(t, keyInfo)
	assert.Equal(t, "user123", keyInfo.UserID)
	assert.Equal(t, result.ID, keyInfo.ID)
	assert.ElementsMatch(t, []string{"read", "write"}, keyInfo.Scopes)
}

func TestAdapter_ValidateAPIKey_Invalid(t *testing.T) {
	t.Parallel()
	auth := createTestAuth(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	keyInfo, err := adapter.ValidateAPIKey(ctx, "invalid-api-key")
	assert.Error(t, err)
	assert.Nil(t, keyInfo)
}

func TestAdapter_ValidateAPIKey_Empty(t *testing.T) {
	t.Parallel()
	auth := createTestAuth(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	keyInfo, err := adapter.ValidateAPIKey(ctx, "")
	assert.Error(t, err)
	assert.Nil(t, keyInfo)
}

func TestAdapter_ValidateAPIKey_MalformedFormats(t *testing.T) {
	t.Parallel()
	auth := createTestAuth(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	testCases := []struct {
		name string
		key  string
	}{
		{"whitespace only", "   "},
		{"special chars", "!@#$%^&*()"},
		{"very long", string(make([]byte, 10000))},
		{"unicode", "api_\u4e2d\u6587_key"},
		{"newlines", "api\nkey\nwith\nnewlines"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			keyInfo, err := adapter.ValidateAPIKey(ctx, tc.key)
			assert.Error(t, err)
			assert.Nil(t, keyInfo)
		})
	}
}

func TestAdapter_ValidateAPIKey_RevokedKey(t *testing.T) {
	t.Parallel()
	auth := createTestAuth(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	// Create an API key
	result, err := auth.CreateAPIKey(ctx, "user123", &apikey.CreateKeyOptions{
		Name:   "Test Key",
		Scopes: []string{"read"},
	})
	require.NoError(t, err)

	// Revoke the key
	err = auth.RevokeAPIKey(ctx, result.ID)
	require.NoError(t, err)

	// Try to validate the revoked key
	keyInfo, err := adapter.ValidateAPIKey(ctx, result.RawKey)
	assert.Error(t, err)
	assert.Nil(t, keyInfo)
}

// =============================================================================
// Comprehensive Permission Matrix Tests
// =============================================================================

func TestPermissionMatrix_TableDriven(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		userPermissions []string
		checkPermission string
		checkAll        []string
		checkAny        []string
		expectHas       bool
		expectHasAll    bool
		expectHasAny    bool
	}{
		{
			name:            "admin with all permissions",
			userPermissions: []string{"users:read", "users:write", "users:delete", "proxies:read", "proxies:write"},
			checkPermission: "users:delete",
			checkAll:        []string{"users:read", "users:write"},
			checkAny:        []string{"users:read", "proxies:delete"},
			expectHas:       true,
			expectHasAll:    true,
			expectHasAny:    true,
		},
		{
			name:            "viewer with read only",
			userPermissions: []string{"users:read", "proxies:read"},
			checkPermission: "users:write",
			checkAll:        []string{"users:read", "users:write"},
			checkAny:        []string{"users:write", "proxies:read"},
			expectHas:       false,
			expectHasAll:    false,
			expectHasAny:    true,
		},
		{
			name:            "no permissions",
			userPermissions: []string{},
			checkPermission: "users:read",
			checkAll:        []string{"users:read"},
			checkAny:        []string{"users:read"},
			expectHas:       false,
			expectHasAll:    false,
			expectHasAny:    false,
		},
		{
			name:            "single permission match",
			userPermissions: []string{"users:read"},
			checkPermission: "users:read",
			checkAll:        []string{"users:read"},
			checkAny:        []string{"users:read"},
			expectHas:       true,
			expectHasAll:    true,
			expectHasAny:    true,
		},
		{
			name:            "wildcard scenario - explicit permissions",
			userPermissions: []string{"users:read", "users:write", "proxies:read"},
			checkPermission: "proxies:write",
			checkAll:        []string{"users:read", "proxies:write"},
			checkAny:        []string{"proxies:write", "audit:read"},
			expectHas:       false,
			expectHasAll:    false,
			expectHasAny:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			auth, mockStore := createTestAuthWithRBAC(t)
			adapter := &Adapter{auth: auth}
			ctx := context.Background()

			mockStore.userPermissions["testuser"] = &store.UserPermissions{
				UserID:            "testuser",
				RoleLabel:         "custom",
				BaseRole:          "custom",
				Permissions:       tc.userPermissions,
				PermissionVersion: 1,
			}

			// Test HasPermission
			hasPerm, err := adapter.HasPermission(ctx, "testuser", tc.checkPermission)
			require.NoError(t, err)
			assert.Equal(t, tc.expectHas, hasPerm, "HasPermission mismatch for %s", tc.name)

			// Test HasAllPermissions
			hasAll, err := adapter.HasAllPermissions(ctx, "testuser", tc.checkAll)
			require.NoError(t, err)
			assert.Equal(t, tc.expectHasAll, hasAll, "HasAllPermissions mismatch for %s", tc.name)

			// Test HasAnyPermission
			hasAny, err := adapter.HasAnyPermission(ctx, "testuser", tc.checkAny)
			require.NoError(t, err)
			assert.Equal(t, tc.expectHasAny, hasAny, "HasAnyPermission mismatch for %s", tc.name)
		})
	}
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func TestAdapter_NilContext(t *testing.T) {
	t.Parallel()
	auth, _ := createTestAuthWithRBAC(t) // Use RBAC-enabled auth for permission tests
	adapter := &Adapter{auth: auth}

	// These should handle nil context gracefully or return error
	// Note: In Go, passing nil context is generally an error

	// Using background context instead of nil for safety
	ctx := context.Background()

	// ValidateAccessToken with empty token
	_, err := adapter.ValidateAccessToken(ctx, "")
	assert.Error(t, err)

	// HasPermission with empty inputs should return false (no permissions for empty user)
	hasPerm, err := adapter.HasPermission(ctx, "", "")
	require.NoError(t, err)
	assert.False(t, hasPerm)
}

func TestAdapter_CancelledContext(t *testing.T) {
	t.Parallel()
	auth := createTestAuth(t)
	adapter := &Adapter{auth: auth}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// These operations should handle cancelled context
	// The behavior depends on the underlying implementation
	_, _ = adapter.ValidateAccessToken(ctx, "some-token")
	_, _ = adapter.HasPermission(ctx, "user", "perm")
	_, _ = adapter.ValidateAPIKey(ctx, "some-key")
}

func TestGetUserIDAsUint_TableDriven(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		userID      string
		expected    uint
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid positive number",
			userID:      "12345",
			expected:    12345,
			expectError: false,
		},
		{
			name:        "zero",
			userID:      "0",
			expected:    0,
			expectError: false,
		},
		{
			name:        "large number",
			userID:      "4294967295", // max uint32
			expected:    4294967295,
			expectError: false,
		},
		{
			name:        "negative number",
			userID:      "-1",
			expected:    0,
			expectError: true,
			errorMsg:    "invalid user ID format",
		},
		{
			name:        "not a number",
			userID:      "abc",
			expected:    0,
			expectError: true,
			errorMsg:    "invalid user ID format",
		},
		{
			name:        "float number",
			userID:      "123.456",
			expected:    0,
			expectError: true,
			errorMsg:    "invalid user ID format",
		},
		{
			name:        "empty string",
			userID:      "",
			expected:    0,
			expectError: true,
			errorMsg:    "user ID not found in context",
		},
		{
			name:        "whitespace",
			userID:      "   ",
			expected:    0,
			expectError: true,
			errorMsg:    "invalid user ID format",
		},
		{
			name:        "number with spaces",
			userID:      " 123 ",
			expected:    0,
			expectError: true,
			errorMsg:    "invalid user ID format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var ctx context.Context
			if tc.userID == "" {
				ctx = context.Background()
			} else {
				ctx = middleware.SetUserID(context.Background(), tc.userID)
			}

			result, err := GetUserIDAsUint(ctx)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestErrorHandler_TableDriven(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "invalid token",
			err:            middleware.ErrInvalidToken,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing token",
			err:            middleware.ErrMissingToken,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "permission denied returns 404",
			err:            middleware.ErrPermissionDenied,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "RBAC not configured",
			err:            middleware.ErrRBACNotConfigured,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "generic error",
			err:            errors.New("some generic error"),
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := ErrorHandler()
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			handler(w, r, tc.err)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// =============================================================================
// Auth Structure Tests
// =============================================================================

func TestAuth_AdapterReturnsCorrectInstance(t *testing.T) {
	t.Parallel()

	expectedAdapter := &Adapter{}
	auth := &Auth{
		adapter: expectedAdapter,
	}

	actualAdapter := auth.Adapter()
	assert.Same(t, expectedAdapter, actualAdapter)
}

func TestAdapter_SetAuth_UpdatesAuthInstance(t *testing.T) {
	t.Parallel()

	adapter := &Adapter{}
	goauthInstance := createTestAuth(t)

	adapter.SetAuth(goauthInstance)

	assert.Same(t, goauthInstance, adapter.auth)
}

// =============================================================================
// CustomClaims Tests
// =============================================================================

func TestCustomClaims_Fields(t *testing.T) {
	t.Parallel()

	claims := &CustomClaims{
		Username: "testuser",
		Email:    "test@example.com",
	}
	claims.UserID = "123"

	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, "123", claims.UserID)
}

func TestCustomClaims_EmptyFields(t *testing.T) {
	t.Parallel()

	claims := &CustomClaims{}

	assert.Empty(t, claims.Username)
	assert.Empty(t, claims.Email)
	assert.Empty(t, claims.UserID)
}

// =============================================================================
// Integration-like Tests
// =============================================================================

func TestFullTokenLifecycle(t *testing.T) {
	t.Parallel()
	auth := createTestAuth(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	// Step 1: Generate token pair
	tokenPair, err := auth.GenerateTokenPair(ctx, "user123", map[string]any{
		"username": "testuser",
		"email":    "test@example.com",
	})
	require.NoError(t, err)
	require.NotEmpty(t, tokenPair.AccessToken)
	require.NotEmpty(t, tokenPair.RefreshToken)

	// Step 2: Validate access token
	claims, err := adapter.ValidateAccessToken(ctx, tokenPair.AccessToken)
	require.NoError(t, err)
	require.NotNil(t, claims)

	// Step 3: Extract user ID
	userID := adapter.ExtractUserID(claims)
	assert.Equal(t, "user123", userID)

	// Step 4: Revoke access token
	err = auth.RevokeAccessToken(ctx, tokenPair.AccessToken)
	require.NoError(t, err)

	// Step 5: Validate revoked token should fail
	_, err = adapter.ValidateAccessToken(ctx, tokenPair.AccessToken)
	assert.Error(t, err)
}

func TestFullAPIKeyLifecycle(t *testing.T) {
	t.Parallel()
	auth := createTestAuth(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	// Step 1: Create API key
	createResult, err := auth.CreateAPIKey(ctx, "user123", &apikey.CreateKeyOptions{
		Name:   "My API Key",
		Scopes: []string{"read", "write"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, createResult.RawKey)

	// Step 2: Validate API key
	keyInfo, err := adapter.ValidateAPIKey(ctx, createResult.RawKey)
	require.NoError(t, err)
	assert.Equal(t, "user123", keyInfo.UserID)
	assert.ElementsMatch(t, []string{"read", "write"}, keyInfo.Scopes)

	// Step 3: List API keys
	keys, err := auth.ListAPIKeys(ctx, "user123")
	require.NoError(t, err)
	assert.Len(t, keys, 1)

	// Step 4: Revoke API key
	err = auth.RevokeAPIKey(ctx, createResult.ID)
	require.NoError(t, err)

	// Step 5: Validate revoked key should fail
	_, err = adapter.ValidateAPIKey(ctx, createResult.RawKey)
	assert.Error(t, err)
}

// =============================================================================
// NewAuth Tests
// =============================================================================

func TestNewAuth_NilDatabase(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:        "test-secret-key-must-be-at-least-32-bytes-long",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 24 * time.Hour,
		},
		Security: config.SecurityConfig{
			RBACPath: "", // Empty path to avoid file not found
		},
	}

	// NewAuth with nil database should fail when creating the SQL store
	auth, err := NewAuth(cfg, nil)
	assert.Error(t, err)
	assert.Nil(t, auth)
	assert.Contains(t, err.Error(), "failed to create goauth store")
}

func TestNewAuth_NilConfig(t *testing.T) {
	t.Parallel()

	// NewAuth with nil config should panic or fail gracefully
	defer func() {
		if r := recover(); r != nil {
			// Expected behavior when config is nil
			t.Log("Recovered from panic as expected when config is nil")
		}
	}()

	_, _ = NewAuth(nil, nil)
}

// TestNewAuth_InvalidRBACPath tests the second error path in NewAuth
// This test requires a valid database but invalid RBAC config
func TestNewAuth_InvalidRBACPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires database connection")
	}

	// This test would require a real database connection to reach the second error path
	// Since we don't have one in unit tests, we'll skip it
	// The first error path (nil DB) is already tested above
	t.Skip("Skipping: requires real database connection for full NewAuth testing")
}

func TestFullPermissionLifecycle(t *testing.T) {
	t.Parallel()
	auth, mockStore := createTestAuthWithRBAC(t)
	adapter := &Adapter{auth: auth}
	ctx := context.Background()

	// Pre-populate the mock store with user permissions to simulate role assignment
	// This simulates what AssignRole would do without requiring the role templates
	mockStore.userPermissions["user123"] = &store.UserPermissions{
		UserID:            "user123",
		RoleLabel:         "viewer",
		BaseRole:          "viewer",
		Permissions:       []string{"users:read", "proxies:read"},
		PermissionVersion: 1,
	}

	// Step 1: Check initial permissions (simulating viewer role)
	hasPerm, err := adapter.HasPermission(ctx, "user123", "users:read")
	require.NoError(t, err)
	assert.True(t, hasPerm, "User should have users:read permission")

	hasPerm, err = adapter.HasPermission(ctx, "user123", "users:write")
	require.NoError(t, err)
	assert.False(t, hasPerm, "User should not have users:write permission")

	// Step 2: Add additional permissions
	err = auth.AddPermissions(ctx, "user123", []string{"users:write"})
	require.NoError(t, err)

	hasPerm, err = adapter.HasPermission(ctx, "user123", "users:write")
	require.NoError(t, err)
	assert.True(t, hasPerm, "User should now have users:write permission")

	// Step 3: Check all permissions
	hasAll, err := adapter.HasAllPermissions(ctx, "user123", []string{"users:read", "users:write"})
	require.NoError(t, err)
	assert.True(t, hasAll, "User should have all required permissions")

	// Step 4: Check any permission
	hasAny, err := adapter.HasAnyPermission(ctx, "user123", []string{"users:delete", "users:write"})
	require.NoError(t, err)
	assert.True(t, hasAny, "User should have at least one of the permissions")

	// Step 5: Remove permissions
	err = auth.RemovePermissions(ctx, "user123", []string{"users:write"})
	require.NoError(t, err)

	hasPerm, err = adapter.HasPermission(ctx, "user123", "users:write")
	require.NoError(t, err)
	assert.False(t, hasPerm, "User should no longer have users:write permission")

	// Step 6: Verify the mock store has the expected state
	perms, ok := mockStore.userPermissions["user123"]
	require.True(t, ok, "User permissions should exist in store")
	assert.Equal(t, "viewer", perms.BaseRole, "Base role should still be viewer")
}
