package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service"
)

// =============================================================================
// Mock Implementations for ACL Handler Tests
// =============================================================================

// aclMockACLService is a mock implementation of ACLServiceInterface for ACL handler tests
type aclMockACLService struct {
	// Group Management
	CreateGroupFunc         func(group *models.ACLGroup, userID int) error
	GetGroupFunc            func(id int) (*models.ACLGroup, error)
	GetGroupByNameFunc      func(name string) (*models.ACLGroup, error)
	ListGroupsFunc          func(req service.ListACLGroupsRequest) (*models.ACLGroupListResponse, error)
	UpdateGroupFunc         func(id int, group *models.ACLGroup) error
	DeleteGroupFunc         func(id int) error
	DeleteGroupWithSyncFunc func(id int, syncCallback service.SyncCallback) error

	// IP Rules
	AddIPRuleFunc    func(groupID int, rule *models.ACLIPRule) error
	UpdateIPRuleFunc func(ruleID int, rule *models.ACLIPRule) error
	DeleteIPRuleFunc func(ruleID int) error

	// Basic Auth
	AddBasicAuthUserFunc        func(groupID int, username, password string) error
	UpdateBasicAuthPasswordFunc func(userID int, password string) error
	DeleteBasicAuthUserFunc     func(userID int) error

	// External Providers
	AddExternalProviderFunc    func(groupID int, provider *models.ACLExternalProvider) error
	UpdateExternalProviderFunc func(providerID int, provider *models.ACLExternalProvider) error
	DeleteExternalProviderFunc func(providerID int) error

	// Waygates Auth Config
	GetWaygatesAuthFunc       func(groupID int) (*models.ACLWaygatesAuth, error)
	ConfigureWaygatesAuthFunc func(groupID int, config *models.ACLWaygatesAuth) error

	// Proxy Assignment
	AssignToProxyFunc         func(groupID, proxyID int, path string, priority int, enabled bool) error
	UpdateProxyAssignmentFunc func(assignmentID int, path string, priority int, enabled bool) error
	RemoveFromProxyFunc       func(groupID, proxyID int) error
	GetProxyACLFunc           func(proxyID int) ([]models.ProxyACLAssignment, error)
	GetGroupUsageFunc         func(groupID int) ([]models.ProxyACLAssignment, error)

	// Branding
	GetBrandingFunc    func() (*models.ACLBranding, error)
	UpdateBrandingFunc func(branding *models.ACLBranding) error

	// OAuth Provider Restrictions
	GetOAuthProviderRestrictionsFunc   func(groupID int) ([]models.ACLOAuthProviderRestriction, error)
	SetOAuthProviderRestrictionFunc    func(groupID int, provider string, allowedEmails, allowedDomains []string, enabled bool) error
	DeleteOAuthProviderRestrictionFunc func(groupID int, provider string) error

	// Access Verification
	VerifyAccessFunc func(req *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error)

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

// Implement ACLServiceInterface
func (m *aclMockACLService) CreateGroup(group *models.ACLGroup, userID int) error {
	if m.CreateGroupFunc != nil {
		return m.CreateGroupFunc(group, userID)
	}
	return nil
}

func (m *aclMockACLService) GetGroup(id int) (*models.ACLGroup, error) {
	if m.GetGroupFunc != nil {
		return m.GetGroupFunc(id)
	}
	return &models.ACLGroup{ID: id, Name: "test-group"}, nil
}

func (m *aclMockACLService) GetGroupByName(name string) (*models.ACLGroup, error) {
	if m.GetGroupByNameFunc != nil {
		return m.GetGroupByNameFunc(name)
	}
	return nil, nil
}

func (m *aclMockACLService) ListGroups(req service.ListACLGroupsRequest) (*models.ACLGroupListResponse, error) {
	if m.ListGroupsFunc != nil {
		return m.ListGroupsFunc(req)
	}
	return &models.ACLGroupListResponse{}, nil
}

func (m *aclMockACLService) UpdateGroup(id int, group *models.ACLGroup) error {
	if m.UpdateGroupFunc != nil {
		return m.UpdateGroupFunc(id, group)
	}
	return nil
}

func (m *aclMockACLService) DeleteGroup(id int) error {
	if m.DeleteGroupFunc != nil {
		return m.DeleteGroupFunc(id)
	}
	return nil
}

func (m *aclMockACLService) DeleteGroupWithSync(id int, syncCallback service.SyncCallback) error {
	if m.DeleteGroupWithSyncFunc != nil {
		return m.DeleteGroupWithSyncFunc(id, syncCallback)
	}
	return nil
}

func (m *aclMockACLService) AddIPRule(groupID int, rule *models.ACLIPRule) error {
	if m.AddIPRuleFunc != nil {
		return m.AddIPRuleFunc(groupID, rule)
	}
	return nil
}

func (m *aclMockACLService) UpdateIPRule(ruleID int, rule *models.ACLIPRule) error {
	if m.UpdateIPRuleFunc != nil {
		return m.UpdateIPRuleFunc(ruleID, rule)
	}
	return nil
}

func (m *aclMockACLService) DeleteIPRule(ruleID int) error {
	if m.DeleteIPRuleFunc != nil {
		return m.DeleteIPRuleFunc(ruleID)
	}
	return nil
}

func (m *aclMockACLService) AddBasicAuthUser(groupID int, username, password string) error {
	if m.AddBasicAuthUserFunc != nil {
		return m.AddBasicAuthUserFunc(groupID, username, password)
	}
	return nil
}

func (m *aclMockACLService) UpdateBasicAuthPassword(userID int, password string) error {
	if m.UpdateBasicAuthPasswordFunc != nil {
		return m.UpdateBasicAuthPasswordFunc(userID, password)
	}
	return nil
}

func (m *aclMockACLService) DeleteBasicAuthUser(userID int) error {
	if m.DeleteBasicAuthUserFunc != nil {
		return m.DeleteBasicAuthUserFunc(userID)
	}
	return nil
}

func (m *aclMockACLService) AddExternalProvider(groupID int, provider *models.ACLExternalProvider) error {
	if m.AddExternalProviderFunc != nil {
		return m.AddExternalProviderFunc(groupID, provider)
	}
	return nil
}

func (m *aclMockACLService) UpdateExternalProvider(providerID int, provider *models.ACLExternalProvider) error {
	if m.UpdateExternalProviderFunc != nil {
		return m.UpdateExternalProviderFunc(providerID, provider)
	}
	return nil
}

func (m *aclMockACLService) DeleteExternalProvider(providerID int) error {
	if m.DeleteExternalProviderFunc != nil {
		return m.DeleteExternalProviderFunc(providerID)
	}
	return nil
}

func (m *aclMockACLService) GetWaygatesAuth(groupID int) (*models.ACLWaygatesAuth, error) {
	if m.GetWaygatesAuthFunc != nil {
		return m.GetWaygatesAuthFunc(groupID)
	}
	return nil, nil
}

func (m *aclMockACLService) ConfigureWaygatesAuth(groupID int, config *models.ACLWaygatesAuth) error {
	if m.ConfigureWaygatesAuthFunc != nil {
		return m.ConfigureWaygatesAuthFunc(groupID, config)
	}
	return nil
}

func (m *aclMockACLService) AssignToProxy(groupID, proxyID int, path string, priority int, enabled bool) error {
	if m.AssignToProxyFunc != nil {
		return m.AssignToProxyFunc(groupID, proxyID, path, priority, enabled)
	}
	return nil
}

func (m *aclMockACLService) UpdateProxyAssignment(assignmentID int, path string, priority int, enabled bool) error {
	if m.UpdateProxyAssignmentFunc != nil {
		return m.UpdateProxyAssignmentFunc(assignmentID, path, priority, enabled)
	}
	return nil
}

func (m *aclMockACLService) RemoveFromProxy(groupID, proxyID int) error {
	if m.RemoveFromProxyFunc != nil {
		return m.RemoveFromProxyFunc(groupID, proxyID)
	}
	return nil
}

func (m *aclMockACLService) GetProxyACL(proxyID int) ([]models.ProxyACLAssignment, error) {
	if m.GetProxyACLFunc != nil {
		return m.GetProxyACLFunc(proxyID)
	}
	return nil, nil
}

func (m *aclMockACLService) GetGroupUsage(groupID int) ([]models.ProxyACLAssignment, error) {
	if m.GetGroupUsageFunc != nil {
		return m.GetGroupUsageFunc(groupID)
	}
	return nil, nil
}

func (m *aclMockACLService) GetBranding() (*models.ACLBranding, error) {
	if m.GetBrandingFunc != nil {
		return m.GetBrandingFunc()
	}
	return &models.ACLBranding{}, nil
}

func (m *aclMockACLService) UpdateBranding(branding *models.ACLBranding) error {
	if m.UpdateBrandingFunc != nil {
		return m.UpdateBrandingFunc(branding)
	}
	return nil
}

func (m *aclMockACLService) GetOAuthProviderRestrictions(groupID int) ([]models.ACLOAuthProviderRestriction, error) {
	if m.GetOAuthProviderRestrictionsFunc != nil {
		return m.GetOAuthProviderRestrictionsFunc(groupID)
	}
	return []models.ACLOAuthProviderRestriction{}, nil
}

func (m *aclMockACLService) SetOAuthProviderRestriction(groupID int, provider string, allowedEmails, allowedDomains []string, enabled bool) error {
	if m.SetOAuthProviderRestrictionFunc != nil {
		return m.SetOAuthProviderRestrictionFunc(groupID, provider, allowedEmails, allowedDomains, enabled)
	}
	return nil
}

func (m *aclMockACLService) DeleteOAuthProviderRestriction(groupID int, provider string) error {
	if m.DeleteOAuthProviderRestrictionFunc != nil {
		return m.DeleteOAuthProviderRestrictionFunc(groupID, provider)
	}
	return nil
}

func (m *aclMockACLService) VerifyAccess(req *service.ACLVerifyRequest) (*service.ACLVerifyResponse, error) {
	if m.VerifyAccessFunc != nil {
		return m.VerifyAccessFunc(req)
	}
	return nil, nil
}

func (m *aclMockACLService) GetAuthOptionsForProxy(hostname string) (*service.AuthOptionsResponse, error) {
	if m.GetAuthOptionsForProxyFunc != nil {
		return m.GetAuthOptionsForProxyFunc(hostname)
	}
	return nil, nil
}

func (m *aclMockACLService) CreateSession(userID int, proxyID *int, ip, userAgent string, ttl int) (*models.ACLSession, error) {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(userID, proxyID, ip, userAgent, ttl)
	}
	return &models.ACLSession{SessionToken: "test-token", ExpiresAt: time.Now().Add(24 * time.Hour)}, nil
}

