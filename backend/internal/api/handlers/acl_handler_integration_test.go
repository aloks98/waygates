package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aloks98/goauth/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service"
)

// =============================================================================
// Mock ACL Service
// =============================================================================

// MockACLService is a mock implementation of ACLServiceInterface for testing
type MockACLService struct {
	// Group Management
	CreateGroupFunc         func(group *models.ACLGroup, createdBy int) error
	GetGroupFunc            func(id int) (*models.ACLGroup, error)
	GetGroupByNameFunc      func(name string) (*models.ACLGroup, error)
	ListGroupsFunc          func(params service.ListACLGroupsRequest) (*models.ACLGroupListResponse, error)
	UpdateGroupFunc         func(id int, updates *models.ACLGroup) error
	DeleteGroupFunc         func(id int) error
	DeleteGroupWithSyncFunc func(id int, syncFn service.SyncCallback) error

	// IP Rules
	AddIPRuleFunc    func(groupID int, rule *models.ACLIPRule) error
	UpdateIPRuleFunc func(id int, rule *models.ACLIPRule) error
	DeleteIPRuleFunc func(id int) error

	// Basic Auth
	AddBasicAuthUserFunc        func(groupID int, username, password string) error
	UpdateBasicAuthPasswordFunc func(id int, password string) error
	DeleteBasicAuthUserFunc     func(id int) error

	// External Providers
	AddExternalProviderFunc    func(groupID int, provider *models.ACLExternalProvider) error
	UpdateExternalProviderFunc func(id int, provider *models.ACLExternalProvider) error
	DeleteExternalProviderFunc func(id int) error

	// Waygates Auth
	GetWaygatesAuthFunc       func(groupID int) (*models.ACLWaygatesAuth, error)
	ConfigureWaygatesAuthFunc func(groupID int, config *models.ACLWaygatesAuth) error

	// Proxy Assignment
	AssignToProxyFunc         func(proxyID, groupID int, pathPattern string, priority int, enabled bool) error
	UpdateProxyAssignmentFunc func(id int, pathPattern string, priority int, enabled bool) error
	RemoveFromProxyFunc       func(proxyID, groupID int) error
	GetProxyACLFunc           func(proxyID int) ([]models.ProxyACLAssignment, error)
	GetGroupUsageFunc         func(groupID int) ([]models.ProxyACLAssignment, error)

	// Branding
	GetBrandingFunc    func() (*models.ACLBranding, error)
	UpdateBrandingFunc func(branding *models.ACLBranding) error

	// OAuth Provider Restrictions
	GetOAuthProviderRestrictionsFunc   func(groupID int) ([]models.ACLOAuthProviderRestriction, error)
	SetOAuthProviderRestrictionFunc    func(groupID int, provider string, emails, domains []string, enabled bool) error
	DeleteOAuthProviderRestrictionFunc func(groupID int, provider string) error

	// Access Verification
	VerifyAccessFunc func(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error)

	// Auth Options
	GetAuthOptionsForProxyFunc func(hostname string) (*service.AuthOptionsResponse, error)

	// Session Management
	CreateSessionFunc           func(userID int, proxyID *int, ip, userAgent string, ttl int) (*models.ACLSession, error)
	CreateOAuthSessionFunc      func(email, provider string, proxyID *int, ip, userAgent string, ttl int) (*models.ACLSession, error)
	CreateSessionWithParamsFunc func(params service.CreateSessionParams) (*models.ACLSession, error)
	ValidateSessionFunc         func(token string) (*models.ACLSession, error)
	RevokeSessionFunc           func(token string) error
	RevokeUserSessionsFunc      func(userID int) error
	CleanupExpiredSessionsFunc  func() (int64, error)
}

// CreateGroup implements ACLServiceInterface.
func (m *MockACLService) CreateGroup(group *models.ACLGroup, createdBy int) error {
	if m.CreateGroupFunc != nil {
		return m.CreateGroupFunc(group, createdBy)
	}
	return nil
}

// GetGroup implements ACLServiceInterface.
func (m *MockACLService) GetGroup(id int) (*models.ACLGroup, error) {
	if m.GetGroupFunc != nil {
		return m.GetGroupFunc(id)
	}
	// Return a valid mock group by default for testing
	return &models.ACLGroup{
		ID:   id,
		Name: "mock-group",
	}, nil
}

// GetGroupByName implements ACLServiceInterface.
func (m *MockACLService) GetGroupByName(name string) (*models.ACLGroup, error) {
	if m.GetGroupByNameFunc != nil {
		return m.GetGroupByNameFunc(name)
	}
	return nil, service.ErrACLGroupNotFound
}

// ListGroups implements ACLServiceInterface.
func (m *MockACLService) ListGroups(params service.ListACLGroupsRequest) (*models.ACLGroupListResponse, error) {
	if m.ListGroupsFunc != nil {
		return m.ListGroupsFunc(params)
	}
	return &models.ACLGroupListResponse{Items: []models.ACLGroup{}, Total: 0}, nil
}

// UpdateGroup implements ACLServiceInterface.
func (m *MockACLService) UpdateGroup(id int, updates *models.ACLGroup) error {
	if m.UpdateGroupFunc != nil {
		return m.UpdateGroupFunc(id, updates)
	}
	return nil
}

// DeleteGroup implements ACLServiceInterface.
func (m *MockACLService) DeleteGroup(id int) error {
	if m.DeleteGroupFunc != nil {
		return m.DeleteGroupFunc(id)
	}
	return nil
}

// DeleteGroupWithSync implements ACLServiceInterface.
func (m *MockACLService) DeleteGroupWithSync(id int, syncFn service.SyncCallback) error {
	if m.DeleteGroupWithSyncFunc != nil {
		return m.DeleteGroupWithSyncFunc(id, syncFn)
	}
	return nil
}

// AddIPRule implements ACLServiceInterface.
func (m *MockACLService) AddIPRule(groupID int, rule *models.ACLIPRule) error {
	if m.AddIPRuleFunc != nil {
		return m.AddIPRuleFunc(groupID, rule)
	}
	return nil
}

// UpdateIPRule implements ACLServiceInterface.
func (m *MockACLService) UpdateIPRule(id int, rule *models.ACLIPRule) error {
	if m.UpdateIPRuleFunc != nil {
		return m.UpdateIPRuleFunc(id, rule)
	}
	return nil
}

// DeleteIPRule implements ACLServiceInterface.
func (m *MockACLService) DeleteIPRule(id int) error {
	if m.DeleteIPRuleFunc != nil {
		return m.DeleteIPRuleFunc(id)
	}
	return nil
}

// AddBasicAuthUser implements ACLServiceInterface.
func (m *MockACLService) AddBasicAuthUser(groupID int, username, password string) error {
	if m.AddBasicAuthUserFunc != nil {
		return m.AddBasicAuthUserFunc(groupID, username, password)
	}
	return nil
}

// UpdateBasicAuthPassword implements ACLServiceInterface.
func (m *MockACLService) UpdateBasicAuthPassword(id int, password string) error {
	if m.UpdateBasicAuthPasswordFunc != nil {
		return m.UpdateBasicAuthPasswordFunc(id, password)
	}
	return nil
}

