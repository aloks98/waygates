package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/auth"
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/utils"
	"github.com/aloks98/waygates/backend/internal/validation"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	auth       AuthProvider
	userRepo   repository.UserRepositoryInterface
	bcryptCost int
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authInstance AuthProvider, userRepo repository.UserRepositoryInterface, bcryptCost int) *AuthHandler {
	return &AuthHandler{
		auth:       authInstance,
		userRepo:   userRepo,
		bcryptCost: bcryptCost,
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
		utils.InternalError(w, "Failed to check existing user")
		return
	}

	_, err = h.userRepo.GetByUsernameOrEmail(req.Username)
	if err == nil {
		utils.Conflict(w, "User with this username already exists")
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
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
		utils.InternalError(w, "Failed to process password")
		return
	}

	// Save user to database
	if err := h.userRepo.Create(user); err != nil {
		utils.InternalError(w, "Failed to create user")
		return
	}

	// Assign default role (admin for first user, operator for others)
	ctx := r.Context()
	userIDStr := fmt.Sprintf("%d", user.ID)

	count, _ := h.userRepo.Count()
	role := "operator"
	if count == 1 {
		role = "admin" // First user gets admin role
	}

	if err := h.auth.AssignRole(ctx, userIDStr, role); err != nil {
		// Rollback user creation if role assignment fails
		// This is critical - especially for the first user who needs admin role
		if delErr := h.userRepo.Delete(user.ID); delErr != nil {
			utils.InternalError(w, "Failed to assign role and rollback failed")
			return
		}
		utils.InternalError(w, "Failed to assign user role")
		return
	}

	utils.Created(w, user, "User registered successfully")
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
			utils.Unauthorized(w, "Invalid credentials")
			return
		}
		utils.InternalError(w, "Failed to authenticate")
		return
	}

	// Check password
	if !user.CheckPassword(req.Password) {
		utils.Unauthorized(w, "Invalid credentials")
		return
	}

	// Generate tokens using goauth
	ctx := r.Context()
	userIDStr := fmt.Sprintf("%d", user.ID)

	tokenPair, err := h.auth.GenerateTokenPair(ctx, userIDStr, map[string]any{
		"username": user.Username,
		"email":    user.Email,
	})
	if err != nil {
		utils.InternalError(w, "Failed to generate tokens")
		return
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
	user, err := h.userRepo.GetByID(int(userID))
	if err != nil {
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
		utils.InternalError(w, "Failed to process new password")
		return
	}

	// Update user in database - we need to update the password hash
	if err := h.userRepo.UpdatePassword(int(userID), user.PasswordHash); err != nil {
		utils.InternalError(w, "Failed to update password")
		return
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

	user, err := h.userRepo.GetByID(int(userID))
	if err != nil {
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