func (m *aclMockACLService) CreateOAuthSession(email, provider string, proxyID *int, ip, userAgent string, ttl int) (*models.ACLSession, error) {
	if m.CreateOAuthSessionFunc != nil {
		return m.CreateOAuthSessionFunc(email, provider, proxyID, ip, userAgent, ttl)
	}
	return &models.ACLSession{SessionToken: "test-oauth-token", ExpiresAt: time.Now().Add(24 * time.Hour)}, nil
}

func (m *aclMockACLService) CreateSessionWithParams(params service.CreateSessionParams) (*models.ACLSession, error) {
	if m.CreateSessionWithParamsFunc != nil {
		return m.CreateSessionWithParamsFunc(params)
	}
	return &models.ACLSession{SessionToken: "test-token", ExpiresAt: time.Now().Add(24 * time.Hour)}, nil
}

func (m *aclMockACLService) ValidateSession(token string) (*models.ACLSession, error) {
	if m.ValidateSessionFunc != nil {
		return m.ValidateSessionFunc(token)
	}
	return nil, nil
}

func (m *aclMockACLService) RevokeSession(token string) error {
	if m.RevokeSessionFunc != nil {
		return m.RevokeSessionFunc(token)
	}
	return nil
}

func (m *aclMockACLService) RevokeUserSessions(userID int) error {
	if m.RevokeUserSessionsFunc != nil {
		return m.RevokeUserSessionsFunc(userID)
	}
	return nil
}