// DeleteBasicAuthUser implements ACLServiceInterface.
func (m *MockACLService) DeleteBasicAuthUser(id int) error {
	if m.DeleteBasicAuthUserFunc != nil {
		return m.DeleteBasicAuthUserFunc(id)
	}
	return nil
}

// AddExternalProvider implements ACLServiceInterface.
func (m *MockACLService) AddExternalProvider(groupID int, provider *models.ACLExternalProvider) error {
	if m.AddExternalProviderFunc != nil {
		return m.AddExternalProviderFunc(groupID, provider)
	}
	return nil
}

// UpdateExternalProvider implements ACLServiceInterface.
func (m *MockACLService) UpdateExternalProvider(id int, provider *models.ACLExternalProvider) error {
	if m.UpdateExternalProviderFunc != nil {
		return m.UpdateExternalProviderFunc(id, provider)
	}
	return nil
}

// DeleteExternalProvider implements ACLServiceInterface.
func (m *MockACLService) DeleteExternalProvider(id int) error {
	if m.DeleteExternalProviderFunc != nil {
		return m.DeleteExternalProviderFunc(id)
	}
	return nil
}

// GetWaygatesAuth implements ACLServiceInterface.
func (m *MockACLService) GetWaygatesAuth(groupID int) (*models.ACLWaygatesAuth, error) {
	if m.GetWaygatesAuthFunc != nil {
		return m.GetWaygatesAuthFunc(groupID)
	}
	return nil, service.ErrWaygatesAuthNotFound
}

// ConfigureWaygatesAuth implements ACLServiceInterface.
func (m *MockACLService) ConfigureWaygatesAuth(groupID int, config *models.ACLWaygatesAuth) error {
	if m.ConfigureWaygatesAuthFunc != nil {
		return m.ConfigureWaygatesAuthFunc(groupID, config)
	}
	return nil
}

// AssignToProxy implements ACLServiceInterface.
func (m *MockACLService) AssignToProxy(proxyID, groupID int, pathPattern string, priority int, enabled bool) error {
	if m.AssignToProxyFunc != nil {
		return m.AssignToProxyFunc(proxyID, groupID, pathPattern, priority, enabled)
	}
	return nil
}

// UpdateProxyAssignment implements ACLServiceInterface.
func (m *MockACLService) UpdateProxyAssignment(id int, pathPattern string, priority int, enabled bool) error {
	if m.UpdateProxyAssignmentFunc != nil {
		return m.UpdateProxyAssignmentFunc(id, pathPattern, priority, enabled)
	}
	return nil
}

// RemoveFromProxy implements ACLServiceInterface.
func (m *MockACLService) RemoveFromProxy(proxyID, groupID int) error {
	if m.RemoveFromProxyFunc != nil {
		return m.RemoveFromProxyFunc(proxyID, groupID)
	}
	return nil
}

// GetProxyACL implements ACLServiceInterface.
func (m *MockACLService) GetProxyACL(proxyID int) ([]models.ProxyACLAssignment, error) {
	if m.GetProxyACLFunc != nil {
		return m.GetProxyACLFunc(proxyID)
	}
	return []models.ProxyACLAssignment{}, nil
}

// GetGroupUsage implements ACLServiceInterface.
func (m *MockACLService) GetGroupUsage(groupID int) ([]models.ProxyACLAssignment, error) {
	if m.GetGroupUsageFunc != nil {
		return m.GetGroupUsageFunc(groupID)
	}
	return []models.ProxyACLAssignment{}, nil
}

// GetBranding implements ACLServiceInterface.
func (m *MockACLService) GetBranding() (*models.ACLBranding, error) {
	if m.GetBrandingFunc != nil {
		return m.GetBrandingFunc()
	}
	return &models.ACLBranding{
		ID:              1,
		PrimaryColor:    "#3b82f6",
		BackgroundColor: "#ffffff",
		Title:           "Login Required",
	}, nil
}

// UpdateBranding implements ACLServiceInterface.
func (m *MockACLService) UpdateBranding(branding *models.ACLBranding) error {
	if m.UpdateBrandingFunc != nil {
		return m.UpdateBrandingFunc(branding)
	}
	return nil
}

// GetOAuthProviderRestrictions implements ACLServiceInterface.
func (m *MockACLService) GetOAuthProviderRestrictions(groupID int) ([]models.ACLOAuthProviderRestriction, error) {
	if m.GetOAuthProviderRestrictionsFunc != nil {
		return m.GetOAuthProviderRestrictionsFunc(groupID)
	}
	return []models.ACLOAuthProviderRestriction{}, nil
}

// SetOAuthProviderRestriction implements ACLServiceInterface.
func (m *MockACLService) SetOAuthProviderRestriction(groupID int, provider string, emails, domains []string, enabled bool) error {
	if m.SetOAuthProviderRestrictionFunc != nil {
		return m.SetOAuthProviderRestrictionFunc(groupID, provider, emails, domains, enabled)
	}
	return nil
}

// DeleteOAuthProviderRestriction implements ACLServiceInterface.
func (m *MockACLService) DeleteOAuthProviderRestriction(groupID int, provider string) error {
	if m.DeleteOAuthProviderRestrictionFunc != nil {
		return m.DeleteOAuthProviderRestrictionFunc(groupID, provider)
	}
	return nil
}

// VerifyAccess implements ACLServiceInterface.
func (m *MockACLService) VerifyAccess(request *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
	if m.VerifyAccessFunc != nil {
		return m.VerifyAccessFunc(request)
	}
	return &service.ACLVerifyResponse{Allowed: true}, nil
}

// GetAuthOptionsForProxy implements ACLServiceInterface.
func (m *MockACLService) GetAuthOptionsForProxy(hostname string) (*service.AuthOptionsResponse, error) {
	if m.GetAuthOptionsForProxyFunc != nil {
		return m.GetAuthOptionsForProxyFunc(hostname)
	}
	return nil, nil
}

// CreateSession implements ACLServiceInterface.
func (m *MockACLService) CreateSession(userID int, proxyID *int, ip, userAgent string, ttl int) (*models.ACLSession, error) {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(userID, proxyID, ip, userAgent, ttl)
	}
	return nil, nil
}

// CreateOAuthSession implements ACLServiceInterface.
func (m *MockACLService) CreateOAuthSession(email, provider string, proxyID *int, ip, userAgent string, ttl int) (*models.ACLSession, error) {
	if m.CreateOAuthSessionFunc != nil {
		return m.CreateOAuthSessionFunc(email, provider, proxyID, ip, userAgent, ttl)
	}
	return nil, nil
}

// CreateSessionWithParams implements ACLServiceInterface.
func (m *MockACLService) CreateSessionWithParams(params service.CreateSessionParams) (*models.ACLSession, error) {
	if m.CreateSessionWithParamsFunc != nil {
		return m.CreateSessionWithParamsFunc(params)
	}
	return nil, nil
}

// ValidateSession implements ACLServiceInterface.
func (m *MockACLService) ValidateSession(token string) (*models.ACLSession, error) {
	if m.ValidateSessionFunc != nil {
		return m.ValidateSessionFunc(token)
	}
	return nil, service.ErrSessionNotFound
}

