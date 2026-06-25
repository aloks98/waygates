package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aloks98/goauth/store"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// RoleManager is the subset of goauth used to read/assign a user's role.
type RoleManager interface {
	AssignRole(ctx context.Context, userID, role string) error
	GetUserPermissions(ctx context.Context, userID string) (*store.UserPermissions, error)
}

// Sentinel errors for user service operations.
var (
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidRole      = errors.New("invalid role")
	ErrCannotModifySelf = errors.New("you cannot perform this action on your own account")
	ErrLastAdmin        = errors.New("cannot remove or demote the last administrator")
	ErrUsernameTaken    = errors.New("username already in use")
	ErrEmailTaken       = errors.New("email already in use")
)

var validRoles = map[string]bool{"admin": true, "operator": true, "viewer": true}

// UserWithRole embeds a User and its role label from goauth.
type UserWithRole struct {
	models.User
	Role string `json:"role"`
}

// CreateUserInput holds fields required to create a new user.
type CreateUserInput struct {
	Name, Username, Email, Role, Password string
	MustChangePassword                    bool
}

// UpdateUserInput holds mutable profile fields for updating a user.
type UpdateUserInput struct {
	Name, Email, Role string
	Active            bool
}

type userService struct {
	repo       repository.UserRepositoryInterface
	roles      RoleManager
	audit      AuditServiceInterface
	bcryptCost int
	logger     *zap.Logger
}

// NewUserService creates a new UserService.
func NewUserService(
	repo repository.UserRepositoryInterface,
	roles RoleManager,
	audit AuditServiceInterface,
	bcryptCost int,
	logger *zap.Logger,
) UserService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &userService{
		repo:       repo,
		roles:      roles,
		audit:      audit,
		bcryptCost: bcryptCost,
		logger:     logger.Named("user-service"),
	}
}

// roleOf returns the role label for a user, or "" on any error.
func (s *userService) roleOf(ctx context.Context, userID int) string {
	perms, err := s.roles.GetUserPermissions(ctx, strconv.Itoa(userID))
	if err != nil || perms == nil {
		return ""
	}
	return perms.RoleLabel
}

// withRoles attaches each user's role (one GetUserPermissions call per user).
func (s *userService) withRoles(ctx context.Context, users []models.User) []UserWithRole {
	out := make([]UserWithRole, 0, len(users))
	for i := range users {
		out = append(out, UserWithRole{User: users[i], Role: s.roleOf(ctx, users[i].ID)})
	}
	return out
}

// adminCount counts active admin users by iterating all users.
// Only users whose role is "admin" AND whose Active field is true are counted.
// Guards run before mutation, so a user being deleted/deactivated is still
// active at count time — counting active admins pre-mutation is the correct check.
func (s *userService) adminCount(ctx context.Context) (int, error) {
	users, err := s.repo.List()
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range users {
		if users[i].Active && s.roleOf(ctx, users[i].ID) == "admin" {
			n++
		}
	}
	return n, nil
}

// mapUniqueErr inspects a GORM/DB error for unique-constraint violations and
// returns the matching sentinel error (ErrUsernameTaken or ErrEmailTaken).
// PostgreSQL duplicate-key errors include the constraint name; e.g.:
//
//	ERROR: duplicate key value violates unique constraint "users_username_key"
//	ERROR: duplicate key value violates unique constraint "users_email_key"
//
// We also check for the column name substring in case the driver surfaces a
// different format (e.g. SQLite's "UNIQUE constraint failed: users.username").
func mapUniqueErr(err error) error {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
		if strings.Contains(msg, "username") {
			return ErrUsernameTaken
		}
		if strings.Contains(msg, "email") {
			return ErrEmailTaken
		}
	}
	return fmt.Errorf("database error: %w", err)
}

// loadAppUser fetches a user by ID and returns it together with its role.
// If the user is missing or has no role (i.e. is an ACL/roleless user), it
// returns ErrUserNotFound so that ACL-auth accounts are never reachable via
// the user-management API.
func (s *userService) loadAppUser(ctx context.Context, id int) (*models.User, string, error) {
	u, err := s.repo.GetByID(id)
	if err != nil || u == nil {
		return nil, "", ErrUserNotFound
	}
	role := s.roleOf(ctx, id)
	if role == "" {
		return nil, "", ErrUserNotFound
	}
	return u, role, nil
}

// List returns all app users (those with a role) with their roles attached.
// ACL-auth end-users that have no goauth role assigned are excluded.
func (s *userService) List(ctx context.Context) ([]UserWithRole, error) {
	users, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	all := s.withRoles(ctx, users)
	out := make([]UserWithRole, 0, len(all))
	for i := range all {
		if all[i].Role != "" {
			out = append(out, all[i])
		}
	}
	return out, nil
}

// Get returns a single app user with their role attached.
// Returns ErrUserNotFound for ACL/roleless users.
func (s *userService) Get(ctx context.Context, id int) (*UserWithRole, error) {
	u, role, err := s.loadAppUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return &UserWithRole{User: *u, Role: role}, nil
}

