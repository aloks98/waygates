package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aloks98/goauth/middleware"
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