// RevokeSession implements ACLServiceInterface.
func (m *MockACLService) RevokeSession(token string) error {
	if m.RevokeSessionFunc != nil {
		return m.RevokeSessionFunc(token)
	}
	return nil
}

// RevokeUserSessions implements ACLServiceInterface.
func (m *MockACLService) RevokeUserSessions(userID int) error {
	if m.RevokeUserSessionsFunc != nil {
		return m.RevokeUserSessionsFunc(userID)
	}
	return nil
}

// CleanupExpiredSessions implements ACLServiceInterface.
func (m *MockACLService) CleanupExpiredSessions() (int64, error) {
	if m.CleanupExpiredSessionsFunc != nil {
		return m.CleanupExpiredSessionsFunc()
	}
	return 0, nil
}

// MockACLRepository is a mock implementation of ACLRepositoryInterface for testing
type MockACLRepository struct {
	ListIPRulesFunc               func(groupID int) ([]models.ACLIPRule, error)
	GetIPRuleByIDFunc             func(id int) (*models.ACLIPRule, error)
	ListBasicAuthUsersFunc        func(groupID int) ([]models.ACLBasicAuthUser, error)
	GetBasicAuthUserByIDFunc      func(id int) (*models.ACLBasicAuthUser, error)
	ListExternalProvidersFunc     func(groupID int) ([]models.ACLExternalProvider, error)
	GetExternalProviderByIDFunc   func(id int) (*models.ACLExternalProvider, error)
	GetProxyACLAssignmentByIDFunc func(id int) (*models.ProxyACLAssignment, error)
}

func (m *MockACLRepository) CreateGroup(_ *models.ACLGroup) error              { return nil }
func (m *MockACLRepository) GetGroupByID(_ int) (*models.ACLGroup, error)      { return nil, nil }
func (m *MockACLRepository) GetGroupByName(_ string) (*models.ACLGroup, error) { return nil, nil }
func (m *MockACLRepository) ListGroups(_ repository.ACLGroupListParams) ([]models.ACLGroup, int64, error) {
	return nil, 0, nil
}
func (m *MockACLRepository) UpdateGroup(_ *models.ACLGroup) error   { return nil }
func (m *MockACLRepository) DeleteGroup(_ int) error                { return nil }
func (m *MockACLRepository) CreateIPRule(_ *models.ACLIPRule) error { return nil }
func (m *MockACLRepository) GetIPRuleByID(id int) (*models.ACLIPRule, error) {
	if m.GetIPRuleByIDFunc != nil {
		return m.GetIPRuleByIDFunc(id)
	}
	// Return a default rule with group ID 1 for tests that don't need specific behavior
	return &models.ACLIPRule{ID: id, ACLGroupID: 1}, nil
}
func (m *MockACLRepository) ListIPRules(groupID int) ([]models.ACLIPRule, error) {
	if m.ListIPRulesFunc != nil {
		return m.ListIPRulesFunc(groupID)
	}
	return []models.ACLIPRule{}, nil
}
func (m *MockACLRepository) UpdateIPRule(_ *models.ACLIPRule) error               { return nil }
func (m *MockACLRepository) DeleteIPRule(_ int) error                             { return nil }
func (m *MockACLRepository) CreateBasicAuthUser(_ *models.ACLBasicAuthUser) error { return nil }
func (m *MockACLRepository) GetBasicAuthUserByID(id int) (*models.ACLBasicAuthUser, error) {
	if m.GetBasicAuthUserByIDFunc != nil {
		return m.GetBasicAuthUserByIDFunc(id)
	}
	// Return a default user with group ID 1 for tests that don't need specific behavior
	return &models.ACLBasicAuthUser{ID: id, ACLGroupID: 1}, nil
}
func (m *MockACLRepository) GetBasicAuthUser(_ int, _ string) (*models.ACLBasicAuthUser, error) {
	return nil, nil
}
func (m *MockACLRepository) ListBasicAuthUsers(groupID int) ([]models.ACLBasicAuthUser, error) {
	if m.ListBasicAuthUsersFunc != nil {
		return m.ListBasicAuthUsersFunc(groupID)
	}
	return []models.ACLBasicAuthUser{}, nil
}
func (m *MockACLRepository) UpdateBasicAuthUser(_ *models.ACLBasicAuthUser) error { return nil }
func (m *MockACLRepository) DeleteBasicAuthUser(_ int) error                      { return nil }
func (m *MockACLRepository) CreateExternalProvider(_ *models.ACLExternalProvider) error {
	return nil
}
func (m *MockACLRepository) GetExternalProviderByID(id int) (*models.ACLExternalProvider, error) {
	if m.GetExternalProviderByIDFunc != nil {
		return m.GetExternalProviderByIDFunc(id)
	}
	// Return a default provider with group ID 1 for tests that don't need specific behavior
	return &models.ACLExternalProvider{ID: id, ACLGroupID: 1}, nil
}
func (m *MockACLRepository) ListExternalProviders(groupID int) ([]models.ACLExternalProvider, error) {
	if m.ListExternalProvidersFunc != nil {
		return m.ListExternalProvidersFunc(groupID)
	}
	return []models.ACLExternalProvider{}, nil
}
func (m *MockACLRepository) UpdateExternalProvider(_ *models.ACLExternalProvider) error {
	return nil
}
func (m *MockACLRepository) DeleteExternalProvider(_ int) error { return nil }
func (m *MockACLRepository) GetWaygatesAuth(_ int) (*models.ACLWaygatesAuth, error) {
	return nil, nil
}
func (m *MockACLRepository) CreateWaygatesAuth(_ *models.ACLWaygatesAuth) error { return nil }
func (m *MockACLRepository) UpdateWaygatesAuth(_ *models.ACLWaygatesAuth) error { return nil }
func (m *MockACLRepository) DeleteWaygatesAuth(_ int) error                     { return nil }
func (m *MockACLRepository) GetOAuthProviderRestrictions(_ int) ([]models.ACLOAuthProviderRestriction, error) {
	return []models.ACLOAuthProviderRestriction{}, nil
}
func (m *MockACLRepository) GetOAuthProviderRestriction(_ int, _ string) (*models.ACLOAuthProviderRestriction, error) {
	return nil, nil
}
func (m *MockACLRepository) CreateOAuthProviderRestriction(_ *models.ACLOAuthProviderRestriction) error {
	return nil
}
func (m *MockACLRepository) UpdateOAuthProviderRestriction(_ *models.ACLOAuthProviderRestriction) error {
	return nil
}
func (m *MockACLRepository) DeleteOAuthProviderRestriction(_ int, _ string) error {
	return nil
}
func (m *MockACLRepository) CreateProxyACLAssignment(_ *models.ProxyACLAssignment) error {
	return nil
}
func (m *MockACLRepository) GetProxyACLAssignments(_ int) ([]models.ProxyACLAssignment, error) {
	return nil, nil
}
func (m *MockACLRepository) GetProxyACLAssignmentsByGroup(_ int) ([]models.ProxyACLAssignment, error) {
	return nil, nil
}
func (m *MockACLRepository) UpdateProxyACLAssignment(_ *models.ProxyACLAssignment) error {
	return nil
}
func (m *MockACLRepository) DeleteProxyACLAssignment(_ int) error { return nil }
func (m *MockACLRepository) GetProxyACLAssignmentByID(id int) (*models.ProxyACLAssignment, error) {
	if m.GetProxyACLAssignmentByIDFunc != nil {
		return m.GetProxyACLAssignmentByIDFunc(id)
	}
	return nil, nil
}
func (m *MockACLRepository) DeleteProxyACLAssignmentByProxyAndGroup(_, _ int) error {
	return nil
}
func (m *MockACLRepository) GetBranding() (*models.ACLBranding, error)  { return nil, nil }
func (m *MockACLRepository) UpdateBranding(_ *models.ACLBranding) error { return nil }
func (m *MockACLRepository) CreateSession(_ *models.ACLSession) error   { return nil }
func (m *MockACLRepository) GetSessionByToken(_ string) (*models.ACLSession, error) {
	return nil, nil
}
func (m *MockACLRepository) DeleteSession(_ string) error          { return nil }
func (m *MockACLRepository) DeleteExpiredSessions() (int64, error) { return 0, nil }
func (m *MockACLRepository) DeleteUserSessions(_ int) error        { return nil }
func (m *MockACLRepository) DeleteProxySessions(_ int) error       { return nil }

