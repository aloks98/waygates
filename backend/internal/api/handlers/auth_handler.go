package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/aloks98/homelab-proxy/backend/internal/service"
	"github.com/aloks98/homelab-proxy/backend/internal/utils"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
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
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}

	if req.Name == "" || req.Username == "" || req.Email == "" || req.Password == "" {
		utils.BadRequest(w, "Name, username, email, and password are required", nil)
		return
	}

	user, err := h.authService.RegisterUser(req.Name, req.Username, req.Email, req.Password)
	if err != nil {
		if err == service.ErrUserAlreadyExists {
			utils.Conflict(w, "User with this email or username already exists")
			return
		}
		utils.InternalError(w, "Failed to register user")
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

	accessToken, refreshToken, err := h.authService.LoginUser(req.Identifier, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			utils.Unauthorized(w, "Invalid credentials")
			return
		}
		utils.InternalError(w, "Failed to log in")
		return
	}

	utils.Success(w, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, "Login successful")
}
