package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	chimw "github.com/aloks98/goauth/middleware/chi"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// UsersHandler handles user management HTTP requests.
type UsersHandler struct {
	svc    service.UserService
	logger *zap.Logger
}

// NewUsersHandler creates a new UsersHandler.
func NewUsersHandler(svc service.UserService, logger *zap.Logger) *UsersHandler {
	return &UsersHandler{svc: svc, logger: logger}
}

// mapUserError translates service sentinel errors to the appropriate HTTP response.
// Returns true if the error was handled (caller should return immediately).
func mapUserError(w http.ResponseWriter, err error, logger *zap.Logger) bool {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		utils.NotFound(w, err.Error())
	case errors.Is(err, service.ErrInvalidRole),
		errors.Is(err, service.ErrUsernameTaken),
		errors.Is(err, service.ErrEmailTaken),
		errors.Is(err, service.ErrCannotModifySelf),
		errors.Is(err, service.ErrLastAdmin):
		utils.BadRequest(w, err.Error(), nil)
	default:
		if logger != nil {
			logger.Error("user service error", zap.Error(err))
		}
		utils.InternalError(w, "An unexpected error occurred")
	}
	return true
}

// validRoles is the set of accepted role values at the HTTP layer.
var validRolesSet = map[string]bool{"admin": true, "operator": true, "viewer": true}

// List returns all users.
func (h *UsersHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.List(r.Context())
	if err != nil {
		mapUserError(w, err, h.logger)
		return
	}
	if users == nil {
		users = []service.UserWithRole{}
	}
	utils.Success(w, users, "Users retrieved successfully")
}

// Get returns a single user by ID.
func (h *UsersHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUserID(w, r)
	if !ok {
		return
	}
	user, err := h.svc.Get(r.Context(), id)
	if err != nil {
		mapUserError(w, err, h.logger)
		return
	}
	utils.Success(w, user, "User retrieved successfully")
}

// createUserRequest is the request body for user creation.
type createUserRequest struct {
	Name               string `json:"name"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	Role               string `json:"role"`
	Password           string `json:"password"`
	MustChangePassword bool   `json:"must_change_password"`
}

// Create creates a new user.
func (h *UsersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}

	// Required field validation.
	if strings.TrimSpace(req.Name) == "" {
		utils.BadRequest(w, "name is required", nil)
		return
	}
	if strings.TrimSpace(req.Username) == "" {
		utils.BadRequest(w, "username is required", nil)
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		utils.BadRequest(w, "email is required", nil)
		return
	}
	if req.Password == "" {
		utils.BadRequest(w, "password is required", nil)
		return
	}
	if len(req.Password) < 8 || len(req.Password) > 128 {
		utils.BadRequest(w, "password must be at least 8 characters", nil)
		return
	}
	if !validRolesSet[req.Role] {
		utils.BadRequest(w, service.ErrInvalidRole.Error(), nil)
		return
	}

	actorID, _ := strconv.Atoi(chimw.UserID(r))
	in := service.CreateUserInput{
		Name:               strings.TrimSpace(req.Name),
		Username:           strings.TrimSpace(req.Username),
		Email:              strings.TrimSpace(req.Email),
		Role:               req.Role,
		Password:           req.Password,
		MustChangePassword: req.MustChangePassword,
	}
	user, err := h.svc.Create(r.Context(), in, actorID, getClientIP(r), r.UserAgent())
	if err != nil {
		mapUserError(w, err, h.logger)
		return
	}
	utils.Created(w, user, "User created successfully")
}

// updateUserRequest is the request body for user updates.
type updateUserRequest struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Active bool   `json:"active"`
}

// Update applies profile changes to an existing user.
func (h *UsersHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUserID(w, r)
	if !ok {
		return
	}
	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		utils.BadRequest(w, "name is required", nil)
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		utils.BadRequest(w, "email is required", nil)
		return
	}
	if !validRolesSet[req.Role] {
		utils.BadRequest(w, service.ErrInvalidRole.Error(), nil)
		return
	}

	actorID, _ := strconv.Atoi(chimw.UserID(r))
	in := service.UpdateUserInput{
		Name:   strings.TrimSpace(req.Name),
		Email:  strings.TrimSpace(req.Email),
		Role:   req.Role,
		Active: req.Active,
	}
	user, err := h.svc.Update(r.Context(), id, in, actorID, getClientIP(r), r.UserAgent())
	if err != nil {
		mapUserError(w, err, h.logger)
		return
	}
	utils.Success(w, user, "User updated successfully")
}

// resetPasswordRequest is the request body for password reset.
type resetPasswordRequest struct {
	Password           string `json:"password"`
	MustChangePassword bool   `json:"must_change_password"`
}

// ResetPassword sets a new password for a user.
func (h *UsersHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUserID(w, r)
	if !ok {
		return
	}
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}

	if req.Password == "" {
		utils.BadRequest(w, "password is required", nil)
		return
	}
	if len(req.Password) < 8 || len(req.Password) > 128 {
		utils.BadRequest(w, "password must be at least 8 characters", nil)
		return
	}

	actorID, _ := strconv.Atoi(chimw.UserID(r))
	err := h.svc.ResetPassword(r.Context(), id, req.Password, req.MustChangePassword, actorID, getClientIP(r), r.UserAgent())
	if err != nil {
		mapUserError(w, err, h.logger)
		return
	}
	utils.Success(w, nil, "Password reset successfully")
}

// Delete removes a user.
func (h *UsersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUserID(w, r)
	if !ok {
		return
	}
	actorID, _ := strconv.Atoi(chimw.UserID(r))
	err := h.svc.Delete(r.Context(), id, actorID, getClientIP(r), r.UserAgent())
	if err != nil {
		mapUserError(w, err, h.logger)
		return
	}
	utils.Success(w, nil, "User deleted successfully")
}

// parseUserID extracts and parses the {id} URL parameter.
// It writes a 400 response and returns false on failure.
func parseUserID(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		utils.BadRequest(w, "invalid user id", nil)
		return 0, false
	}
	return id, true
}