// GetDB implements ACLRepositoryInterface.
func (m *MockACLRepository) GetDB() *gorm.DB { return nil }

// DeleteGroupWithTx implements ACLRepositoryInterface.
func (m *MockACLRepository) DeleteGroupWithTx(_ *gorm.DB, _ int) error { return nil }

// GetProxyACLAssignmentsByGroupWithTx implements ACLRepositoryInterface.
func (m *MockACLRepository) GetProxyACLAssignmentsByGroupWithTx(_ *gorm.DB, _ int) ([]models.ProxyACLAssignment, error) {
	return nil, nil
}

// Ensure mock implements interface
var _ service.ACLServiceInterface = (*MockACLService)(nil)

// =============================================================================
// Test Helpers
// =============================================================================

// setupACLTestRouter creates a router with ACL handler for testing
func setupACLTestRouter(mockService *MockACLService, mockRepo *MockACLRepository) *chi.Mux {
	handler := NewACLHandler(mockService, mockRepo, nil, nil, nil)
	r := chi.NewRouter()

	// Group routes
	r.Get("/api/acl/groups", handler.ListGroups)
	r.Post("/api/acl/groups", handler.CreateGroup)
	r.Get("/api/acl/groups/{id}", handler.GetGroup)
	r.Put("/api/acl/groups/{id}", handler.UpdateGroup)
	r.Delete("/api/acl/groups/{id}", handler.DeleteGroup)
	r.Get("/api/acl/groups/{id}/usage", handler.GetGroupUsage)

	// IP Rules
	r.Get("/api/acl/groups/{id}/ip-rules", handler.ListIPRules)
	r.Post("/api/acl/groups/{id}/ip-rules", handler.AddIPRule)
	r.Put("/api/acl/ip-rules/{id}", handler.UpdateIPRule)
	r.Delete("/api/acl/ip-rules/{id}", handler.DeleteIPRule)

	// Basic Auth
	r.Get("/api/acl/groups/{id}/basic-auth", handler.ListBasicAuthUsers)
	r.Post("/api/acl/groups/{id}/basic-auth", handler.AddBasicAuthUser)
	r.Put("/api/acl/basic-auth/{id}", handler.UpdateBasicAuthUser)
	r.Delete("/api/acl/basic-auth/{id}", handler.DeleteBasicAuthUser)

	// External Providers
	r.Get("/api/acl/groups/{id}/providers", handler.ListExternalProviders)
	r.Post("/api/acl/groups/{id}/providers", handler.AddExternalProvider)
	r.Put("/api/acl/providers/{id}", handler.UpdateExternalProvider)
	r.Delete("/api/acl/providers/{id}", handler.DeleteExternalProvider)

	// Waygates Auth
	r.Get("/api/acl/groups/{id}/waygates-auth", handler.GetWaygatesAuth)
	r.Put("/api/acl/groups/{id}/waygates-auth", handler.ConfigureWaygatesAuth)

	// Branding
	r.Get("/api/acl/branding", handler.GetBranding)
	r.Put("/api/acl/branding", handler.UpdateBranding)

	return r
}

// createAuthenticatedRequest creates a request with a mock user ID context
func createAuthenticatedRequest(method, url string, body interface{}) *http.Request {
	var bodyReader *bytes.Buffer
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonBody)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, url, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	// Set user ID in context (simulating goauth middleware)
	ctx := middleware.SetUserID(req.Context(), "1")
	req = req.WithContext(ctx)
	return req
}

// =============================================================================
// ACL Group Tests
// =============================================================================

