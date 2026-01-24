package auth

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/aloks98/goauth"
	"github.com/aloks98/goauth/middleware"
	sqlstore "github.com/aloks98/goauth/store/sql"

	"github.com/aloks98/waygates/backend/internal/config"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// CustomClaims extends StandardClaims with application-specific fields
type CustomClaims struct {
	goauth.StandardClaims
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
}

// Auth wraps goauth.Auth with application-specific functionality
type Auth struct {
	*goauth.Auth[*CustomClaims]
	adapter *Adapter
}

// Adapter implements the middleware interfaces for goauth
type Adapter struct {
	auth *goauth.Auth[*CustomClaims]
}

// SetAuth sets the goauth instance on the adapter
func (a *Adapter) SetAuth(auth *goauth.Auth[*CustomClaims]) {
	a.auth = auth
}

// NewAuth creates a new Auth instance with goauth configured
func NewAuth(cfg *config.Config, db *sql.DB) (*Auth, error) {
	// Create SQL store with existing database connection (PostgreSQL)
	store, err := sqlstore.New(&sqlstore.Config{
		Dialect:     sqlstore.PostgreSQL,
		DB:          db,
		TablePrefix: "goauth_",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create goauth store: %w", err)
	}

	// Create goauth instance
	auth, err := goauth.New[*CustomClaims](
		goauth.WithSecret(cfg.JWT.Secret),
		goauth.WithStore(store),
		goauth.WithAccessTokenTTL(cfg.JWT.AccessExpiry),
		goauth.WithRefreshTokenTTL(cfg.JWT.RefreshExpiry),
		goauth.WithRBACFromFile(cfg.Security.RBACPath),
		goauth.WithAutoMigrate(true),
		goauth.WithRoleSyncOnStartup(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create goauth instance: %w", err)
	}

	return &Auth{
		Auth:    auth,
		adapter: &Adapter{auth: auth},
	}, nil
}

// Adapter returns the middleware adapter
func (a *Auth) Adapter() *Adapter {
	return a.adapter
}

// ValidateAccessToken implements middleware.TokenValidator
func (a *Adapter) ValidateAccessToken(ctx context.Context, tokenString string) (interface{}, error) {
	return a.auth.ValidateAccessToken(ctx, tokenString)
}

// ExtractUserID implements middleware.ClaimsExtractor
func (a *Adapter) ExtractUserID(claims interface{}) string {
	if c, ok := claims.(*CustomClaims); ok {
		return c.UserID
	}
	// Also handle the token.Claims type from goauth
	if c, ok := claims.(interface{ GetUserID() string }); ok {
		return c.GetUserID()
	}
	return ""
}

// ExtractPermissions implements middleware.ClaimsExtractor
func (a *Adapter) ExtractPermissions(_ interface{}) []string {
	// Permissions are checked via RBAC, not stored in token
	return nil
}

// HasPermission implements middleware.PermissionChecker
func (a *Adapter) HasPermission(ctx context.Context, userID, permission string) (bool, error) {
	return a.auth.HasPermission(ctx, userID, permission)
}

// HasAllPermissions implements middleware.PermissionChecker
func (a *Adapter) HasAllPermissions(ctx context.Context, userID string, permissions []string) (bool, error) {
	return a.auth.HasAllPermissions(ctx, userID, permissions)
}

// HasAnyPermission implements middleware.PermissionChecker
func (a *Adapter) HasAnyPermission(ctx context.Context, userID string, permissions []string) (bool, error) {
	return a.auth.HasAnyPermission(ctx, userID, permissions)
}

// ValidateAPIKey implements middleware.APIKeyValidator
func (a *Adapter) ValidateAPIKey(ctx context.Context, rawKey string) (*middleware.APIKeyInfo, error) {
	result, err := a.auth.ValidateAPIKey(ctx, rawKey)
	if err != nil {
		return nil, err
	}
	return &middleware.APIKeyInfo{
		ID:     result.Key.ID,
		UserID: result.UserID,
		Scopes: result.Key.Scopes,
	}, nil
}

// ErrorHandler returns a custom error handler that uses our response utilities
func ErrorHandler() middleware.ErrorHandler {
	return func(w http.ResponseWriter, _ *http.Request, err error) {
		code := middleware.ErrorToHTTPStatus(err)
		switch code {
		case http.StatusUnauthorized:
			utils.Unauthorized(w, err.Error())
		case http.StatusForbidden:
			// Return 404 instead of 403 to avoid leaking information about protected resources
			utils.NotFound(w, "Not found")
		default:
			utils.Unauthorized(w, "Authentication failed")
		}
	}
}

// GetUserIDFromContext retrieves the user ID from the request context
func GetUserIDFromContext(ctx context.Context) string {
	return middleware.GetUserID(ctx)
}

// GetUserIDAsUint retrieves the user ID as uint from the request context
func GetUserIDAsUint(ctx context.Context) (uint, error) {
	userIDStr := middleware.GetUserID(ctx)
	if userIDStr == "" {
		return 0, fmt.Errorf("user ID not found in context")
	}
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID format: %w", err)
	}
	return uint(userID), nil
}

// Ensure Adapter implements the required interfaces
var (
	_ middleware.TokenValidator    = (*Adapter)(nil)
	_ middleware.ClaimsExtractor   = (*Adapter)(nil)
	_ middleware.PermissionChecker = (*Adapter)(nil)
	_ middleware.APIKeyValidator   = (*Adapter)(nil)
)
