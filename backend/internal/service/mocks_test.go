package service

import (
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// MockProxyGroupRepository is a mock implementation of
// repository.ProxyGroupRepositoryInterface, shared between
// proxy_group_service_test.go and proxy_service_test.go (both package
// `service` — Go allows only one definition per package).
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
