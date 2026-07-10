package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// NOTE: do NOT define `ptr` here. Task 1 Step 8 already added it to
// proxy_service_test.go, and both files are package `service` — a second
// definition will not compile.

type MockProxyGroupRepository struct {
	ListAllFunc            func() ([]models.ProxyGroup, error)
	GetByIDFunc            func(id int) (*models.ProxyGroup, error)
	CreateFunc             func(g *models.ProxyGroup) error
	UpdateFunc             func(g *models.ProxyGroup) error
	DeleteFunc             func(id int) error
	MemberCountFunc        func(id int) (int64, error)
	UpdateBaseDomainTxFunc func(groupID int, newBase *string) error
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
func (m *MockProxyGroupRepository) List(repository.ProxyGroupListParams) ([]models.ProxyGroup, int64, error) {
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
func (m *MockProxyGroupRepository) Update(g *models.ProxyGroup) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(g)
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
func (m *MockProxyGroupRepository) ListMembers(int) ([]models.Proxy, error) { return nil, nil }
func (m *MockProxyGroupRepository) UpdateBaseDomainTx(groupID int, newBase *string) error {
	if m.UpdateBaseDomainTxFunc != nil {
		return m.UpdateBaseDomainTxFunc(groupID, newBase)
	}
	return nil
}
func (m *MockProxyGroupRepository) ListACLAssignments(int) ([]models.ProxyGroupACLAssignment, error) {
	return nil, nil
}
func (m *MockProxyGroupRepository) CreateACLAssignment(*models.ProxyGroupACLAssignment) error {
	return nil
}
func (m *MockProxyGroupRepository) UpdateACLAssignment(*models.ProxyGroupACLAssignment) error {
	return nil
}
func (m *MockProxyGroupRepository) DeleteACLAssignment(int, int) error { return nil }

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

// A rename goes through the transactional path, not a plain Update.
func TestProxyGroupService_BaseDomainChangeUsesTransaction(t *testing.T) {
	var txCalled bool
	repo := &MockProxyGroupRepository{
		GetByIDFunc: func(id int) (*models.ProxyGroup, error) {
			return &models.ProxyGroup{ID: id, Name: "internal", BaseDomain: ptr("group.acme.in")}, nil
		},
		UpdateBaseDomainTxFunc: func(int, *string) error { txCalled = true; return nil },
	}
	svc := newGroupService(repo, &MockGroupSyncer{})

	require.NoError(t, svc.UpdateGroup(&models.ProxyGroup{ID: 3, Name: "internal", BaseDomain: ptr("g2.acme.in")}))
	assert.True(t, txCalled, "base_domain change must go through UpdateBaseDomainTx")
}

// A failed rename must not rebuild the config.
func TestProxyGroupService_FailedRenameDoesNotRebuild(t *testing.T) {
	syncer := &MockGroupSyncer{}
	repo := &MockProxyGroupRepository{
		GetByIDFunc: func(id int) (*models.ProxyGroup, error) {
			return &models.ProxyGroup{ID: id, BaseDomain: ptr("group.acme.in")}, nil
		},
		UpdateBaseDomainTxFunc: func(int, *string) error { return errors.New("hostname conflict") },
	}
	svc := newGroupService(repo, syncer)

	require.Error(t, svc.UpdateGroup(&models.ProxyGroup{ID: 3, BaseDomain: ptr("g2.acme.in")}))
	assert.Zero(t, syncer.Calls, "a failed rename must leave the served config alone")
}
