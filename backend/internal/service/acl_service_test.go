package service

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// =============================================================================
// Mock ACL Repository
// =============================================================================

// MockACLRepository is a mock implementation of ACLRepositoryInterface
type MockACLRepository struct {
	// Group methods
	CreateGroupFunc    func(group *models.ACLGroup) error
	GetGroupByIDFunc   func(id int) (*models.ACLGroup, error)
	GetGroupByNameFunc func(name string) (*models.ACLGroup, error)
	ListGroupsFunc     func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error)
	UpdateGroupFunc    func(group *models.ACLGroup) error
	DeleteGroupFunc    func(id int) error

	// IP Rule methods
	CreateIPRuleFunc  func(rule *models.ACLIPRule) error
	GetIPRuleByIDFunc func(id int) (*models.ACLIPRule, error)
	ListIPRulesFunc   func(groupID int) ([]models.ACLIPRule, error)
	UpdateIPRuleFunc  func(rule *models.ACLIPRule) error
	DeleteIPRuleFunc  func(id int) error

	// Basic Auth methods
	CreateBasicAuthUserFunc  func(user *models.ACLBasicAuthUser) error
	GetBasicAuthUserByIDFunc func(id int) (*models.ACLBasicAuthUser, error)
	GetBasicAuthUserFunc     func(groupID int, username string) (*models.ACLBasicAuthUser, error)
	ListBasicAuthUsersFunc   func(groupID int) ([]models.ACLBasicAuthUser, error)
	UpdateBasicAuthUserFunc  func(user *models.ACLBasicAuthUser) error
	DeleteBasicAuthUserFunc  func(id int) error

	// External Provider methods
	CreateExternalProviderFunc  func(provider *models.ACLExternalProvider) error
	GetExternalProviderByIDFunc func(id int) (*models.ACLExternalProvider, error)
	ListExternalProvidersFunc   func(groupID int) ([]models.ACLExternalProvider, error)
	UpdateExternalProviderFunc  func(provider *models.ACLExternalProvider) error
	DeleteExternalProviderFunc  func(id int) error

	// Waygates Auth methods
	GetWaygatesAuthFunc    func(groupID int) (*models.ACLWaygatesAuth, error)
	CreateWaygatesAuthFunc func(auth *models.ACLWaygatesAuth) error
	UpdateWaygatesAuthFunc func(auth *models.ACLWaygatesAuth) error
	DeleteWaygatesAuthFunc func(groupID int) error

	// Proxy ACL Assignment methods
	CreateProxyACLAssignmentFunc                func(assignment *models.ProxyACLAssignment) error
	GetProxyACLAssignmentByIDFunc               func(id int) (*models.ProxyACLAssignment, error)
	GetProxyACLAssignmentsFunc                  func(proxyID int) ([]models.ProxyACLAssignment, error)
	GetProxyACLAssignmentsByGroupFunc           func(groupID int) ([]models.ProxyACLAssignment, error)
	UpdateProxyACLAssignmentFunc                func(assignment *models.ProxyACLAssignment) error
	DeleteProxyACLAssignmentFunc                func(id int) error
	DeleteProxyACLAssignmentByProxyAndGroupFunc func(proxyID, groupID int) error

	// Branding methods
	GetBrandingFunc    func() (*models.ACLBranding, error)
	UpdateBrandingFunc func(branding *models.ACLBranding) error

	// Session methods
	CreateSessionFunc         func(session *models.ACLSession) error
	GetSessionByTokenFunc     func(token string) (*models.ACLSession, error)
	DeleteSessionFunc         func(token string) error
	DeleteExpiredSessionsFunc func() (int64, error)
	DeleteUserSessionsFunc    func(userID int) error
	DeleteProxySessionsFunc   func(proxyID int) error
}

// ACL Group methods
func (m *MockACLRepository) CreateGroup(group *models.ACLGroup) error {
	if m.CreateGroupFunc != nil {
		return m.CreateGroupFunc(group)
	}
	return nil
}

