package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// NOTE: do NOT define `ptr` here. Task 1 Step 8 already added it to
// proxy_service_test.go, and both files are package `service` — a second
// definition will not compile.

type MockProxyGroupRepository struct {
	ListAllFunc             func() ([]models.ProxyGroup, error)
	ListFunc                func(repository.ProxyGroupListParams) ([]models.ProxyGroup, int64, error)
	GetByIDFunc             func(id int) (*models.ProxyGroup, error)
	CreateFunc              func(g *models.ProxyGroup) error
	DeleteFunc              func(id int) error
	MemberCountFunc         func(id int) (int64, error)
	ListMembersFunc         func(id int) ([]models.Proxy, error)
	UpdateGroupTxFunc       func(g *models.ProxyGroup, baseDomainChanged bool) error
	ListACLAssignmentsFunc  func(groupID int) ([]models.ProxyGroupACLAssignment, error)
	CreateACLAssignmentFunc func(a *models.ProxyGroupACLAssignment) error
	UpdateACLAssignmentFunc func(a *models.ProxyGroupACLAssignment) error
	DeleteACLAssignmentFunc func(groupID, aclGroupID int) error
}

func (m *MockProxyGroupRepository) ListAll() ([]models.ProxyGroup, error) {
	if m.ListAllFunc != nil {
		return m.ListAllFunc()
	}
	return nil, nil
}
func (m *MockProxyGroupRepository) ListAllACLAssignments() ([]models.ProxyGroupACLAssignment, error) {
	return nil, nil
}
func (m *MockProxyGroupRepository) List(p repository.ProxyGroupListParams) ([]models.ProxyGroup, int64, error) {
	if m.ListFunc != nil {
		return m.ListFunc(p)
	}
	return nil, 0, nil
}
func (m *MockProxyGroupRepository) GetByID(id int) (*models.ProxyGroup, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return &models.ProxyGroup{ID: id}, nil
}
func (m *MockProxyGroupRepository) Create(g *models.ProxyGroup) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(g)
	}
	return nil
}
func (m *MockProxyGroupRepository) Delete(id int) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}
func (m *MockProxyGroupRepository) MemberCount(id int) (int64, error) {
	if m.MemberCountFunc != nil {
		return m.MemberCountFunc(id)
	}
	return 0, nil
}
func (m *MockProxyGroupRepository) ListMembers(id int) ([]models.Proxy, error) {
	if m.ListMembersFunc != nil {
		return m.ListMembersFunc(id)
	}
	return nil, nil
}
func (m *MockProxyGroupRepository) UpdateGroupTx(g *models.ProxyGroup, baseDomainChanged bool) error {
	if m.UpdateGroupTxFunc != nil {
		return m.UpdateGroupTxFunc(g, baseDomainChanged)
	}
	return nil
}
func (m *MockProxyGroupRepository) ListACLAssignments(groupID int) ([]models.ProxyGroupACLAssignment, error) {
	if m.ListACLAssignmentsFunc != nil {
		return m.ListACLAssignmentsFunc(groupID)
	}
	return nil, nil
}
func (m *MockProxyGroupRepository) CreateACLAssignment(a *models.ProxyGroupACLAssignment) error {
	if m.CreateACLAssignmentFunc != nil {
		return m.CreateACLAssignmentFunc(a)
	}
	return nil
}
func (m *MockProxyGroupRepository) UpdateACLAssignment(a *models.ProxyGroupACLAssignment) error {
	if m.UpdateACLAssignmentFunc != nil {
		return m.UpdateACLAssignmentFunc(a)
	}
	return nil
}
func (m *MockProxyGroupRepository) DeleteACLAssignment(groupID, aclGroupID int) error {
	if m.DeleteACLAssignmentFunc != nil {
		return m.DeleteACLAssignmentFunc(groupID, aclGroupID)
	}
	return nil
}

type MockGroupSyncer struct {
	RebuildAllFunc func() error
	Calls          int
}

func (m *MockGroupSyncer) RebuildAll() error {
	m.Calls++
	if m.RebuildAllFunc != nil {
		return m.RebuildAllFunc()
	}
	return nil
}

func newGroupService(repo repository.ProxyGroupRepositoryInterface, syncer GroupSyncer) *ProxyGroupService {
	return NewProxyGroupService(ProxyGroupServiceConfig{Repo: repo, Syncer: syncer})
}

