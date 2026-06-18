package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/auth"
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
	"github.com/aloks98/waygates/backend/internal/validation"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	auth         AuthProvider
	userRepo     repository.UserRepositoryInterface
	auditService service.AuditServiceInterface
	bcryptCost   int
	logger       *zap.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authInstance AuthProvider, userRepo repository.UserRepositoryInterface, auditService service.AuditServiceInterface, bcryptCost int, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		auth:         authInstance,
		userRepo:     userRepo,
		auditService: auditService,
		bcryptCost:   bcryptCost,
		logger:       logger,
	}
}

// RegisterRequest is the request body for user registration
type RegisterRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req validation.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}

	// Validate request using validator library
	if err := validation.ValidateStruct(&req); err != nil {
		utils.BadRequest(w, err.Error(), nil)
		return
	}

	// Check if user already exists
	_, err := h.userRepo.GetByUsernameOrEmail(req.Email)
	if err == nil {
		utils.Conflict(w, "User with this email already exists")
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		if h.logger != nil {
			h.logger.Error("Failed to check existing user by email",
				zap.String("email", req.Email),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to check existing user")
		return
	}

	_, err = h.userRepo.GetByUsernameOrEmail(req.Username)
	if err == nil {
		utils.Conflict(w, "User with this username already exists")
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		if h.logger != nil {
			h.logger.Error("Failed to check existing user by username",
				zap.String("username", req.Username),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to check existing user")
		return
	}

	// Create new user
	user := &models.User{
		Name:     req.Name,
		Username: req.Username,
		Email:    req.Email,
	}

	// Hash password
	if err := user.SetPassword(req.Password, h.bcryptCost); err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to hash password during registration",
				zap.String("username", req.Username),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to process password")
		return
	}

	// Create the user and assign its role. This is serialized so the
	// "first user becomes admin" decision is race-free (see the method).
	if err := h.createUserAndAssignRole(r.Context(), user); err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to register user",
				zap.String("username", req.Username),
				zap.String("email", req.Email),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to register user")
		return
	}

	// Log audit event
	if h.auditService != nil {
		_ = h.auditService.LogRegister(r.Context(), user.ID, user.Username, getClientIP(r), r.UserAgent())
	}

	utils.Created(w, user, "User registered successfully")
}

// firstUserRegistrationMu serializes user creation + role assignment during
// registration so the "first user becomes admin" decision (which depends on the
// user count) is not subject to a race between concurrent registrations.
// Waygates runs as a single backend instance, so a process-level mutex suffices.
var firstUserRegistrationMu sync.Mutex

// createUserAndAssignRole persists the user and assigns its role atomically. The
// first registered user (count == 1) becomes admin; subsequent users become
// operator. On role-assignment failure the user creation is rolled back.
func (h *AuthHandler) createUserAndAssignRole(ctx context.Context, user *models.User) error {
	firstUserRegistrationMu.Lock()
	defer firstUserRegistrationMu.Unlock()

	if err := h.userRepo.Create(user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	count, _ := h.userRepo.Count()
	role := "operator"
	if count == 1 {
		role = "admin"
	}

	if err := h.auth.AssignRole(ctx, fmt.Sprintf("%d", user.ID), role); err != nil {
		if delErr := h.userRepo.Delete(user.ID); delErr != nil {
			return fmt.Errorf("assign role failed and rollback failed: %w", errors.Join(err, delErr))
		}
		return fmt.Errorf("assign role: %w", err)
	}
	return nil
}

// LoginRequest is the request body for user login
type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// LoginResponse is the response body for successful login
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}

	if req.Identifier == "" || req.Password == "" {
		utils.BadRequest(w, "Identifier and password are required", nil)
		return
	}

	// Get user by username or email
	user, err := h.userRepo.GetByUsernameOrEmail(req.Identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Log failed login attempt
			if h.auditService != nil {
				_ = h.auditService.LogLoginFailed(ctx, req.Identifier, getClientIP(r), r.UserAgent(), "User not found")
			}
			if h.logger != nil {
				h.logger.Info("Login failed: user not found",
					zap.String("identifier", req.Identifier),
					zap.String("client_ip", getClientIP(r)))
			}
			utils.Unauthorized(w, "Invalid credentials")
			return
		}
		if h.logger != nil {
			h.logger.Error("Failed to get user during login",
				zap.String("identifier", req.Identifier),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to authenticate")
		return
	}

	// Check password
	if !user.CheckPassword(req.Password) {
		// Log failed login attempt
		if h.auditService != nil {
			_ = h.auditService.LogLoginFailed(ctx, req.Identifier, getClientIP(r), r.UserAgent(), "Invalid password")
		}
		if h.logger != nil {
			h.logger.Info("Login failed: invalid password",
				zap.String("identifier", req.Identifier),
				zap.String("client_ip", getClientIP(r)))
		}
		utils.Unauthorized(w, "Invalid credentials")
		return
	}

	// Generate tokens using goauth
	userIDStr := fmt.Sprintf("%d", user.ID)

	tokenPair, err := h.auth.GenerateTokenPair(ctx, userIDStr, map[string]any{
		"username": user.Username,
		"email":    user.Email,
	})
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to generate token pair",
				zap.Int("user_id", user.ID),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to generate tokens")
		return
	}

	// Log successful login
	if h.auditService != nil {
		_ = h.auditService.LogLogin(ctx, user.ID, user.Username, getClientIP(r), r.UserAgent())
	}

	utils.Success(w, LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, "Login successful")
}