// Create creates a new user, assigns their role, and logs the audit event.
func (s *userService) Create(ctx context.Context, in CreateUserInput, actorID int, ip, ua string) (*UserWithRole, error) {
	if !validRoles[in.Role] {
		return nil, ErrInvalidRole
	}
	u := &models.User{
		Name:               in.Name,
		Username:           in.Username,
		Email:              in.Email,
		Active:             true,
		MustChangePassword: in.MustChangePassword,
	}
	if err := u.SetPassword(in.Password, s.bcryptCost); err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	if err := s.repo.Create(u); err != nil {
		return nil, mapUniqueErr(err)
	}
	if err := s.roles.AssignRole(ctx, strconv.Itoa(u.ID), in.Role); err != nil {
		_ = s.repo.Delete(u.ID) // best-effort rollback
		return nil, fmt.Errorf("assign role: %w", err)
	}
	s.logUser(ctx, models.AuditActionUserCreate, u, actorID, ip, ua)
	return &UserWithRole{User: *u, Role: in.Role}, nil
}

// Update applies profile changes to an existing user, enforcing guards
// against self-mutation and last-admin demotion/deactivation.
// Returns ErrUserNotFound for ACL/roleless users.
func (s *userService) Update(ctx context.Context, id int, in UpdateUserInput, actorID int, ip, ua string) (*UserWithRole, error) {
	if !validRoles[in.Role] {
		return nil, ErrInvalidRole
	}
	u, oldRole, err := s.loadAppUser(ctx, id)
	if err != nil {
		return nil, err
	}

	losingAdmin := oldRole == "admin" && in.Role != "admin"
	deactivating := u.Active && !in.Active

	// Last-admin guard: cannot demote or deactivate the last active admin.
	// Note: the deactivating branch only enters when oldRole == "admin" because
	// non-admin deactivation does not satisfy this condition and is allowed.
	if (losingAdmin || deactivating) && oldRole == "admin" {
		n, countErr := s.adminCount(ctx)
		if countErr != nil {
			return nil, countErr
		}
		if n <= 1 {
			return nil, ErrLastAdmin
		}
	}
	// Self guard: cannot demote or deactivate yourself.
	if id == actorID && (losingAdmin || deactivating) {
		return nil, ErrCannotModifySelf
	}

	// Capture pre-mutation active state for activation audit.
	wasActive := u.Active

	u.Name, u.Email, u.Active = in.Name, in.Email, in.Active
	if err := s.repo.Update(u); err != nil {
		return nil, mapUniqueErr(err)
	}

	if in.Role != oldRole {
		if err := s.roles.AssignRole(ctx, strconv.Itoa(id), in.Role); err != nil {
			return nil, fmt.Errorf("assign role: %w", err)
		}
		s.logUser(ctx, models.AuditActionUserRoleChange, u, actorID, ip, ua)
	}

	// Activation-state transition audit.
	if wasActive && !in.Active {
		s.logUser(ctx, models.AuditActionUserDeactivate, u, actorID, ip, ua)
	} else if !wasActive && in.Active {
		s.logUser(ctx, models.AuditActionUserActivate, u, actorID, ip, ua)
	}

	s.logUser(ctx, models.AuditActionUserUpdate, u, actorID, ip, ua)
	return &UserWithRole{User: *u, Role: in.Role}, nil
}

// ResetPassword sets a new password for the user and optionally forces a
// must-change on next login.
// Returns ErrUserNotFound for ACL/roleless users.
func (s *userService) ResetPassword(ctx context.Context, id int, password string, mustChange bool, actorID int, ip, ua string) error {
	u, _, err := s.loadAppUser(ctx, id)
	if err != nil {
		return err
	}
	if err := u.SetPassword(password, s.bcryptCost); err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.repo.UpdatePassword(id, u.PasswordHash); err != nil {
		return err
	}
	u.MustChangePassword = mustChange
	if err := s.repo.Update(u); err != nil {
		return err
	}
	s.logUser(ctx, models.AuditActionUserPasswordReset, u, actorID, ip, ua)
	return nil
}

// Delete removes a user, enforcing self and last-admin guards.
// Returns ErrUserNotFound for ACL/roleless users.
func (s *userService) Delete(ctx context.Context, id, actorID int, ip, ua string) error {
	if id == actorID {
		return ErrCannotModifySelf
	}
	u, role, err := s.loadAppUser(ctx, id)
	if err != nil {
		return err
	}
	if role == "admin" {
		n, countErr := s.adminCount(ctx)
		if countErr != nil {
			return countErr
		}
		if n <= 1 {
			return ErrLastAdmin
		}
	}
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	s.logUser(ctx, models.AuditActionUserDelete, u, actorID, ip, ua)
	return nil
}

// logUser fires a fire-and-forget audit event for a user action.
func (s *userService) logUser(ctx context.Context, action string, u *models.User, actorID int, ip, ua string) {
	aid := actorID
	uid := u.ID
	_ = s.audit.LogEvent(ctx, models.AuditEvent{
		UserID:       &aid,
		Action:       action,
		ResourceType: "user",
		ResourceID:   &uid,
		ResourceName: u.Username,
		IPAddress:    ip,
		UserAgent:    ua,
		Status:       "success",
	})
}