func TestProxyGroupService_DeleteBlockedWhenGroupHasMembers(t *testing.T) {
	repo := &MockProxyGroupRepository{
		MemberCountFunc: func(int) (int64, error) { return 7, nil },
		DeleteFunc:      func(int) error { t.Fatal("Delete must not be called"); return nil },
	}
	svc := newGroupService(repo, &MockGroupSyncer{})

	err := svc.DeleteGroup(3)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGroupHasMembers)
	assert.Contains(t, err.Error(), "7")
}

func TestProxyGroupService_DeleteSucceedsWhenEmptyAndRebuilds(t *testing.T) {
	syncer := &MockGroupSyncer{}
	svc := newGroupService(&MockProxyGroupRepository{}, syncer)

	require.NoError(t, svc.DeleteGroup(3))
	assert.Equal(t, 1, syncer.Calls, "deleting a group must rebuild the whole config")
}

func TestProxyGroupService_UpdateRebuildsFullConfig(t *testing.T) {
	syncer := &MockGroupSyncer{}
	svc := newGroupService(&MockProxyGroupRepository{}, syncer)

	require.NoError(t, svc.UpdateGroup(&models.ProxyGroup{ID: 3, Name: "internal"}))
	assert.Equal(t, 1, syncer.Calls)
}

// A rename goes through the single transactional write path, with
// baseDomainChanged=true so the repository knows to re-home members.
func TestProxyGroupService_BaseDomainChangeUsesTransaction(t *testing.T) {
	var txCalled bool
	var gotChanged bool
	repo := &MockProxyGroupRepository{
		GetByIDFunc: func(id int) (*models.ProxyGroup, error) {
			return &models.ProxyGroup{ID: id, Name: "internal", BaseDomain: ptr("group.acme.in")}, nil
		},
		UpdateGroupTxFunc: func(g *models.ProxyGroup, baseDomainChanged bool) error {
			txCalled = true
			gotChanged = baseDomainChanged
			return nil
		},
	}
	svc := newGroupService(repo, &MockGroupSyncer{})

	require.NoError(t, svc.UpdateGroup(&models.ProxyGroup{ID: 3, Name: "internal", BaseDomain: ptr("g2.acme.in")}))
	assert.True(t, txCalled, "base_domain change must go through UpdateGroupTx")
	assert.True(t, gotChanged, "UpdateGroupTx must be told the base_domain changed, so it re-homes members")
}

// A failed update must not rebuild the config.
func TestProxyGroupService_FailedRenameDoesNotRebuild(t *testing.T) {
	syncer := &MockGroupSyncer{}
	repo := &MockProxyGroupRepository{
		GetByIDFunc: func(id int) (*models.ProxyGroup, error) {
			return &models.ProxyGroup{ID: id, BaseDomain: ptr("group.acme.in")}, nil
		},
		UpdateGroupTxFunc: func(*models.ProxyGroup, bool) error { return errors.New("hostname conflict") },
	}
	svc := newGroupService(repo, syncer)

	require.Error(t, svc.UpdateGroup(&models.ProxyGroup{ID: 3, BaseDomain: ptr("g2.acme.in")}))
	assert.Zero(t, syncer.Calls, "a failed update must leave the served config alone")
}

// =============================================================================
// Error classification: unique-violation -> typed sentinel error.
//
// service.ErrGroupNameConflict was declared but never produced before this
// task (Task 4's review flagged it). These tests exercise the classification
// this task adds: repository.IsUniqueViolation(err, <constraint>) matched
// against a *pgconn.PgError, not string-sniffing.
// =============================================================================

func uniqueViolation(constraint string) error {
	return &pgconn.PgError{Code: "23505", ConstraintName: constraint}
}

func TestProxyGroupService_CreateGroupNameConflict(t *testing.T) {
	repo := &MockProxyGroupRepository{
		CreateFunc: func(*models.ProxyGroup) error {
			// Wrapped, the way the repository's `.Error` from GORM naturally is —
			// classification must still find the *pgconn.PgError through the chain.
			return fmt.Errorf("insert: %w", uniqueViolation("uq_proxy_groups_name"))
		},
	}
	svc := newGroupService(repo, &MockGroupSyncer{})

	err := svc.CreateGroup(&models.ProxyGroup{Name: "taken"}, 1)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGroupNameConflict)
}