// RefreshTokenRequest is the request body for token refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse is the response body for successful token refresh
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}

	if req.RefreshToken == "" {
		utils.BadRequest(w, "Refresh token is required", nil)
		return
	}

	// Refresh tokens using goauth (includes rotation)
	ctx := r.Context()
	tokenPair, err := h.auth.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		if h.logger != nil {
			h.logger.Info("Token refresh failed",
				zap.Error(err))
		}
		utils.Unauthorized(w, "Invalid or expired refresh token")
		return
	}

	utils.Success(w, RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, "Token refreshed successfully")
}

// LogoutRequest is the optional request body for logout
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

// Logout handles user logout by revoking access and refresh tokens
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get access token from Authorization header
	accessToken := r.Header.Get("Authorization")
	if len(accessToken) > 7 && accessToken[:7] == "Bearer " {
		accessToken = accessToken[7:]
	}

	if accessToken == "" {
		utils.BadRequest(w, "No token provided", nil)
		return
	}

	ctx := r.Context()

	// Get user info for audit log before logout
	userID, _ := auth.GetUserIDAsUint(ctx)

	// Revoke access token (ignore errors - token might already be revoked or expired)
	_ = h.auth.RevokeAccessToken(ctx, accessToken)

	// Try to parse request body for refresh token (optional)
	var req LogoutRequest
	if r.Body != nil {
		// Ignore decode errors - refresh token is optional
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	// Revoke refresh token if provided (ignore errors - token might already be revoked or expired)
	if req.RefreshToken != "" {
		_ = h.auth.RevokeRefreshToken(ctx, req.RefreshToken)
	}

	// Log audit event
	if h.auditService != nil && userID > 0 {
		// nolint:gosec // G115: userID from JWT claims will never exceed int max in practice
		if user, err := h.userRepo.GetByID(int(userID)); err == nil {
			_ = h.auditService.LogLogout(ctx, int(userID), user.Username, getClientIP(r), r.UserAgent())
		}
	}

	utils.Success(w, nil, "Logged out successfully")
}

// ChangePasswordRequest is the request body for changing password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword handles password change for authenticated users
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := auth.GetUserIDAsUint(ctx)
	if err != nil {
		utils.Unauthorized(w, "Not authenticated")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		utils.BadRequest(w, "Current password and new password are required", nil)
		return
	}

	if len(req.NewPassword) < 8 {
		utils.BadRequest(w, "New password must be at least 8 characters", nil)
		return
	}

	// Get the current user
	// nolint:gosec // G115: userID from JWT claims will never exceed int max in practice
	user, err := h.userRepo.GetByID(int(userID))
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to get user for password change",
				zap.Uint("user_id", userID),
				zap.Error(err))
		}
		utils.NotFound(w, "User not found")
		return
	}

	// Verify current password
	if !user.CheckPassword(req.CurrentPassword) {
		utils.Unauthorized(w, "Current password is incorrect")
		return
	}

	// Set new password
	if err := user.SetPassword(req.NewPassword, h.bcryptCost); err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to hash new password",
				zap.Uint("user_id", userID),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to process new password")
		return
	}

	// Update user in database - we need to update the password hash
	// nolint:gosec // G115: userID from JWT claims will never exceed int max in practice
	if err := h.userRepo.UpdatePassword(int(userID), user.PasswordHash); err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to update password in database",
				zap.Uint("user_id", userID),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to update password")
		return
	}

	// Log audit event
	if h.auditService != nil {
		// nolint:gosec // G115: userID from JWT claims will never exceed int max in practice
		_ = h.auditService.LogPasswordChange(ctx, int(userID), user.Username, getClientIP(r), r.UserAgent())
	}

	utils.Success(w, nil, "Password changed successfully")
}

// GetMe returns the current user's information
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := auth.GetUserIDAsUint(ctx)
	if err != nil {
		utils.Unauthorized(w, "Not authenticated")
		return
	}

	// nolint:gosec // G115: userID from JWT claims will never exceed int max in practice
	user, err := h.userRepo.GetByID(int(userID))
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to get user for GetMe",
				zap.Uint("user_id", userID),
				zap.Error(err))
		}
		utils.NotFound(w, "User not found")
		return
	}

	// Get user permissions
	userIDStr := fmt.Sprintf("%d", userID)
	perms, permErr := h.auth.GetUserPermissions(ctx, userIDStr)

	response := map[string]any{
		"id":       user.ID,
		"name":     user.Name,
		"username": user.Username,
		"email":    user.Email,
	}

	switch {
	case permErr != nil:
		// Permissions could not be retrieved - user might not have a role assigned
		response["role"] = nil
		response["permissions"] = []string{}
	case perms != nil:
		response["role"] = perms.RoleLabel
		response["permissions"] = perms.Permissions
	default:
		// No permissions found but no error - explicitly indicate no role
		response["role"] = nil
		response["permissions"] = []string{}
	}

	utils.Success(w, response, "")
}
