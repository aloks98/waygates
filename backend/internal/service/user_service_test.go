package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aloks98/goauth/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// ---- fakes / mocks ----

// fakeRM is a RoleManager test double that maps string userID → role label.
// It records the last AssignRole call for assertion.
type fakeRM struct {
	roles          map[string]string
	assignErr      error
	assignedUserID string
	assignedRole   string
}

func (f *fakeRM) AssignRole(_ context.Context, userID string, role string) error {
	f.assignedUserID = userID
	f.assignedRole = role
	return f.assignErr
}

func (f *fakeRM) GetUserPermissions(_ context.Context, userID string) (*store.UserPermissions, error) {
	role, ok := f.roles[userID]
	if !ok {
		return nil, nil
	}
	return &store.UserPermissions{UserID: userID, RoleLabel: role}, nil
}

// mockUserRepo is an inline mock for UserRepositoryInterface.
type mockUserRepo struct {
	createFn           func(user *models.User) error
	getByEmailFn       func(email string) (*models.User, error)
	getByUserOrEmailFn func(identifier string) (*models.User, error)
	getByIDFn          func(id int) (*models.User, error)
	countFn            func() (int64, error)
	deleteFn           func(id int) error
	updatePasswordFn   func(id int, passwordHash string) error
	listFn             func() ([]models.User, error)
	updateFn           func(user *models.User) error
	updateLastLoginFn  func(id int, t time.Time) error
}

func (m *mockUserRepo) Create(u *models.User) error {
	if m.createFn != nil {
		return m.createFn(u)
	}
	return nil
}
func (m *mockUserRepo) GetByEmail(email string) (*models.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(email)
	}
	return nil, nil
}
func (m *mockUserRepo) GetByUsernameOrEmail(id string) (*models.User, error) {
	if m.getByUserOrEmailFn != nil {
		return m.getByUserOrEmailFn(id)
	}
	return nil, nil
}
func (m *mockUserRepo) GetByID(id int) (*models.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return nil, nil
}
func (m *mockUserRepo) Count() (int64, error) {
	if m.countFn != nil {
		return m.countFn()
	}
	return 0, nil
}
func (m *mockUserRepo) Delete(id int) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}
func (m *mockUserRepo) UpdatePassword(id int, passwordHash string) error {
	if m.updatePasswordFn != nil {
		return m.updatePasswordFn(id, passwordHash)
	}
	return nil
}
func (m *mockUserRepo) List() ([]models.User, error) {
	if m.listFn != nil {
		return m.listFn()
	}
	return []models.User{}, nil
}
func (m *mockUserRepo) Update(u *models.User) error {
	if m.updateFn != nil {
		return m.updateFn(u)
	}
	return nil
}
func (m *mockUserRepo) UpdateLastLogin(id int, t time.Time) error {
	if m.updateLastLoginFn != nil {
		return m.updateLastLoginFn(id, t)
	}
	return nil
}

// Ensure mockUserRepo satisfies the interface at compile time.
var _ repository.UserRepositoryInterface = (*mockUserRepo)(nil)

// mockAuditSvc is a minimal AuditServiceInterface test double that records
// LogEvent calls. All other methods are no-ops.
type mockAuditSvc struct {
	logEventFn func(ctx context.Context, e models.AuditEvent) error
	actions    []string
}

func (m *mockAuditSvc) LogEvent(ctx context.Context, e models.AuditEvent) error {
	m.actions = append(m.actions, e.Action)
	if m.logEventFn != nil {
		return m.logEventFn(ctx, e)
	}
	return nil
}