func TestProxyGroupService_CreateGroupOtherDBErrorIsNotNameConflict(t *testing.T) {
	repo := &MockProxyGroupRepository{
		CreateFunc: func(*models.ProxyGroup) error { return errors.New("connection reset") },
	}
	svc := newGroupService(repo, &MockGroupSyncer{})

	err := svc.CreateGroup(&models.ProxyGroup{Name: "internal"}, 1)

	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrGroupNameConflict), "a non-unique-violation error must not be misclassified as a name conflict")
}

func TestProxyGroupService_UpdateGroupNameConflict(t *testing.T) {
	repo := &MockProxyGroupRepository{
		GetByIDFunc: func(id int) (*models.ProxyGroup, error) { return &models.ProxyGroup{ID: id, Name: "internal"}, nil },
		UpdateGroupTxFunc: func(*models.ProxyGroup, bool) error {
			return fmt.Errorf("updating proxy group: %w", uniqueViolation("uq_proxy_groups_name"))
		},
	}
	syncer := &MockGroupSyncer{}
	svc := newGroupService(repo, syncer)

	err := svc.UpdateGroup(&models.ProxyGroup{ID: 3, Name: "taken"})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGroupNameConflict)
	assert.Zero(t, syncer.Calls, "a failed rename must not rebuild the config")
}

func TestProxyGroupService_UpdateGroupHostnameConflictNamesTheHostname(t *testing.T) {
	repo := &MockProxyGroupRepository{
		GetByIDFunc: func(id int) (*models.ProxyGroup, error) {
			return &models.ProxyGroup{ID: id, BaseDomain: ptr("group.acme.in")}, nil
		},
		UpdateGroupTxFunc: func(*models.ProxyGroup, bool) error {
			return &repository.HostnameConflictError{Hostname: "abc.g2.acme.in"}
		},
	}
	svc := newGroupService(repo, &MockGroupSyncer{})

	err := svc.UpdateGroup(&models.ProxyGroup{ID: 3, BaseDomain: ptr("g2.acme.in")})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrHostnameConflict)
	assert.Contains(t, err.Error(), "abc.g2.acme.in", "the 409 must name the colliding hostname")
}

// A zero-value ListGroups request (no page/limit query params) must not
// translate into a GORM `LIMIT 0` — before this fix, ListGroups had no
// clamping at all, so an omitted page/limit would silently return zero rows.
func TestProxyGroupService_ListGroupsDefaultsPageAndLimit(t *testing.T) {
	var gotParams repository.ProxyGroupListParams
	repo := &MockProxyGroupRepository{
		ListFunc: func(p repository.ProxyGroupListParams) ([]models.ProxyGroup, int64, error) {
			gotParams = p
			return []models.ProxyGroup{{ID: 1}}, 1, nil
		},
	}
	svc := newGroupService(repo, &MockGroupSyncer{})

	resp, err := svc.ListGroups(repository.ProxyGroupListParams{})

	require.NoError(t, err)
	assert.Equal(t, 1, gotParams.Page)
	assert.Equal(t, 20, gotParams.Limit)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.Limit)
	assert.Equal(t, int64(1), resp.Total)
	assert.Equal(t, 1, resp.TotalPages)
}

// =============================================================================
// ACL assignment (nested under a proxy group)
// =============================================================================

func TestProxyGroupService_ListACLAssignmentsGroupNotFound(t *testing.T) {
	repo := &MockProxyGroupRepository{
		GetByIDFunc: func(int) (*models.ProxyGroup, error) { return nil, errors.New("record not found") },
	}
	svc := newGroupService(repo, &MockGroupSyncer{})

	_, err := svc.ListACLAssignments(99)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGroupNotFound)
}

func TestProxyGroupService_AssignACLToGroupNotFound(t *testing.T) {
	repo := &MockProxyGroupRepository{
		GetByIDFunc: func(int) (*models.ProxyGroup, error) { return nil, errors.New("record not found") },
	}
	svc := newGroupService(repo, &MockGroupSyncer{})

	err := svc.AssignACLToGroup(99, 1, "/*", 0)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGroupNotFound)
}