func TestACLHandler_ListGroups_Success(t *testing.T) {
	mockService := &MockACLService{
		ListGroupsFunc: func(_ service.ListACLGroupsRequest) (*models.ACLGroupListResponse, error) {
			return &models.ACLGroupListResponse{
				Items: []models.ACLGroup{
					{ID: 1, Name: "Test Group", CombinationMode: models.ACLCombinationModeAny},
					{ID: 2, Name: "Admin Group", CombinationMode: models.ACLCombinationModeAll},
				},
				Total:      2,
				Page:       1,
				Limit:      20,
				TotalPages: 1,
			}, nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, true, response["success"])
}

func TestACLHandler_ListGroups_WithPagination(t *testing.T) {
	var capturedReq service.ListACLGroupsRequest
	mockService := &MockACLService{
		ListGroupsFunc: func(params service.ListACLGroupsRequest) (*models.ACLGroupListResponse, error) {
			capturedReq = params
			return &models.ACLGroupListResponse{Items: []models.ACLGroup{}, Total: 0}, nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups?page=2&limit=50&search=admin", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 2, capturedReq.Page)
	assert.Equal(t, 50, capturedReq.Limit)
	assert.Equal(t, "admin", capturedReq.Search)
}

func TestACLHandler_ListGroups_InvalidPage(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups?page=-1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_ListGroups_InvalidLimit(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups?limit=101", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_GetGroup_Success(t *testing.T) {
	mockService := &MockACLService{
		GetGroupFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{
				ID:              id,
				Name:            "Test Group",
				CombinationMode: models.ACLCombinationModeAny,
			}, nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_GetGroup_NotFound(t *testing.T) {
	mockService := &MockACLService{
		GetGroupFunc: func(_ int) (*models.ACLGroup, error) {
			return nil, service.ErrACLGroupNotFound
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups/999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestACLHandler_GetGroup_InvalidID(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups/invalid", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_CreateGroup_ValidationError_EmptyName(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	body := `{"name": ""}`
	req := createAuthenticatedRequest(http.MethodPost, "/api/acl/groups", nil)
	req.Body = httpReadCloser(body)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_CreateGroup_InvalidCombinationMode(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	body := `{"name": "Test", "combination_mode": "invalid"}`
	req := createAuthenticatedRequest(http.MethodPost, "/api/acl/groups", nil)
	req.Body = httpReadCloser(body)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_CreateGroup_DuplicateName(t *testing.T) {
	mockService := &MockACLService{
		CreateGroupFunc: func(_ *models.ACLGroup, _ int) error {
			return service.ErrACLGroupNameExists
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{"name": "Existing Group"}`
	req := createAuthenticatedRequest(http.MethodPost, "/api/acl/groups", nil)
	req.Body = httpReadCloser(body)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestACLHandler_UpdateGroup_Success(t *testing.T) {
	mockService := &MockACLService{
		UpdateGroupFunc: func(_ int, _ *models.ACLGroup) error {
			return nil
		},
		GetGroupFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{
				ID:              id,
				Name:            "Updated Group",
				CombinationMode: models.ACLCombinationModeAny,
			}, nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{"name": "Updated Group", "combination_mode": "any"}`
	req := httptest.NewRequest(http.MethodPut, "/api/acl/groups/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_UpdateGroup_NotFound(t *testing.T) {
	mockService := &MockACLService{
		UpdateGroupFunc: func(_ int, _ *models.ACLGroup) error {
			return service.ErrACLGroupNotFound
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{"name": "Updated Group"}`
	req := httptest.NewRequest(http.MethodPut, "/api/acl/groups/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestACLHandler_UpdateGroup_ValidationError(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	body := `{"name": ""}`
	req := httptest.NewRequest(http.MethodPut, "/api/acl/groups/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_DeleteGroup_Success(t *testing.T) {
	mockService := &MockACLService{
		DeleteGroupFunc: func(_ int) error {
			return nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodDelete, "/api/acl/groups/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_DeleteGroup_NotFound(t *testing.T) {
	mockService := &MockACLService{
		DeleteGroupWithSyncFunc: func(_ int, _ service.SyncCallback) error {
			return service.ErrACLGroupNotFound
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodDelete, "/api/acl/groups/999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =============================================================================
// IP Rule Tests
// =============================================================================

func TestACLHandler_ListIPRules_Success(t *testing.T) {
	mockService := &MockACLService{
		GetGroupFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id, Name: "Test"}, nil
		},
	}
	mockRepo := &MockACLRepository{
		ListIPRulesFunc: func(groupID int) ([]models.ACLIPRule, error) {
			return []models.ACLIPRule{
				{ID: 1, ACLGroupID: groupID, RuleType: "allow", CIDR: "192.168.1.0/24"},
				{ID: 2, ACLGroupID: groupID, RuleType: "deny", CIDR: "10.0.0.0/8"},
			}, nil
		},
	}

	r := setupACLTestRouter(mockService, mockRepo)
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups/1/ip-rules", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_ListIPRules_GroupNotFound(t *testing.T) {
	mockService := &MockACLService{
		GetGroupFunc: func(_ int) (*models.ACLGroup, error) {
			return nil, service.ErrACLGroupNotFound
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups/999/ip-rules", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestACLHandler_AddIPRule_Success(t *testing.T) {
	mockService := &MockACLService{
		AddIPRuleFunc: func(_ int, rule *models.ACLIPRule) error {
			rule.ID = 1
			return nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{"rule_type": "allow", "cidr": "192.168.1.0/24", "priority": 10}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/ip-rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestACLHandler_AddIPRule_InvalidCIDR(t *testing.T) {
	mockService := &MockACLService{
		AddIPRuleFunc: func(_ int, _ *models.ACLIPRule) error {
			return service.ErrInvalidCIDR
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{"rule_type": "allow", "cidr": "invalid-cidr", "priority": 10}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/ip-rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_AddIPRule_GroupNotFound(t *testing.T) {
	mockService := &MockACLService{
		AddIPRuleFunc: func(_ int, _ *models.ACLIPRule) error {
			return service.ErrACLGroupNotFound
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{"rule_type": "allow", "cidr": "192.168.1.0/24", "priority": 10}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/999/ip-rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestACLHandler_AddIPRule_MissingRuleType(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	body := `{"cidr": "192.168.1.0/24", "priority": 10}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/ip-rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_AddIPRule_MissingCIDR(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	body := `{"rule_type": "allow", "priority": 10}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/ip-rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_AddIPRule_InvalidRuleType(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	body := `{"rule_type": "invalid", "cidr": "192.168.1.0/24", "priority": 10}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/ip-rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_UpdateIPRule_Success(t *testing.T) {
	mockService := &MockACLService{
		UpdateIPRuleFunc: func(_ int, _ *models.ACLIPRule) error {
			return nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{"rule_type": "deny", "cidr": "10.0.0.0/8", "priority": 5}`
	req := httptest.NewRequest(http.MethodPut, "/api/acl/ip-rules/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_UpdateIPRule_NotFound(t *testing.T) {
	mockService := &MockACLService{
		UpdateIPRuleFunc: func(_ int, _ *models.ACLIPRule) error {
			return service.ErrIPRuleNotFound
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{"rule_type": "deny", "cidr": "10.0.0.0/8", "priority": 5}`
	req := httptest.NewRequest(http.MethodPut, "/api/acl/ip-rules/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestACLHandler_DeleteIPRule_Success(t *testing.T) {
	mockService := &MockACLService{
		DeleteIPRuleFunc: func(_ int) error {
			return nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodDelete, "/api/acl/ip-rules/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_DeleteIPRule_NotFound(t *testing.T) {
	mockService := &MockACLService{
		DeleteIPRuleFunc: func(_ int) error {
			return service.ErrIPRuleNotFound
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodDelete, "/api/acl/ip-rules/999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =============================================================================
// Basic Auth Tests
// =============================================================================

func TestACLHandler_ListBasicAuthUsers_Success(t *testing.T) {
	mockService := &MockACLService{
		GetGroupFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id, Name: "Test"}, nil
		},
	}
	mockRepo := &MockACLRepository{
		ListBasicAuthUsersFunc: func(groupID int) ([]models.ACLBasicAuthUser, error) {
			return []models.ACLBasicAuthUser{
				{ID: 1, ACLGroupID: groupID, Username: "user1"},
				{ID: 2, ACLGroupID: groupID, Username: "user2"},
			}, nil
		},
	}

	r := setupACLTestRouter(mockService, mockRepo)
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups/1/basic-auth", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_AddBasicAuthUser_Success(t *testing.T) {
	mockService := &MockACLService{
		AddBasicAuthUserFunc: func(_ int, _, _ string) error {
			return nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{"username": "testuser", "password": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/basic-auth", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestACLHandler_AddBasicAuthUser_DuplicateUsername(t *testing.T) {
	mockService := &MockACLService{
		AddBasicAuthUserFunc: func(_ int, _, _ string) error {
			return service.ErrBasicAuthUserExists
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{"username": "existinguser", "password": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/basic-auth", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestACLHandler_AddBasicAuthUser_MissingUsername(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	body := `{"password": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/basic-auth", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_AddBasicAuthUser_MissingPassword(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	body := `{"username": "testuser"}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/basic-auth", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_AddBasicAuthUser_PasswordTooShort(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	body := `{"username": "testuser", "password": "short"}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/basic-auth", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_UpdateBasicAuthUser_Success(t *testing.T) {
	mockService := &MockACLService{
		UpdateBasicAuthPasswordFunc: func(_ int, _ string) error {
			return nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{"password": "newpassword123"}`
	req := httptest.NewRequest(http.MethodPut, "/api/acl/basic-auth/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_UpdateBasicAuthUser_NotFound(t *testing.T) {
	mockService := &MockACLService{
		UpdateBasicAuthPasswordFunc: func(_ int, _ string) error {
			return service.ErrBasicAuthUserNotFound
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{"password": "newpassword123"}`
	req := httptest.NewRequest(http.MethodPut, "/api/acl/basic-auth/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestACLHandler_DeleteBasicAuthUser_Success(t *testing.T) {
	mockService := &MockACLService{
		DeleteBasicAuthUserFunc: func(_ int) error {
			return nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodDelete, "/api/acl/basic-auth/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_DeleteBasicAuthUser_NotFound(t *testing.T) {
	mockService := &MockACLService{
		DeleteBasicAuthUserFunc: func(_ int) error {
			return service.ErrBasicAuthUserNotFound
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodDelete, "/api/acl/basic-auth/999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =============================================================================
// External Provider Tests
// =============================================================================

func TestACLHandler_ListExternalProviders_Success(t *testing.T) {
	mockService := &MockACLService{
		GetGroupFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id, Name: "Test"}, nil
		},
	}
	mockRepo := &MockACLRepository{
		ListExternalProvidersFunc: func(groupID int) ([]models.ACLExternalProvider, error) {
			return []models.ACLExternalProvider{
				{ID: 1, ACLGroupID: groupID, Name: "Authelia", ProviderType: "authelia"},
			}, nil
		},
	}

	r := setupACLTestRouter(mockService, mockRepo)
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups/1/providers", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_AddExternalProvider_Success(t *testing.T) {
	mockService := &MockACLService{
		AddExternalProviderFunc: func(_ int, provider *models.ACLExternalProvider) error {
			provider.ID = 1
			return nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{
		"provider_type": "authelia",
		"name": "My Authelia",
		"verify_url": "https://auth.example.com/api/verify"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/providers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestACLHandler_AddExternalProvider_InvalidProviderType(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	body := `{
		"provider_type": "invalid",
		"name": "Invalid Provider",
		"verify_url": "https://auth.example.com/api/verify"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/providers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_AddExternalProvider_MissingName(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	body := `{
		"provider_type": "authelia",
		"verify_url": "https://auth.example.com/api/verify"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/providers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_AddExternalProvider_MissingVerifyURL(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	body := `{
		"provider_type": "authelia",
		"name": "My Provider"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/providers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_UpdateExternalProvider_Success(t *testing.T) {
	mockService := &MockACLService{
		UpdateExternalProviderFunc: func(_ int, _ *models.ACLExternalProvider) error {
			return nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{
		"provider_type": "authentik",
		"name": "Updated Provider",
		"verify_url": "https://auth.example.com/api/verify"
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/acl/providers/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_UpdateExternalProvider_NotFound(t *testing.T) {
	mockService := &MockACLService{
		UpdateExternalProviderFunc: func(_ int, _ *models.ACLExternalProvider) error {
			return service.ErrExternalProviderNotFound
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{
		"provider_type": "authentik",
		"name": "Updated Provider",
		"verify_url": "https://auth.example.com/api/verify"
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/acl/providers/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestACLHandler_DeleteExternalProvider_Success(t *testing.T) {
	mockService := &MockACLService{
		DeleteExternalProviderFunc: func(_ int) error {
			return nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodDelete, "/api/acl/providers/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_DeleteExternalProvider_NotFound(t *testing.T) {
	mockService := &MockACLService{
		DeleteExternalProviderFunc: func(_ int) error {
			return service.ErrExternalProviderNotFound
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodDelete, "/api/acl/providers/999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =============================================================================
// Waygates Auth Tests
// =============================================================================

func TestACLHandler_GetWaygatesAuth_Success(t *testing.T) {
	mockService := &MockACLService{
		GetWaygatesAuthFunc: func(groupID int) (*models.ACLWaygatesAuth, error) {
			return &models.ACLWaygatesAuth{
				ID:         1,
				ACLGroupID: groupID,
				Enabled:    true,
				SessionTTL: 86400,
			}, nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups/1/waygates-auth", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_GetWaygatesAuth_NotConfigured(t *testing.T) {
	mockService := &MockACLService{
		GetWaygatesAuthFunc: func(_ int) (*models.ACLWaygatesAuth, error) {
			return nil, service.ErrWaygatesAuthNotFound
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups/1/waygates-auth", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Not configured should return 200 with null data
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_GetWaygatesAuth_GroupNotFound(t *testing.T) {
	mockService := &MockACLService{
		GetWaygatesAuthFunc: func(_ int) (*models.ACLWaygatesAuth, error) {
			return nil, service.ErrACLGroupNotFound
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups/999/waygates-auth", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestACLHandler_ConfigureWaygatesAuth_Success(t *testing.T) {
	mockService := &MockACLService{
		ConfigureWaygatesAuthFunc: func(_ int, _ *models.ACLWaygatesAuth) error {
			return nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{
		"enabled": true,
		"allowed_users": ["admin", "user1"],
		"require_2fa": false,
		"session_ttl": 3600
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/acl/groups/1/waygates-auth", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_ConfigureWaygatesAuth_GroupNotFound(t *testing.T) {
	mockService := &MockACLService{
		ConfigureWaygatesAuthFunc: func(_ int, _ *models.ACLWaygatesAuth) error {
			return service.ErrACLGroupNotFound
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{"enabled": true}`
	req := httptest.NewRequest(http.MethodPut, "/api/acl/groups/999/waygates-auth", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =============================================================================
// Branding Tests
// =============================================================================

func TestACLHandler_GetBranding_Success(t *testing.T) {
	mockService := &MockACLService{
		GetBrandingFunc: func() (*models.ACLBranding, error) {
			return &models.ACLBranding{
				ID:              1,
				PrimaryColor:    "#3b82f6",
				BackgroundColor: "#ffffff",
				Title:           "Login Required",
			}, nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodGet, "/api/acl/branding", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, true, response["success"])
}

func TestACLHandler_UpdateBranding_Success(t *testing.T) {
	mockService := &MockACLService{
		UpdateBrandingFunc: func(_ *models.ACLBranding) error {
			return nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{
		"primary_color": "#ff0000",
		"background_color": "#000000",
		"title": "Custom Login"
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/acl/branding", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_UpdateBranding_DefaultValues(t *testing.T) {
	var capturedBranding *models.ACLBranding
	mockService := &MockACLService{
		UpdateBrandingFunc: func(branding *models.ACLBranding) error {
			capturedBranding = branding
			return nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	body := `{}`
	req := httptest.NewRequest(http.MethodPut, "/api/acl/branding", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Check that default values were set
	assert.Equal(t, "#3b82f6", capturedBranding.PrimaryColor)
	assert.Equal(t, "#ffffff", capturedBranding.BackgroundColor)
	assert.Equal(t, "Login Required", capturedBranding.Title)
}

// =============================================================================
// Group Usage Tests
// =============================================================================

func TestACLHandler_GetGroupUsage_Success(t *testing.T) {
	mockService := &MockACLService{
		GetGroupFunc: func(id int) (*models.ACLGroup, error) {
			return &models.ACLGroup{ID: id, Name: "Test"}, nil
		},
		GetGroupUsageFunc: func(groupID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{
				{ID: 1, ProxyID: 1, ACLGroupID: groupID, PathPattern: "/*"},
				{ID: 2, ProxyID: 2, ACLGroupID: groupID, PathPattern: "/api/*"},
			}, nil
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups/1/usage", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestACLHandler_GetGroupUsage_GroupNotFound(t *testing.T) {
	mockService := &MockACLService{
		GetGroupFunc: func(_ int) (*models.ACLGroup, error) {
			return nil, service.ErrACLGroupNotFound
		},
	}

	r := setupACLTestRouter(mockService, &MockACLRepository{})
	req := httptest.NewRequest(http.MethodGet, "/api/acl/groups/999/usage", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func TestACLHandler_InvalidJSON(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})
	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/ip-rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_InvalidIDFormat(t *testing.T) {
	r := setupACLTestRouter(&MockACLService{}, &MockACLRepository{})

	testCases := []struct {
		name   string
		method string
		url    string
	}{
		{"invalid group id", http.MethodGet, "/api/acl/groups/abc"},
		{"invalid ip rule id", http.MethodDelete, "/api/acl/ip-rules/xyz"},
		{"invalid basic auth id", http.MethodDelete, "/api/acl/basic-auth/foo"},
		{"invalid provider id", http.MethodDelete, "/api/acl/providers/bar"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// Helper function for creating io.ReadCloser from string
func httpReadCloser(s string) interface {
	Read([]byte) (int, error)
	Close() error
} {
	return &stringReadCloser{s: s}
}

type stringReadCloser struct {
	s   string
	pos int
}

func (src *stringReadCloser) Read(p []byte) (int, error) {
	if src.pos >= len(src.s) {
		return 0, fmt.Errorf("EOF")
	}
	n := copy(p, src.s[src.pos:])
	src.pos += n
	return n, nil
}

func (src *stringReadCloser) Close() error {
	return nil
}

// =============================================================================
// GetAuthOptions Tests (Priority 1 - 0% coverage)
// =============================================================================

// MockSyncService is a mock implementation of SyncServiceInterface for handler tests
type MockSyncService struct {
	SyncProxyByIDFunc  func(proxyID int) error
	SyncProxyByIDCalls []int // Track calls for verification
}

func (m *MockSyncService) GetStatus() service.SyncStatus {
	return service.SyncStatus{}
}

func (m *MockSyncService) FullSync() error {
	return nil
}

func (m *MockSyncService) SyncProxyByID(proxyID int) error {
	m.SyncProxyByIDCalls = append(m.SyncProxyByIDCalls, proxyID)
	if m.SyncProxyByIDFunc != nil {
		return m.SyncProxyByIDFunc(proxyID)
	}
	return nil
}

// setupACLTestRouterWithSync creates a router with ACL handler including sync service for testing
func setupACLTestRouterWithSync(mockService *MockACLService, mockRepo *MockACLRepository, mockSync *MockSyncService) *chi.Mux {
	// Handle nil sync service properly - pass nil interface, not nil pointer wrapped in interface
	var syncService service.SyncServiceInterface
	if mockSync != nil {
		syncService = mockSync
	}
	handler := NewACLHandler(mockService, mockRepo, syncService, nil, nil)
	r := chi.NewRouter()

	// Group routes
	r.Get("/api/acl/groups", handler.ListGroups)
	r.Post("/api/acl/groups", handler.CreateGroup)
	r.Get("/api/acl/groups/{id}", handler.GetGroup)
	r.Put("/api/acl/groups/{id}", handler.UpdateGroup)
	r.Delete("/api/acl/groups/{id}", handler.DeleteGroup)
	r.Get("/api/acl/groups/{id}/usage", handler.GetGroupUsage)

	// IP Rules
	r.Get("/api/acl/groups/{id}/ip-rules", handler.ListIPRules)
	r.Post("/api/acl/groups/{id}/ip-rules", handler.AddIPRule)
	r.Put("/api/acl/ip-rules/{id}", handler.UpdateIPRule)
	r.Delete("/api/acl/ip-rules/{id}", handler.DeleteIPRule)

	// Basic Auth
	r.Get("/api/acl/groups/{id}/basic-auth", handler.ListBasicAuthUsers)
	r.Post("/api/acl/groups/{id}/basic-auth", handler.AddBasicAuthUser)
	r.Put("/api/acl/basic-auth/{id}", handler.UpdateBasicAuthUser)
	r.Delete("/api/acl/basic-auth/{id}", handler.DeleteBasicAuthUser)

	// External Providers
	r.Get("/api/acl/groups/{id}/providers", handler.ListExternalProviders)
	r.Post("/api/acl/groups/{id}/providers", handler.AddExternalProvider)
	r.Put("/api/acl/providers/{id}", handler.UpdateExternalProvider)
	r.Delete("/api/acl/providers/{id}", handler.DeleteExternalProvider)

	// Waygates Auth
	r.Get("/api/acl/groups/{id}/waygates-auth", handler.GetWaygatesAuth)
	r.Put("/api/acl/groups/{id}/waygates-auth", handler.ConfigureWaygatesAuth)

	// Branding
	r.Get("/api/acl/branding", handler.GetBranding)
	r.Put("/api/acl/branding", handler.UpdateBranding)

	// Auth Options (PUBLIC endpoint)
	r.Get("/api/acl/options", handler.GetAuthOptions)

	return r
}

func TestACLHandler_GetAuthOptions_Success(t *testing.T) {
	mockService := &MockACLService{
		GetAuthOptionsForProxyFunc: func(hostname string) (*service.AuthOptionsResponse, error) {
			return &service.AuthOptionsResponse{
				Hostname:     hostname,
				ProxyID:      1,
				RequiresAuth: true,
				WaygatesAuth: &service.UnionWaygatesAuth{Enabled: true},
				OAuthProviders: []service.UnionOAuthProvider{
					{ID: "google", Name: "Google", Enabled: true},
				},
				BasicAuthEnabled: false,
			}, nil
		},
	}

	r := setupACLTestRouterWithSync(mockService, &MockACLRepository{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/acl/options?hostname=example.com", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, true, response["success"])

	// Verify data structure
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "example.com", data["hostname"])
	assert.Equal(t, float64(1), data["proxy_id"])
	assert.Equal(t, true, data["requires_auth"])
}

func TestACLHandler_GetAuthOptions_MissingHostname(t *testing.T) {
	r := setupACLTestRouterWithSync(&MockACLService{}, &MockACLRepository{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/acl/options", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, false, response["success"])

	// Error response uses structured format with code and message
	errData := response["error"].(map[string]interface{})
	assert.Equal(t, "VALIDATION_ERROR", errData["code"])
	assert.Contains(t, errData["message"], "hostname")
}

func TestACLHandler_GetAuthOptions_EmptyHostname(t *testing.T) {
	r := setupACLTestRouterWithSync(&MockACLService{}, &MockACLRepository{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/acl/options?hostname=", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestACLHandler_GetAuthOptions_ProxyNotFound(t *testing.T) {
	mockService := &MockACLService{
		GetAuthOptionsForProxyFunc: func(_ string) (*service.AuthOptionsResponse, error) {
			return nil, fmt.Errorf("proxy not found")
		},
	}

	r := setupACLTestRouterWithSync(mockService, &MockACLRepository{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/acl/options?hostname=unknown.com", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, false, response["success"])

	// Error response uses structured format with code and message
	errData := response["error"].(map[string]interface{})
	assert.Equal(t, "NOT_FOUND", errData["code"])
	assert.Contains(t, errData["message"], "proxy not found")
}

func TestACLHandler_GetAuthOptions_NoAuthRequired(t *testing.T) {
	mockService := &MockACLService{
		GetAuthOptionsForProxyFunc: func(hostname string) (*service.AuthOptionsResponse, error) {
			return &service.AuthOptionsResponse{
				Hostname:     hostname,
				ProxyID:      2,
				RequiresAuth: false,
			}, nil
		},
	}

	r := setupACLTestRouterWithSync(mockService, &MockACLRepository{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/acl/options?hostname=public.example.com", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, true, response["success"])

	data := response["data"].(map[string]interface{})
	assert.Equal(t, false, data["requires_auth"])
}

func TestACLHandler_GetAuthOptions_BasicAuthOnly(t *testing.T) {
	mockService := &MockACLService{
		GetAuthOptionsForProxyFunc: func(hostname string) (*service.AuthOptionsResponse, error) {
			return &service.AuthOptionsResponse{
				Hostname:         hostname,
				ProxyID:          3,
				RequiresAuth:     true,
				BasicAuthEnabled: true,
			}, nil
		},
	}

	r := setupACLTestRouterWithSync(mockService, &MockACLRepository{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/acl/options?hostname=basic.example.com", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, true, data["basic_auth_enabled"])
	assert.Equal(t, true, data["requires_auth"])
}

// =============================================================================
// syncProxiesUsingGroup Tests (Priority 1 - 0% coverage)
// =============================================================================

func TestACLHandler_SyncProxiesUsingGroup_MultipleProxies(t *testing.T) {
	mockSync := &MockSyncService{}
	mockService := &MockACLService{
		GetGroupUsageFunc: func(groupID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{
				{ID: 1, ProxyID: 10, ACLGroupID: groupID},
				{ID: 2, ProxyID: 20, ACLGroupID: groupID},
				{ID: 3, ProxyID: 30, ACLGroupID: groupID},
			}, nil
		},
		AddIPRuleFunc: func(_ int, rule *models.ACLIPRule) error {
			rule.ID = 1
			return nil
		},
	}

	r := setupACLTestRouterWithSync(mockService, &MockACLRepository{}, mockSync)
	body := `{"rule_type": "allow", "cidr": "192.168.1.0/24", "priority": 10}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/ip-rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Verify that all 3 proxies were synced
	assert.Equal(t, 3, len(mockSync.SyncProxyByIDCalls))
	assert.Contains(t, mockSync.SyncProxyByIDCalls, 10)
	assert.Contains(t, mockSync.SyncProxyByIDCalls, 20)
	assert.Contains(t, mockSync.SyncProxyByIDCalls, 30)
}

func TestACLHandler_SyncProxiesUsingGroup_SyncErrors_ContinuesWithOthers(t *testing.T) {
	syncErrorCount := 0
	mockSync := &MockSyncService{
		SyncProxyByIDFunc: func(proxyID int) error {
			// First proxy fails, but we continue
			if proxyID == 10 {
				syncErrorCount++
				return fmt.Errorf("sync failed for proxy %d", proxyID)
			}
			return nil
		},
	}
	mockService := &MockACLService{
		GetGroupUsageFunc: func(groupID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{
				{ID: 1, ProxyID: 10, ACLGroupID: groupID}, // This one will fail
				{ID: 2, ProxyID: 20, ACLGroupID: groupID}, // Should still sync
				{ID: 3, ProxyID: 30, ACLGroupID: groupID}, // Should still sync
			}, nil
		},
		AddIPRuleFunc: func(_ int, rule *models.ACLIPRule) error {
			rule.ID = 1
			return nil
		},
	}

	r := setupACLTestRouterWithSync(mockService, &MockACLRepository{}, mockSync)
	body := `{"rule_type": "deny", "cidr": "10.0.0.0/8", "priority": 5}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/ip-rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Request should still succeed even though one proxy sync failed
	assert.Equal(t, http.StatusCreated, w.Code)

	// All 3 proxies should have been attempted
	assert.Equal(t, 3, len(mockSync.SyncProxyByIDCalls))

	// One error should have occurred
	assert.Equal(t, 1, syncErrorCount)
}

func TestACLHandler_SyncProxiesUsingGroup_NilSyncService(t *testing.T) {
	// When syncService is nil, we should not panic and operations should still work
	mockService := &MockACLService{
		GetGroupUsageFunc: func(groupID int) ([]models.ProxyACLAssignment, error) {
			return []models.ProxyACLAssignment{
				{ID: 1, ProxyID: 10, ACLGroupID: groupID},
			}, nil
		},
		AddIPRuleFunc: func(_ int, rule *models.ACLIPRule) error {
			rule.ID = 1
			return nil
		},
	}

	// Pass nil for sync service
	r := setupACLTestRouterWithSync(mockService, &MockACLRepository{}, nil)
	body := `{"rule_type": "allow", "cidr": "172.16.0.0/12", "priority": 10}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/ip-rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Should not panic and should complete successfully
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestACLHandler_SyncProxiesUsingGroup_GetGroupUsageError(t *testing.T) {
	mockSync := &MockSyncService{}
	mockService := &MockACLService{
		GetGroupUsageFunc: func(_ int) ([]models.ProxyACLAssignment, error) {
			// Error getting group usage - should be handled gracefully
			return nil, fmt.Errorf("database error")
		},
		AddIPRuleFunc: func(_ int, rule *models.ACLIPRule) error {
			rule.ID = 1
			return nil
		},
	}

	r := setupACLTestRouterWithSync(mockService, &MockACLRepository{}, mockSync)
	body := `{"rule_type": "bypass", "cidr": "192.168.100.0/24", "priority": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/ip-rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Request should still succeed (sync is best-effort, not blocking)
	assert.Equal(t, http.StatusCreated, w.Code)

	// No proxies should have been synced due to the error
	assert.Equal(t, 0, len(mockSync.SyncProxyByIDCalls))
}

func TestACLHandler_SyncProxiesUsingGroup_NoProxiesUsing(t *testing.T) {
	mockSync := &MockSyncService{}
	mockService := &MockACLService{
		GetGroupUsageFunc: func(_ int) ([]models.ProxyACLAssignment, error) {
			// No proxies using this group
			return []models.ProxyACLAssignment{}, nil
		},
		AddIPRuleFunc: func(_ int, rule *models.ACLIPRule) error {
			rule.ID = 1
			return nil
		},
	}

	r := setupACLTestRouterWithSync(mockService, &MockACLRepository{}, mockSync)
	body := `{"rule_type": "allow", "cidr": "10.10.0.0/16", "priority": 100}`
	req := httptest.NewRequest(http.MethodPost, "/api/acl/groups/1/ip-rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// No proxies should have been synced
	assert.Equal(t, 0, len(mockSync.SyncProxyByIDCalls))
}
