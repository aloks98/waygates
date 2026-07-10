package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/service/mocks"
)

// =============================================================================
// NewProxyGroupHandler
// =============================================================================

func TestNewProxyGroupHandler(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyGroupService{}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	require.NotNil(t, handler)
	assert.Equal(t, mockService, handler.service)
}

// =============================================================================
// DeleteGroup — 409 with member count, 404
// =============================================================================

// RED (pre-implementation): with no error mapping in DeleteGroup, a
// service.ErrGroupHasMembers would fall through to the generic 500 branch and
// this test would see 500, not 409. GREEN below confirms the switch in
// handlers/proxy_group.go maps it to 409 and preserves the member count in
// the response body.
func TestDeleteGroup_MemberCountConflict(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyGroupService{
		DeleteGroupFunc: func(_ int) error {
			return fmt.Errorf("%w: 7 member proxies; reassign or remove them first", service.ErrGroupHasMembers)
		},
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Delete("/api/proxy-groups/{id}", handler.DeleteGroup)

	req := httptest.NewRequest(http.MethodDelete, "/api/proxy-groups/3", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "7", "the 409 must carry the member count so the UI can report it without a second round trip")
}

func TestDeleteGroup_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyGroupService{
		DeleteGroupFunc: func(_ int) error { return service.ErrGroupNotFound },
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Delete("/api/proxy-groups/{id}", handler.DeleteGroup)

	req := httptest.NewRequest(http.MethodDelete, "/api/proxy-groups/999", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteGroup_InvalidID(t *testing.T) {
	t.Parallel()
	handler := NewProxyGroupHandler(&mocks.MockProxyGroupService{}, nil, nil)

	r := chi.NewRouter()
	r.Delete("/api/proxy-groups/{id}", handler.DeleteGroup)

	req := httptest.NewRequest(http.MethodDelete, "/api/proxy-groups/abc", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteGroup_Success_LogsAudit(t *testing.T) {
	t.Parallel()
	auditCalled := false
	mockService := &mocks.MockProxyGroupService{
		DeleteGroupFunc: func(_ int) error { return nil },
	}
	mockAudit := &mocks.MockAuditService{
		LogProxyGroupDeleteFunc: func(_ context.Context, userID int, groupID int, _, _ string) error {
			auditCalled = true
			assert.Equal(t, 123, userID)
			assert.Equal(t, 3, groupID)
			return nil
		},
	}
	handler := NewProxyGroupHandler(mockService, mockAudit, nil)

	r := chi.NewRouter()
	r.Delete("/api/proxy-groups/{id}", handler.DeleteGroup)

	req := requestWithUserID(http.MethodDelete, "/api/proxy-groups/3", nil, "123")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, auditCalled)
}

// =============================================================================
// GetGroup — 404
// =============================================================================

func TestGetGroup_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyGroupService{
		GetGroupByIDFunc: func(_ int) (*models.ProxyGroup, error) { return nil, service.ErrGroupNotFound },
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/proxy-groups/{id}", handler.GetGroup)

	req := httptest.NewRequest(http.MethodGet, "/api/proxy-groups/999", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetGroup_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyGroupService{
		GetGroupByIDFunc: func(id int) (*models.ProxyGroup, error) { return &models.ProxyGroup{ID: id, Name: "internal"}, nil },
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/proxy-groups/{id}", handler.GetGroup)

	req := httptest.NewRequest(http.MethodGet, "/api/proxy-groups/3", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "internal")
}

// =============================================================================
// CreateGroup — 409 on name conflict
// =============================================================================

// RED (pre-implementation): before CreateGroup mapped
// service.ErrGroupNameConflict, this fell through to the 500 branch.
func TestCreateGroup_NameConflict(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyGroupService{
		CreateGroupFunc: func(_ *models.ProxyGroup, _ int) error { return service.ErrGroupNameConflict },
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxy-groups", handler.CreateGroup)

	body, _ := json.Marshal(map[string]string{"name": "taken"})
	req := requestWithUserID(http.MethodPost, "/api/proxy-groups", body, "123")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestCreateGroup_MissingName(t *testing.T) {
	t.Parallel()
	handler := NewProxyGroupHandler(&mocks.MockProxyGroupService{}, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxy-groups", handler.CreateGroup)

	body, _ := json.Marshal(map[string]string{})
	req := requestWithUserID(http.MethodPost, "/api/proxy-groups", body, "123")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateGroup_NoUserID(t *testing.T) {
	t.Parallel()
	handler := NewProxyGroupHandler(&mocks.MockProxyGroupService{}, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxy-groups", handler.CreateGroup)

	body, _ := json.Marshal(map[string]string{"name": "internal"})
	req := httptest.NewRequest(http.MethodPost, "/api/proxy-groups", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateGroup_Success_LogsAudit(t *testing.T) {
	t.Parallel()
	auditCalled := false
	mockService := &mocks.MockProxyGroupService{
		CreateGroupFunc: func(g *models.ProxyGroup, userID int) error {
			g.ID = 5
			assert.Equal(t, 123, userID)
			return nil
		},
	}
	mockAudit := &mocks.MockAuditService{
		LogProxyGroupCreateFunc: func(_ context.Context, userID int, group *models.ProxyGroup, _, _ string) error {
			auditCalled = true
			assert.Equal(t, 123, userID)
			assert.Equal(t, "internal", group.Name)
			return nil
		},
	}
	handler := NewProxyGroupHandler(mockService, mockAudit, nil)

	r := chi.NewRouter()
	r.Post("/api/proxy-groups", handler.CreateGroup)

	body, _ := json.Marshal(map[string]string{"name": "internal"})
	req := requestWithUserID(http.MethodPost, "/api/proxy-groups", body, "123")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.True(t, auditCalled)
}

// =============================================================================
// UpdateGroup — 404, ErrBaseDomainRequiredByMembers -> 409, ErrHostnameConflict
// -> 409 naming the hostname, ErrGroupNameConflict -> 409, and the rehome
// audit trail.
// =============================================================================

func TestUpdateGroup_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyGroupService{
		GetGroupByIDFunc: func(_ int) (*models.ProxyGroup, error) { return nil, service.ErrGroupNotFound },
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxy-groups/{id}", handler.UpdateGroup)

	body, _ := json.Marshal(map[string]string{"name": "internal"})
	req := httptest.NewRequest(http.MethodPut, "/api/proxy-groups/999", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateGroup_BaseDomainRequiredByMembers(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyGroupService{
		GetGroupByIDFunc: func(id int) (*models.ProxyGroup, error) {
			return &models.ProxyGroup{ID: id, Name: "internal", BaseDomain: ptr("acme.in")}, nil
		},
		UpdateGroupFunc: func(_ *models.ProxyGroup) error { return service.ErrBaseDomainRequiredByMembers },
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxy-groups/{id}", handler.UpdateGroup)

	body, _ := json.Marshal(map[string]interface{}{"name": "internal", "base_domain": nil})
	req := httptest.NewRequest(http.MethodPut, "/api/proxy-groups/3", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestUpdateGroup_NameConflict(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyGroupService{
		GetGroupByIDFunc: func(id int) (*models.ProxyGroup, error) { return &models.ProxyGroup{ID: id, Name: "internal"}, nil },
		UpdateGroupFunc:  func(_ *models.ProxyGroup) error { return service.ErrGroupNameConflict },
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxy-groups/{id}", handler.UpdateGroup)

	body, _ := json.Marshal(map[string]string{"name": "taken"})
	req := httptest.NewRequest(http.MethodPut, "/api/proxy-groups/3", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

// RED (pre-implementation): before the repository/service typed the driver
// error, a hostname collision from the re-home transaction surfaced as a raw
// 500, and this assertion on both the status code and the hostname appearing
// in the body would fail.
func TestUpdateGroup_HostnameConflictNamesTheHostname(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyGroupService{
		GetGroupByIDFunc: func(id int) (*models.ProxyGroup, error) {
			return &models.ProxyGroup{ID: id, Name: "internal", BaseDomain: ptr("group.acme.in")}, nil
		},
		UpdateGroupFunc: func(_ *models.ProxyGroup) error {
			return fmt.Errorf("%w: abc.g2.acme.in", service.ErrHostnameConflict)
		},
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxy-groups/{id}", handler.UpdateGroup)

	body, _ := json.Marshal(map[string]interface{}{"name": "internal", "base_domain": "g2.acme.in"})
	req := httptest.NewRequest(http.MethodPut, "/api/proxy-groups/3", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "abc.g2.acme.in")
}

// A base_domain rewrite must log the affected proxy IDs. This test verifies
// UpdateGroup actually looks at which members are label-addressed (only those
// get re-homed) and reports exactly that set.
func TestUpdateGroup_RehomeLogsAffectedProxyIDs(t *testing.T) {
	t.Parallel()
	label1, label2 := "abc", "def"
	mockService := &mocks.MockProxyGroupService{
		GetGroupByIDFunc: func(id int) (*models.ProxyGroup, error) {
			return &models.ProxyGroup{ID: id, Name: "internal", BaseDomain: ptr("group.acme.in")}, nil
		},
		UpdateGroupFunc: func(_ *models.ProxyGroup) error { return nil },
		ListMembersFunc: func(_ int) ([]models.Proxy, error) {
			return []models.Proxy{
				{ID: 10, HostnameLabel: &label1},
				{ID: 11, HostnameLabel: &label2},
				{ID: 12, HostnameLabel: nil}, // not label-addressed: must NOT be reported as rehomed
			}, nil
		},
	}
	var gotOldBase, gotNewBase string
	var gotIDs []int
	mockAudit := &mocks.MockAuditService{
		LogProxyGroupUpdateFunc: func(_ context.Context, _ int, _, _ *models.ProxyGroup, _, _ string) error { return nil },
		LogProxyGroupRehomeFunc: func(_ context.Context, userID, groupID int, oldBase, newBase string, proxyIDs []int, _, _ string) error {
			assert.Equal(t, 123, userID)
			assert.Equal(t, 3, groupID)
			gotOldBase, gotNewBase = oldBase, newBase
			gotIDs = proxyIDs
			return nil
		},
	}
	handler := NewProxyGroupHandler(mockService, mockAudit, nil)

	r := chi.NewRouter()
	r.Put("/api/proxy-groups/{id}", handler.UpdateGroup)

	body, _ := json.Marshal(map[string]interface{}{"name": "internal", "base_domain": "g2.acme.in"})
	req := requestWithUserID(http.MethodPut, "/api/proxy-groups/3", body, "123")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "group.acme.in", gotOldBase)
	assert.Equal(t, "g2.acme.in", gotNewBase)
	assert.ElementsMatch(t, []int{10, 11}, gotIDs, "only label-addressed members are re-homed")
}

// When base_domain doesn't change, no rehome event should be logged.
func TestUpdateGroup_NoRehomeWhenBaseDomainUnchanged(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyGroupService{
		GetGroupByIDFunc: func(id int) (*models.ProxyGroup, error) {
			return &models.ProxyGroup{ID: id, Name: "internal", BaseDomain: ptr("group.acme.in")}, nil
		},
		UpdateGroupFunc: func(_ *models.ProxyGroup) error { return nil },
	}
	rehomeCalled := false
	mockAudit := &mocks.MockAuditService{
		LogProxyGroupUpdateFunc: func(_ context.Context, _ int, _, _ *models.ProxyGroup, _, _ string) error { return nil },
		LogProxyGroupRehomeFunc: func(context.Context, int, int, string, string, []int, string, string) error {
			rehomeCalled = true
			return nil
		},
	}
	handler := NewProxyGroupHandler(mockService, mockAudit, nil)

	r := chi.NewRouter()
	r.Put("/api/proxy-groups/{id}", handler.UpdateGroup)

	body, _ := json.Marshal(map[string]interface{}{"name": "renamed", "base_domain": "group.acme.in"})
	req := requestWithUserID(http.MethodPut, "/api/proxy-groups/3", body, "123")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, rehomeCalled, "an update that doesn't touch base_domain must not log a rehome event")
}

// =============================================================================
// ACL assignment routes
// =============================================================================

func TestGetGroupACL_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyGroupService{
		ListACLAssignmentsFunc: func(_ int) ([]models.ProxyGroupACLAssignment, error) { return nil, service.ErrGroupNotFound },
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/proxy-groups/{id}/acl", handler.GetGroupACL)

	req := httptest.NewRequest(http.MethodGet, "/api/proxy-groups/999/acl", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAssignACLToGroup_InvalidACLGroupID(t *testing.T) {
	t.Parallel()
	handler := NewProxyGroupHandler(&mocks.MockProxyGroupService{}, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxy-groups/{id}/acl", handler.AssignACLToGroup)

	body, _ := json.Marshal(map[string]int{"acl_group_id": 0})
	req := httptest.NewRequest(http.MethodPost, "/api/proxy-groups/3/acl", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAssignACLToGroup_GroupNotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyGroupService{
		AssignACLToGroupFunc: func(_, _ int, _ string, _ int, _ bool) error { return service.ErrGroupNotFound },
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxy-groups/{id}/acl", handler.AssignACLToGroup)

	body, _ := json.Marshal(map[string]int{"acl_group_id": 5})
	req := httptest.NewRequest(http.MethodPost, "/api/proxy-groups/999/acl", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAssignACLToGroup_Conflict(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyGroupService{
		AssignACLToGroupFunc: func(_, _ int, _ string, _ int, _ bool) error { return service.ErrGroupACLAssignmentExists },
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxy-groups/{id}/acl", handler.AssignACLToGroup)

	body, _ := json.Marshal(map[string]int{"acl_group_id": 5})
	req := httptest.NewRequest(http.MethodPost, "/api/proxy-groups/3/acl", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestAssignACLToGroup_Success_LogsAuditAndReturnsList(t *testing.T) {
	t.Parallel()
	auditCalled := false
	mockService := &mocks.MockProxyGroupService{
		AssignACLToGroupFunc: func(_, _ int, _ string, _ int, _ bool) error { return nil },
		ListACLAssignmentsFunc: func(_ int) ([]models.ProxyGroupACLAssignment, error) {
			return []models.ProxyGroupACLAssignment{{ID: 1, ACLGroupID: 5}}, nil
		},
	}
	mockAudit := &mocks.MockAuditService{
		LogEventFunc: func(_ context.Context, e models.AuditEvent) error {
			auditCalled = true
			assert.Equal(t, models.AuditActionProxyGroupACLAssign, e.Action)
			return nil
		},
	}
	handler := NewProxyGroupHandler(mockService, mockAudit, nil)

	r := chi.NewRouter()
	r.Post("/api/proxy-groups/{id}/acl", handler.AssignACLToGroup)

	body, _ := json.Marshal(map[string]int{"acl_group_id": 5})
	req := requestWithUserID(http.MethodPost, "/api/proxy-groups/3/acl", body, "123")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.True(t, auditCalled)
	assert.True(t, strings.Contains(rec.Body.String(), `"acl_group_id":5`))
}

// TestAssignACLToGroup_EnabledFalse mirrors the proxy-side handler test
// (proxy_acl_handler_integration_test.go): an explicit "enabled": false must
// reach the service, not be dropped by the request struct.
func TestAssignACLToGroup_EnabledFalse(t *testing.T) {
	t.Parallel()
	var capturedEnabled bool
	mockService := &mocks.MockProxyGroupService{
		AssignACLToGroupFunc: func(_, _ int, _ string, _ int, enabled bool) error {
			capturedEnabled = enabled
			return nil
		},
		ListACLAssignmentsFunc: func(_ int) ([]models.ProxyGroupACLAssignment, error) {
			return []models.ProxyGroupACLAssignment{}, nil
		},
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxy-groups/{id}/acl", handler.AssignACLToGroup)

	body, _ := json.Marshal(map[string]interface{}{"acl_group_id": 5, "enabled": false})
	req := httptest.NewRequest(http.MethodPost, "/api/proxy-groups/3/acl", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.False(t, capturedEnabled)
}

// TestAssignACLToGroup_EnabledOmittedDefaultsTrue proves an omitted "enabled"
// field defaults to true, preserving pre-existing behavior.
func TestAssignACLToGroup_EnabledOmittedDefaultsTrue(t *testing.T) {
	t.Parallel()
	var capturedEnabled bool
	mockService := &mocks.MockProxyGroupService{
		AssignACLToGroupFunc: func(_, _ int, _ string, _ int, enabled bool) error {
			capturedEnabled = enabled
			return nil
		},
		ListACLAssignmentsFunc: func(_ int) ([]models.ProxyGroupACLAssignment, error) {
			return []models.ProxyGroupACLAssignment{}, nil
		},
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxy-groups/{id}/acl", handler.AssignACLToGroup)

	body, _ := json.Marshal(map[string]int{"acl_group_id": 5})
	req := httptest.NewRequest(http.MethodPost, "/api/proxy-groups/3/acl", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.True(t, capturedEnabled)
}

func TestUpdateGroupACLAssignment_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyGroupService{
		UpdateGroupACLAssignmentFunc: func(_, _ int, _ string, _ int, _ bool) error {
			return service.ErrGroupACLAssignmentNotFound
		},
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxy-groups/{id}/acl/{assignmentId}", handler.UpdateGroupACLAssignment)

	body, _ := json.Marshal(map[string]interface{}{"path_pattern": "/*", "priority": 0, "enabled": true})
	req := httptest.NewRequest(http.MethodPut, "/api/proxy-groups/3/acl/999", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateGroupACLAssignment_Success_LogsAudit(t *testing.T) {
	t.Parallel()
	auditCalled := false
	mockService := &mocks.MockProxyGroupService{
		UpdateGroupACLAssignmentFunc: func(_, _ int, _ string, _ int, _ bool) error { return nil },
	}
	mockAudit := &mocks.MockAuditService{
		LogEventFunc: func(_ context.Context, e models.AuditEvent) error {
			auditCalled = true
			assert.Equal(t, models.AuditActionProxyGroupACLUpdate, e.Action)
			return nil
		},
	}
	handler := NewProxyGroupHandler(mockService, mockAudit, nil)

	r := chi.NewRouter()
	r.Put("/api/proxy-groups/{id}/acl/{assignmentId}", handler.UpdateGroupACLAssignment)

	body, _ := json.Marshal(map[string]interface{}{"path_pattern": "/admin/*", "priority": 1, "enabled": false})
	req := requestWithUserID(http.MethodPut, "/api/proxy-groups/3/acl/5", body, "123")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, auditCalled)
}

func TestRemoveACLFromGroup_Success_LogsAudit(t *testing.T) {
	t.Parallel()
	auditCalled := false
	mockService := &mocks.MockProxyGroupService{
		RemoveACLFromGroupFunc: func(_, _ int) error { return nil },
	}
	mockAudit := &mocks.MockAuditService{
		LogEventFunc: func(_ context.Context, e models.AuditEvent) error {
			auditCalled = true
			assert.Equal(t, models.AuditActionProxyGroupACLRemove, e.Action)
			return nil
		},
	}
	handler := NewProxyGroupHandler(mockService, mockAudit, nil)

	r := chi.NewRouter()
	r.Delete("/api/proxy-groups/{id}/acl/{aclGroupId}", handler.RemoveACLFromGroup)

	req := requestWithUserID(http.MethodDelete, "/api/proxy-groups/3/acl/5", nil, "123")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, auditCalled)
}

// =============================================================================
// ListGroups — query param wiring
// =============================================================================

func TestListGroups_WithQueryParams(t *testing.T) {
	t.Parallel()
	var gotParams repository.ProxyGroupListParams
	mockService := &mocks.MockProxyGroupService{
		ListGroupsFunc: func(params repository.ProxyGroupListParams) (*models.ProxyGroupListResponse, error) {
			gotParams = params
			return &models.ProxyGroupListResponse{Items: []models.ProxyGroup{}, Total: 0}, nil
		},
	}
	handler := NewProxyGroupHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxy-groups?page=2&limit=10&search=int&sort=name&order=asc", nil)
	rec := httptest.NewRecorder()
	handler.ListGroups(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 2, gotParams.Page)
	assert.Equal(t, 10, gotParams.Limit)
	assert.Equal(t, "int", gotParams.Search)
	assert.Equal(t, "name", gotParams.Sort)
	assert.Equal(t, "asc", gotParams.Order)
}