func TestProxyGroupService_AssignACLToGroupSuccessRebuildsConfig(t *testing.T) {
	var created *models.ProxyGroupACLAssignment
	repo := &MockProxyGroupRepository{
		CreateACLAssignmentFunc: func(a *models.ProxyGroupACLAssignment) error {
			created = a
			return nil
		},
	}
	syncer := &MockGroupSyncer{}
	svc := newGroupService(repo, syncer)

	err := svc.AssignACLToGroup(3, 7, "", 5)

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, 3, created.ProxyGroupID)
	assert.Equal(t, 7, created.ACLGroupID)
	assert.Equal(t, "/*", created.PathPattern, "empty path pattern must default to /*")
	assert.True(t, created.Enabled, "a new assignment must start enabled")
	assert.Equal(t, 1, syncer.Calls, "assigning an ACL must rebuild the whole config")
}

func TestProxyGroupService_AssignACLToGroupDuplicateConflict(t *testing.T) {
	repo := &MockProxyGroupRepository{
		CreateACLAssignmentFunc: func(*models.ProxyGroupACLAssignment) error {
			return uniqueViolation("uq_pgaa_group_acl")
		},
	}
	syncer := &MockGroupSyncer{}
	svc := newGroupService(repo, syncer)

	err := svc.AssignACLToGroup(3, 7, "/*", 0)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGroupACLAssignmentExists)
	assert.Zero(t, syncer.Calls)
}

func TestProxyGroupService_AssignACLToGroupRollsBackOnSyncFailure(t *testing.T) {
	var deletedGroupID, deletedACLGroupID int
	repo := &MockProxyGroupRepository{
		DeleteACLAssignmentFunc: func(groupID, aclGroupID int) error {
			deletedGroupID, deletedACLGroupID = groupID, aclGroupID
			return nil
		},
	}
	syncer := &MockGroupSyncer{RebuildAllFunc: func() error { return errors.New("caddy reload failed") }}
	svc := newGroupService(repo, syncer)

	err := svc.AssignACLToGroup(3, 7, "/*", 0)

	require.Error(t, err)
	assert.Equal(t, 3, deletedGroupID, "a sync failure must roll back the row it just inserted")
	assert.Equal(t, 7, deletedACLGroupID)
}

func TestProxyGroupService_UpdateGroupACLAssignmentRejectsWrongGroup(t *testing.T) {
	repo := &MockProxyGroupRepository{
		ListACLAssignmentsFunc: func(int) ([]models.ProxyGroupACLAssignment, error) {
			// Group 3 owns assignment 5, not 99 — a caller must not be able to
			// update assignment 5 via group 3's route using a fabricated ID it
			// doesn't own.
			return []models.ProxyGroupACLAssignment{{ID: 5, ProxyGroupID: 3}}, nil
		},
		UpdateACLAssignmentFunc: func(*models.ProxyGroupACLAssignment) error {
			t.Fatal("UpdateACLAssignment must not be called for an assignment the group doesn't own")
			return nil
		},
	}
	svc := newGroupService(repo, &MockGroupSyncer{})

	err := svc.UpdateGroupACLAssignment(3, 99, "/*", 0, true)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGroupACLAssignmentNotFound)
}

func TestProxyGroupService_UpdateGroupACLAssignmentSuccessRebuildsConfig(t *testing.T) {
	repo := &MockProxyGroupRepository{
		ListACLAssignmentsFunc: func(int) ([]models.ProxyGroupACLAssignment, error) {
			return []models.ProxyGroupACLAssignment{{ID: 5, ProxyGroupID: 3}}, nil
		},
	}
	syncer := &MockGroupSyncer{}
	svc := newGroupService(repo, syncer)

	err := svc.UpdateGroupACLAssignment(3, 5, "/admin/*", 2, false)

	require.NoError(t, err)
	assert.Equal(t, 1, syncer.Calls)
}

func TestProxyGroupService_RemoveACLFromGroupRebuildsConfig(t *testing.T) {
	syncer := &MockGroupSyncer{}
	svc := newGroupService(&MockProxyGroupRepository{}, syncer)

	require.NoError(t, svc.RemoveACLFromGroup(3, 7))
	assert.Equal(t, 1, syncer.Calls)
}