func (m *MockACLRepository) GetGroupByID(id int) (*models.ACLGroup, error) {
	if m.GetGroupByIDFunc != nil {
		return m.GetGroupByIDFunc(id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockACLRepository) GetGroupByName(name string) (*models.ACLGroup, error) {
	if m.GetGroupByNameFunc != nil {
		return m.GetGroupByNameFunc(name)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockACLRepository) ListGroups(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
	if m.ListGroupsFunc != nil {
		return m.ListGroupsFunc(params)
	}
	return []models.ACLGroup{}, 0, nil
}

func (m *MockACLRepository) UpdateGroup(group *models.ACLGroup) error {
	if m.UpdateGroupFunc != nil {
		return m.UpdateGroupFunc(group)
	}
	return nil
}

func (m *MockACLRepository) DeleteGroup(id int) error {
	if m.DeleteGroupFunc != nil {
		return m.DeleteGroupFunc(id)
	}
	return nil
}

// IP Rule methods
func (m *MockACLRepository) CreateIPRule(rule *models.ACLIPRule) error {
	if m.CreateIPRuleFunc != nil {
		return m.CreateIPRuleFunc(rule)
	}
	return nil
}

func (m *MockACLRepository) GetIPRuleByID(id int) (*models.ACLIPRule, error) {
	if m.GetIPRuleByIDFunc != nil {
		return m.GetIPRuleByIDFunc(id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockACLRepository) ListIPRules(groupID int) ([]models.ACLIPRule, error) {
	if m.ListIPRulesFunc != nil {
		return m.ListIPRulesFunc(groupID)
	}
	return []models.ACLIPRule{}, nil
}

func (m *MockACLRepository) UpdateIPRule(rule *models.ACLIPRule) error {
	if m.UpdateIPRuleFunc != nil {
		return m.UpdateIPRuleFunc(rule)
	}
	return nil
}

func (m *MockACLRepository) DeleteIPRule(id int) error {
	if m.DeleteIPRuleFunc != nil {
		return m.DeleteIPRuleFunc(id)
	}
	return nil
}

// Basic Auth methods
func (m *MockACLRepository) CreateBasicAuthUser(user *models.ACLBasicAuthUser) error {
	if m.CreateBasicAuthUserFunc != nil {
		return m.CreateBasicAuthUserFunc(user)
	}
	return nil
}

func (m *MockACLRepository) GetBasicAuthUserByID(id int) (*models.ACLBasicAuthUser, error) {
	if m.GetBasicAuthUserByIDFunc != nil {
		return m.GetBasicAuthUserByIDFunc(id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockACLRepository) GetBasicAuthUser(groupID int, username string) (*models.ACLBasicAuthUser, error) {
	if m.GetBasicAuthUserFunc != nil {
		return m.GetBasicAuthUserFunc(groupID, username)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockACLRepository) ListBasicAuthUsers(groupID int) ([]models.ACLBasicAuthUser, error) {
	if m.ListBasicAuthUsersFunc != nil {
		return m.ListBasicAuthUsersFunc(groupID)
	}
	return []models.ACLBasicAuthUser{}, nil
}

func (m *MockACLRepository) UpdateBasicAuthUser(user *models.ACLBasicAuthUser) error {
	if m.UpdateBasicAuthUserFunc != nil {
		return m.UpdateBasicAuthUserFunc(user)
	}
	return nil
}

func (m *MockACLRepository) DeleteBasicAuthUser(id int) error {
	if m.DeleteBasicAuthUserFunc != nil {
		return m.DeleteBasicAuthUserFunc(id)
	}
	return nil
}

// External Provider methods
func (m *MockACLRepository) CreateExternalProvider(provider *models.ACLExternalProvider) error {
	if m.CreateExternalProviderFunc != nil {
		return m.CreateExternalProviderFunc(provider)
	}
	return nil
}

func (m *MockACLRepository) GetExternalProviderByID(id int) (*models.ACLExternalProvider, error) {
	if m.GetExternalProviderByIDFunc != nil {
		return m.GetExternalProviderByIDFunc(id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockACLRepository) ListExternalProviders(groupID int) ([]models.ACLExternalProvider, error) {
	if m.ListExternalProvidersFunc != nil {
		return m.ListExternalProvidersFunc(groupID)
	}
	return []models.ACLExternalProvider{}, nil
}

func (m *MockACLRepository) UpdateExternalProvider(provider *models.ACLExternalProvider) error {
	if m.UpdateExternalProviderFunc != nil {
		return m.UpdateExternalProviderFunc(provider)
	}
	return nil
}

func (m *MockACLRepository) DeleteExternalProvider(id int) error {
	if m.DeleteExternalProviderFunc != nil {
		return m.DeleteExternalProviderFunc(id)
	}
	return nil
}

// Waygates Auth methods
func (m *MockACLRepository) GetWaygatesAuth(groupID int) (*models.ACLWaygatesAuth, error) {
	if m.GetWaygatesAuthFunc != nil {
		return m.GetWaygatesAuthFunc(groupID)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockACLRepository) CreateWaygatesAuth(auth *models.ACLWaygatesAuth) error {
	if m.CreateWaygatesAuthFunc != nil {
		return m.CreateWaygatesAuthFunc(auth)
	}
	return nil
}

func (m *MockACLRepository) UpdateWaygatesAuth(auth *models.ACLWaygatesAuth) error {
	if m.UpdateWaygatesAuthFunc != nil {
		return m.UpdateWaygatesAuthFunc(auth)
	}
	return nil
}

func (m *MockACLRepository) DeleteWaygatesAuth(groupID int) error {
	if m.DeleteWaygatesAuthFunc != nil {
		return m.DeleteWaygatesAuthFunc(groupID)
	}
	return nil
}

// Proxy ACL Assignment methods
func (m *MockACLRepository) CreateProxyACLAssignment(assignment *models.ProxyACLAssignment) error {
	if m.CreateProxyACLAssignmentFunc != nil {
		return m.CreateProxyACLAssignmentFunc(assignment)
	}
	return nil
}

func (m *MockACLRepository) GetProxyACLAssignmentByID(id int) (*models.ProxyACLAssignment, error) {
	if m.GetProxyACLAssignmentByIDFunc != nil {
		return m.GetProxyACLAssignmentByIDFunc(id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockACLRepository) GetProxyACLAssignments(proxyID int) ([]models.ProxyACLAssignment, error) {
	if m.GetProxyACLAssignmentsFunc != nil {
		return m.GetProxyACLAssignmentsFunc(proxyID)
	}
	return []models.ProxyACLAssignment{}, nil
}

func (m *MockACLRepository) GetProxyACLAssignmentsByGroup(groupID int) ([]models.ProxyACLAssignment, error) {
	if m.GetProxyACLAssignmentsByGroupFunc != nil {
		return m.GetProxyACLAssignmentsByGroupFunc(groupID)
	}
	return []models.ProxyACLAssignment{}, nil
}

func (m *MockACLRepository) UpdateProxyACLAssignment(assignment *models.ProxyACLAssignment) error {
	if m.UpdateProxyACLAssignmentFunc != nil {
		return m.UpdateProxyACLAssignmentFunc(assignment)
	}
	return nil
}

func (m *MockACLRepository) DeleteProxyACLAssignment(id int) error {
	if m.DeleteProxyACLAssignmentFunc != nil {
		return m.DeleteProxyACLAssignmentFunc(id)
	}
	return nil
}

func (m *MockACLRepository) DeleteProxyACLAssignmentByProxyAndGroup(proxyID, groupID int) error {
	if m.DeleteProxyACLAssignmentByProxyAndGroupFunc != nil {
		return m.DeleteProxyACLAssignmentByProxyAndGroupFunc(proxyID, groupID)
	}
	return nil
}

// Branding methods
func (m *MockACLRepository) GetBranding() (*models.ACLBranding, error) {
	if m.GetBrandingFunc != nil {
		return m.GetBrandingFunc()
	}
	return &models.ACLBranding{}, nil
}

func (m *MockACLRepository) UpdateBranding(branding *models.ACLBranding) error {
	if m.UpdateBrandingFunc != nil {
		return m.UpdateBrandingFunc(branding)
	}
	return nil
}

// Session methods
func (m *MockACLRepository) CreateSession(session *models.ACLSession) error {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(session)
	}
	return nil
}

func (m *MockACLRepository) GetSessionByToken(token string) (*models.ACLSession, error) {
	if m.GetSessionByTokenFunc != nil {
		return m.GetSessionByTokenFunc(token)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockACLRepository) DeleteSession(token string) error {
	if m.DeleteSessionFunc != nil {
		return m.DeleteSessionFunc(token)
	}
	return nil
}

func (m *MockACLRepository) DeleteExpiredSessions() (int64, error) {
	if m.DeleteExpiredSessionsFunc != nil {
		return m.DeleteExpiredSessionsFunc()
	}
	return 0, nil
}

func (m *MockACLRepository) DeleteUserSessions(userID int) error {
	if m.DeleteUserSessionsFunc != nil {
		return m.DeleteUserSessionsFunc(userID)
	}
	return nil
}

func (m *MockACLRepository) DeleteProxySessions(proxyID int) error {
	if m.DeleteProxySessionsFunc != nil {
		return m.DeleteProxySessionsFunc(proxyID)
	}
	return nil
}

// Ensure mock implements interface
var _ repository.ACLRepositoryInterface = (*MockACLRepository)(nil)

// =============================================================================
// Service Tests
// =============================================================================

func TestNewACLService(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{}
	proxyRepo := &MockProxyRepository{}

	svc := NewACLService(ACLServiceConfig{
		ACLRepo:   aclRepo,
		ProxyRepo: proxyRepo,
		Logger:    nil, // Should use nop logger
	})

	if svc == nil {
		t.Fatal("Expected non-nil service")
	}
	if svc.aclRepo != aclRepo {
		t.Error("Expected aclRepo to be set")
	}
	if svc.proxyRepo != proxyRepo {
		t.Error("Expected proxyRepo to be set")
	}
}

// =============================================================================
// Group Management Tests
// =============================================================================

func TestCreateGroup_Success(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByNameFunc: func(name string) (*models.ACLGroup, error) {
			return nil, gorm.ErrRecordNotFound
		},
		CreateGroupFunc: func(group *models.ACLGroup) error {
			group.ID = 1
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	group := &models.ACLGroup{
		Name:            "Test Group",
		CombinationMode: models.ACLCombinationModeAny,
	}

	err := svc.CreateGroup(group, 1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if group.ID != 1 {
		t.Error("Expected ID to be set")
	}
	if group.CreatedBy != 1 {
		t.Error("Expected CreatedBy to be set")
	}
}

func TestCreateGroup_ValidationError(t *testing.T) {
	t.Parallel()

	svc := NewACLService(ACLServiceConfig{ACLRepo: &MockACLRepository{}})

	group := &models.ACLGroup{
		Name:            "", // Empty name
		CombinationMode: models.ACLCombinationModeAny,
	}

	err := svc.CreateGroup(group, 1)
	if err == nil {
		t.Error("Expected validation error")
	}
}

func TestCreateGroup_NameExists(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByNameFunc: func(name string) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: 1, Name: name}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	group := &models.ACLGroup{
		Name:            "Existing Group",
		CombinationMode: models.ACLCombinationModeAny,
	}

	err := svc.CreateGroup(group, 1)
	if !errors.Is(err, ErrACLGroupNameExists) {
		t.Errorf("Expected ErrACLGroupNameExists, got: %v", err)
	}
}

func TestCreateGroup_DefaultCombinationMode(t *testing.T) {
	t.Parallel()

	// Note: The service currently validates BEFORE setting defaults.
	// This means an empty CombinationMode will fail validation.
	// Test documents this actual behavior.

	aclRepo := &MockACLRepository{}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	group := &models.ACLGroup{
		Name:            "Test Group",
		CombinationMode: "", // Empty - validation will fail
	}

	// Validation happens before default is set, so this fails
	err := svc.CreateGroup(group, 1)
	if err == nil {
		t.Error("Expected validation error for empty combination mode")
	}
}

func TestCreateGroup_WithExplicitCombinationMode(t *testing.T) {
	t.Parallel()

	var createdGroup *models.ACLGroup

	aclRepo := &MockACLRepository{
		GetGroupByNameFunc: func(name string) (*models.ACLGroup, error) {
			return nil, gorm.ErrRecordNotFound
		},
		CreateGroupFunc: func(group *models.ACLGroup) error {
			createdGroup = group
			group.ID = 1
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	group := &models.ACLGroup{
		Name:            "Test Group",
		CombinationMode: models.ACLCombinationModeAll, // Explicit mode
	}

	err := svc.CreateGroup(group, 1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if createdGroup.CombinationMode != models.ACLCombinationModeAll {
		t.Errorf("Expected combination mode 'all', got: %s", createdGroup.CombinationMode)
	}
}

func TestGetGroup_Success(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id, Name: "Test Group"}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	group, err := svc.GetGroup(1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if group.ID != 1 {
		t.Errorf("Expected ID 1, got: %d", group.ID)
	}
}

func TestGetGroup_NotFound(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	_, err := svc.GetGroup(999)
	if !errors.Is(err, ErrACLGroupNotFound) {
		t.Errorf("Expected ErrACLGroupNotFound, got: %v", err)
	}
}

func TestListGroups_Success(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		ListGroupsFunc: func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
			return []models.ACLGroup{
				{ID: 1, Name: "Group 1"},
				{ID: 2, Name: "Group 2"},
			}, 2, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	response, err := svc.ListGroups(ListACLGroupsRequest{Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(response.Items) != 2 {
		t.Errorf("Expected 2 items, got: %d", len(response.Items))
	}
	if response.Total != 2 {
		t.Errorf("Expected total 2, got: %d", response.Total)
	}
}

func TestListGroups_PaginationDefaults(t *testing.T) {
	t.Parallel()

	var capturedParams repository.ACLGroupListParams

	aclRepo := &MockACLRepository{
		ListGroupsFunc: func(params repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
			capturedParams = params
			return []models.ACLGroup{}, 0, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	// Test with invalid pagination
	_, err := svc.ListGroups(ListACLGroupsRequest{Page: 0, Limit: 0})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if capturedParams.Page != 1 {
		t.Errorf("Expected page 1, got: %d", capturedParams.Page)
	}
	if capturedParams.Limit != 20 {
		t.Errorf("Expected limit 20, got: %d", capturedParams.Limit)
	}
}

func TestUpdateGroup_Success(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id, Name: "Old Name", CombinationMode: models.ACLCombinationModeAny}, nil
		},
		GetGroupByNameFunc: func(name string) (*models.ACLGroup, error) {
			return nil, gorm.ErrRecordNotFound
		},
		UpdateGroupFunc: func(group *models.ACLGroup) error {
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.UpdateGroup(1, &models.ACLGroup{Name: "New Name", CombinationMode: models.ACLCombinationModeAll})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestUpdateGroup_NotFound(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.UpdateGroup(999, &models.ACLGroup{Name: "Test"})
	if !errors.Is(err, ErrACLGroupNotFound) {
		t.Errorf("Expected ErrACLGroupNotFound, got: %v", err)
	}
}

func TestUpdateGroup_NameConflict(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id, Name: "Old Name", CombinationMode: models.ACLCombinationModeAny}, nil
		},
		GetGroupByNameFunc: func(name string) (*models.ACLGroup, error) {
			if name == "Existing Name" {
				return &models.ACLGroup{ID: 2, Name: name}, nil
			}
			return nil, gorm.ErrRecordNotFound
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.UpdateGroup(1, &models.ACLGroup{Name: "Existing Name"})
	if !errors.Is(err, ErrACLGroupNameExists) {
		t.Errorf("Expected ErrACLGroupNameExists, got: %v", err)
	}
}

func TestDeleteGroup_Success(t *testing.T) {
	t.Parallel()

	deleteCalled := false

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id, Name: "Test"}, nil
		},
		DeleteGroupFunc: func(id int) error {
			deleteCalled = true
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.DeleteGroup(1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !deleteCalled {
		t.Error("Expected DeleteGroup to be called")
	}
}

func TestDeleteGroup_NotFound(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.DeleteGroup(999)
	if !errors.Is(err, ErrACLGroupNotFound) {
		t.Errorf("Expected ErrACLGroupNotFound, got: %v", err)
	}
}

// =============================================================================
// IP Rule Tests
// =============================================================================

func TestAddIPRule_Success(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id}, nil
		},
		CreateIPRuleFunc: func(rule *models.ACLIPRule) error {
			rule.ID = 1
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	rule := &models.ACLIPRule{
		RuleType: models.ACLIPRuleTypeAllow,
		CIDR:     "192.168.1.0/24",
	}

	err := svc.AddIPRule(1, rule)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if rule.ACLGroupID != 1 {
		t.Error("Expected ACLGroupID to be set")
	}
}

func TestAddIPRule_GroupNotFound(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.AddIPRule(999, &models.ACLIPRule{RuleType: models.ACLIPRuleTypeAllow, CIDR: "192.168.1.0/24"})
	if !errors.Is(err, ErrACLGroupNotFound) {
		t.Errorf("Expected ErrACLGroupNotFound, got: %v", err)
	}
}

func TestAddIPRule_InvalidCIDR(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	rule := &models.ACLIPRule{
		RuleType: models.ACLIPRuleTypeAllow,
		CIDR:     "invalid-cidr",
	}

	err := svc.AddIPRule(1, rule)
	if !errors.Is(err, ErrInvalidCIDR) {
		t.Errorf("Expected ErrInvalidCIDR, got: %v", err)
	}
}

func TestUpdateIPRule_Success(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetIPRuleByIDFunc: func(id int) (*models.ACLIPRule, error) {
			return &models.ACLIPRule{ID: id, RuleType: models.ACLIPRuleTypeAllow, CIDR: "192.168.1.0/24"}, nil
		},
		UpdateIPRuleFunc: func(rule *models.ACLIPRule) error {
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.UpdateIPRule(1, &models.ACLIPRule{RuleType: models.ACLIPRuleTypeDeny, CIDR: "10.0.0.0/8"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestDeleteIPRule_Success(t *testing.T) {
	t.Parallel()

	deleteCalled := false

	aclRepo := &MockACLRepository{
		GetIPRuleByIDFunc: func(id int) (*models.ACLIPRule, error) {
			return &models.ACLIPRule{ID: id}, nil
		},
		DeleteIPRuleFunc: func(id int) error {
			deleteCalled = true
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.DeleteIPRule(1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !deleteCalled {
		t.Error("Expected DeleteIPRule to be called")
	}
}

// =============================================================================
// Basic Auth Tests
// =============================================================================

func TestAddBasicAuthUser_Success(t *testing.T) {
	t.Parallel()

	var createdUser *models.ACLBasicAuthUser

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id}, nil
		},
		GetBasicAuthUserFunc: func(groupID int, username string) (*models.ACLBasicAuthUser, error) {
			return nil, gorm.ErrRecordNotFound
		},
		CreateBasicAuthUserFunc: func(user *models.ACLBasicAuthUser) error {
			createdUser = user
			user.ID = 1
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.AddBasicAuthUser(1, "admin", "password123")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if createdUser.Username != "admin" {
		t.Error("Expected username to be set")
	}
	if createdUser.PasswordHash == "" {
		t.Error("Expected password hash to be set")
	}
	if createdUser.PasswordHash == "password123" {
		t.Error("Password should be hashed, not plain text")
	}
}

func TestAddBasicAuthUser_EmptyUsername(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.AddBasicAuthUser(1, "", "password123")
	if !errors.Is(err, models.ErrBasicAuthUsernameEmpty) {
		t.Errorf("Expected ErrBasicAuthUsernameEmpty, got: %v", err)
	}
}

func TestAddBasicAuthUser_EmptyPassword(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.AddBasicAuthUser(1, "admin", "")
	if !errors.Is(err, models.ErrBasicAuthPasswordEmpty) {
		t.Errorf("Expected ErrBasicAuthPasswordEmpty, got: %v", err)
	}
}

func TestAddBasicAuthUser_UserExists(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id}, nil
		},
		GetBasicAuthUserFunc: func(groupID int, username string) (*models.ACLBasicAuthUser, error) {
			return &models.ACLBasicAuthUser{ID: 1, Username: username}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.AddBasicAuthUser(1, "admin", "password")
	if !errors.Is(err, ErrBasicAuthUserExists) {
		t.Errorf("Expected ErrBasicAuthUserExists, got: %v", err)
	}
}

func TestUpdateBasicAuthPassword_Success(t *testing.T) {
	t.Parallel()

	var updatedUser *models.ACLBasicAuthUser

	aclRepo := &MockACLRepository{
		GetBasicAuthUserByIDFunc: func(id int) (*models.ACLBasicAuthUser, error) {
			return &models.ACLBasicAuthUser{ID: id, Username: "admin"}, nil
		},
		UpdateBasicAuthUserFunc: func(user *models.ACLBasicAuthUser) error {
			updatedUser = user
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.UpdateBasicAuthPassword(1, "newpassword")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if updatedUser.PasswordHash == "" {
		t.Error("Expected password hash to be updated")
	}
}

func TestDeleteBasicAuthUser_Success(t *testing.T) {
	t.Parallel()

	deleteCalled := false

	aclRepo := &MockACLRepository{
		GetBasicAuthUserByIDFunc: func(id int) (*models.ACLBasicAuthUser, error) {
			return &models.ACLBasicAuthUser{ID: id}, nil
		},
		DeleteBasicAuthUserFunc: func(id int) error {
			deleteCalled = true
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.DeleteBasicAuthUser(1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !deleteCalled {
		t.Error("Expected DeleteBasicAuthUser to be called")
	}
}

// =============================================================================
// External Provider Tests
// =============================================================================

func TestAddExternalProvider_Success(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id}, nil
		},
		CreateExternalProviderFunc: func(provider *models.ACLExternalProvider) error {
			provider.ID = 1
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	provider := &models.ACLExternalProvider{
		ProviderType: models.ACLProviderTypeAuthelia,
		Name:         "Authelia",
		VerifyURL:    "https://auth.example.com/verify",
	}

	err := svc.AddExternalProvider(1, provider)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if provider.ACLGroupID != 1 {
		t.Error("Expected ACLGroupID to be set")
	}
}

func TestAddExternalProvider_ValidationError(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	provider := &models.ACLExternalProvider{
		ProviderType: "invalid",
		Name:         "Test",
		VerifyURL:    "https://example.com",
	}

	err := svc.AddExternalProvider(1, provider)
	if err == nil {
		t.Error("Expected validation error")
	}
}

// =============================================================================
// Waygates Auth Tests
// =============================================================================

func TestConfigureWaygatesAuth_Create(t *testing.T) {
	t.Parallel()

	createCalled := false

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id}, nil
		},
		GetWaygatesAuthFunc: func(groupID int) (*models.ACLWaygatesAuth, error) {
			return nil, gorm.ErrRecordNotFound
		},
		CreateWaygatesAuthFunc: func(auth *models.ACLWaygatesAuth) error {
			createCalled = true
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.ConfigureWaygatesAuth(1, &models.ACLWaygatesAuth{Enabled: true})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !createCalled {
		t.Error("Expected CreateWaygatesAuth to be called")
	}
}

func TestConfigureWaygatesAuth_Update(t *testing.T) {
	t.Parallel()

	updateCalled := false

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id}, nil
		},
		GetWaygatesAuthFunc: func(groupID int) (*models.ACLWaygatesAuth, error) {
			return &models.ACLWaygatesAuth{ID: 1, ACLGroupID: groupID}, nil
		},
		UpdateWaygatesAuthFunc: func(auth *models.ACLWaygatesAuth) error {
			updateCalled = true
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.ConfigureWaygatesAuth(1, &models.ACLWaygatesAuth{Enabled: true})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !updateCalled {
		t.Error("Expected UpdateWaygatesAuth to be called")
	}
}

func TestConfigureWaygatesAuth_DefaultSessionTTL(t *testing.T) {
	t.Parallel()

	var createdAuth *models.ACLWaygatesAuth

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id}, nil
		},
		GetWaygatesAuthFunc: func(groupID int) (*models.ACLWaygatesAuth, error) {
			return nil, gorm.ErrRecordNotFound
		},
		CreateWaygatesAuthFunc: func(auth *models.ACLWaygatesAuth) error {
			createdAuth = auth
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.ConfigureWaygatesAuth(1, &models.ACLWaygatesAuth{Enabled: true, SessionTTL: 0})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if createdAuth.SessionTTL != 86400 {
		t.Errorf("Expected default SessionTTL 86400, got: %d", createdAuth.SessionTTL)
	}
}

// =============================================================================
// Proxy Assignment Tests
// =============================================================================

func TestAssignToProxy_Success(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id}, nil
		},
		CreateProxyACLAssignmentFunc: func(assignment *models.ProxyACLAssignment) error {
			assignment.ID = 1
			return nil
		},
	}
	proxyRepo := &MockProxyRepository{
		GetByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{ID: id}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo, ProxyRepo: proxyRepo})

	err := svc.AssignToProxy(1, 2, "/api/*", 10)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestAssignToProxy_ProxyNotFound(t *testing.T) {
	t.Parallel()

	proxyRepo := &MockProxyRepository{
		GetByIDFunc: func(id int) (*models.Proxy, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: &MockACLRepository{}, ProxyRepo: proxyRepo})

	err := svc.AssignToProxy(999, 1, "/*", 0)
	if !errors.Is(err, ErrProxyNotFound) {
		t.Errorf("Expected ErrProxyNotFound, got: %v", err)
	}
}

func TestAssignToProxy_GroupNotFound(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	proxyRepo := &MockProxyRepository{
		GetByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{ID: id}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo, ProxyRepo: proxyRepo})

	err := svc.AssignToProxy(1, 999, "/*", 0)
	if !errors.Is(err, ErrACLGroupNotFound) {
		t.Errorf("Expected ErrACLGroupNotFound, got: %v", err)
	}
}

func TestAssignToProxy_InvalidPathPattern(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id}, nil
		},
	}
	proxyRepo := &MockProxyRepository{
		GetByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{ID: id}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo, ProxyRepo: proxyRepo})

	err := svc.AssignToProxy(1, 2, "invalid?path#fragment", 0)
	if !errors.Is(err, ErrInvalidPathPattern) {
		t.Errorf("Expected ErrInvalidPathPattern, got: %v", err)
	}
}

func TestRemoveFromProxy_Success(t *testing.T) {
	t.Parallel()

	deleteCalled := false

	aclRepo := &MockACLRepository{
		DeleteProxyACLAssignmentByProxyAndGroupFunc: func(proxyID, groupID int) error {
			deleteCalled = true
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.RemoveFromProxy(1, 2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !deleteCalled {
		t.Error("Expected DeleteProxyACLAssignmentByProxyAndGroup to be called")
	}
}

// =============================================================================
// Branding Tests
// =============================================================================

func TestGetBranding_Success(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetBrandingFunc: func() (*models.ACLBranding, error) {
			return &models.ACLBranding{ID: 1, Title: "Login"}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	branding, err := svc.GetBranding()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if branding.Title != "Login" {
		t.Errorf("Expected title 'Login', got: %s", branding.Title)
	}
}

func TestUpdateBranding_Success(t *testing.T) {
	t.Parallel()

	updateCalled := false

	aclRepo := &MockACLRepository{
		UpdateBrandingFunc: func(branding *models.ACLBranding) error {
			updateCalled = true
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.UpdateBranding(&models.ACLBranding{Title: "New Title"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !updateCalled {
		t.Error("Expected UpdateBranding to be called")
	}
}

// =============================================================================
// Session Tests
// =============================================================================

func TestCreateSession_Success(t *testing.T) {
	t.Parallel()

	var createdSession *models.ACLSession

	aclRepo := &MockACLRepository{
		CreateSessionFunc: func(session *models.ACLSession) error {
			createdSession = session
			session.ID = 1
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	session, err := svc.CreateSession(1, nil, "192.168.1.1", "Mozilla/5.0", 3600)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if session.ID != 1 {
		t.Error("Expected session ID to be set")
	}
	if createdSession.SessionToken == "" {
		t.Error("Expected session token to be generated")
	}
	if createdSession.UserID == nil || *createdSession.UserID != 1 {
		t.Error("Expected UserID to be set to 1")
	}
}

func TestCreateSession_DefaultTTL(t *testing.T) {
	t.Parallel()

	var createdSession *models.ACLSession

	aclRepo := &MockACLRepository{
		CreateSessionFunc: func(session *models.ACLSession) error {
			createdSession = session
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	_, err := svc.CreateSession(1, nil, "", "", 0) // TTL = 0 should default to 86400
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedExpiry := time.Now().Add(86400 * time.Second)
	if createdSession.ExpiresAt.Sub(expectedExpiry) > time.Second {
		t.Error("Expected default TTL of 86400 seconds")
	}
}

func TestValidateSession_Success(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetSessionByTokenFunc: func(token string) (*models.ACLSession, error) {
			return &models.ACLSession{
				ID:           1,
				SessionToken: token,
				ExpiresAt:    time.Now().Add(1 * time.Hour),
			}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	session, err := svc.ValidateSession("valid-token")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if session.ID != 1 {
		t.Error("Expected session to be returned")
	}
}

func TestValidateSession_NotFound(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		GetSessionByTokenFunc: func(token string) (*models.ACLSession, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	_, err := svc.ValidateSession("invalid-token")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Expected ErrSessionNotFound, got: %v", err)
	}
}

func TestValidateSession_EmptyToken(t *testing.T) {
	t.Parallel()

	svc := NewACLService(ACLServiceConfig{ACLRepo: &MockACLRepository{}})

	_, err := svc.ValidateSession("")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Expected ErrSessionNotFound, got: %v", err)
	}
}

func TestValidateSession_Expired(t *testing.T) {
	t.Parallel()

	deleteCalled := false

	aclRepo := &MockACLRepository{
		GetSessionByTokenFunc: func(token string) (*models.ACLSession, error) {
			return &models.ACLSession{
				ID:           1,
				SessionToken: token,
				ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired
			}, nil
		},
		DeleteSessionFunc: func(token string) error {
			deleteCalled = true
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	_, err := svc.ValidateSession("expired-token")
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("Expected ErrSessionExpired, got: %v", err)
	}

	if !deleteCalled {
		t.Error("Expected expired session to be deleted")
	}
}

func TestRevokeSession_Success(t *testing.T) {
	t.Parallel()

	deleteCalled := false

	aclRepo := &MockACLRepository{
		DeleteSessionFunc: func(token string) error {
			deleteCalled = true
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.RevokeSession("token")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !deleteCalled {
		t.Error("Expected DeleteSession to be called")
	}
}

func TestRevokeUserSessions_Success(t *testing.T) {
	t.Parallel()

	deleteCalled := false

	aclRepo := &MockACLRepository{
		DeleteUserSessionsFunc: func(userID int) error {
			deleteCalled = true
			return nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	err := svc.RevokeUserSessions(1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !deleteCalled {
		t.Error("Expected DeleteUserSessions to be called")
	}
}

func TestCleanupExpiredSessions_Success(t *testing.T) {
	t.Parallel()

	aclRepo := &MockACLRepository{
		DeleteExpiredSessionsFunc: func() (int64, error) {
			return 5, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo})

	count, err := svc.CleanupExpiredSessions()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5, got: %d", count)
	}
}

// =============================================================================
// VerifyAccess Tests
// =============================================================================

func TestVerifyAccess_NoProxyFound(t *testing.T) {
	t.Parallel()

	proxyRepo := &MockProxyRepository{
		GetByHostnameFunc: func(hostname string) (*models.Proxy, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: &MockACLRepository{}, ProxyRepo: proxyRepo})

	response, err := svc.VerifyAccess(&ACLVerifyRequest{Host: "unknown.example.com"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if response.Allowed {
		t.Error("Expected access to be denied")
	}
	if response.Reason != "proxy not found for hostname" {
		t.Errorf("Expected reason 'proxy not found for hostname', got: %s", response.Reason)
	}
}

func TestVerifyAccess_NoACLAssignments(t *testing.T) {
	t.Parallel()

	proxyRepo := &MockProxyRepository{
		GetByHostnameFunc: func(hostname string) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Hostname: hostname}, nil
		},
	}
	aclRepo := &MockACLRepository{
		GetProxyACLAssignmentsFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{}, nil // No assignments
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo, ProxyRepo: proxyRepo})

	response, err := svc.VerifyAccess(&ACLVerifyRequest{Host: "example.com", Path: "/"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !response.Allowed {
		t.Error("Expected access to be allowed when no ACL assignments")
	}
}

func TestVerifyAccess_IPDeny(t *testing.T) {
	t.Parallel()

	proxyRepo := &MockProxyRepository{
		GetByHostnameFunc: func(hostname string) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Hostname: hostname}, nil
		},
	}
	aclRepo := &MockACLRepository{
		GetProxyACLAssignmentsFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{
				{ID: 1, ProxyID: proxyID, ACLGroupID: 1, PathPattern: "/*", Enabled: true},
			}, nil
		},
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{
				ID:              id,
				CombinationMode: models.ACLCombinationModeAny,
				IPRules: []models.ACLIPRule{
					{ID: 1, RuleType: models.ACLIPRuleTypeDeny, CIDR: "192.168.1.0/24"},
				},
			}, nil
		},
		GetBrandingFunc: func() (*models.ACLBranding, error) {
			return &models.ACLBranding{}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo, ProxyRepo: proxyRepo})

	response, err := svc.VerifyAccess(&ACLVerifyRequest{
		Host:     "example.com",
		Path:     "/api",
		RemoteIP: "192.168.1.100",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if response.Allowed {
		t.Error("Expected access to be denied by IP rule")
	}
}

func TestVerifyAccess_IPBypass(t *testing.T) {
	t.Parallel()

	proxyRepo := &MockProxyRepository{
		GetByHostnameFunc: func(hostname string) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Hostname: hostname}, nil
		},
	}
	aclRepo := &MockACLRepository{
		GetProxyACLAssignmentsFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{
				{ID: 1, ProxyID: proxyID, ACLGroupID: 1, PathPattern: "/*", Enabled: true},
			}, nil
		},
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{
				ID:              id,
				CombinationMode: models.ACLCombinationModeIPBypass,
				IPRules: []models.ACLIPRule{
					{ID: 1, RuleType: models.ACLIPRuleTypeBypass, CIDR: "10.0.0.0/8"},
				},
			}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo, ProxyRepo: proxyRepo})

	response, err := svc.VerifyAccess(&ACLVerifyRequest{
		Host:     "example.com",
		Path:     "/api",
		RemoteIP: "10.0.0.50",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !response.Allowed {
		t.Error("Expected access to be allowed by IP bypass")
	}
}

func TestVerifyAccess_IPAllow(t *testing.T) {
	t.Parallel()

	proxyRepo := &MockProxyRepository{
		GetByHostnameFunc: func(hostname string) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Hostname: hostname}, nil
		},
	}
	aclRepo := &MockACLRepository{
		GetProxyACLAssignmentsFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{
				{ID: 1, ProxyID: proxyID, ACLGroupID: 1, PathPattern: "/*", Enabled: true},
			}, nil
		},
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{
				ID:              id,
				CombinationMode: models.ACLCombinationModeIPBypass,
				IPRules: []models.ACLIPRule{
					{ID: 1, RuleType: models.ACLIPRuleTypeAllow, CIDR: "192.168.0.0/16"},
				},
			}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo, ProxyRepo: proxyRepo})

	response, err := svc.VerifyAccess(&ACLVerifyRequest{
		Host:     "example.com",
		Path:     "/",
		RemoteIP: "192.168.1.100",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !response.Allowed {
		t.Error("Expected access to be allowed by IP allow rule")
	}
}

func TestVerifyAccess_BasicAuth(t *testing.T) {
	t.Parallel()

	// Create a user with a known password hash
	testUser := &models.ACLBasicAuthUser{ID: 1, Username: "admin"}
	_ = testUser.SetPassword("password123", 10)

	proxyRepo := &MockProxyRepository{
		GetByHostnameFunc: func(hostname string) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Hostname: hostname}, nil
		},
	}
	aclRepo := &MockACLRepository{
		GetProxyACLAssignmentsFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{
				{ID: 1, ProxyID: proxyID, ACLGroupID: 1, PathPattern: "/*", Enabled: true},
			}, nil
		},
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{
				ID:              id,
				CombinationMode: models.ACLCombinationModeAny,
				BasicAuthUsers:  []models.ACLBasicAuthUser{*testUser},
			}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo, ProxyRepo: proxyRepo})

	response, err := svc.VerifyAccess(&ACLVerifyRequest{
		Host:     "example.com",
		Path:     "/",
		RemoteIP: "1.2.3.4",
		BasicAuth: &BasicAuthCredentials{
			Username: "admin",
			Password: "password123",
		},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !response.Allowed {
		t.Error("Expected access to be allowed with correct basic auth")
	}
}

func TestVerifyAccess_BasicAuth_WrongPassword(t *testing.T) {
	t.Parallel()

	// Create a user with a known password hash
	testUser := &models.ACLBasicAuthUser{ID: 1, Username: "admin"}
	_ = testUser.SetPassword("correctpassword", 10)

	proxyRepo := &MockProxyRepository{
		GetByHostnameFunc: func(hostname string) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Hostname: hostname}, nil
		},
	}
	aclRepo := &MockACLRepository{
		GetProxyACLAssignmentsFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{
				{ID: 1, ProxyID: proxyID, ACLGroupID: 1, PathPattern: "/*", Enabled: true},
			}, nil
		},
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{
				ID:              id,
				CombinationMode: models.ACLCombinationModeAny,
				BasicAuthUsers:  []models.ACLBasicAuthUser{*testUser},
			}, nil
		},
		GetBrandingFunc: func() (*models.ACLBranding, error) {
			return &models.ACLBranding{}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo, ProxyRepo: proxyRepo})

	response, err := svc.VerifyAccess(&ACLVerifyRequest{
		Host:     "example.com",
		Path:     "/",
		RemoteIP: "1.2.3.4",
		BasicAuth: &BasicAuthCredentials{
			Username: "admin",
			Password: "wrongpassword",
		},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if response.Allowed {
		t.Error("Expected access to be denied with wrong password")
	}
}

func TestVerifyAccess_WaygatesSession(t *testing.T) {
	t.Parallel()

	proxyRepo := &MockProxyRepository{
		GetByHostnameFunc: func(hostname string) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Hostname: hostname}, nil
		},
	}
	aclRepo := &MockACLRepository{
		GetProxyACLAssignmentsFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{
				{ID: 1, ProxyID: proxyID, ACLGroupID: 1, PathPattern: "/*", Enabled: true},
			}, nil
		},
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{
				ID:              id,
				CombinationMode: models.ACLCombinationModeAny,
				WaygatesAuth: &models.ACLWaygatesAuth{
					Enabled: true,
				},
			}, nil
		},
		GetSessionByTokenFunc: func(token string) (*models.ACLSession, error) {
			userID := 1
			return &models.ACLSession{
				ID:           1,
				SessionToken: token,
				UserID:       &userID,
				ExpiresAt:    time.Now().Add(1 * time.Hour),
				User: &models.User{
					ID:       1,
					Username: "testuser",
					Email:    "test@example.com",
				},
			}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo, ProxyRepo: proxyRepo})

	response, err := svc.VerifyAccess(&ACLVerifyRequest{
		Host:         "example.com",
		Path:         "/",
		RemoteIP:     "1.2.3.4",
		SessionToken: "valid-session-token",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !response.Allowed {
		t.Error("Expected access to be allowed with valid session")
	}
	if response.User == nil {
		t.Error("Expected user to be set in response")
	}
	if response.Headers["X-Auth-User"] != "testuser" {
		t.Error("Expected X-Auth-User header to be set")
	}
}

func TestVerifyAccess_PathNoMatch(t *testing.T) {
	t.Parallel()

	proxyRepo := &MockProxyRepository{
		GetByHostnameFunc: func(hostname string) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Hostname: hostname}, nil
		},
	}
	aclRepo := &MockACLRepository{
		GetProxyACLAssignmentsFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{
				{ID: 1, ProxyID: proxyID, ACLGroupID: 1, PathPattern: "/api/*", Enabled: true},
			}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo, ProxyRepo: proxyRepo})

	response, err := svc.VerifyAccess(&ACLVerifyRequest{
		Host:     "example.com",
		Path:     "/public/resource", // Doesn't match /api/*
		RemoteIP: "1.2.3.4",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !response.Allowed {
		t.Error("Expected access to be allowed when path doesn't match ACL patterns")
	}
}

func TestVerifyAccess_DisabledAssignment(t *testing.T) {
	t.Parallel()

	proxyRepo := &MockProxyRepository{
		GetByHostnameFunc: func(hostname string) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Hostname: hostname}, nil
		},
	}
	aclRepo := &MockACLRepository{
		GetProxyACLAssignmentsFunc: func(proxyID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{
				{ID: 1, ProxyID: proxyID, ACLGroupID: 1, PathPattern: "/*", Enabled: false}, // Disabled
			}, nil
		},
		GetGroupByIDFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{
				ID:              id,
				CombinationMode: models.ACLCombinationModeAny,
				IPRules: []models.ACLIPRule{
					{RuleType: models.ACLIPRuleTypeDeny, CIDR: "0.0.0.0/0"}, // Deny all
				},
			}, nil
		},
		GetBrandingFunc: func() (*models.ACLBranding, error) {
			return &models.ACLBranding{}, nil
		},
	}

	svc := NewACLService(ACLServiceConfig{ACLRepo: aclRepo, ProxyRepo: proxyRepo})

	response, err := svc.VerifyAccess(&ACLVerifyRequest{
		Host:     "example.com",
		Path:     "/",
		RemoteIP: "1.2.3.4",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Disabled assignments should be skipped, but access should be denied due to no valid auth
	if response.Allowed {
		t.Error("Expected access to be denied when no enabled rules grant access")
	}
}

// =============================================================================
// Path Matching Tests
// =============================================================================

func TestMatchPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		pattern  string
		path     string
		expected bool
	}{
		// Exact matches
		{"/api", "/api", true},
		{"/", "/", true},

		// Wildcard patterns
		{"/*", "/anything", true},
		{"/*", "/", true},
		{"*", "/anything", true},

		// Prefix patterns - note: /api/* matches /api because it's a prefix match
		{"/api/*", "/api/users", true},
		{"/api/*", "/api/users/123", true},
		{"/api/*", "/api", true}, // Matches because /api has prefix /api
		{"/api/*", "/other", false},

		// Middle wildcard patterns
		{"/api/*/users", "/api/v1/users", true},
		{"/api/*/users", "/api/v2/users", true},

		// Non-matches
		{"/api", "/other", false},
		{"/specific/path", "/different/path", false},
	}

	for _, tc := range testCases {
		t.Run(tc.pattern+"_"+tc.path, func(t *testing.T) {
			result := matchPath(tc.pattern, tc.path)
			if result != tc.expected {
				t.Errorf("matchPath(%q, %q) = %v, expected %v", tc.pattern, tc.path, result, tc.expected)
			}
		})
	}
}

// =============================================================================
// Email Pattern Matching Tests
// =============================================================================

func TestMatchEmailPattern(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		email    string
		pattern  string
		expected bool
	}{
		// Wildcard
		{"user@example.com", "*", true},
		{"anyone@anywhere.com", "*", true},

		// Domain patterns
		{"user@company.com", "*@company.com", true},
		{"admin@company.com", "*@company.com", true},
		{"user@other.com", "*@company.com", false},

		// Exact matches
		{"user@example.com", "user@example.com", true},
		{"other@example.com", "user@example.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.email+"_"+tc.pattern, func(t *testing.T) {
			result := matchEmailPattern(tc.email, tc.pattern)
			if result != tc.expected {
				t.Errorf("matchEmailPattern(%q, %q) = %v, expected %v", tc.email, tc.pattern, result, tc.expected)
			}
		})
	}
}

// =============================================================================
// CIDR Validation Tests
// =============================================================================

func TestValidateCIDR(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		cidr    string
		isValid bool
	}{
		// Valid CIDRs
		{"192.168.1.0/24", true},
		{"10.0.0.0/8", true},
		{"172.16.0.0/12", true},
		{"0.0.0.0/0", true},
		{"2001:db8::/32", true},

		// Valid single IPs (not CIDR but accepted)
		{"192.168.1.1", true},
		{"10.0.0.1", true},

		// Invalid
		{"invalid", false},
		{"192.168.1.0/33", false}, // Invalid mask
		{"256.256.256.256", false},
	}

	for _, tc := range testCases {
		t.Run(tc.cidr, func(t *testing.T) {
			err := validateCIDR(tc.cidr)
			if tc.isValid && err != nil {
				t.Errorf("Expected %q to be valid, got error: %v", tc.cidr, err)
			}
			if !tc.isValid && err == nil {
				t.Errorf("Expected %q to be invalid", tc.cidr)
			}
		})
	}
}

// =============================================================================
// Path Pattern Validation Tests
// =============================================================================

func TestValidatePathPattern(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		pattern string
		isValid bool
	}{
		// Valid patterns
		{"", true}, // Empty is valid (defaults to /*)
		{"/*", true},
		{"/api/*", true},
		{"/api/v1/users", true},
		{"/api/*/users", true},
		{"*", true},

		// Invalid patterns
		{"api/users", false},    // Doesn't start with /
		{"/api?query=1", false}, // Contains ?
		{"/api#hash", false},    // Contains #
	}

	for _, tc := range testCases {
		t.Run(tc.pattern, func(t *testing.T) {
			err := validatePathPattern(tc.pattern)
			if tc.isValid && err != nil {
				t.Errorf("Expected %q to be valid, got error: %v", tc.pattern, err)
			}
			if !tc.isValid && err == nil {
				t.Errorf("Expected %q to be invalid", tc.pattern)
			}
		})
	}
}

// =============================================================================
// Service Error Constants Tests
// =============================================================================

func TestServiceErrorConstants(t *testing.T) {
	t.Parallel()

	errors := map[string]error{
		"ErrACLGroupNotFound":         ErrACLGroupNotFound,
		"ErrACLGroupNameExists":       ErrACLGroupNameExists,
		"ErrIPRuleNotFound":           ErrIPRuleNotFound,
		"ErrBasicAuthUserNotFound":    ErrBasicAuthUserNotFound,
		"ErrBasicAuthUserExists":      ErrBasicAuthUserExists,
		"ErrExternalProviderNotFound": ErrExternalProviderNotFound,
		"ErrWaygatesAuthNotFound":     ErrWaygatesAuthNotFound,
		"ErrProxyACLNotFound":         ErrProxyACLNotFound,
		"ErrProxyACLExists":           ErrProxyACLExists,
		"ErrSessionNotFound":          ErrSessionNotFound,
		"ErrSessionExpired":           ErrSessionExpired,
		"ErrAccessDenied":             ErrAccessDenied,
		"ErrInvalidCIDR":              ErrInvalidCIDR,
		"ErrInvalidPathPattern":       ErrInvalidPathPattern,
		"ErrProxyNotFoundForHost":     ErrProxyNotFoundForHost,
	}

	for name, err := range errors {
		t.Run(name, func(t *testing.T) {
			if err == nil {
				t.Errorf("Expected %s to be non-nil", name)
			}
			if err.Error() == "" {
				t.Errorf("Expected %s to have a message", name)
			}
		})
	}
}

// =============================================================================
// Constants Tests
// =============================================================================

func TestBcryptCost(t *testing.T) {
	t.Parallel()

	if BcryptCost != 14 {
		t.Errorf("Expected BcryptCost 14, got: %d", BcryptCost)
	}
}

func TestSessionTokenLength(t *testing.T) {
	t.Parallel()

	if SessionTokenLength != 32 {
		t.Errorf("Expected SessionTokenLength 32, got: %d", SessionTokenLength)
	}
}