func (m *aclMockACLService) CleanupExpiredSessions() (int64, error) {
	if m.CleanupExpiredSessionsFunc != nil {
		return m.CleanupExpiredSessionsFunc()
	}
	return 0, nil
}

var _ service.ACLServiceInterface = (*aclMockACLService)(nil)

// aclMockAuditService is a mock implementation of AuditServiceInterface
type aclMockAuditService struct{}

func (m *aclMockAuditService) LogEvent(_ context.Context, _ models.AuditEvent) error { return nil }
func (m *aclMockAuditService) GetConfig() (*models.AuditConfig, error)               { return nil, nil }
func (m *aclMockAuditService) SetConfig(_ *models.AuditConfig) error                 { return nil }
func (m *aclMockAuditService) InvalidateConfigCache()                                {}
func (m *aclMockAuditService) ListAuditLogs(_ repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
	return &models.AuditLogListResponse{}, nil
}
func (m *aclMockAuditService) GetAuditLogByID(_ int) (*models.AuditLog, error) { return nil, nil }
func (m *aclMockAuditService) GetStats() (*models.AuditLogStats, error)        { return nil, nil }
func (m *aclMockAuditService) LogProxyCreate(_ context.Context, _ int, _ *models.Proxy, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogProxyUpdate(_ context.Context, _ int, _ *models.Proxy, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogProxyDelete(_ context.Context, _, _ int, _, _, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogProxyEnable(_ context.Context, _ int, _ *models.Proxy, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogProxyDisable(_ context.Context, _ int, _ *models.Proxy, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogProxyGroupCreate(_ context.Context, _ int, _ *models.ProxyGroup, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogProxyGroupUpdate(_ context.Context, _ int, _, _ *models.ProxyGroup, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogProxyGroupDelete(_ context.Context, _, _ int, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogProxyGroupRehome(_ context.Context, _, _ int, _, _ string, _ []int, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogLogin(_ context.Context, _ int, _, _, _ string) error { return nil }
func (m *aclMockAuditService) LogLoginFailed(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogLogout(_ context.Context, _ int, _, _, _ string) error { return nil }
func (m *aclMockAuditService) LogRegister(_ context.Context, _ int, _, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogPasswordChange(_ context.Context, _ int, _, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogSettingsUpdate(_ context.Context, _ int, _, _, _, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogSyncStarted(_ context.Context) error                   { return nil }
func (m *aclMockAuditService) LogSyncCompleted(_ context.Context, _ int) error          { return nil }
func (m *aclMockAuditService) LogSyncFailed(_ context.Context, _ string) error          { return nil }
func (m *aclMockAuditService) LogSystemStartup(_ context.Context) error                 { return nil }
func (m *aclMockAuditService) LogCaddyReload(_ context.Context, _ bool, _ string) error { return nil }
func (m *aclMockAuditService) LogACLGroupCreate(_ context.Context, _ int, _ *models.ACLGroup, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLGroupUpdate(_ context.Context, _ int, _ *models.ACLGroup, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLGroupDelete(_ context.Context, _, _ int, _, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLIPRuleAdd(_ context.Context, _, _ int, _ string, _ *models.ACLIPRule, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLIPRuleUpdate(_ context.Context, _ int, _ *models.ACLIPRule, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLIPRuleDelete(_ context.Context, _, _ int, _, _, _, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLBasicAuthAdd(_ context.Context, _, _ int, _, _, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLBasicAuthUpdate(_ context.Context, _, _ int, _, _, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLBasicAuthDelete(_ context.Context, _, _ int, _, _, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLWaygatesAuthUpdate(_ context.Context, _, _ int, _ string, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLAssignmentCreate(_ context.Context, _, _ int, _ string, _ int, _, _, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLAssignmentUpdate(_ context.Context, _ int, _ *models.ProxyACLAssignment, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLAssignmentDelete(_ context.Context, _, _ int, _ string, _ int, _, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLBrandingUpdate(_ context.Context, _ int, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLSessionRevoke(_ context.Context, _, _ int, _, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLOAuthRestrictionSet(_ context.Context, _, _ int, _, _ string, _ *models.ACLOAuthProviderRestriction, _ bool, _, _ []string, _, _ string) error {
	return nil
}
func (m *aclMockAuditService) LogACLOAuthRestrictionDelete(_ context.Context, _, _ int, _, _, _, _ string) error {
	return nil
}

var _ service.AuditServiceInterface = (*aclMockAuditService)(nil)

// =============================================================================
// Test Helpers
// =============================================================================

func createTestACLHandler(t *testing.T) (*ACLHandler, *aclMockACLService) {
	t.Helper()

	mockService := &aclMockACLService{}
	mockAudit := &aclMockAuditService{}
	logger := zap.NewNop()

	handler := NewACLHandler(mockService, nil, nil, mockAudit, logger)

	return handler, mockService
}

func setACLChiURLParams(r *http.Request, params map[string]string) *http.Request {
	ctx := chi.NewRouteContext()
	for key, value := range params {
		ctx.URLParams.Add(key, value)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, ctx))
}

// =============================================================================
// TestGetOAuthProviderRestrictions
// =============================================================================

func TestACLHandler_GetOAuthProviderRestrictions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		groupID        string
		setupMock      func(*aclMockACLService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:    "success - returns restrictions",
			groupID: "1",
			setupMock: func(m *aclMockACLService) {
				m.GetOAuthProviderRestrictionsFunc = func(groupID int) ([]models.ACLOAuthProviderRestriction, error) {
					return []models.ACLOAuthProviderRestriction{
						{
							ID:             1,
							ACLGroupID:     groupID,
							Provider:       "google",
							AllowedEmails:  []string{"user@example.com"},
							AllowedDomains: []string{"example.com"},
							Enabled:        true,
						},
						{
							ID:             2,
							ACLGroupID:     groupID,
							Provider:       "github",
							AllowedEmails:  []string{},
							AllowedDomains: []string{"github.com"},
							Enabled:        false,
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response struct {
					Success bool                                 `json:"success"`
					Data    []models.ACLOAuthProviderRestriction `json:"data"`
				}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.True(t, response.Success)
				assert.Len(t, response.Data, 2)
				assert.Equal(t, "google", response.Data[0].Provider)
				assert.Equal(t, "github", response.Data[1].Provider)
			},
		},
		{
			name:           "invalid group ID",
			groupID:        "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "group not found",
			groupID: "999",
			setupMock: func(m *aclMockACLService) {
				m.GetOAuthProviderRestrictionsFunc = func(_ int) ([]models.ACLOAuthProviderRestriction, error) {
					return nil, service.ErrACLGroupNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:    "empty restrictions",
			groupID: "1",
			setupMock: func(m *aclMockACLService) {
				m.GetOAuthProviderRestrictionsFunc = func(_ int) ([]models.ACLOAuthProviderRestriction, error) {
					return []models.ACLOAuthProviderRestriction{}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response struct {
					Success bool                                 `json:"success"`
					Data    []models.ACLOAuthProviderRestriction `json:"data"`
				}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.True(t, response.Success)
				assert.Len(t, response.Data, 0)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler, mockService := createTestACLHandler(t)

			if tc.setupMock != nil {
				tc.setupMock(mockService)
			}

			req := httptest.NewRequest(http.MethodGet, "/api/acl/groups/"+tc.groupID+"/oauth-restrictions", nil)
			req = setACLChiURLParams(req, map[string]string{"id": tc.groupID})

			rec := httptest.NewRecorder()
			handler.GetOAuthProviderRestrictions(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)

			if tc.checkResponse != nil {
				tc.checkResponse(t, rec)
			}
		})
	}
}

// =============================================================================
// TestSetOAuthProviderRestriction
// =============================================================================

func TestACLHandler_SetOAuthProviderRestriction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		groupID        string
		provider       string
		requestBody    interface{}
		setupMock      func(*aclMockACLService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:     "success - set restriction",
			groupID:  "1",
			provider: "google",
			requestBody: SetOAuthProviderRestrictionRequest{
				AllowedEmails:  []string{"user@example.com", "admin@example.com"},
				AllowedDomains: []string{"example.com"},
				Enabled:        true,
			},
			setupMock: func(m *aclMockACLService) {
				m.GetGroupFunc = func(_ int) (*models.ACLGroup, error) {
					return &models.ACLGroup{ID: 1, Name: "test-group"}, nil
				}
				m.SetOAuthProviderRestrictionFunc = func(_ int, _ string, _, _ []string, _ bool) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response struct {
					Success bool   `json:"success"`
					Message string `json:"message"`
				}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.True(t, response.Success)
				assert.Contains(t, response.Message, "successfully")
			},
		},
		{
			name:           "invalid group ID",
			groupID:        "invalid",
			provider:       "google",
			requestBody:    SetOAuthProviderRestrictionRequest{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid provider",
			groupID:  "1",
			provider: "invalid-provider",
			requestBody: SetOAuthProviderRestrictionRequest{
				AllowedEmails: []string{"user@example.com"},
				Enabled:       true,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "group not found",
			groupID:  "999",
			provider: "google",
			requestBody: SetOAuthProviderRestrictionRequest{
				AllowedEmails: []string{"user@example.com"},
				Enabled:       true,
			},
			setupMock: func(m *aclMockACLService) {
				m.GetGroupFunc = func(_ int) (*models.ACLGroup, error) {
					return &models.ACLGroup{ID: 999, Name: "test-group"}, nil
				}
				m.SetOAuthProviderRestrictionFunc = func(_ int, _ string, _, _ []string, _ bool) error {
					return service.ErrACLGroupNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid request body",
			groupID:        "1",
			provider:       "google",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "empty emails and domains",
			groupID:  "1",
			provider: "github",
			requestBody: SetOAuthProviderRestrictionRequest{
				AllowedEmails:  []string{},
				AllowedDomains: []string{},
				Enabled:        false,
			},
			setupMock: func(m *aclMockACLService) {
				m.GetGroupFunc = func(_ int) (*models.ACLGroup, error) {
					return &models.ACLGroup{ID: 1, Name: "test-group"}, nil
				}
				m.SetOAuthProviderRestrictionFunc = func(_ int, _ string, _, _ []string, _ bool) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler, mockService := createTestACLHandler(t)

			if tc.setupMock != nil {
				tc.setupMock(mockService)
			}

			var body []byte
			switch v := tc.requestBody.(type) {
			case string:
				body = []byte(v)
			default:
				var err error
				body, err = json.Marshal(tc.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPut, "/api/acl/groups/"+tc.groupID+"/oauth-restrictions/"+tc.provider, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = setACLChiURLParams(req, map[string]string{"id": tc.groupID, "provider": tc.provider})

			rec := httptest.NewRecorder()
			handler.SetOAuthProviderRestriction(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)

			if tc.checkResponse != nil {
				tc.checkResponse(t, rec)
			}
		})
	}
}

// =============================================================================
// TestDeleteOAuthProviderRestriction
// =============================================================================

func TestACLHandler_DeleteOAuthProviderRestriction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		groupID        string
		provider       string
		setupMock      func(*aclMockACLService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:     "success - delete restriction",
			groupID:  "1",
			provider: "google",
			setupMock: func(m *aclMockACLService) {
				m.GetGroupFunc = func(_ int) (*models.ACLGroup, error) {
					return &models.ACLGroup{ID: 1, Name: "test-group"}, nil
				}
				m.DeleteOAuthProviderRestrictionFunc = func(_ int, _ string) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response struct {
					Success bool   `json:"success"`
					Message string `json:"message"`
				}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.True(t, response.Success)
				assert.Contains(t, response.Message, "deleted")
			},
		},
		{
			name:           "invalid group ID",
			groupID:        "invalid",
			provider:       "google",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid provider",
			groupID:        "1",
			provider:       "invalid-provider",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "group not found",
			groupID:  "999",
			provider: "google",
			setupMock: func(m *aclMockACLService) {
				m.GetGroupFunc = func(_ int) (*models.ACLGroup, error) {
					return &models.ACLGroup{ID: 999, Name: "test-group"}, nil
				}
				m.DeleteOAuthProviderRestrictionFunc = func(_ int, _ string) error {
					return service.ErrACLGroupNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "restriction not found",
			groupID:  "1",
			provider: "github",
			setupMock: func(m *aclMockACLService) {
				m.GetGroupFunc = func(_ int) (*models.ACLGroup, error) {
					return &models.ACLGroup{ID: 1, Name: "test-group"}, nil
				}
				m.DeleteOAuthProviderRestrictionFunc = func(_ int, _ string) error {
					return service.ErrOAuthProviderRestrictionNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler, mockService := createTestACLHandler(t)

			if tc.setupMock != nil {
				tc.setupMock(mockService)
			}

			req := httptest.NewRequest(http.MethodDelete, "/api/acl/groups/"+tc.groupID+"/oauth-restrictions/"+tc.provider, nil)
			req = setACLChiURLParams(req, map[string]string{"id": tc.groupID, "provider": tc.provider})

			rec := httptest.NewRecorder()
			handler.DeleteOAuthProviderRestriction(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)

			if tc.checkResponse != nil {
				tc.checkResponse(t, rec)
			}
		})
	}
}

// =============================================================================
// TestBuildACLGroupChanges
// =============================================================================

func TestBuildACLGroupChanges(t *testing.T) {
	t.Parallel()

	// Helper to create string pointer
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name           string
		oldGroup       *models.ACLGroup
		newGroup       *models.ACLGroup
		expectedKeys   []string
		unexpectedKeys []string
	}{
		{
			name: "no changes",
			oldGroup: &models.ACLGroup{
				Name:            "test-group",
				Description:     strPtr("Test description"),
				CombinationMode: "any",
			},
			newGroup: &models.ACLGroup{
				Name:            "test-group",
				Description:     strPtr("Test description"),
				CombinationMode: "any",
			},
			expectedKeys:   []string{},
			unexpectedKeys: []string{"name", "description", "combination_mode"},
		},
		{
			name: "name changed",
			oldGroup: &models.ACLGroup{
				Name:            "old-name",
				Description:     strPtr("Test description"),
				CombinationMode: "any",
			},
			newGroup: &models.ACLGroup{
				Name:            "new-name",
				Description:     strPtr("Test description"),
				CombinationMode: "any",
			},
			expectedKeys:   []string{"name"},
			unexpectedKeys: []string{"description", "combination_mode"},
		},
		{
			name: "description changed",
			oldGroup: &models.ACLGroup{
				Name:            "test-group",
				Description:     strPtr("Old description"),
				CombinationMode: "any",
			},
			newGroup: &models.ACLGroup{
				Name:            "test-group",
				Description:     strPtr("New description"),
				CombinationMode: "any",
			},
			expectedKeys:   []string{"description"},
			unexpectedKeys: []string{"name", "combination_mode"},
		},
		{
			name: "combination mode changed",
			oldGroup: &models.ACLGroup{
				Name:            "test-group",
				Description:     strPtr("Test description"),
				CombinationMode: "any",
			},
			newGroup: &models.ACLGroup{
				Name:            "test-group",
				Description:     strPtr("Test description"),
				CombinationMode: "all",
			},
			expectedKeys:   []string{"combination_mode"},
			unexpectedKeys: []string{"name", "description"},
		},
		{
			name: "multiple changes",
			oldGroup: &models.ACLGroup{
				Name:            "old-name",
				Description:     strPtr("Old description"),
				CombinationMode: "any",
			},
			newGroup: &models.ACLGroup{
				Name:            "new-name",
				Description:     strPtr("New description"),
				CombinationMode: "all",
			},
			expectedKeys:   []string{"name", "description", "combination_mode"},
			unexpectedKeys: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			changes := buildACLGroupChanges(tc.oldGroup, tc.newGroup)

			for _, key := range tc.expectedKeys {
				assert.Contains(t, changes, key, "Expected key %s to be present", key)
			}

			for _, key := range tc.unexpectedKeys {
				assert.NotContains(t, changes, key, "Unexpected key %s should not be present", key)
			}
		})
	}
}

// =============================================================================
// TestBuildIPRuleChanges
// =============================================================================

func TestBuildIPRuleChanges(t *testing.T) {
	t.Parallel()

	// Helper to create string pointer
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name           string
		oldRule        *models.ACLIPRule
		newRule        *models.ACLIPRule
		expectedKeys   []string
		unexpectedKeys []string
	}{
		{
			name: "no changes",
			oldRule: &models.ACLIPRule{
				RuleType:    "allow",
				CIDR:        "192.168.1.0/24",
				Description: strPtr("Test rule"),
				Priority:    10,
			},
			newRule: &models.ACLIPRule{
				RuleType:    "allow",
				CIDR:        "192.168.1.0/24",
				Description: strPtr("Test rule"),
				Priority:    10,
			},
			expectedKeys:   []string{},
			unexpectedKeys: []string{"rule_type", "cidr", "description", "priority"},
		},
		{
			name: "rule type changed",
			oldRule: &models.ACLIPRule{
				RuleType: "allow",
				CIDR:     "192.168.1.0/24",
			},
			newRule: &models.ACLIPRule{
				RuleType: "deny",
				CIDR:     "192.168.1.0/24",
			},
			expectedKeys:   []string{"rule_type"},
			unexpectedKeys: []string{"cidr"},
		},
		{
			name: "CIDR changed",
			oldRule: &models.ACLIPRule{
				RuleType: "allow",
				CIDR:     "192.168.1.0/24",
			},
			newRule: &models.ACLIPRule{
				RuleType: "allow",
				CIDR:     "10.0.0.0/16",
			},
			expectedKeys:   []string{"cidr"},
			unexpectedKeys: []string{"rule_type"},
		},
		{
			name: "description changed",
			oldRule: &models.ACLIPRule{
				CIDR:        "192.168.1.0/24",
				Description: strPtr("Old description"),
			},
			newRule: &models.ACLIPRule{
				CIDR:        "192.168.1.0/24",
				Description: strPtr("New description"),
			},
			expectedKeys: []string{"description"},
		},
		{
			name: "priority changed",
			oldRule: &models.ACLIPRule{
				CIDR:     "192.168.1.0/24",
				Priority: 10,
			},
			newRule: &models.ACLIPRule{
				CIDR:     "192.168.1.0/24",
				Priority: 20,
			},
			expectedKeys: []string{"priority"},
		},
		{
			name: "description nil to value",
			oldRule: &models.ACLIPRule{
				CIDR:        "192.168.1.0/24",
				Description: nil,
			},
			newRule: &models.ACLIPRule{
				CIDR:        "192.168.1.0/24",
				Description: strPtr("New description"),
			},
			expectedKeys: []string{"description"},
		},
		{
			name: "multiple changes",
			oldRule: &models.ACLIPRule{
				RuleType:    "allow",
				CIDR:        "192.168.1.0/24",
				Description: strPtr("Old rule"),
				Priority:    10,
			},
			newRule: &models.ACLIPRule{
				RuleType:    "deny",
				CIDR:        "10.0.0.0/16",
				Description: strPtr("New rule"),
				Priority:    20,
			},
			expectedKeys: []string{"rule_type", "cidr", "description", "priority"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			changes := buildIPRuleChanges(tc.oldRule, tc.newRule)

			for _, key := range tc.expectedKeys {
				assert.Contains(t, changes, key, "Expected key %s to be present", key)
			}

			for _, key := range tc.unexpectedKeys {
				assert.NotContains(t, changes, key, "Unexpected key %s should not be present", key)
			}
		})
	}
}

// =============================================================================
// TestBuildWaygatesAuthChanges
// =============================================================================

func TestBuildWaygatesAuthChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		oldConfig      *models.ACLWaygatesAuth
		newConfig      *models.ACLWaygatesAuth
		expectedKeys   []string
		unexpectedKeys []string
	}{
		{
			name:      "nil old config - first time configuration",
			oldConfig: nil,
			newConfig: &models.ACLWaygatesAuth{
				Enabled:          true,
				AllowedRoles:     []string{"admin", "user"},
				AllowedDomains:   []string{"example.com"},
				AllowedProviders: []string{"google"},
				AllowedEmails:    []string{"user@example.com"},
				AllowedUsers:     []string{"admin"},
			},
			expectedKeys: []string{"enabled", "allowed_roles", "allowed_domains", "allowed_providers", "allowed_emails", "allowed_users"},
		},
		{
			name: "no changes",
			oldConfig: &models.ACLWaygatesAuth{
				Enabled:          true,
				AllowedRoles:     []string{"admin"},
				AllowedDomains:   []string{"example.com"},
				AllowedProviders: []string{"google"},
				AllowedEmails:    []string{"user@example.com"},
				AllowedUsers:     []string{"admin"},
			},
			newConfig: &models.ACLWaygatesAuth{
				Enabled:          true,
				AllowedRoles:     []string{"admin"},
				AllowedDomains:   []string{"example.com"},
				AllowedProviders: []string{"google"},
				AllowedEmails:    []string{"user@example.com"},
				AllowedUsers:     []string{"admin"},
			},
			expectedKeys:   []string{},
			unexpectedKeys: []string{"enabled", "allowed_roles", "allowed_domains"},
		},
		{
			name: "enabled changed",
			oldConfig: &models.ACLWaygatesAuth{
				Enabled: true,
			},
			newConfig: &models.ACLWaygatesAuth{
				Enabled: false,
			},
			expectedKeys:   []string{"enabled"},
			unexpectedKeys: []string{"allowed_roles"},
		},
		{
			name: "allowed roles changed",
			oldConfig: &models.ACLWaygatesAuth{
				AllowedRoles: []string{"admin"},
			},
			newConfig: &models.ACLWaygatesAuth{
				AllowedRoles: []string{"admin", "user"},
			},
			expectedKeys: []string{"allowed_roles"},
		},
		{
			name: "allowed domains changed",
			oldConfig: &models.ACLWaygatesAuth{
				AllowedDomains: []string{"old.com"},
			},
			newConfig: &models.ACLWaygatesAuth{
				AllowedDomains: []string{"new.com"},
			},
			expectedKeys: []string{"allowed_domains"},
		},
		{
			name: "allowed providers changed",
			oldConfig: &models.ACLWaygatesAuth{
				AllowedProviders: []string{"google"},
			},
			newConfig: &models.ACLWaygatesAuth{
				AllowedProviders: []string{"google", "github"},
			},
			expectedKeys: []string{"allowed_providers"},
		},
		{
			name: "allowed emails changed",
			oldConfig: &models.ACLWaygatesAuth{
				AllowedEmails: []string{"old@example.com"},
			},
			newConfig: &models.ACLWaygatesAuth{
				AllowedEmails: []string{"new@example.com"},
			},
			expectedKeys: []string{"allowed_emails"},
		},
		{
			name: "allowed users changed",
			oldConfig: &models.ACLWaygatesAuth{
				AllowedUsers: []string{"user1"},
			},
			newConfig: &models.ACLWaygatesAuth{
				AllowedUsers: []string{"user1", "user2"},
			},
			expectedKeys: []string{"allowed_users"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			changes := buildWaygatesAuthChanges(tc.oldConfig, tc.newConfig)

			for _, key := range tc.expectedKeys {
				assert.Contains(t, changes, key, "Expected key %s to be present", key)
			}

			for _, key := range tc.unexpectedKeys {
				assert.NotContains(t, changes, key, "Unexpected key %s should not be present", key)
			}
		})
	}
}

// =============================================================================
// TestBuildBrandingChanges
// =============================================================================

func TestBuildBrandingChanges(t *testing.T) {
	t.Parallel()

	// Helper to create string pointer
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name           string
		oldBranding    *models.ACLBranding
		newBranding    *models.ACLBranding
		expectedKeys   []string
		unexpectedKeys []string
	}{
		{
			name:        "nil old branding - first time configuration",
			oldBranding: nil,
			newBranding: &models.ACLBranding{
				Title:           "My App",
				Subtitle:        strPtr("Welcome to my app"),
				LogoURL:         strPtr("https://example.com/logo.png"),
				PrimaryColor:    "#FF5733",
				BackgroundColor: "#FFFFFF",
			},
			expectedKeys: []string{"title", "subtitle", "logo_url", "primary_color", "background_color"},
		},
		{
			name: "no changes",
			oldBranding: &models.ACLBranding{
				Title:    "My App",
				Subtitle: strPtr("Welcome"),
			},
			newBranding: &models.ACLBranding{
				Title:    "My App",
				Subtitle: strPtr("Welcome"),
			},
			expectedKeys:   []string{},
			unexpectedKeys: []string{"title", "subtitle"},
		},
		{
			name: "title changed",
			oldBranding: &models.ACLBranding{
				Title: "Old Title",
			},
			newBranding: &models.ACLBranding{
				Title: "New Title",
			},
			expectedKeys: []string{"title"},
		},
		{
			name: "subtitle changed",
			oldBranding: &models.ACLBranding{
				Subtitle: strPtr("Old subtitle"),
			},
			newBranding: &models.ACLBranding{
				Subtitle: strPtr("New subtitle"),
			},
			expectedKeys: []string{"subtitle"},
		},
		{
			name: "logo URL changed",
			oldBranding: &models.ACLBranding{
				LogoURL: strPtr("https://old.com/logo.png"),
			},
			newBranding: &models.ACLBranding{
				LogoURL: strPtr("https://new.com/logo.png"),
			},
			expectedKeys: []string{"logo_url"},
		},
		{
			name: "primary color changed",
			oldBranding: &models.ACLBranding{
				PrimaryColor: "#000000",
			},
			newBranding: &models.ACLBranding{
				PrimaryColor: "#FFFFFF",
			},
			expectedKeys: []string{"primary_color"},
		},
		{
			name: "background color changed",
			oldBranding: &models.ACLBranding{
				BackgroundColor: "#000000",
			},
			newBranding: &models.ACLBranding{
				BackgroundColor: "#FFFFFF",
			},
			expectedKeys: []string{"background_color"},
		},
		{
			name: "logo URL nil to value",
			oldBranding: &models.ACLBranding{
				LogoURL: nil,
			},
			newBranding: &models.ACLBranding{
				LogoURL: strPtr("https://new.com/logo.png"),
			},
			expectedKeys: []string{"logo_url"},
		},
		{
			name: "multiple changes",
			oldBranding: &models.ACLBranding{
				Title:        "Old Title",
				PrimaryColor: "#000000",
				Subtitle:     strPtr("Old subtitle"),
			},
			newBranding: &models.ACLBranding{
				Title:        "New Title",
				PrimaryColor: "#FFFFFF",
				Subtitle:     strPtr("New subtitle"),
			},
			expectedKeys: []string{"title", "primary_color", "subtitle"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			changes := buildBrandingChanges(tc.oldBranding, tc.newBranding)

			for _, key := range tc.expectedKeys {
				assert.Contains(t, changes, key, "Expected key %s to be present", key)
			}

			for _, key := range tc.unexpectedKeys {
				assert.NotContains(t, changes, key, "Unexpected key %s should not be present", key)
			}
		})
	}
}

// =============================================================================
// TestJoinStrings
// =============================================================================

func TestJoinStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "nil slice",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: "",
		},
		{
			name:     "single element",
			input:    []string{"admin"},
			expected: "admin",
		},
		{
			name:     "multiple elements",
			input:    []string{"admin", "user", "guest"},
			expected: "admin,user,guest",
		},
		{
			name:     "elements with spaces",
			input:    []string{"admin role", "user role"},
			expected: "admin role,user role",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := joinStrings(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// TestValidOAuthProviders
// =============================================================================

func TestValidOAuthProviders(t *testing.T) {
	t.Parallel()

	validProviders := []string{"google", "github", "microsoft", "gitlab"}
	invalidProviders := []string{"facebook", "twitter", "linkedin", "invalid", "", "Google", "GITHUB"}

	for _, provider := range validProviders {
		t.Run("valid_"+provider, func(t *testing.T) {
			t.Parallel()
			assert.True(t, validOAuthProviders[provider], "Provider %s should be valid", provider)
		})
	}

	for _, provider := range invalidProviders {
		t.Run("invalid_"+provider, func(t *testing.T) {
			t.Parallel()
			assert.False(t, validOAuthProviders[provider], "Provider %s should be invalid", provider)
		})
	}
}