func (m *mockAuditSvc) GetConfig() (*models.AuditConfig, error) {
	return models.DefaultAuditConfig(), nil
}
func (m *mockAuditSvc) SetConfig(_ *models.AuditConfig) error { return nil }
func (m *mockAuditSvc) InvalidateConfigCache()                {}
func (m *mockAuditSvc) ListAuditLogs(_ repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
	return &models.AuditLogListResponse{}, nil
}
func (m *mockAuditSvc) GetAuditLogByID(_ int) (*models.AuditLog, error) { return nil, nil }
func (m *mockAuditSvc) GetStats() (*models.AuditLogStats, error)        { return &models.AuditLogStats{}, nil }
func (m *mockAuditSvc) LogProxyCreate(_ context.Context, _ int, _ *models.Proxy, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogProxyUpdate(_ context.Context, _ int, _ *models.Proxy, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogProxyDelete(_ context.Context, _ int, _ int, _, _, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogProxyEnable(_ context.Context, _ int, _ *models.Proxy, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogProxyDisable(_ context.Context, _ int, _ *models.Proxy, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogLogin(_ context.Context, _ int, _, _, _ string) error    { return nil }
func (m *mockAuditSvc) LogLoginFailed(_ context.Context, _, _, _, _ string) error  { return nil }
func (m *mockAuditSvc) LogLogout(_ context.Context, _ int, _, _, _ string) error   { return nil }
func (m *mockAuditSvc) LogRegister(_ context.Context, _ int, _, _, _ string) error { return nil }
func (m *mockAuditSvc) LogPasswordChange(_ context.Context, _ int, _, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogSettingsUpdate(_ context.Context, _ int, _, _, _, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogSyncStarted(_ context.Context) error                   { return nil }
func (m *mockAuditSvc) LogSyncCompleted(_ context.Context, _ int) error          { return nil }
func (m *mockAuditSvc) LogSyncFailed(_ context.Context, _ string) error          { return nil }
func (m *mockAuditSvc) LogSystemStartup(_ context.Context) error                 { return nil }
func (m *mockAuditSvc) LogCaddyReload(_ context.Context, _ bool, _ string) error { return nil }
func (m *mockAuditSvc) LogACLGroupCreate(_ context.Context, _ int, _ *models.ACLGroup, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLGroupUpdate(_ context.Context, _ int, _ *models.ACLGroup, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLGroupDelete(_ context.Context, _ int, _ int, _, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLIPRuleAdd(_ context.Context, _ int, _ int, _ string, _ *models.ACLIPRule, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLIPRuleUpdate(_ context.Context, _ int, _ *models.ACLIPRule, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLIPRuleDelete(_ context.Context, _ int, _ int, _, _, _, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLBasicAuthAdd(_ context.Context, _ int, _ int, _, _, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLBasicAuthUpdate(_ context.Context, _ int, _ int, _, _, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLBasicAuthDelete(_ context.Context, _ int, _ int, _, _, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLWaygatesAuthUpdate(_ context.Context, _ int, _ int, _ string, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLAssignmentCreate(_ context.Context, _ int, _ int, _ string, _ int, _, _, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLAssignmentUpdate(_ context.Context, _ int, _ *models.ProxyACLAssignment, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLAssignmentDelete(_ context.Context, _ int, _ int, _ string, _ int, _, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLBrandingUpdate(_ context.Context, _ int, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLSessionRevoke(_ context.Context, _ int, _ int, _, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLOAuthRestrictionSet(_ context.Context, _ int, _ int, _, _ string, _ *models.ACLOAuthProviderRestriction, _ bool, _ []string, _ []string, _, _ string) error {
	return nil
}
func (m *mockAuditSvc) LogACLOAuthRestrictionDelete(_ context.Context, _ int, _ int, _, _, _, _ string) error {
	return nil
}

// Ensure mockAuditSvc satisfies the interface at compile time.
var _ AuditServiceInterface = (*mockAuditSvc)(nil)

// ---- helpers ----

func newTestSvc(repo repository.UserRepositoryInterface, rm RoleManager, audit AuditServiceInterface) UserService {
	return NewUserService(repo, rm, audit, 4, zap.NewNop())
}

func makeUser(id int, username string, active bool) models.User {
	return models.User{
		ID:       id,
		Username: username,
		Name:     username,
		Email:    username + "@example.com",
		Active:   active,
	}
}

// ---- Create ----

func TestUserService_Create_AssignsRoleAndAudits(t *testing.T) {
	repo := &mockUserRepo{
		createFn: func(u *models.User) error {
			u.ID = 10
			return nil
		},
	}
	rm := &fakeRM{roles: map[string]string{}}
	audit := &mockAuditSvc{}

	svc := newTestSvc(repo, rm, audit)
	in := CreateUserInput{
		Name:     "Alice",
		Username: "alice",
		Email:    "alice@example.com",
		Role:     "operator",
		Password: "secret123",
	}

	result, err := svc.Create(context.Background(), in, 1, "127.0.0.1", "test-agent")
	require.NoError(t, err)
	assert.Equal(t, "operator", result.Role)
	assert.Equal(t, "alice", result.Username)
	assert.Contains(t, audit.actions, models.AuditActionUserCreate)
	// Verify AssignRole was called with the new user's ID and the requested role.
	assert.Equal(t, "10", rm.assignedUserID, "AssignRole must be called with the new user's ID")
	assert.Equal(t, "operator", rm.assignedRole, "AssignRole must be called with the requested role")
}

func TestUserService_Create_InvalidRole(t *testing.T) {
	repo := &mockUserRepo{}
	rm := &fakeRM{roles: map[string]string{}}
	audit := &mockAuditSvc{}

	svc := newTestSvc(repo, rm, audit)
	in := CreateUserInput{Role: "superuser", Password: "x"}
	_, err := svc.Create(context.Background(), in, 1, "", "")
	assert.ErrorIs(t, err, ErrInvalidRole)
}

// ---- Delete ----

func TestUserService_Delete_Self_ReturnsErrCannotModifySelf(t *testing.T) {
	repo := &mockUserRepo{}
	rm := &fakeRM{roles: map[string]string{}}
	audit := &mockAuditSvc{}

	svc := newTestSvc(repo, rm, audit)
	err := svc.Delete(context.Background(), 5, 5, "127.0.0.1", "agent")
	assert.ErrorIs(t, err, ErrCannotModifySelf)
}

func TestUserService_Delete_LastAdmin_ReturnsErrLastAdmin(t *testing.T) {
	target := makeUser(2, "bob", true)
	repo := &mockUserRepo{
		getByIDFn: func(id int) (*models.User, error) {
			if id == 2 {
				cp := target
				return &cp, nil
			}
			return nil, errors.New("not found")
		},
		listFn: func() ([]models.User, error) {
			return []models.User{target}, nil
		},
	}
	rm := &fakeRM{roles: map[string]string{"2": "admin"}}
	audit := &mockAuditSvc{}

	svc := newTestSvc(repo, rm, audit)
	err := svc.Delete(context.Background(), 2, 1, "127.0.0.1", "agent")
	assert.ErrorIs(t, err, ErrLastAdmin)
}

func TestUserService_Delete_NonLastAdmin_Succeeds(t *testing.T) {
	user2 := makeUser(2, "bob", true)
	user3 := makeUser(3, "carol", true)
	var deletedID int
	repo := &mockUserRepo{
		getByIDFn: func(id int) (*models.User, error) {
			if id == 2 {
				cp := user2
				return &cp, nil
			}
			return nil, errors.New("not found")
		},
		listFn: func() ([]models.User, error) {
			return []models.User{user2, user3}, nil
		},
		deleteFn: func(id int) error {
			deletedID = id
			return nil
		},
	}
	rm := &fakeRM{roles: map[string]string{"2": "admin", "3": "admin"}}
	audit := &mockAuditSvc{}

	svc := newTestSvc(repo, rm, audit)
	err := svc.Delete(context.Background(), 2, 1, "127.0.0.1", "agent")
	require.NoError(t, err)
	assert.Equal(t, 2, deletedID)
	assert.Contains(t, audit.actions, models.AuditActionUserDelete)
}

// ---- Update ----

func TestUserService_Update_DemoteLastAdmin_ReturnsErrLastAdmin(t *testing.T) {
	admin := makeUser(2, "bob", true)
	repo := &mockUserRepo{
		getByIDFn: func(id int) (*models.User, error) {
			if id == 2 {
				cp := admin
				return &cp, nil
			}
			return nil, errors.New("not found")
		},
		listFn: func() ([]models.User, error) {
			return []models.User{admin}, nil
		},
	}
	rm := &fakeRM{roles: map[string]string{"2": "admin"}}
	audit := &mockAuditSvc{}

	svc := newTestSvc(repo, rm, audit)
	in := UpdateUserInput{Name: "Bob", Email: "bob@example.com", Role: "viewer", Active: true}
	_, err := svc.Update(context.Background(), 2, in, 1, "127.0.0.1", "agent")
	assert.ErrorIs(t, err, ErrLastAdmin)
}

func TestUserService_Update_DeactivateSelf_ReturnsErrCannotModifySelf(t *testing.T) {
	actor := makeUser(1, "alice", true)
	other := makeUser(2, "bob", true)
	repo := &mockUserRepo{
		getByIDFn: func(id int) (*models.User, error) {
			if id == 1 {
				cp := actor
				return &cp, nil
			}
			return nil, errors.New("not found")
		},
		listFn: func() ([]models.User, error) {
			// Two admins — last-admin check passes; self-guard fires next.
			return []models.User{actor, other}, nil
		},
	}
	rm := &fakeRM{roles: map[string]string{"1": "admin", "2": "admin"}}
	audit := &mockAuditSvc{}

	svc := newTestSvc(repo, rm, audit)
	in := UpdateUserInput{Name: "Alice", Email: "alice@example.com", Role: "admin", Active: false}
	_, err := svc.Update(context.Background(), 1, in, 1, "127.0.0.1", "agent")
	assert.ErrorIs(t, err, ErrCannotModifySelf)
}

// ---- ResetPassword ----

func TestUserService_ResetPassword_SetsMustChangePassword(t *testing.T) {
	user := makeUser(3, "carol", true)
	var updatedUser *models.User
	repo := &mockUserRepo{
		getByIDFn: func(id int) (*models.User, error) {
			if id == 3 {
				cp := user
				return &cp, nil
			}
			return nil, errors.New("not found")
		},
		updatePasswordFn: func(_ int, _ string) error { return nil },
		updateFn: func(u *models.User) error {
			updatedUser = u
			return nil
		},
	}
	rm := &fakeRM{roles: map[string]string{"3": "viewer"}}
	audit := &mockAuditSvc{}

	svc := newTestSvc(repo, rm, audit)
	err := svc.ResetPassword(context.Background(), 3, "newpassword123!", true, 1, "127.0.0.1", "agent")
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.True(t, updatedUser.MustChangePassword)
	assert.Contains(t, audit.actions, models.AuditActionUserPasswordReset)
}

// ---- Update happy path ----

func TestUserService_Update_HappyPath_TwoAdmins(t *testing.T) {
	adminA := makeUser(2, "bob", true)
	adminB := makeUser(3, "carol", true)
	var updatedUser *models.User
	repo := &mockUserRepo{
		getByIDFn: func(id int) (*models.User, error) {
			if id == 2 {
				cp := adminA
				return &cp, nil
			}
			return nil, errors.New("not found")
		},
		listFn: func() ([]models.User, error) {
			return []models.User{adminA, adminB}, nil
		},
		updateFn: func(u *models.User) error {
			updatedUser = u
			return nil
		},
	}
	rm := &fakeRM{roles: map[string]string{"2": "admin", "3": "admin"}}
	audit := &mockAuditSvc{}

	svc := newTestSvc(repo, rm, audit)
	// Role and Active unchanged — just editing name/email.
	in := UpdateUserInput{Name: "Bobby", Email: "bobby@example.com", Role: "admin", Active: true}
	result, err := svc.Update(context.Background(), 2, in, 1, "127.0.0.1", "agent")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, updatedUser, "repo.Update must be called")
	assert.Equal(t, "Bobby", updatedUser.Name)
	assert.Equal(t, "bobby@example.com", updatedUser.Email)
	assert.Contains(t, audit.actions, models.AuditActionUserUpdate)
}

// ---- ACL/roleless user exclusion ----

// TestUserService_List_ExcludesRolelessUsers verifies that users with no goauth
// role (ACL-auth accounts) are filtered out of the List result.
func TestUserService_List_ExcludesRolelessUsers(t *testing.T) {
	appUser := makeUser(1, "admin-user", true)
	aclUser := makeUser(2, "acl-oauth-user", true)

	repo := &mockUserRepo{
		listFn: func() ([]models.User, error) {
			return []models.User{appUser, aclUser}, nil
		},
	}
	// appUser has role "admin"; aclUser has no role entry (returns "").
	rm := &fakeRM{roles: map[string]string{"1": "admin"}}
	audit := &mockAuditSvc{}

	svc := newTestSvc(repo, rm, audit)
	result, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, result, 1, "roleless ACL user must be excluded")
	assert.Equal(t, "admin-user", result[0].Username)
}

// TestUserService_Get_RolelessUser_ReturnsErrUserNotFound verifies that
// attempting to Get an ACL/roleless user returns ErrUserNotFound.
func TestUserService_Get_RolelessUser_ReturnsErrUserNotFound(t *testing.T) {
	aclUser := makeUser(5, "acl-user", true)
	repo := &mockUserRepo{
		getByIDFn: func(id int) (*models.User, error) {
			if id == 5 {
				cp := aclUser
				return &cp, nil
			}
			return nil, errors.New("not found")
		},
	}
	rm := &fakeRM{roles: map[string]string{}} // no role for user 5
	audit := &mockAuditSvc{}

	svc := newTestSvc(repo, rm, audit)
	_, err := svc.Get(context.Background(), 5)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

// TestUserService_Update_RolelessUser_ReturnsErrUserNotFound verifies that
// updating an ACL/roleless user is rejected with ErrUserNotFound.
func TestUserService_Update_RolelessUser_ReturnsErrUserNotFound(t *testing.T) {
	aclUser := makeUser(6, "acl-user", true)
	repo := &mockUserRepo{
		getByIDFn: func(id int) (*models.User, error) {
			if id == 6 {
				cp := aclUser
				return &cp, nil
			}
			return nil, errors.New("not found")
		},
	}
	rm := &fakeRM{roles: map[string]string{}} // no role for user 6
	audit := &mockAuditSvc{}

	svc := newTestSvc(repo, rm, audit)
	in := UpdateUserInput{Name: "ACL User", Email: "acl@example.com", Role: "viewer", Active: true}
	_, err := svc.Update(context.Background(), 6, in, 1, "127.0.0.1", "agent")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

// TestUserService_Delete_RolelessUser_ReturnsErrUserNotFound verifies that
// deleting an ACL/roleless user is rejected with ErrUserNotFound.
func TestUserService_Delete_RolelessUser_ReturnsErrUserNotFound(t *testing.T) {
	aclUser := makeUser(7, "acl-user", true)
	repo := &mockUserRepo{
		getByIDFn: func(id int) (*models.User, error) {
			if id == 7 {
				cp := aclUser
				return &cp, nil
			}
			return nil, errors.New("not found")
		},
	}
	rm := &fakeRM{roles: map[string]string{}} // no role for user 7
	audit := &mockAuditSvc{}

	svc := newTestSvc(repo, rm, audit)
	err := svc.Delete(context.Background(), 7, 1, "127.0.0.1", "agent")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

// TestUserService_ResetPassword_RolelessUser_ReturnsErrUserNotFound verifies
// that resetting the password of an ACL/roleless user is rejected.
func TestUserService_ResetPassword_RolelessUser_ReturnsErrUserNotFound(t *testing.T) {
	aclUser := makeUser(8, "acl-user", true)
	repo := &mockUserRepo{
		getByIDFn: func(id int) (*models.User, error) {
			if id == 8 {
				cp := aclUser
				return &cp, nil
			}
			return nil, errors.New("not found")
		},
	}
	rm := &fakeRM{roles: map[string]string{}} // no role for user 8
	audit := &mockAuditSvc{}

	svc := newTestSvc(repo, rm, audit)
	err := svc.ResetPassword(context.Background(), 8, "newpass123!", false, 1, "127.0.0.1", "agent")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

// ---- Active-admin-count regression (Fix 1) ----

// TestUserService_Delete_ActiveAdminCountFix verifies that a deactivated admin
// does NOT count toward the active-admin total. With one active admin (A) and
// one inactive admin (B), deleting A must return ErrLastAdmin.
func TestUserService_Delete_ActiveAdminCountFix(t *testing.T) {
	activeAdmin := makeUser(2, "active-admin", true)
	inactiveAdmin := makeUser(3, "inactive-admin", false)

	repo := &mockUserRepo{
		getByIDFn: func(id int) (*models.User, error) {
			if id == 2 {
				cp := activeAdmin
				return &cp, nil
			}
			return nil, errors.New("not found")
		},
		listFn: func() ([]models.User, error) {
			// Two admins in the role store, but only one is active.
			return []models.User{activeAdmin, inactiveAdmin}, nil
		},
	}
	rm := &fakeRM{roles: map[string]string{"2": "admin", "3": "admin"}}
	audit := &mockAuditSvc{}

	svc := newTestSvc(repo, rm, audit)
	// actorID != 2 so self-guard does not fire.
	err := svc.Delete(context.Background(), 2, 1, "127.0.0.1", "agent")
	assert.ErrorIs(t, err, ErrLastAdmin,
		"deleting the only active admin must be blocked even when a deactivated admin exists")
}
